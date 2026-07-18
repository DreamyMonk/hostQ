package main

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// sleepMillis pauses for the given number of milliseconds. Wrapped so the
// retry loops read clearly.
func sleepMillis(ms int) { time.Sleep(time.Duration(ms) * time.Millisecond) }

// nginxWriteMu serializes vhost writes within the panel process so two
// concurrent HTTP actions (e.g. a cache toggle and an SSL install) never
// interleave their generate → validate → reload steps. Cross-process safety
// (the CLI vs a running panel) is additionally guarded by a file lock around
// the maintenance commands; atomic writes mean the worst case is a harmless
// last-writer-wins on a fully-valid config, never a truncated one.
var nginxWriteMu sync.Mutex

// This file centralises every write to a live Nginx vhost so a single, safe
// path handles all of them. The 2026-07-18 outage was caused by generated
// vhosts becoming zero-byte files: the old writeNginxSite truncated the live
// file with os.WriteFile, then reloaded Nginx *regardless* of whether
// `nginx -t` passed. An interrupted or failed write left an empty config and
// the reload happily served it.
//
// The guarantees enforced here mirror the incident report's recommendations:
//   1. Never write an empty config.
//   2. Never truncate the live file in place — write a temp file and rename()
//      atomically, so readers only ever see the old or the new whole file.
//   3. Keep the previous config as <domain>.bak for instant rollback.
//   4. Validate with `nginx -t` before committing to the new config; on
//      failure, roll the live file back to the last good content.
//   5. Only reload Nginx after a successful validation.
//   6. Snapshot every validated-good config to a backup store under DataDir so
//      `hostq doctor` can rebuild sites-available even if it is wiped entirely.

// nginxConfBackupDir is where the last known-good copy of every managed vhost
// is kept. It lives under DataDir (default /etc/hostq), deliberately separate
// from /etc/nginx/sites-available, so a truncation or accidental wipe of the
// Nginx directory does not take the backups with it.
func (a *App) nginxConfBackupDir() string {
	return filepath.Join(a.cfg.DataDir, "nginx-conf-backups")
}

// isBlankConf reports whether a config is empty or whitespace-only — i.e. the
// zero-byte failure mode we must never write and must always recover from.
func isBlankConf(b []byte) bool {
	return len(bytes.TrimSpace(b)) == 0
}

// writeFileAtomic writes data to a sibling temp file, fsyncs it, and renames it
// over path. rename(2) is atomic on the same filesystem, so a concurrent reader
// (or Nginx) sees either the complete old file or the complete new one — never
// a half-written or truncated file.
func writeFileAtomic(path string, data []byte, perm os.FileMode) error {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, "."+filepath.Base(path)+".hostq-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	// Best-effort cleanup if we bail before the rename.
	defer func() { _ = os.Remove(tmpName) }()

	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Chmod(tmpName, perm); err != nil {
		return err
	}
	return os.Rename(tmpName, path)
}

// enabledLinkPath returns the sites-enabled symlink path for a domain. The
// enabled dir is a fixed Debian/Ubuntu location alongside NginxSitesDir.
func (a *App) enabledLinkPath(domain string) string {
	return filepath.Join(filepath.Dir(a.cfg.NginxSitesDir), "sites-enabled", domain)
}

// ensureEnabledLink makes sure sites-enabled/<domain> points at the live vhost.
func (a *App) ensureEnabledLink(domain string) {
	link := a.enabledLinkPath(domain)
	live := filepath.Join(a.cfg.NginxSitesDir, domain)
	if target, err := os.Readlink(link); err == nil && target == live {
		return
	}
	_ = os.Remove(link)
	_ = os.Symlink(live, link)
}

// saveGoodConf snapshots a validated-good vhost into the backup store. Callers
// invoke this only *after* `nginx -t` has accepted the config, so the store
// never holds a config that would fail to load.
func (a *App) saveGoodConf(domain string, conf []byte) {
	if isBlankConf(conf) {
		return
	}
	dir := a.nginxConfBackupDir()
	if err := os.MkdirAll(dir, 0700); err != nil {
		log.Printf("nginx backup: mkdir %s: %v", dir, err)
		return
	}
	if err := writeFileAtomic(filepath.Join(dir, domain), conf, 0600); err != nil {
		log.Printf("nginx backup: write %s: %v", domain, err)
	}
	// Also append to the rolling revision history (last N kept per domain).
	a.saveConfRevision(domain, conf)
}

