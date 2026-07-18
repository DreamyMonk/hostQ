package main

import (
	"fmt"
	"html/template"
	"log"
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

// docrootRe accepts a relative path under the site's parent like "htdocs",
// "public", "public_html", "www", "src/public", etc. Rejects ".." or absolute
// paths so the caller can't escape /var/www/<domain>/.
var docrootRe = regexp.MustCompile(`^[a-zA-Z0-9._][a-zA-Z0-9._-]*(/[a-zA-Z0-9._-]+)*$`)

// siteAdd renders the mode-picker page (Blank vs Install WordPress) and
// handles the create + optional WP install in one shot.
func (a *App) siteAdd(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		a.render(w, "siteadd", map[string]any{"Title": "Add website"})
		return
	}
	domain := strings.ToLower(strings.TrimSpace(r.FormValue("domain")))
	domain = strings.TrimPrefix(domain, "www.")
	if !domainRe.MatchString(domain) {
		a.render(w, "siteadd", map[string]any{"Title": "Add website", "Error": "Invalid domain", "Domain": r.FormValue("domain")})
		return
	}
	docroot := strings.TrimSpace(r.FormValue("docroot"))
	docroot = strings.Trim(docroot, "/")
	if docroot == "" {
		docroot = "htdocs"
	}
	if !docrootRe.MatchString(docroot) {
		a.render(w, "siteadd", map[string]any{"Title": "Add website", "Error": "Invalid document root", "Domain": domain})
		return
	}
	root := filepath.Join(a.cfg.WebRoot, domain, docroot)
	_ = os.MkdirAll(root, 0755)

	mode := r.FormValue("mode")
	if mode != "wordpress" && mode != "laravel" {
		mode = "blank"
	}

	if mode == "laravel" {
		// Laravel one-shot: composer scaffold + DB + .env + key. The web
		// docroot becomes <root>/public, wired inside installLaravel.
		out, err := a.installLaravel(LaravelInstallParams{
			Domain:     domain,
			ProjectDir: root,
			PHPVersion: "8.4",
		})
		if err != nil {
			a.render(w, "siteadd", map[string]any{"Title": "Add website", "Error": "Laravel install failed", "Domain": domain, "Output": out})
			return
		}
		a.audit("site.create", "success", domain)
		http.Redirect(w, r, "/site?domain="+domain+"&output="+queryEscape("Site created and Laravel installed."), http.StatusSeeOther)
		return
	}

	if mode == "blank" {
		index := filepath.Join(root, "index.html")
		if _, err := os.Stat(index); err != nil {
			_ = os.WriteFile(index, []byte("<h1>"+template.HTMLEscapeString(domain)+"</h1><p>Managed by hostQ</p>"), 0644)
		}
		a.writeNginxSite(domain, root, false, "8.4")
		a.audit("site.create", "success", domain)
		http.Redirect(w, r, "/site?domain="+domain, http.StatusSeeOther)
		return
	}

	// WordPress one-shot: create the site shell, then run wp-cli through the
	// shared installer (DB + core + config) and surface its log on the WP
	// page if anything fails.
	a.writeNginxSite(domain, root, false, "8.4")
	a.audit("site.create", "success", domain)
	wpTitle := strings.TrimSpace(r.FormValue("wp_title"))
	if wpTitle == "" {
		wpTitle = domain
	}
	out, err := a.installWordPress(WPInstallParams{
		Domain:     domain,
		Root:       root,
		Title:      wpTitle,
		AdminUser:  safeName(r.FormValue("wp_user")),
		AdminPass:  strings.TrimSpace(r.FormValue("wp_pass")),
		AdminEmail: strings.TrimSpace(r.FormValue("wp_email")),
	})
	if err != nil {
		http.Redirect(w, r, "/wordpress?output="+queryEscape(out), http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, "/site?domain="+domain+"&output="+queryEscape("Site created and WordPress installed."), http.StatusSeeOther)
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
	if tab == "overview" {
		// Lightweight per-site "applications detected" data: any WP install
		// rooted at this site's docroot.
		var siteWP *WordPressInfo
		for _, w := range a.listWordPress() {
			if w.Domain == site.Domain {
				wp := w
				siteWP = &wp
				break
			}
		}
		data["WPOnSite"] = siteWP
		// Same idea for Laravel apps rooted under this site.
		var siteLaravel *LaravelInfo
		for _, l := range a.listLaravel() {
			if l.Domain == site.Domain {
				lv := l
				siteLaravel = &lv
				break
			}
		}
		data["LaravelOnSite"] = siteLaravel
		// Hero card visual: derive a stable hue from the domain so each site
		// gets a recognisable colour without us needing screenshots.
		h := 0
		for _, b := range []byte(site.Domain) {
			h = (h*31 + int(b)) & 0xffff
		}
		data["HeroHue"] = h % 360
		first := site.Domain
		if first != "" {
			first = strings.ToUpper(first[:1])
		}
		data["HeroLetter"] = first
		data["ServerHostname"] = a.systemStats().Hostname
	}
	switch tab {
	case "database":
		dbs := a.listDatabasesForSite(site.Domain)
		data["Databases"] = dbs
		data["DBPrefix"] = dbPrefixForSite(site.Domain)
		// Build the "attach existing" dropdown: every DB on the server that
		// isn't already linked to this site.
		shown := map[string]bool{}
		for _, db := range dbs {
			shown[db.Name] = true
		}
		available := []DatabaseInfo{}
		for _, db := range a.listDatabases() {
			if !shown[db.Name] {
				available = append(available, db)
			}
		}
		data["AttachableDBs"] = available
		data["AttachedDBs"] = a.siteAttachedDBs(site.Domain)
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
		data["PHPExtensions"] = phpExtensions(site.PHPVersion)
		raw := a.loadSitePHPIni(site.Domain)
		data["PHPIniRaw"] = raw
		data["PHPIniValues"] = parseSitePHPIni(raw)
		data["PHPManagedKeys"] = managedPHPKeys
	case "security":
		if report, err := a.loadScanReport(site.Domain); err == nil {
			data["Scan"] = report
		}
	case "nginx":
		data["NginxExtra"] = a.loadExtraNginx(site.Domain)
		if vhost, err := os.ReadFile(filepath.Join(a.cfg.NginxSitesDir, site.Domain)); err == nil {
			data["NginxVhost"] = string(vhost)
		}
	case "analytics":
		data["Analytics"] = a.siteAnalytics(site.Domain, 7)
	case "cache":
		data["RedisStats"] = a.redisStats()
		data["FastCGICacheDir"] = "/var/cache/nginx/hostq-fastcgi"
	case "domains":
		data["Aliases"] = a.loadAliases(site.Domain)
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
	case "backend-apache":
		if !apacheInstalled() {
			http.Redirect(w, r, "/site?domain="+domain+"&tab=nginx&output="+queryEscape("Install Apache from Services & Packages first."), http.StatusSeeOther)
			return
		}
		if err := a.ensureApacheHybrid(); err != nil {
			http.Redirect(w, r, "/site?domain="+domain+"&tab=nginx&output="+queryEscape("Apache setup failed: "+err.Error()), http.StatusSeeOther)
			return
		}
		_ = a.setSiteBackend(domain, "apache")
		// FastCGI cache is an Nginx-FastCGI concept; force it off for Apache.
		a.writeNginxSite(site.Domain, site.Root, false, site.PHPVersion)
		a.cache.invalidate("sites")
		a.audit("site.backend-apache", "success", domain)
		http.Redirect(w, r, "/site?domain="+domain+"&tab=nginx&output="+queryEscape("Switched "+domain+" to the Apache backend — .htaccess is now active."), http.StatusSeeOther)
		return
	case "backend-nginx":
		_ = a.setSiteBackend(domain, "nginx")
		a.removeApacheSite(domain)
		a.writeNginxSite(site.Domain, site.Root, site.Cache, site.PHPVersion)
		a.cache.invalidate("sites")
		a.audit("site.backend-nginx", "success", domain)
		http.Redirect(w, r, "/site?domain="+domain+"&tab=nginx&output="+queryEscape("Switched "+domain+" back to the Nginx backend."), http.StatusSeeOther)
		return
	case "permissions":
		_ = exec.Command("chown", "-R", "www-data:www-data", filepath.Join(a.cfg.WebRoot, domain)).Run()
		_ = exec.Command("find", filepath.Join(a.cfg.WebRoot, domain), "-type", "d", "-exec", "chmod", "755", "{}", ";").Run()
		_ = exec.Command("find", filepath.Join(a.cfg.WebRoot, domain), "-type", "f", "-exec", "chmod", "644", "{}", ";").Run()
	case "php-save-managed":
		_ = r.ParseForm()
		form := map[string]string{}
		for _, m := range managedPHPKeys {
			form[m.Key] = r.FormValue("php_" + m.Key)
		}
		content := buildSitePHPIniFromForm(a.loadSitePHPIni(domain), form)
		if err := a.saveSitePHPIni(domain, content); err == nil {
			a.writeNginxSite(domain, site.Root, site.Cache, site.PHPVersion)
			a.audit("php.ini-managed", "success", domain)
		}
		http.Redirect(w, r, "/site?domain="+domain+"&tab=php&output="+queryEscape("PHP overrides saved + Nginx reloaded."), http.StatusSeeOther)
		return
	case "php-save-full":
		content := r.FormValue("ini")
		if err := a.saveSitePHPIni(domain, content); err != nil {
			http.Redirect(w, r, "/site?domain="+domain+"&tab=php&output="+queryEscape("save failed: "+err.Error()), http.StatusSeeOther)
			return
		}
		a.writeNginxSite(domain, site.Root, site.Cache, site.PHPVersion)
		a.audit("php.ini-full", "success", domain)
		http.Redirect(w, r, "/site?domain="+domain+"&tab=php&output="+queryEscape("Raw PHP overrides saved + Nginx reloaded."), http.StatusSeeOther)
		return
	case "alias-add":
		alias := strings.ToLower(strings.TrimSpace(r.FormValue("alias")))
		alias = strings.TrimPrefix(alias, "www.")
		if err := a.addAlias(domain, alias); err == nil {
			a.writeNginxSite(domain, site.Root, site.Cache, site.PHPVersion)
		}
		http.Redirect(w, r, "/site?domain="+domain+"&tab=domains", http.StatusSeeOther)
		return
	case "alias-remove":
		alias := strings.ToLower(strings.TrimSpace(r.FormValue("alias")))
		if err := a.removeAlias(domain, alias); err == nil {
			a.writeNginxSite(domain, site.Root, site.Cache, site.PHPVersion)
		}
		http.Redirect(w, r, "/site?domain="+domain+"&tab=domains", http.StatusSeeOther)
		return
	case "backup":
		_, _ = a.createSiteBackup(site)
	case "delete":
		_ = os.Remove(filepath.Join("/etc/nginx/sites-enabled", domain))
		_ = os.Remove(filepath.Join(a.cfg.NginxSitesDir, domain))
		// Remove the source-of-truth record and the good-config backup so the
		// site isn't resurrected by a rebuild/self-heal after deletion.
		a.removeSiteMeta(domain)
		_ = os.Remove(filepath.Join(a.nginxConfBackupDir(), domain))
		// Tear down the Apache backend + persisted backend choice, if any.
		a.removeApacheSite(domain)
		_ = a.setSiteBackend(domain, "nginx")
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

// hasUserLocationSlash returns true if the (already-sanitised) extra config
// declares its own `location / { ... }` so writeNginxSite can skip the
// panel default.
var locationSlashRe = regexp.MustCompile(`(?m)^\s*location\s+/\s*\{`)

func hasUserLocationSlash(extra string) bool {
	if strings.TrimSpace(extra) == "" {
		return false
	}
	return locationSlashRe.MatchString(extra)
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
		// writeNginxSite now validates the generated vhost with `nginx -t` and
		// only reloads on success. If the custom snippet is bad it returns the
		// error, so restore the previous snippet and regenerate to return both
		// the include file and the live vhost to the last good state.
		if err := a.writeNginxSite(domain, site.Root, site.Cache, site.PHPVersion); err != nil {
			_ = a.saveExtraNginx(domain, prev)
			_ = a.writeNginxSite(domain, site.Root, site.Cache, site.PHPVersion)
			output = "nginx -t failed; rolled back. " + err.Error()
			a.audit("nginx.save", "failure", domain)
		} else {
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
		// Skip hostQ sidecars (.prev rollback copies, temp files, hidden) so a
		// rollback snapshot never shows up as a phantom duplicate site.
		if isSidecarName(entry.Name()) {
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
		if phpVersion == "" {
			// Apache-backed vhosts proxy to Apache instead of php-fpm, so the
			// socket line is absent — fall back to the header comment.
			phpVersion = firstMatch(text, `hostQ php: (\d\.\d)`)
		}
		backend := firstMatch(text, `hostQ backend: (\w+)`)
		if domain == "" {
			domain = entry.Name()
		}
		if phpVersion == "" {
			phpVersion = "8.4"
		}
		if backend == "" {
			backend = "nginx"
		}
		_, enabledErr := os.Stat(filepath.Join("/etc/nginx/sites-enabled", entry.Name()))
		sites = append(sites, Site{
			Domain: domain, Root: root, Enabled: enabledErr == nil, PHPVersion: phpVersion,
			Backend: backend,
			SSL:     strings.Contains(text, "ssl_certificate") && a.certExists(domain),
			Cache:   strings.Contains(text, "hostQ fastcgi cache: on"),
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

func (a *App) writeNginxSite(domain, root string, cache bool, phpVersion string) error {
	if !phpVersionRe.MatchString(phpVersion) {
		phpVersion = "8.4"
	}
	// Commit the source-of-truth record first (begin transaction). The Nginx
	// vhost below is a disposable artifact regenerated from this metadata by
	// `hostq rebuild` / `hostq repair`.
	if err := a.saveSiteMeta(SiteMeta{Domain: domain, Root: root, Cache: cache, PHPVersion: phpVersion, Updated: time.Now().Format(time.RFC3339)}); err != nil {
		log.Printf("writeNginxSite %s: metadata save failed: %v", domain, err)
	}
	// Web backend: "apache" means Nginx front-proxies dynamic requests to the
	// site's loopback Apache vhost (which handles PHP + .htaccess); "nginx"
	// (default) means Nginx talks to php-fpm directly. Read from disk so no
	// caller has to thread the choice through.
	backend := a.siteBackend(domain)
	if backend == "apache" && apacheInstalled() {
		// Keep the Apache vhost in sync on every render (alias/PHP/root change).
		_ = a.writeApacheSite(domain, root, phpVersion)
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
	// If the user supplies their own `location /` block in extra.conf, suppress
	// the panel default so nginx doesn't reject "duplicate location" and so
	// the user can choose, e.g., $uri.php fallback over /index.php front
	// controller. Anything more nuanced (regex location overrides, etc.) the
	// user can paste freely.
	defaultLocationSlash := "    location / { try_files $uri $uri/ /index.php?$query_string; }\n"
	if hasUserLocationSlash(a.loadExtraNginx(domain)) {
		defaultLocationSlash = "    # location / is overridden by the site's custom Nginx config.\n"
	}
	accessLog := fmt.Sprintf("    access_log /var/log/nginx/%s.access.log combined;\n", domain)
	// Per-site PHP_VALUE overrides — populated from /etc/hostq/sites/<domain>.php.ini
	// when the operator edits the PHP overrides on the PHP tab. Empty → no extra
	// fastcgi_param line is emitted, keeping the vhost output identical to before.
	phpValueBlock := renderPHPValueBlock(a.loadSitePHPIni(domain))
	siteBody := fmt.Sprintf(`%s    root %s;
    index index.php index.html;
%s    location ~ \.php$ {
        include snippets/fastcgi-php.conf;
        fastcgi_pass unix:/run/php/php%s-fpm.sock;
        fastcgi_param SCRIPT_FILENAME $realpath_root$fastcgi_script_name;
        include fastcgi_params;%s%s
    }
%s%s`, accessLog, root, defaultLocationSlash, phpVersion, cacheBlock, phpValueBlock, pmaInclude, extraInclude)

	// Apache backend: hand the whole site to the loopback Apache vhost so its
	// .htaccess rules apply. Nginx proxies everything, keeping TLS and the
	// panel front-door intact. Per-site PHP overrides, the FastCGI cache and
	// the custom Nginx include are all Nginx-FastCGI concepts — and the
	// include could collide with this single `location /` — so we deliberately
	// omit them here. Rewrite rules live in .htaccess in this mode.
	if backend == "apache" {
		siteBody = fmt.Sprintf(`%s    root %s;
    location / {
        proxy_pass http://%s;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
`, accessLog, root, apacheBackendAddr)
	}

	cacheLabel := "off"
	if cache {
		cacheLabel = "on"
	}

	// Preserve SSL across rewrites: if a Let's Encrypt cert exists for the
	// domain, emit a 443 server block ourselves and 301 the 80 block. Prior
	// versions only wrote port 80, so any panel action that called this
	// helper (cache toggle, PHP switch, WordPress install) silently wiped
	// the SSL config certbot had injected.
	// Aliases — extra hostnames the user attached via the Domains tab. They
	// land in server_name on every block so the existing vhost serves them
	// too (no extra docroot, no extra cert — alias semantics).
	aliasSuffix := ""
	if aliases := a.loadAliases(domain); len(aliases) > 0 {
		aliasSuffix = " " + strings.Join(aliases, " ")
	}
	hasSSL := a.certExists(domain)
	sslLabel := "off"
	port80 := fmt.Sprintf(`server {
    listen 80;
    server_name %s www.%s%s;
%s}
`, domain, domain, aliasSuffix, siteBody)
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
    server_name %s www.%s%s;
    return 301 https://$host$request_uri;
}
`, domain, domain, aliasSuffix)
		port443 = fmt.Sprintf(`server {
    listen 443 ssl http2;
    server_name %s www.%s%s;
    ssl_certificate /etc/letsencrypt/live/%s/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/%s/privkey.pem;
%s%s}
`, domain, domain, aliasSuffix, domain, domain, sslIncludes, siteBody)
	}
	conf := fmt.Sprintf("# hostQ managed - %s\n# hostQ backend: %s\n# hostQ php: %s\n# hostQ fastcgi cache: %s\n# hostQ ssl: %s\n%s%s",
		domain, backend, phpVersion, cacheLabel, sslLabel, port80, port443)

	// Publish through the safe path: atomic write, `nginx -t` validation,
	// automatic rollback to the last good config on failure, and reload only
	// on success. This replaces the old truncate-then-reload-regardless write
	// that could leave a zero-byte vhost live (see nginxsafe.go).
	err := a.applyNginxConf(domain, []byte(conf))
	if err != nil {
		log.Printf("writeNginxSite %s: %v", domain, err)
	}
	a.cache.invalidate("sites")
	return err
}
