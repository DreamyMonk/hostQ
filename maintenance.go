package main

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// This file implements the operator-facing recovery/maintenance CLI:
//
//   hostq doctor    detect + restore zero-byte/missing vhosts, validate, reload
//   hostq repair    rebuild every vhost from metadata (+ restore any legacy
//                   ones with no metadata), validate, reload
//   hostq rebuild   regenerate every vhost from /etc/hostq/sites/*.json
//   hostq validate  nginx -t (+ php-fpm test) without changing anything
//   hostq status    one-shot health summary
//
// The mutating commands (doctor/repair/rebuild) run under a file lock so two
// invocations — or an invocation racing an SSL renewal — can't regenerate
// Nginx at the same time.

// runRepair is the strongest recovery path: it regenerates every site from its
// authoritative JSON metadata, falls back to the good-config backup store for
// any legacy site that has no metadata yet, then validates and reloads Nginx.
func (a *App) runRepair() DoctorResult {
	res := DoctorResult{}
	a.cleanupStaleTemps()
	_ = os.MkdirAll(a.nginxConfBackupDir(), 0700)

	// Primary: rebuild from source of truth. writeNginxSite validates + reloads
	// each site individually and rolls back any that fail.
	for _, r := range a.rebuildFromMetadata() {
		if r.OK {
			res.Restored = append(res.Restored, r.Domain)
		} else {
			res.Notes = append(res.Notes, fmt.Sprintf("%s: %s", r.Domain, r.Err))
		}
	}

	// Fallback: any config still zero-byte/missing with no metadata — restore
	// its last known-good copy from the backup store.
	fixable, unbacked := a.scanBrokenConfigs()
	res.Missing = unbacked
	if len(fixable) > 0 {
		res.Restored = append(res.Restored, a.restoreConfigs(fixable)...)
	}

	out, err := exec.Command("nginx", "-t").CombinedOutput()
	res.NginxTest = strings.TrimSpace(tail(string(out), 600))
	res.NginxOK = err == nil
	if !res.NginxOK {
		res.Notes = append(res.Notes, "nginx -t failed — Nginx left untouched to preserve the running state")
		return res
	}
	if isNginxActive() {
		res.Reloaded = exec.Command("systemctl", "reload", "nginx").Run() == nil
	} else if exec.Command("systemctl", "start", "nginx").Run() == nil {
		res.Reloaded = true
		res.Notes = append(res.Notes, "nginx was down — started it")
	} else {
		res.Notes = append(res.Notes, "nginx -t passed but nginx would not start; check nothing else holds ports 80/443 (e.g. Apache)")
	}
	return res
}

// runRebuild regenerates every vhost from metadata and reports per-site
// results. It is idempotent: repeated runs produce identical configs.
func (a *App) runRebuild() []RebuildResult {
	a.cleanupStaleTemps()
	return a.rebuildFromMetadata()
}

// ValidateResult holds the outcome of a read-only validation pass.
type ValidateResult struct {
	NginxOK   bool
	NginxOut  string
	PHPChecks []string // e.g. "php8.4-fpm: config OK"
	EmptyConf []string // managed vhosts that are zero-byte right now
	AllOK     bool
}

// runValidate checks Nginx syntax, each installed PHP-FPM pool config, and
// flags any zero-byte managed vhost — without modifying anything.
func (a *App) runValidate() ValidateResult {
	res := ValidateResult{}
	out, err := exec.Command("nginx", "-t").CombinedOutput()
	res.NginxOK = err == nil
	res.NginxOut = strings.TrimSpace(tail(string(out), 600))

	for _, v := range []string{"8.2", "8.3", "8.4", "8.5"} {
		bin := "php-fpm" + v
		if _, err := exec.LookPath(bin); err != nil {
			continue
		}
		if out, err := exec.Command(bin, "-t").CombinedOutput(); err != nil {
			res.PHPChecks = append(res.PHPChecks, fmt.Sprintf("%s: FAILED — %s", bin, strings.TrimSpace(tail(string(out), 200))))
		} else {
			res.PHPChecks = append(res.PHPChecks, bin+": config OK")
		}
	}

	if entries, err := os.ReadDir(a.cfg.NginxSitesDir); err == nil {
		for _, e := range entries {
			if e.IsDir() {
				continue
			}
			data, _ := os.ReadFile(filepath.Join(a.cfg.NginxSitesDir, e.Name()))
			if isBlankConf(data) {
				res.EmptyConf = append(res.EmptyConf, e.Name())
			}
		}
	}

	res.AllOK = res.NginxOK && len(res.EmptyConf) == 0
	for _, c := range res.PHPChecks {
		if strings.Contains(c, "FAILED") {
			res.AllOK = false
		}
	}
	return res
}