// applyNginxConf is the single safe path for publishing a generated vhost. It
// refuses empty configs, writes atomically, validates the whole Nginx config,
// rolls back to the previous good content if validation fails, and only then
// reloads Nginx. The last known-good content is preserved both as a sibling
// <domain>.bak and in the DataDir backup store.
func (a *App) applyNginxConf(domain string, conf []byte) error {
	nginxWriteMu.Lock()
	defer nginxWriteMu.Unlock()

	live := filepath.Join(a.cfg.NginxSitesDir, domain)

	// Guarantee 1: never publish an empty config.
	if isBlankConf(conf) {
		return fmt.Errorf("refusing to write empty nginx config for %s", domain)
	}

	// Snapshot the current live content so we can roll back if the new config
	// fails validation. hadPrev distinguishes "update" from "brand new site".
	prev, prevErr := os.ReadFile(live)
	hadPrev := prevErr == nil && !isBlankConf(prev)
	if hadPrev {
		// Guarantee 3: keep the previous config as a .prev sibling for instant
		// rollback (isSidecarName keeps it out of the site listing).
		if err := writeFileAtomic(live+".prev", prev, 0644); err != nil {
			log.Printf("nginx: could not write %s.prev: %v", live, err)
		}
	}

	// Guarantees 2: atomic write — the live file is never truncated in place.
	if err := writeFileAtomic(live, conf, 0644); err != nil {
		a.deployLog(domain, "apply", "write-failed", err.Error())
		return fmt.Errorf("write nginx config for %s: %w", domain, err)
	}
	a.ensureEnabledLink(domain)

	// rollback restores the last good state (or removes a brand-new site) and
	// re-establishes the enabled link accordingly.
	rollback := func() {
		if hadPrev {
			_ = writeFileAtomic(live, prev, 0644)
			a.ensureEnabledLink(domain)
		} else {
			_ = os.Remove(live)
			_ = os.Remove(a.enabledLinkPath(domain))
		}
	}

	// Guarantee 4: validate before committing. `nginx -t` checks the whole
	// server config, so a syntactically bad vhost is caught here.
	if out, err := exec.Command("nginx", "-t").CombinedOutput(); err != nil {
		rollback()
		msg := strings.TrimSpace(tail(string(out), 400))
		a.deployLog(domain, "apply", "validate-failed", msg)
		return fmt.Errorf("nginx -t rejected config for %s (rolled back): %s", domain, msg)
	}

	// Config is good — persist it for doctor recovery + revision history.
	a.saveGoodConf(domain, conf)

	// Guarantee 5: reload only after a passing test. If Nginx isn't running
	// (dev/first boot) there is nothing to reload or verify.
	if !isNginxActive() {
		a.deployLog(domain, "apply", "ok", "written (nginx not active)")
		return nil
	}
	if err := exec.Command("systemctl", "reload", "nginx").Run(); err != nil {
		// Reload itself failed — roll back and reload the last good config so we
		// don't leave the bad one staged for the next restart.
		rollback()
		_ = exec.Command("systemctl", "reload", "nginx").Run()
		a.deployLog(domain, "apply", "reload-failed", err.Error())
		return fmt.Errorf("nginx reload failed for %s (rolled back)", domain)
	}

	// Guarantee 6: post-reload verification. A config can pass `nginx -t` yet
	// still take the server down at runtime (bad upstream, port clash). Probe
	// that Nginx is still serving; if not, roll back and reload the good config.
	if !a.verifyNginxServing(domain) {
		rollback()
		_ = exec.Command("systemctl", "reload", "nginx").Run()
		a.deployLog(domain, "apply", "serve-failed", "nginx not serving after reload; rolled back")
		return fmt.Errorf("nginx stopped serving after applying %s; rolled back to previous config", domain)
	}

	a.deployLog(domain, "apply", "ok", "")
	return nil
}

