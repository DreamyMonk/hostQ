package main

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// MalfixReport is the persistent integrity-check result for a WP install.
type MalfixReport struct {
	Domain        string              `json:"domain"`
	Path          string              `json:"path"`
	When          string              `json:"when"`
	Took          string              `json:"took"`
	CoreOK        bool                `json:"core_ok"`
	CoreFailed    []string            `json:"core_failed"`
	PluginsFailed map[string][]string `json:"plugins_failed"`
	ThemesFailed  map[string][]string `json:"themes_failed"`
	Summary       string              `json:"summary"`
}

var coreFailRe = regexp.MustCompile(`(?i)file (?:doesn't|does not) verify against checksum:\s*(.+)`)
var pluginFailRe = regexp.MustCompile(`(?i)file (?:doesn't|does not) verify against checksum:\s*(.+?)\s*\(plugin:\s*([^)]+)\)`)
var themeFailRe = regexp.MustCompile(`(?i)file (?:doesn't|does not) verify against checksum:\s*(.+?)\s*\(theme:\s*([^)]+)\)`)

func (a *App) runMalfix(site Site) MalfixReport {
	start := time.Now()
	report := MalfixReport{
		Domain:        site.Domain,
		Path:          site.Root,
		When:          start.Format(time.RFC3339),
		PluginsFailed: map[string][]string{},
		ThemesFailed:  map[string][]string{},
	}
	if _, err := os.Stat(filepath.Join(site.Root, "wp-config.php")); err != nil {
		report.Summary = "Not a WordPress install."
		report.Took = time.Since(start).Round(time.Millisecond).String()
		_ = a.saveMalfixReport(report)
		return report
	}

	// Core checksums
	core := wpRun(site.Root, "core", "verify-checksums")
	report.CoreOK = strings.Contains(strings.ToLower(core), "verifies against checksums") &&
		!strings.Contains(strings.ToLower(core), "doesn't verify") &&
		!strings.Contains(strings.ToLower(core), "does not verify")
	for _, line := range strings.Split(core, "\n") {
		// Skip plugin/theme lines (handled below)
		if strings.Contains(line, "(plugin:") || strings.Contains(line, "(theme:") {
			continue
		}
		if m := coreFailRe.FindStringSubmatch(line); m != nil {
			report.CoreFailed = append(report.CoreFailed, strings.TrimSpace(m[1]))
		}
	}

	// Plugins
	pl := wpRun(site.Root, "plugin", "verify-checksums", "--all")
	for _, line := range strings.Split(pl, "\n") {
		if m := pluginFailRe.FindStringSubmatch(line); m != nil {
			slug := strings.TrimSpace(m[2])
			report.PluginsFailed[slug] = append(report.PluginsFailed[slug], strings.TrimSpace(m[1]))
		}
	}

	// Themes (verify-checksums for themes ships in newer wp-cli; tolerate failure)
	th := wpRun(site.Root, "theme", "verify-checksums", "--all")
	for _, line := range strings.Split(th, "\n") {
		if m := themeFailRe.FindStringSubmatch(line); m != nil {
			slug := strings.TrimSpace(m[2])
			report.ThemesFailed[slug] = append(report.ThemesFailed[slug], strings.TrimSpace(m[1]))
		}
	}

	report.Summary = malfixSummary(report)
	report.Took = time.Since(start).Round(time.Millisecond).String()
	_ = a.saveMalfixReport(report)
	return report
}

func malfixSummary(r MalfixReport) string {
	parts := []string{}
	if !r.CoreOK || len(r.CoreFailed) > 0 {
		if len(r.CoreFailed) > 0 {
			parts = append(parts, itoa(len(r.CoreFailed))+" core file(s) altered")
		} else {
			parts = append(parts, "core integrity unverified")
		}
	} else {
		parts = append(parts, "core OK")
	}
	if n := len(r.PluginsFailed); n > 0 {
		parts = append(parts, itoa(n)+" plugin(s) altered")
	}
	if n := len(r.ThemesFailed); n > 0 {
		parts = append(parts, itoa(n)+" theme(s) altered")
	}
	if len(parts) == 0 {
		return "all good"
	}
	return strings.Join(parts, " · ")
}

func (a *App) malfixReportPath(domain string) string {
	return filepath.Join(a.cfg.DataDir, "malfix", domain+".json")
}

