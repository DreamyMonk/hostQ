package main

import (
	"html/template"
	"net/http"
	"os"
	"strings"
)

func (a *App) security(w http.ResponseWriter, r *http.Request) {
	domain := strings.ToLower(strings.TrimSpace(r.FormValue("domain")))
	site, ok := a.findSite(domain)
	if !ok {
		http.Redirect(w, r, "/sites", http.StatusSeeOther)
		return
	}
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/site?domain="+domain+"&tab=security", http.StatusSeeOther)
		return
	}
	action := r.FormValue("action")
	output := ""
	switch action {
	case "scan":
		report := a.runSiteScan(site)
		output = "Scan complete: " +
			itoa(report.Critical) + " critical · " +
			itoa(report.High) + " high · " +
			itoa(report.Medium) + " medium · " +
			itoa(report.Scanned) + " files scanned in " + report.Took
		a.audit("security.scan", "success", site.Domain)
	case "quarantine":
		abs := r.FormValue("abs")
		if dst, err := a.quarantineFile(site.Domain, abs); err != nil {
			output = "Quarantine failed: " + err.Error()
			a.audit("security.quarantine", "failure", site.Domain+" "+abs)
		} else {
			output = "Quarantined → " + dst
			a.audit("security.quarantine", "success", abs)
			a.removeFinding(site.Domain, abs)
		}
	case "delete":
		abs := r.FormValue("abs")
		if !a.canMutateWebPath(abs) {
			output = "Refusing to delete: path is not mutable"
		} else if err := os.Remove(abs); err != nil {
			output = "Delete failed: " + err.Error()
			a.audit("security.delete", "failure", abs)
		} else {
			output = "Deleted " + abs
			a.audit("security.delete", "success", abs)
			a.removeFinding(site.Domain, abs)
		}
	}
	http.Redirect(w, r, "/site?domain="+site.Domain+"&tab=security&output="+template.URLQueryEscaper(output), http.StatusSeeOther)
}

// removeFinding drops a single finding from the on-disk scan report so the next
// page render no longer lists a file you already quarantined or deleted.
func (a *App) removeFinding(domain, abs string) {
	report, err := a.loadScanReport(domain)
	if err != nil {
		return
	}
	kept := report.Findings[:0]
	for _, f := range report.Findings {
		if f.AbsPath != abs {
			kept = append(kept, f)
		}
	}
	report.Findings = kept
	report.Critical, report.High, report.Medium, report.Low = 0, 0, 0, 0
	for _, f := range kept {
		switch f.Severity {
		case "critical":
			report.Critical++
		case "high":
			report.High++
		case "medium":
			report.Medium++
		default:
			report.Low++
		}
	}
	_ = a.saveScanReport(*report)
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var out []byte
	for n > 0 {
		out = append([]byte{byte('0' + n%10)}, out...)
		n /= 10
	}
	return string(out)
}