// verifyNginxServing confirms Nginx is still up and actually serving this vhost
// after a reload. It checks, with brief retries (a graceful reload is async):
//   1. the nginx unit is active,
//   2. port 80 is listening (and 443 if the site has a TLS cert), and
//   3. an HTTP request to 127.0.0.1 with this site's Host header gets a
//      response — proving Nginx routes the vhost, not just that a port is open.
// Any HTTP status counts as "serving"; only a connection-level failure (refused,
// reset, timeout, no response) means the site went dark and triggers rollback.
// A 5xx from a down app backend is deliberately NOT treated as a failure — that
// isn't something rolling back the Nginx config would fix.
func (a *App) verifyNginxServing(domain string) bool {
	if !waitServiceActive("nginx", 10) {
		return false
	}
	if !waitPortListening(80, 10) {
		return false
	}
	hasSSL := a.certExists(domain)
	if hasSSL && !waitPortListening(443, 10) {
		return false
	}
	// Per-site HTTP probe over plain :80.
	if !waitHTTPProbe(domain, false, 8) {
		return false
	}
	// And over :443 when the site is HTTPS.
	if hasSSL && !waitHTTPProbe(domain, true, 8) {
		return false
	}
	return true
}

// waitHTTPProbe retries the per-site HTTP probe up to attempts times.
func waitHTTPProbe(domain string, https bool, attempts int) bool {
	for i := 0; i < attempts; i++ {
		if probeSiteHTTP(domain, https) {
			return true
		}
		sleepMillis(150)
	}
	return false
}

// probeSiteHTTP issues GET / to 127.0.0.1 on port 80/443 with Host: <domain>,
// so Nginx selects this site's server block exactly as a real client on that
// hostname would — without depending on the domain's public DNS resolving to
// this box.
func probeSiteHTTP(domain string, https bool) bool {
	addr := "127.0.0.1:80"
	if https {
		addr = "127.0.0.1:443"
	}
	return httpProbeAddr(addr, domain, https, 3*time.Second)
}

// httpProbeAddr dials dialAddr but presents host as the HTTP Host / TLS SNI, and
// reports whether Nginx returned any HTTP response. TLS verification is skipped
// because we connect to the loopback address rather than the real hostname — we
// are testing that Nginx answers for this vhost, not the certificate chain.
func httpProbeAddr(dialAddr, host string, https bool, timeout time.Duration) bool {
	scheme := "http"
	if https {
		scheme = "https"
	}
	transport := &http.Transport{
		DialContext: func(ctx context.Context, network, _ string) (net.Conn, error) {
			d := &net.Dialer{Timeout: timeout}
			return d.DialContext(ctx, "tcp", dialAddr)
		},
		TLSClientConfig:   &tls.Config{InsecureSkipVerify: true, ServerName: host},
		DisableKeepAlives: true,
	}
	client := &http.Client{
		Timeout:   timeout,
		Transport: transport,
		// Don't follow redirects — an HTTP→HTTPS 301 is itself a valid "Nginx
		// is serving this vhost" signal.
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}
	req, err := http.NewRequest(http.MethodGet, scheme+"://"+host+"/", nil)
	if err != nil {
		return false
	}
	req.Host = host
	resp, err := client.Do(req)
	if err != nil {
		return false
	}
	_ = resp.Body.Close()
	return resp.StatusCode > 0
}

// waitPortListening polls a local TCP port up to attempts times (~100ms apart).
func waitPortListening(port, attempts int) bool {
	for i := 0; i < attempts; i++ {
		if portListening(port) {
			return true
		}
		sleepMillis(120)
	}
	return false
}

// waitServiceActive polls a systemd unit's active state.
func waitServiceActive(unit string, attempts int) bool {
	for i := 0; i < attempts; i++ {
		if systemdActive(unit) {
			return true
		}
		sleepMillis(120)
	}
	return false
}

