package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

// runRestoreCLI dispatches the `hostq restore ...` command:
//
//	hostq restore                  restore every domain from its latest good backup
//	hostq restore <domain>         restore one domain from its latest good backup
//	hostq restore <domain> list    list that domain's saved revisions
//	hostq restore <domain> #N      restore a specific revision (N from `list`)
//
// It exits non-zero when the resulting config fails validation.
func (a *App) runRestoreCLI(args []string) {
	// No domain → restore everything from latest good.
	if len(args) == 0 {
		var res RestoreResult
		a.withConfigLock(func() { res = a.runRestore("") })
		printRestoreResult(res)
		if !res.NginxOK {
			os.Exit(1)
		}
		return
	}
	domain := args[0]
	if !domainRe.MatchString(domain) {
		fmt.Printf("invalid domain %q\n", domain)
		os.Exit(1)
	}

	// `list` sub-verb — show revision history, change nothing.
	if len(args) >= 2 && args[1] == "list" {
		a.printConfRevisions(domain)
		return
	}

	// `#N` / `N` — restore a specific historical revision.
	if len(args) >= 2 {
		n, err := strconv.Atoi(strings.TrimPrefix(args[1], "#"))
		if err != nil {
			fmt.Println("usage: hostq restore <domain> [list|#N]")
			os.Exit(1)
		}
		rev, ok := a.revisionByIndex(domain, n)
		if !ok {
			fmt.Printf("no revision #%d for %s (try: hostq restore %s list)\n", n, domain, domain)
			os.Exit(1)
		}
		var res RestoreResult
		a.withConfigLock(func() {
			if a.restoreRevision(domain, rev) {
				res.Restored = []string{domain}
			} else {
				res.Skipped = []string{domain}
			}
			a.finishRestore(&res)
		})
		printRestoreResult(res)
		if !res.NginxOK {
			os.Exit(1)
		}
		return
	}

	// Just a domain → restore its latest good backup.
	var res RestoreResult
	a.withConfigLock(func() { res = a.runRestore(domain) })
	printRestoreResult(res)
	if !res.NginxOK {
		os.Exit(1)
	}
}

// Rolling configuration history + explicit backup/restore. The backup store
// (nginxConfBackupDir) already keeps the single latest known-good copy of each
// vhost as a flat file <domain>; this file adds a per-domain revision history
// under <domain>.history/ so an operator can roll back to an earlier good
// config, not just the most recent one.

// maxConfRevisions bounds the per-site history so it never grows unbounded.
const maxConfRevisions = 20

// confHistoryDir is the per-domain revision directory inside the backup store.
func (a *App) confHistoryDir(domain string) string {
	return filepath.Join(a.nginxConfBackupDir(), domain+".history")
}

// saveConfRevision appends a timestamped revision for a validated-good config,
// skipping the write if it is identical to the newest revision (so idempotent
// rebuilds don't churn the history), then prunes to maxConfRevisions.
func (a *App) saveConfRevision(domain string, conf []byte) {
	if !domainRe.MatchString(domain) || isBlankConf(conf) {
		return
	}
	dir := a.confHistoryDir(domain)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return
	}
	revs := a.listConfRevisions(domain)
	if len(revs) > 0 {
		if data, err := os.ReadFile(filepath.Join(dir, revs[len(revs)-1])); err == nil &&
			bytes.Equal(bytes.TrimSpace(data), bytes.TrimSpace(conf)) {
			return // unchanged since last revision
		}
	}
	// Name = zero-padded monotonic sequence + human timestamp. The sequence
	// prefix guarantees stable chronological ordering and unique filenames even
	// when the wall clock is coarse or steps backward — a bare timestamp could
	// collide and silently overwrite a revision.
	seq := 1
	for _, n := range revs {
		if dash := strings.IndexByte(n, '-'); dash > 0 {
			if v, err := strconv.Atoi(n[:dash]); err == nil && v >= seq {
				seq = v + 1
			}
		}
	}
	name := fmt.Sprintf("%06d-%s.conf", seq, time.Now().UTC().Format("20060102T150405"))
	if err := writeFileAtomic(filepath.Join(dir, name), conf, 0600); err != nil {
		return
	}
	a.pruneConfRevisions(domain)
}

// listConfRevisions returns revision filenames oldest-first (lexical == time).
func (a *App) listConfRevisions(domain string) []string {
	entries, err := os.ReadDir(a.confHistoryDir(domain))
	if err != nil {
		return nil
	}
	names := []string{}
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".conf") {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)
	return names
}

// pruneConfRevisions removes the oldest revisions beyond maxConfRevisions.
func (a *App) pruneConfRevisions(domain string) {
	names := a.listConfRevisions(domain)
	if len(names) <= maxConfRevisions {
		return
	}
	for _, old := range names[:len(names)-maxConfRevisions] {
		_ = os.Remove(filepath.Join(a.confHistoryDir(domain), old))
	}
}

// ---- hostq backup -------------------------------------------------------

// runBackup snapshots every currently-valid managed vhost into the store as a
// fresh revision (and updates the latest-good flat copy). Returns the domains
// captured.
func (a *App) runBackup() []string {
	a.cleanupStaleTemps()
	saved := []string{}
	entries, err := os.ReadDir(a.cfg.NginxSitesDir)
	if err != nil {
		return saved
	}
	for _, e := range entries {
		if e.IsDir() || isSidecarName(e.Name()) {
			continue
		}
		domain := e.Name()
		data, err := os.ReadFile(filepath.Join(a.cfg.NginxSitesDir, domain))
		if err != nil || isBlankConf(data) || !bytes.Contains(data, []byte("hostQ managed")) {
			continue
		}
		a.saveGoodConf(domain, data) // updates latest flat copy + revision
		a.deployLog(domain, "backup", "ok", "")
		saved = append(saved, domain)
	}
	sort.Strings(saved)
	return saved
}

