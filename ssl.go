package main

import (
	"fmt"
	"html/template"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

func (a *App) ssl(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		a.sslAction(w, r)
		return
	}
	a.render(w, "ssl", map[string]any{
		"Title":        "SSL",
		"Certificates": a.listCertificates(),
		"Sites":        a.listSites(),
		"Created":      r.URL.Query().Get("created"),
		"Output":       r.URL.Query().Get("output"),
		"Site":         strings.TrimSpace(r.URL.Query().Get("site")),
	})
}

func (a *App) sslAction(w http.ResponseWriter, r *http.Request) {
	domain := strings.ToLower(strings.TrimSpace(r.FormValue("domain")))
	action := r.FormValue("action")
	if domain != "" && !domainRe.MatchString(domain) {
		http.Redirect(w, r, "/ssl", http.StatusSeeOther)
		return
	}
	var result []byte
	var err error
	switch action {
	case "issue":
		email := strings.TrimSpace(r.FormValue("email"))
		if domain != "" && email != "" {
			repair := a.removeBrokenNginxSSL(domain)
			result, err = exec.Command("certbot", "--nginx", "-d", domain, "--email", email, "--agree-tos", "--non-interactive").CombinedOutput()
			if err == nil {
				a.audit("ssl.issue", "success", domain)
			} else {
				a.audit("ssl.issue", "failure", domain)
			}
			http.Redirect(w, r, "/ssl?created="+domain+"&output="+template.URLQueryEscaper(repair+string(result)), http.StatusSeeOther)
			return
		}
	case "renew":
		if domain != "" {
			result, err = exec.Command("certbot", "renew", "--cert-name", domain, "--non-interactive").CombinedOutput()
		} else {
			result, err = exec.Command("certbot", "renew", "--non-interactive").CombinedOutput()
		}
		status := "failure"
		if err == nil {
			status = "success"
		}
		a.audit("ssl.renew", status, domain)
		http.Redirect(w, r, "/ssl?output="+template.URLQueryEscaper(string(result)), http.StatusSeeOther)
		return
	case "delete":
		if domain != "" {
			result, err = exec.Command("certbot", "delete", "--cert-name", domain, "--non-interactive").CombinedOutput()
			status := "failure"
			if err == nil {
				status = "success"
			}
			a.audit("ssl.delete", status, domain)
			http.Redirect(w, r, "/ssl?output="+template.URLQueryEscaper(string(result)), http.StatusSeeOther)
			return
		}
	}
	http.Redirect(w, r, "/ssl", http.StatusSeeOther)
}

func (a *App) certExists(domain string) bool {
	_, certErr := os.Stat(filepath.Join("/etc/letsencrypt/live", domain, "fullchain.pem"))
	_, keyErr := os.Stat(filepath.Join("/etc/letsencrypt/live", domain, "privkey.pem"))
	return certErr == nil && keyErr == nil
}

func (a *App) removeBrokenNginxSSL(domain string) string {
	if a.certExists(domain) {
		return ""
	}
	site, ok := a.findSite(domain)
	if !ok {
		return ""
	}
	configPath := filepath.Join(a.cfg.NginxSitesDir, domain)
	data, err := os.ReadFile(configPath)
	if err != nil || !strings.Contains(string(data), "/etc/letsencrypt/live/") {
		return ""
	}
	backupPath := fmt.Sprintf("%s.broken-ssl-%d.bak", configPath, time.Now().Unix())
	_ = copyFile(configPath, backupPath)
	a.writeNginxSite(site.Domain, site.Root, site.Cache, site.PHPVersion)
	return fmt.Sprintf("Removed stale SSL references from %s. Backup: %s\n", configPath, backupPath)
}

func (a *App) listCertificates() []CertInfo {
	if v, ok := a.cache.get("certs"); ok {
		return v.([]CertInfo)
	}
	out, _ := exec.Command("certbot", "certificates").CombinedOutput()
	certs := []CertInfo{}
	for _, block := range strings.Split(string(out), "Certificate Name:") {
		block = strings.TrimSpace(block)
		if block == "" || strings.HasPrefix(block, "Saving debug log") {
			continue
		}
		name := strings.Fields(block)
		if len(name) == 0 {
			continue
		}
		expiry := strings.TrimSpace(firstMatch(block, `Expiry Date:\s*([^\n(]+)`))
		days := daysLeftFromCertbot(block, expiry)
		status := certStatus(days)
		certs = append(certs, CertInfo{Domain: name[0], Expiry: expiry, Days: days, Status: status})
	}
	sort.Slice(certs, func(i, j int) bool { return certs[i].Domain < certs[j].Domain })
	a.cache.set("certs", certs, 15*time.Second)
	return certs
}

func daysLeftFromCertbot(block, expiry string) int {
	if match := regexp.MustCompile(`(?i)\((?:VALID:\s*)?(\d+)\s+days?\)`).FindStringSubmatch(block); len(match) > 1 {
		days, _ := strconv.Atoi(match[1])
		return days
	}
	match := regexp.MustCompile(`(\d{4}-\d{2}-\d{2})(?:\s+(\d{2}:\d{2}:\d{2}))?`).FindStringSubmatch(expiry)
	if len(match) < 2 {
		return 0
	}
	stamp := match[1] + "T23:59:59Z"
	if len(match) > 2 && match[2] != "" {
		stamp = match[1] + "T" + match[2] + "Z"
	}
	expiresAt, err := time.Parse(time.RFC3339, stamp)
	if err != nil {
		return 0
	}
	days := int(time.Until(expiresAt).Hours() / 24)
	if time.Until(expiresAt) > 0 && time.Until(expiresAt).Hours() > float64(days*24) {
		days++
	}
	if days < 0 {
		return 0
	}
	return days
}

func certStatus(days int) string {
	if days < 7 {
		return "critical"
	}
	if days < 30 {
		return "expiring"
	}
	return "valid"
}
