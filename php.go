package main

import (
	"net/http"
	"os/exec"
	"strings"
)

func (a *App) php(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		a.phpAction(w, r)
		return
	}
	a.render(w, "php", map[string]any{
		"Title": "PHP",
		"PHP":   a.listPHP(),
		"Sites": a.listSites(),
	})
}

func (a *App) listPHP() []PHPInfo {
	versions := []string{"8.2", "8.3", "8.4", "8.5"}
	out := []PHPInfo{}
	for _, version := range versions {
		service := "php" + version + "-fpm"
		status := "missing"
		if data, err := exec.Command("systemctl", "is-active", service).Output(); err == nil {
			status = strings.TrimSpace(string(data))
		}
		out = append(out, PHPInfo{Version: version, Service: service, Status: status})
	}
	return out
}

func (a *App) phpAction(w http.ResponseWriter, r *http.Request) {
	domain := strings.TrimSpace(r.FormValue("domain"))
	version := strings.TrimSpace(r.FormValue("version"))
	if !domainRe.MatchString(domain) || !phpVersionRe.MatchString(version) {
		http.Redirect(w, r, "/php", http.StatusSeeOther)
		return
	}
	site, ok := a.findSite(domain)
	if ok {
		a.writeNginxSite(site.Domain, site.Root, site.Cache, version)
		a.audit("php.switch", "success", domain+"="+version)
	}
	http.Redirect(w, r, "/php", http.StatusSeeOther)
}