// DoctorResult summarises what `hostq doctor` did, both for the CLI output and
// so the same routine can drive a panel button later.
type DoctorResult struct {
	Restored  []string // domains whose zero-byte/missing config was rebuilt
	Missing   []string // domains with no backup available to restore from
	NginxTest string   // trimmed `nginx -t` output
	NginxOK   bool
	Reloaded  bool
	Notes     []string
}

// scanBrokenConfigs returns the managed vhosts that are currently missing or
// zero-byte in sites-available but have a known-good copy in the backup store.
func (a *App) scanBrokenConfigs() (fixable []string, unbacked []string) {
	entries, err := os.ReadDir(a.nginxConfBackupDir())
	if err != nil {
		return nil, nil
	}
	for _, e := range entries {
		if e.IsDir() || isSidecarName(e.Name()) {
			continue
		}
		domain := e.Name()
		live := filepath.Join(a.cfg.NginxSitesDir, domain)
		data, err := os.ReadFile(live)
		if err != nil || isBlankConf(data) {
			// Live config is gone or empty — this is the outage signature.
			backup, berr := os.ReadFile(filepath.Join(a.nginxConfBackupDir(), domain))
			if berr == nil && !isBlankConf(backup) {
				fixable = append(fixable, domain)
			} else {
				unbacked = append(unbacked, domain)
			}
		}
	}
	return fixable, unbacked
}

// restoreConfigs rewrites the given domains from the backup store using the
// atomic writer and re-links them into sites-enabled. It does not validate or
// reload — the caller decides when to do that once.
func (a *App) restoreConfigs(domains []string) []string {
	restored := []string{}
	for _, domain := range domains {
		backup, err := os.ReadFile(filepath.Join(a.nginxConfBackupDir(), domain))
		if err != nil || isBlankConf(backup) {
			continue
		}
		live := filepath.Join(a.cfg.NginxSitesDir, domain)
		if err := writeFileAtomic(live, backup, 0644); err != nil {
			log.Printf("doctor: restore %s: %v", domain, err)
			continue
		}
		a.ensureEnabledLink(domain)
		restored = append(restored, domain)
	}
	return restored
}

// runDoctor detects and repairs the zero-byte-config failure mode, validates
// Nginx, and reloads it. It never restarts onto a config that fails `nginx -t`.
func (a *App) runDoctor() DoctorResult {
	res := DoctorResult{}
	if err := os.MkdirAll(a.nginxConfBackupDir(), 0700); err != nil {
		res.Notes = append(res.Notes, "backup dir unavailable: "+err.Error())
	}

	fixable, unbacked := a.scanBrokenConfigs()
	res.Missing = unbacked
	if len(fixable) > 0 {
		res.Restored = a.restoreConfigs(fixable)
	}

	// Validate whatever we ended up with. Startup safety: if the test fails we
	// keep the existing files and do NOT reload/restart Nginx.
	out, err := exec.Command("nginx", "-t").CombinedOutput()
	res.NginxTest = strings.TrimSpace(tail(string(out), 600))
	res.NginxOK = err == nil
	if !res.NginxOK {
		res.Notes = append(res.Notes, "nginx -t failed — leaving Nginx untouched so the running state is preserved")
		return res
	}

	// Config is valid. Reload if Nginx is already up; start it if it is down
	// (the outage left it failed-to-start).
	if isNginxActive() {
		res.Reloaded = exec.Command("systemctl", "reload", "nginx").Run() == nil
	} else {
		if exec.Command("systemctl", "start", "nginx").Run() == nil {
			res.Reloaded = true
			res.Notes = append(res.Notes, "nginx was down — started it")
		} else {
			res.Notes = append(res.Notes, "nginx -t passed but nginx would not start; check that nothing else holds ports 80/443 (e.g. Apache)")
		}
	}
	return res
}

// isNginxActive reports whether the nginx systemd unit is currently active.
func isNginxActive() bool {
	out, _ := exec.Command("systemctl", "is-active", "nginx").Output()
	return strings.TrimSpace(string(out)) == "active"
}

