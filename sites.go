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
	case "nginx":
		data["NginxExtra"] = a.loadExtraNginx(site.Domain)
		if vhost, err := os.ReadFile(filepath.Join(a.cfg.NginxSitesDir, site.Domain)); err == nil {
			data["NginxVhost"] = string(vhost)
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

// extraNginxPath is where per-site custom Nginx directives are persisted.
// The file is included from inside every hostQ-managed server block so
// rewrite rules survive any panel action that re-renders the vhost (cache
// toggle, PHP switch, SSL install, WordPress install).
func (a *App) extraNginxPath(domain string) string {
	if !domainRe.MatchString(domain) {
		return ""
	}
	return filepath.Join(a.cfg.DataDir, "sites", domain+".extra.conf")
}

func (a *App) loadExtraNginx(domain string) string {
	if p := a.extraNginxPath(domain); p != "" {
		data, _ := os.ReadFile(p)
		return string(data)
	}
	return ""
}

func (a *App) saveExtraNginx(domain, content string) error {
	p := a.extraNginxPath(domain)
	if p == "" {
		return fmt.Errorf("invalid domain")
	}
	cleaned := sanitizeExtraNginx(content)
	if cleaned == "" {
		_ = os.Remove(p)
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(p), 0700); err != nil {
		return err
	}
	return os.WriteFile(p, []byte(cleaned), 0644)
}

// sanitizeExtraNginx makes a user-pasted snippet safe to include inside a
// hostQ-managed server block. It (1) peels off an outer `server { ... }`
// wrapper if the user pasted a full vhost, and (2) drops top-level
// directives that would duplicate what writeNginxSite already emits
// (listen, server_name, root, index, ssl_*).
func sanitizeExtraNginx(content string) string {
	s := strings.TrimSpace(content)
	if s == "" {
		return ""
	}
	s = unwrapOuterServer(s)
	s = stripTopLevelDirectives(s, map[string]bool{
		"listen":              true,
		"server_name":         true,
		"root":                true,
		"index":               true,
		"ssl_certificate":     true,
		"ssl_certificate_key": true,
		"ssl_dhparam":         true,
		"ssl_protocols":       true,
		"ssl_ciphers":         true,
		"ssl_session_cache":   true,
		"ssl_session_timeout": true,
	})
	return strings.TrimSpace(s)
}

// unwrapOuterServer returns the body of a single top-level `server { ... }`
// block when the input starts with one; otherwise returns the input as-is.
func unwrapOuterServer(s string) string {
	trim := strings.TrimSpace(s)
	if !strings.HasPrefix(trim, "server") {
		return s
	}
	// find the opening brace
	rest := strings.TrimSpace(strings.TrimPrefix(trim, "server"))
	if !strings.HasPrefix(rest, "{") {
		return s
	}
	open := strings.Index(s, "{")
	if open < 0 {
		return s
	}
	depth, end := 0, -1
	for i := open; i < len(s); i++ {
		switch s[i] {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				end = i
			}
		}
		if end >= 0 {
			break
		}
	}
	if end < 0 || end <= open+1 {
		return s
	}
	return strings.TrimSpace(s[open+1 : end])
}

// stripTopLevelDirectives removes lines that start (at brace depth 0) with one
// of the named directives. Lines inside nested blocks (location { ... }, if {},
// etc.) are kept untouched so user rewrite logic survives.
func stripTopLevelDirectives(s string, names map[string]bool) string {
	var b strings.Builder
	depthBefore := 0
	for _, line := range strings.Split(s, "\n") {
		drop := false
		trimmed := strings.TrimSpace(line)
		if depthBefore == 0 && trimmed != "" && !strings.HasPrefix(trimmed, "#") {
			// Extract the directive name (first token, may include trailing ; or {).
			tok := trimmed
			for _, sep := range []string{" ", "\t", ";", "{"} {
				if i := strings.Index(tok, sep); i >= 0 {
					tok = tok[:i]
				}
			}
			if names[tok] {
				drop = true
			}
		}
		if !drop {
			b.WriteString(line)
			b.WriteByte('\n')
		}
		for _, c := range line {
			if c == '{' {
				depthBefore++
			} else if c == '}' {
				depthBefore--
			}
		}
	}
	return strings.TrimRight(b.String(), "\n")
}

// siteNginx renders the per-site custom Nginx editor and handles its actions.
// "save" persists the custom block and re-renders the vhost; if nginx -t
// refuses it, the file is rolled back so the running config keeps working.
// "flush" purges the shared FastCGI cache directory and reloads Nginx.
func (a *App) siteNginx(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/sites", http.StatusSeeOther)
		return
	}
	domain := strings.ToLower(strings.TrimSpace(r.FormValue("domain")))
	site, ok := a.findSite(domain)
	if !ok {
		http.Redirect(w, r, "/sites", http.StatusSeeOther)
		return
	}
	action := r.FormValue("action")
	output := ""
	switch action {
	case "save":
		content := r.FormValue("nginx")
		prev := a.loadExtraNginx(domain)
		if err := a.saveExtraNginx(domain, content); err != nil {
			output = "save failed: " + err.Error()
			break
		}
		a.writeNginxSite(domain, site.Root, site.Cache, site.PHPVersion)
		if out, err := exec.Command("nginx", "-t").CombinedOutput(); err != nil {
			_ = a.saveExtraNginx(domain, prev)
			a.writeNginxSite(domain, site.Root, site.Cache, site.PHPVersion)
			output = "nginx -t failed; rolled back. " + strings.TrimSpace(string(out))
			a.audit("nginx.save", "failure", domain)
		} else {
			_ = exec.Command("systemctl", "reload", "nginx").Run()
			output = "Custom Nginx config saved and reloaded."
			a.audit("nginx.save", "success", domain)
		}
	case "flush":
		_ = os.RemoveAll("/var/cache/nginx/hostq-fastcgi")
		_ = os.MkdirAll("/var/cache/nginx/hostq-fastcgi", 0755)
		_ = exec.Command("chown", "-R", "www-data:www-data", "/var/cache/nginx/hostq-fastcgi").Run()
		_ = exec.Command("systemctl", "reload", "nginx").Run()
		output = "FastCGI cache flushed; Nginx reloaded."
		a.audit("nginx.flush", "success", domain)
	default:
		output = "Unknown nginx action: " + action
	}
	http.Redirect(w, r, "/site?domain="+domain+"&tab=nginx&output="+queryEscape(output), http.StatusSeeOther)
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
	pmaInclude := ""
	if _, err := os.Stat("/etc/nginx/snippets/hostq-pma.conf"); err == nil {
		pmaInclude = "    include snippets/hostq-pma.conf;\n"
	}
	extraInclude := ""
	if extraPath := a.extraNginxPath(domain); extraPath != "" {
		if data, err := os.ReadFile(extraPath); err == nil && len(strings.TrimSpace(string(data))) > 0 {
			extraInclude = fmt.Sprintf("    # hostQ custom rules for %s\n    include %s;\n", domain, extraPath)
		}
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
%s%s`, root, phpVersion, cacheBlock, pmaInclude, extraInclude)

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