// StatusReport is a one-shot health snapshot for `hostq status`.
type StatusReport struct {
	NginxActive  bool
	NginxListen  map[int]bool // 80, 443
	PanelActive  bool
	PanelPort    bool
	PHPFPM       []string // active php-fpm units
	Sites        int
	Certs        int
	EmptyConfigs int
	NginxSyntax  bool
	Healthy      bool
	DeployLog    string
}

func (a *App) runStatus() StatusReport {
	s := StatusReport{NginxListen: map[int]bool{}}
	s.NginxActive = isNginxActive()
	s.NginxListen[80] = portListening(80)
	s.NginxListen[443] = portListening(443)
	s.PanelActive = systemdActive("hostq-panel")
	s.PanelPort = portListening(8090)
	for _, v := range []string{"8.2", "8.3", "8.4", "8.5"} {
		if systemdActive("php" + v + "-fpm") {
			s.PHPFPM = append(s.PHPFPM, "php"+v+"-fpm")
		}
	}
	s.Sites = len(a.listSiteMeta())
	if s.Sites == 0 {
		s.Sites = len(a.listSites())
	}
	s.Certs = len(a.listCertificates())
	v := a.runValidate()
	s.NginxSyntax = v.NginxOK
	s.EmptyConfigs = len(v.EmptyConf)

	s.DeployLog = a.deployLogPath()
	s.Healthy = s.NginxActive && s.NginxListen[80] && s.PanelPort && s.NginxSyntax && s.EmptyConfigs == 0
	return s
}

// portListening reports whether something accepts TCP on the given local port.
func portListening(port int) bool {
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", port), 400_000_000)
	if err != nil {
		return false
	}
	_ = conn.Close()
	return true
}

// ---- CLI printers -------------------------------------------------------

func printValidateResult(v ValidateResult) {
	fmt.Println("hostQ validate")
	if v.NginxOK {
		fmt.Println("  nginx -t:  OK")
	} else {
		fmt.Printf("  nginx -t:  FAILED\n%s\n", indentLines(v.NginxOut, "    "))
	}
	for _, c := range v.PHPChecks {
		fmt.Printf("  php-fpm:   %s\n", c)
	}
	if len(v.EmptyConf) > 0 {
		fmt.Printf("  empty configs: %s\n", strings.Join(v.EmptyConf, ", "))
	}
	if v.AllOK {
		fmt.Println("  result:    healthy")
	} else {
		fmt.Println("  result:    problems found")
	}
}

func printRebuildResults(rs []RebuildResult) {
	fmt.Println("hostQ rebuild — regenerating vhosts from /etc/hostq/sites/*.json")
	if len(rs) == 0 {
		fmt.Println("  no site metadata found (nothing to rebuild)")
		return
	}
	for _, r := range rs {
		if r.OK {
			fmt.Printf("  ✓ %s\n", r.Domain)
		} else {
			fmt.Printf("  ✗ %s — %s\n", r.Domain, r.Err)
		}
	}
}

func printStatus(s StatusReport) {
	yn := func(b bool) string {
		if b {
			return "OK"
		}
		return "DOWN"
	}
	fmt.Println("hostQ status")
	fmt.Printf("  nginx          %s (:80 %s, :443 %s)\n", yn(s.NginxActive), yn(s.NginxListen[80]), yn(s.NginxListen[443]))
	fmt.Printf("  panel          %s (:8090 %s)\n", yn(s.PanelActive || s.PanelPort), yn(s.PanelPort))
	if len(s.PHPFPM) > 0 {
		fmt.Printf("  php-fpm        %s\n", strings.Join(s.PHPFPM, ", "))
	} else {
		fmt.Println("  php-fpm        none active")
	}
	fmt.Printf("  nginx syntax   %s\n", yn(s.NginxSyntax))
	fmt.Printf("  sites          %d\n", s.Sites)
	fmt.Printf("  certificates   %d\n", s.Certs)
	if s.EmptyConfigs > 0 {
		fmt.Printf("  empty configs  %d  (run: hostq repair)\n", s.EmptyConfigs)
	}
	if s.Healthy {
		fmt.Println("  health         Healthy")
	} else {
		fmt.Println("  health         Degraded")
	}
	fmt.Printf("  deploy log     %s\n", s.DeployLog)
}
