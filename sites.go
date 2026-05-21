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
	"strings"
	"time"
)

func (a *App) dashboard(w http.ResponseWriter, _ *http.Request) {
	sites := a.listSites()
	services := a.listServices()
	a.render(w, "dashboard", map[string]any{
		"Title":    "Dashboard",
		"Sites":    sites,
		"Services": services,
		"Stats":    a.systemStats(),
	})
}

func (a *App) sites(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		domain := strings.ToLower(strings.TrimSpace(r.FormValue("domain")))
		if !domainRe.MatchString(domain) {
			a.render(w, "sites", map[string]any{"Title": "Sites", "Sites": a.listSites(), "Error": "Invalid domain"})
			return
		}
		root := filepath.Join(a.cfg.WebRoot, domain, "htdocs")
		_ = os.MkdirAll(root, 0755)
		index := filepath.Join(root, "index.html")
		if _, err := os.Stat(index); err != nil {
			_ = os.WriteFile(index, []byte("<h1>"+template.HTMLEscapeString(domain)+"</h1><p>Managed by hostQ</p>"), 0644)
		}
		a.writeNginxSite(domain, root, false, "8.4")
		a.audit("site.create", "success", domain)
		http.Redirect(w, r, "/sites", http.StatusSeeOther)
		return
	}
	a.render(w, "sites", map[string]any{"Title": "Sites", "Sites": a.listSites()})
}

func (a *App) siteManager(w http.ResponseWriter, r *http.Request) {
	domain := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("domain")))
	site, ok := a.findSite(domain)
	if !ok {
		http.Redirect(w, r, "/sites", http.StatusSeeOther)
		return
	}
	tab := strings.TrimSpace(r.URL.Query().Get("tab"))
	if tab == "" {
		tab = "overview"
	}
	data := map[string]any{
		"Title":    "Manage " + site.Domain,
		"Site":     site,
		"Tab":      tab,
		"Output":   r.URL.Query().Get("output"),
		"Created":  r.URL.Query().Get("created"),
		"User":     r.URL.Query().Get("user"),
		"Password": r.URL.Query().Get("password"),
		"DBUser":   r.URL.Query().Get("dbuser"),
		"DBPass":   r.URL.Query().Get("dbpass"),
		"DBName":   r.URL.Query().Get("db"),
	}
	switch tab {
	case "database":
		data["Databases"] = a.listDatabasesForSite(site.Domain)
		data["DBPrefix"] = dbPrefixForSite(site.Domain)
	case "wordpress":
		installs := a.listWordPress()
		filtered := []WordPressInfo{}
		for _, w := range installs {
			if w.Domain == site.Domain {
				filtered = append(filtered, w)
			}
		}
		data["Installs"] = filtered
		if len(filtered) > 0 {
			data["WPManage"] = &filtered[0]
			data["WPUsers"] = a.listWPUsers(filtered[0].Path)
			if report, err := a.loadMalfixReport(site.Domain); err == nil {
				data["Malfix"] = report
			}
		}
	case "ssl":
		data["Certificates"] = a.listCertificates()
	case "backups":
		data["Backups"] = a.listBackups(site.Domain)
		data["Policy"] = a.backupPolicy(site.Domain)
	case "php":
		data["PHP"] = a.listPHP()
	case "security":
		if report, err := a.loadScanReport(site.Domain); err == nil {
			data["Scan"] = report
		}
	}
	a.render(w, "site", data)
}

func (a *App) siteAction(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/sites", http.StatusSeeOther)
		return
	}
	domain := strings.TrimSpace(r.FormValue("domain"))
	action := r.FormValue("action")
	site, ok := a.findSite(domain)
	if !ok {
		http.Redirect(w, r, "/sites", http.StatusSeeOther)
		return
	}
	switch action {
	case "enable":
		_ = os.Remove(filepath.Join("/etc/nginx/sites-enabled", domain))
		_ = os.Symlink(filepath.Join(a.cfg.NginxSitesDir, domain), filepath.Join("/etc/nginx/sites-enabled", domain))
		_ = exec.Command("systemctl", "reload", "nginx").Run()
	case "disable":
		_ = os.Remove(filepath.Join("/etc/nginx/sites-enabled", domain))
		_ = exec.Command("systemctl", "reload", "nginx").Run()
	case "cache-on":
		a.writeNginxSite(site.Domain, site.Root, true, site.PHPVersion)
	case "cache-off":
		a.writeNginxSite(site.Domain, site.Root, false, site.PHPVersion)
	case "permissions":
		_ = exec.Command("chown", "-R", "www-data:www-data", filepath.Join(a.cfg.WebRoot, domain)).Run()
		_ = exec.Command("find", filepath.Join(a.cfg.WebRoot, domain), "-type", "d", "-exec", "chmod", "755", "{}", ";").Run()
		_ = exec.Command("find", filepath.Join(a.cfg.WebRoot, domain), "-type", "f", "-exec", "chmod", "644", "{}", ";").Run()
	case "backup":
		_, _ = a.createSiteBackup(site)
	case "delete":
		_ = os.Remove(filepath.Join("/etc/nginx/sites-enabled", domain))
		_ = os.Remove(filepath.Join(a.cfg.NginxSitesDir, domain))
		siteBase := filepath.Dir(site.Root)
		if a.canMutateWebPath(siteBase) {
			_ = os.RemoveAll(siteBase)
		}
		_ = exec.Command("systemctl", "reload", "nginx").Run()
	}
	a.cache.invalidate("sites")
	a.audit("site."+action, "success", domain)
	http.Redirect(w, r, "/sites", http.StatusSeeOther)
}