func (a *App) saveMalfixReport(report MalfixReport) error {
	dir := filepath.Join(a.cfg.DataDir, "malfix")
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	data, _ := json.MarshalIndent(report, "", "  ")
	return os.WriteFile(a.malfixReportPath(report.Domain), data, 0600)
}

func (a *App) loadMalfixReport(domain string) (*MalfixReport, error) {
	data, err := os.ReadFile(a.malfixReportPath(domain))
	if err != nil {
		return nil, err
	}
	var r MalfixReport
	if err := json.Unmarshal(data, &r); err != nil {
		return nil, err
	}
	if r.PluginsFailed == nil {
		r.PluginsFailed = map[string][]string{}
	}
	if r.ThemesFailed == nil {
		r.ThemesFailed = map[string][]string{}
	}
	return &r, nil
}

// repairCore re-downloads WordPress core over the install, skipping wp-content
// (so themes/plugins/uploads stay). wp-config.php is not part of the core zip
// so it is preserved automatically.
func (a *App) repairCore(site Site) string {
	out := wpRun(site.Root, "core", "download", "--force", "--skip-content")
	return out
}

func (a *App) repairPlugin(site Site, slug string) string {
	if !pluginSlugRe.MatchString(slug) {
		return "Invalid plugin slug."
	}
	return wpRun(site.Root, "plugin", "install", slug, "--force")
}

func (a *App) repairTheme(site Site, slug string) string {
	if !pluginSlugRe.MatchString(slug) {
		return "Invalid theme slug."
	}
	return wpRun(site.Root, "theme", "install", slug, "--force")
}

var pluginSlugRe = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,63}$`)

// Malfix HTTP handler
func (a *App) malfix(w http.ResponseWriter, r *http.Request) {
	domain := strings.ToLower(strings.TrimSpace(r.FormValue("domain")))
	site, ok := a.findSite(domain)
	if !ok {
		http.Redirect(w, r, "/sites", http.StatusSeeOther)
		return
	}
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/site?domain="+domain+"&tab=wordpress", http.StatusSeeOther)
		return
	}
	action := r.FormValue("action")
	slug := strings.ToLower(strings.TrimSpace(r.FormValue("slug")))
	output := ""
	auditTarget := domain
	switch action {
	case "scan":
		report := a.runMalfix(site)
		output = "Integrity scan: " + report.Summary + " · " + report.Took
		a.audit("malfix.scan", "success", domain)
	case "repair-core":
		output = "Core repair:\n" + a.repairCore(site)
		a.audit("malfix.repair-core", "success", domain)
		// re-scan to refresh state
		_ = a.runMalfix(site)
	case "repair-plugin":
		output = "Plugin " + slug + " repair:\n" + a.repairPlugin(site, slug)
		auditTarget = domain + "/" + slug
		a.audit("malfix.repair-plugin", "success", auditTarget)
		_ = a.runMalfix(site)
	case "repair-theme":
		output = "Theme " + slug + " repair:\n" + a.repairTheme(site, slug)
		auditTarget = domain + "/" + slug
		a.audit("malfix.repair-theme", "success", auditTarget)
		_ = a.runMalfix(site)
	case "repair-all":
		var b strings.Builder
		report, err := a.loadMalfixReport(domain)
		if err != nil || report == nil {
			fresh := a.runMalfix(site)
			report = &fresh
		}
		if !report.CoreOK || len(report.CoreFailed) > 0 {
			b.WriteString("Core:\n" + a.repairCore(site) + "\n")
		}
		for plug := range report.PluginsFailed {
			b.WriteString("Plugin " + plug + ":\n" + a.repairPlugin(site, plug) + "\n")
		}
		for theme := range report.ThemesFailed {
			b.WriteString("Theme " + theme + ":\n" + a.repairTheme(site, theme) + "\n")
		}
		_ = a.runMalfix(site)
		a.audit("malfix.repair-all", "success", domain)
		output = "Repair-all complete.\n" + b.String()
	default:
		output = "Unknown action."
	}
	short := output
	if len(short) > 200 {
		short = short[:200] + "…"
	}
	http.Redirect(w, r, "/site?domain="+domain+"&tab=wordpress&output="+queryEscape(short), http.StatusSeeOther)
}