// ---- hostq restore ------------------------------------------------------

// RestoreResult summarises a restore operation for CLI output.
type RestoreResult struct {
	Restored  []string
	Skipped   []string // domains with no backup available
	NginxOK   bool
	NginxTest string
	Reloaded  bool
	Notes     []string
}

// restoreLatest writes the latest-good backup for a domain over the live file.
func (a *App) restoreLatest(domain string) bool {
	backup, err := os.ReadFile(filepath.Join(a.nginxConfBackupDir(), domain))
	if err != nil || isBlankConf(backup) {
		return false
	}
	if err := writeFileAtomic(filepath.Join(a.cfg.NginxSitesDir, domain), backup, 0644); err != nil {
		return false
	}
	a.ensureEnabledLink(domain)
	a.deployLog(domain, "restore", "ok", "latest good backup")
	return true
}

// restoreRevision writes a specific historical revision over the live file.
func (a *App) restoreRevision(domain, revName string) bool {
	data, err := os.ReadFile(filepath.Join(a.confHistoryDir(domain), revName))
	if err != nil || isBlankConf(data) {
		return false
	}
	if err := writeFileAtomic(filepath.Join(a.cfg.NginxSitesDir, domain), data, 0644); err != nil {
		return false
	}
	a.ensureEnabledLink(domain)
	a.deployLog(domain, "restore", "ok", "revision "+revName)
	return true
}

// runRestore restores one domain (latest good) or, when domain is empty, every
// domain in the backup store. It then validates and reloads Nginx once.
func (a *App) runRestore(domain string) RestoreResult {
	res := RestoreResult{}
	targets := []string{}
	if domain != "" {
		targets = []string{domain}
	} else {
		entries, _ := os.ReadDir(a.nginxConfBackupDir())
		for _, e := range entries {
			if !e.IsDir() && !isSidecarName(e.Name()) {
				targets = append(targets, e.Name())
			}
		}
		sort.Strings(targets)
	}
	for _, d := range targets {
		if a.restoreLatest(d) {
			res.Restored = append(res.Restored, d)
		} else {
			res.Skipped = append(res.Skipped, d)
		}
	}
	a.finishRestore(&res)
	return res
}

// finishRestore validates and reloads/starts Nginx after a restore, never
// reloading onto a config that fails `nginx -t`.
func (a *App) finishRestore(res *RestoreResult) {
	out, err := exec.Command("nginx", "-t").CombinedOutput()
	res.NginxTest = strings.TrimSpace(tail(string(out), 600))
	res.NginxOK = err == nil
	if !res.NginxOK {
		res.Notes = append(res.Notes, "nginx -t failed — not reloading")
		return
	}
	if isNginxActive() {
		res.Reloaded = exec.Command("systemctl", "reload", "nginx").Run() == nil
	} else {
		res.Reloaded = exec.Command("systemctl", "start", "nginx").Run() == nil
	}
}

// ---- CLI printers -------------------------------------------------------

func printBackupResult(saved []string) {
	fmt.Println("hostQ backup — snapshotting managed vhosts")
	if len(saved) == 0 {
		fmt.Println("  no managed vhosts to back up")
		return
	}
	for _, d := range saved {
		fmt.Printf("  ✓ %s\n", d)
	}
	fmt.Printf("  %d config(s) captured (history kept: last %d revisions each)\n", len(saved), maxConfRevisions)
}

func printRestoreResult(res RestoreResult) {
	fmt.Println("hostQ restore")
	for _, d := range res.Restored {
		fmt.Printf("  ✓ restored %s\n", d)
	}
	for _, d := range res.Skipped {
		fmt.Printf("  ✗ %s — no backup available\n", d)
	}
	if res.NginxOK {
		fmt.Println("  nginx -t: OK")
	} else {
		fmt.Printf("  nginx -t: FAILED\n%s\n", indentLines(res.NginxTest, "    "))
	}
	if res.Reloaded {
		fmt.Println("  nginx:    reloaded")
	}
	for _, n := range res.Notes {
		fmt.Printf("  note: %s\n", n)
	}
}

// printConfRevisions lists a domain's revision history newest-first with an
// index usable by `hostq restore <domain> #N`.
func (a *App) printConfRevisions(domain string) {
	names := a.listConfRevisions(domain)
	if len(names) == 0 {
		fmt.Printf("no saved revisions for %s\n", domain)
		return
	}
	fmt.Printf("revision history for %s (newest first):\n", domain)
	for i := len(names) - 1; i >= 0; i-- {
		idx := len(names) - i // 1 = newest
		fmt.Printf("  #%d  %s\n", idx, names[i])
	}
	fmt.Printf("restore one with: hostq restore %s #<N>\n", domain)
}

// revisionByIndex maps a 1-based newest-first index to a revision filename.
func (a *App) revisionByIndex(domain string, idx int) (string, bool) {
	names := a.listConfRevisions(domain)
	if idx < 1 || idx > len(names) {
		return "", false
	}
	return names[len(names)-idx], true
}

// isSidecarName reports whether a file in sites-available is a hostQ sidecar
// (rollback copy, temp file, hidden) rather than a live vhost.
func isSidecarName(name string) bool {
	if strings.HasPrefix(name, ".") {
		return true
	}
	for _, suf := range []string{".prev", ".bak", ".tmp", ".history"} {
		if strings.HasSuffix(name, suf) {
			return true
		}
	}
	return false
}