func (a *App) findSite(domain string) (Site, bool) {
	for _, site := range a.listSites() {
		if site.Domain == domain {
			return site, true
		}
	}
	return Site{}, false
}

func (a *App) listSites() []Site {
	if v, ok := a.cache.get("sites"); ok {
		return v.([]Site)
	}
	sites := []Site{}
	entries, err := os.ReadDir(a.cfg.NginxSitesDir)
	if err != nil {
		return sites
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		data, err := os.ReadFile(filepath.Join(a.cfg.NginxSitesDir, entry.Name()))
		if err != nil {
			continue
		}
		text := string(data)
		if !strings.Contains(text, "hostQ managed") {
			continue
		}
		domain := firstMatch(text, `hostQ managed - ([^\n]+)`)
		root := firstMatch(text, `root\s+([^;]+);`)
		phpVersion := firstMatch(text, `php(\d\.\d)-fpm`)
		if domain == "" {
			domain = entry.Name()
		}
		if phpVersion == "" {
			phpVersion = "8.4"
		}
		_, enabledErr := os.Stat(filepath.Join("/etc/nginx/sites-enabled", entry.Name()))
		sites = append(sites, Site{
			Domain: domain, Root: root, Enabled: enabledErr == nil, PHPVersion: phpVersion,
			SSL:   strings.Contains(text, "ssl_certificate") && a.certExists(domain),
			Cache: strings.Contains(text, "hostQ fastcgi cache: on"),
		})
	}
	sort.Slice(sites, func(i, j int) bool { return sites[i].Domain < sites[j].Domain })
	a.cache.set("sites", sites, 3*time.Second)
	return sites
}

func firstMatch(text, expr string) string {
	match := regexp.MustCompile(expr).FindStringSubmatch(text)
	if len(match) > 1 {
		return strings.TrimSpace(match[1])
	}
	return ""
}

func (a *App) writeNginxSite(domain, root string, cache bool, phpVersion string) {
	if !phpVersionRe.MatchString(phpVersion) {
		phpVersion = "8.4"
	}
	cacheBlock := ""
	if cache {
		cacheBlock = `
        fastcgi_cache HOSTQ_FASTCGI;
        fastcgi_cache_valid 200 301 302 10m;
        add_header X-hostQ-Cache $upstream_cache_status always;`
	}
	siteBody := fmt.Sprintf(`    root %s;
    index index.php index.html;
    location / { try_files $uri $uri/ /index.php?$query_string; }
    location ~ \.php$ {
        include snippets/fastcgi-php.conf;
        fastcgi_pass unix:/run/php/php%s-fpm.sock;
        fastcgi_param SCRIPT_FILENAME $realpath_root$fastcgi_script_name;
        include fastcgi_params;%s
    }
`, root, phpVersion, cacheBlock)

	cacheLabel := "off"
	if cache {
		cacheLabel = "on"
	}

	// Preserve SSL across rewrites: if a Let's Encrypt cert exists for the
	// domain, emit a 443 server block ourselves and 301 the 80 block. Prior
	// versions only wrote port 80, so any panel action that called this
	// helper (cache toggle, PHP switch, WordPress install) silently wiped
	// the SSL config certbot had injected.
	hasSSL := a.certExists(domain)
	sslLabel := "off"
	port80 := fmt.Sprintf(`server {
    listen 80;
    server_name %s www.%s;
%s}
`, domain, domain, siteBody)
	port443 := ""
	if hasSSL {
		sslLabel = "on"
		sslIncludes := ""
		if _, err := os.Stat("/etc/letsencrypt/options-ssl-nginx.conf"); err == nil {
			sslIncludes += "    include /etc/letsencrypt/options-ssl-nginx.conf;\n"
		}
		if _, err := os.Stat("/etc/letsencrypt/ssl-dhparams.pem"); err == nil {
			sslIncludes += "    ssl_dhparam /etc/letsencrypt/ssl-dhparams.pem;\n"
		}
		port80 = fmt.Sprintf(`server {
    listen 80;
    server_name %s www.%s;
    return 301 https://$host$request_uri;
}
`, domain, domain)
		port443 = fmt.Sprintf(`server {
    listen 443 ssl http2;
    server_name %s www.%s;
    ssl_certificate /etc/letsencrypt/live/%s/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/%s/privkey.pem;
%s%s}
`, domain, domain, domain, domain, sslIncludes, siteBody)
	}
	conf := fmt.Sprintf("# hostQ managed - %s\n# hostQ fastcgi cache: %s\n# hostQ ssl: %s\n%s%s",
		domain, cacheLabel, sslLabel, port80, port443)

	_ = os.WriteFile(filepath.Join(a.cfg.NginxSitesDir, domain), []byte(conf), 0644)
	_ = os.Remove(filepath.Join("/etc/nginx/sites-enabled", domain))
	_ = os.Symlink(filepath.Join(a.cfg.NginxSitesDir, domain), filepath.Join("/etc/nginx/sites-enabled", domain))
	_ = exec.Command("nginx", "-t").Run()
	_ = exec.Command("systemctl", "reload", "nginx").Run()
	a.cache.invalidate("sites")
}