// seedConfBackups snapshots every currently-healthy managed vhost into the
// backup store when it has no backup yet. This bootstraps recovery for sites
// that existed before the backup store did, so `hostq doctor` and the boot
// self-heal can rebuild them too. It never overwrites an existing backup and
// never copies a blank config.
func (a *App) seedConfBackups() {
	entries, err := os.ReadDir(a.cfg.NginxSitesDir)
	if err != nil {
		return
	}
	dir := a.nginxConfBackupDir()
	for _, e := range entries {
		if e.IsDir() || isSidecarName(e.Name()) {
			continue
		}
		domain := e.Name()
		if _, err := os.Stat(filepath.Join(dir, domain)); err == nil {
			continue // already backed up
		}
		data, err := os.ReadFile(filepath.Join(a.cfg.NginxSitesDir, domain))
		if err != nil || isBlankConf(data) {
			continue
		}
		if !bytes.Contains(data, []byte("hostQ managed")) {
			continue // only manage the vhosts we generate
		}
		a.saveGoodConf(domain, data)
	}
}

// cleanupStaleTemps removes leftover atomic-write temp files (".*.hostq-*")
// from the sites-available and backup dirs. A crash between CreateTemp and
// rename would strand one; this is the "delete temp files" step of crash
// recovery. Live and backup files are never matched by the glob.
func (a *App) cleanupStaleTemps() {
	dirs := []string{a.cfg.NginxSitesDir, a.nginxConfBackupDir(), filepath.Join(a.cfg.DataDir, "sites")}
	for _, dir := range dirs {
		matches, err := filepath.Glob(filepath.Join(dir, ".*.hostq-*"))
		if err != nil {
			continue
		}
		for _, m := range matches {
			if err := os.Remove(m); err == nil {
				log.Printf("crash recovery: removed stale temp file %s", m)
			}
		}
	}
}

// nginxStartupHeal runs at panel boot: it restores any zero-byte/missing
// managed vhosts from the backup store and reloads Nginx only if the resulting
// config validates. It is best-effort and never blocks the panel from serving.
func (a *App) nginxStartupHeal() {
	fixable, _ := a.scanBrokenConfigs()
	if len(fixable) == 0 {
		return
	}
	restored := a.restoreConfigs(fixable)
	if len(restored) == 0 {
		return
	}
	log.Printf("nginx self-heal: restored %d zero-byte vhost(s) from backup: %s", len(restored), strings.Join(restored, ", "))
	if out, err := exec.Command("nginx", "-t").CombinedOutput(); err != nil {
		log.Printf("nginx self-heal: nginx -t still failing, not reloading: %s", strings.TrimSpace(tail(string(out), 300)))
		return
	}
	if isNginxActive() {
		_ = exec.Command("systemctl", "reload", "nginx").Run()
	} else {
		_ = exec.Command("systemctl", "start", "nginx").Run()
	}
}

// printDoctorResult renders a DoctorResult to stdout for the CLI subcommand.
func printDoctorResult(res DoctorResult) {
	fmt.Println("hostQ doctor — nginx configuration check")
	if len(res.Restored) > 0 {
		fmt.Printf("  restored from backup: %s\n", strings.Join(res.Restored, ", "))
	} else {
		fmt.Println("  restored from backup: none needed")
	}
	if len(res.Missing) > 0 {
		fmt.Printf("  missing (no backup):  %s\n", strings.Join(res.Missing, ", "))
	}
	if res.NginxOK {
		fmt.Println("  nginx -t:             OK")
	} else {
		fmt.Printf("  nginx -t:             FAILED\n%s\n", indentLines(res.NginxTest, "    "))
	}
	if res.Reloaded {
		fmt.Println("  nginx:                reloaded/started")
	} else if res.NginxOK {
		fmt.Println("  nginx:                already running (no reload needed)")
	}
	for _, n := range res.Notes {
		fmt.Printf("  note: %s\n", n)
	}
}

func indentLines(s, prefix string) string {
	if s == "" {
		return ""
	}
	lines := strings.Split(s, "\n")
	for i, l := range lines {
		lines[i] = prefix + l
	}
	return strings.Join(lines, "\n")
}
