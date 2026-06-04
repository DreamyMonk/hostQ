package main

import (
	"bytes"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

func (a *App) wordpress(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		a.wordpressAction(w, r)
		return
	}
	installs := a.listWordPress()
	manageDomain := strings.TrimSpace(r.URL.Query().Get("manage"))
	var manage *WordPressInfo
	var users []WordPressUser
	if manageDomain != "" {
		for i := range installs {
			if installs[i].Domain == manageDomain {
				manage = &installs[i]
				users = a.listWPUsers(installs[i].Path)
				break
			}
		}
	}
	a.render(w, "wordpress", map[string]any{
		"Title":    "WordPress",
		"Installs": installs,
		"Output":   r.URL.Query().Get("output"),
		"Site":     strings.TrimSpace(r.URL.Query().Get("site")),
		"Manage":   manage,
		"Users":    users,
	})
}

func (a *App) listWordPress() []WordPressInfo {
	installs := []WordPressInfo{}
	_ = filepath.Walk(a.cfg.WebRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil || info == nil || info.IsDir() || info.Name() != "wp-config.php" {
			return nil
		}
		root := filepath.Dir(path)
		rel, _ := filepath.Rel(a.cfg.WebRoot, root)
		parts := strings.Split(rel, string(os.PathSeparator))
		domain := rel
		if len(parts) > 0 {
			domain = parts[0]
		}
		info2 := WordPressInfo{Domain: domain, Path: root, Status: "installed"}
		info2.Version = wpValue(root, "core", "version")
		info2.SiteURL = wpValue(root, "option", "get", "siteurl")
		installs = append(installs, info2)
		return nil
	})
	sort.Slice(installs, func(i, j int) bool { return installs[i].Domain < installs[j].Domain })
	return installs
}

// wpRun executes a wp-cli command at the given path and returns combined
// output with PHP/Zend warning noise filtered out. Used for logs shown to
// the operator after an action.
func wpRun(path string, args ...string) string {
	full := append([]string{"--path=" + path, "--allow-root", "--skip-themes", "--skip-plugins"}, args...)
	cmd := exec.Command("wp", full...)
	out, _ := cmd.CombinedOutput()
	return cleanWPOutput(string(out))
}

// wpValue runs wp-cli capturing stdout only and returns it trimmed. Stderr
// noise (PHP warnings like "Cannot load Zend OPcache") is discarded so the
// result is safe to display inline as a value.
func wpValue(path string, args ...string) string {
	full := append([]string{"--path=" + path, "--allow-root", "--skip-themes", "--skip-plugins"}, args...)
	cmd := exec.Command("wp", full...)
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	_ = cmd.Run()
	return strings.TrimSpace(cleanWPOutput(stdout.String()))
}

// cleanWPOutput drops the PHP/Zend warning + notice lines that some hosts
// emit on every CLI invocation. They pollute version checks and search-replace
// previews; nobody wants to see them in the panel.
func cleanWPOutput(s string) string {
	lines := strings.Split(s, "\n")
	kept := make([]string, 0, len(lines))
	for _, line := range lines {
		l := strings.TrimSpace(line)
		if l == "" {
			continue
		}
		low := strings.ToLower(l)
		if strings.HasPrefix(low, "php warning") ||
			strings.HasPrefix(low, "php notice") ||
			strings.HasPrefix(low, "php deprecated") ||
			strings.HasPrefix(low, "warning:") ||
			strings.HasPrefix(low, "notice:") ||
			strings.HasPrefix(low, "deprecated:") ||
			strings.HasPrefix(low, "cannot load") ||
			strings.HasPrefix(low, "stack trace:") ||
			strings.HasPrefix(low, "#") && strings.Contains(low, "include(") {
			continue
		}
		kept = append(kept, line)
	}
	return strings.Join(kept, "\n")
}

func (a *App) listWPUsers(path string) []WordPressUser {
	out := wpRun(path, "user", "list", "--fields=ID,user_login,user_email,roles", "--format=csv")
	users := []WordPressUser{}
	lines := strings.Split(strings.TrimSpace(out), "\n")
	for i, line := range lines {
		if i == 0 || line == "" {
			continue
		}
		fields := strings.SplitN(line, ",", 4)
		if len(fields) < 4 {
			continue
		}
		users = append(users, WordPressUser{
			ID: fields[0], Login: fields[1], Email: fields[2], Roles: fields[3],
		})
	}
	return users
}

func (a *App) wordpressAction(w http.ResponseWriter, r *http.Request) {
	action := r.FormValue("action")
	if action == "install" || action == "" {
		a.wordpressInstall(w, r)
		return
	}
	domain := strings.ToLower(strings.TrimSpace(r.FormValue("domain")))
	if !domainRe.MatchString(domain) {
		http.Redirect(w, r, "/wordpress?output="+template.URLQueryEscaper("Invalid domain"), http.StatusSeeOther)
		return
	}
	// Locate install path
	site, ok := a.findSite(domain)
	root := ""
	if ok {
		root = site.Root
	} else {
		root = filepath.Join(a.cfg.WebRoot, domain, "htdocs")
	}
	if _, err := os.Stat(filepath.Join(root, "wp-config.php")); err != nil {
		http.Redirect(w, r, "/wordpress?output="+template.URLQueryEscaper("No WordPress install at "+root), http.StatusSeeOther)
		return
	}

	output := ""
	auditTag := "wordpress." + action
	auditStatus := "success"
	redirect := "/wordpress?manage=" + domain
	switch action {
	case "update-core":
		output = wpRun(root, "core", "update")
		output += "\n" + wpRun(root, "core", "update-db")
	case "flush-cache":
		output = wpRun(root, "cache", "flush")
		output += wpRun(root, "rewrite", "flush")
	case "reset-pass":
		user := safeName(r.FormValue("user"))
		newPass := strings.TrimSpace(r.FormValue("password"))
		if user == "" || len(newPass) < 8 {
			output = "Provide an existing username and a password of 8+ characters."
			auditStatus = "failure"
			break
		}
		output = wpRun(root, "user", "update", user, "--user_pass="+newPass)
		if !strings.Contains(strings.ToLower(output), "success") && !strings.Contains(strings.ToLower(output), "updated") {
			auditStatus = "failure"
		} else {
			output = "Password updated for user '" + user + "'.\n" + output
		}
	case "change-url":
		newURL := strings.TrimSpace(r.FormValue("url"))
		if newURL == "" {
			output = "Provide a target URL like https://example.com"
			auditStatus = "failure"
			break
		}
		old := strings.TrimSpace(wpRun(root, "option", "get", "siteurl"))
		output = wpRun(root, "option", "update", "siteurl", newURL)
		output += wpRun(root, "option", "update", "home", newURL)
		if old != "" && old != newURL {
			output += wpRun(root, "search-replace", old, newURL, "--all-tables-with-prefix")
		}
		output += wpRun(root, "cache", "flush")
	case "delete":
		// Remove WP files + drop the matching database. Keep nginx vhost.
		if !a.canMutateWebPath(root) {
			output = "Refusing to delete: path not under web root."
			auditStatus = "failure"
			break
		}
		dbName := safeDBName(strings.ReplaceAll(domain, ".", "_"))
		if dbName != "" {
			dbUser := dbName
			if len(dbUser) > 32 {
				dbUser = dbUser[:32]
			}
			sql := fmt.Sprintf("DROP DATABASE IF EXISTS %s; DROP USER IF EXISTS %s@'localhost'; FLUSH PRIVILEGES;",
				sqlIdent(dbName), sqlString(dbUser))
			_ = exec.Command("mysql", "-e", sql).Run()
		}
		// Remove WP files only — leave the document root directory empty so the
		// site stays browsable / replaceable. We delete contents not the dir.
		entries, _ := os.ReadDir(root)
		for _, entry := range entries {
			_ = os.RemoveAll(filepath.Join(root, entry.Name()))
		}
		output = "WordPress install removed from " + root + " and database " + dbName + " dropped."
		redirect = "/wordpress"
	default:
		output = "Unknown action: " + action
		auditStatus = "failure"
	}

	a.audit(auditTag, auditStatus, domain)
	http.Redirect(w, r, redirect+"&output="+template.URLQueryEscaper(output), http.StatusSeeOther)
}

// WPInstallParams collects everything installWordPress needs. Exposed as a
// struct so the "one-click WordPress" path on Site → Add can reuse the exact
// same install logic the WordPress page uses.
type WPInstallParams struct {
	Domain     string
	Root       string // docroot; defaults to <WebRoot>/<Domain>/htdocs
	Title      string
	AdminUser  string
	AdminPass  string
	AdminEmail string
}

// installWordPress provisions the DB, runs wp-cli, rewrites the vhost, and
// fixes file ownership. Returns the combined log so the caller can surface it.
func (a *App) installWordPress(p WPInstallParams) (string, error) {
	if !domainRe.MatchString(p.Domain) || p.Title == "" || p.AdminUser == "" || p.AdminPass == "" || !strings.Contains(p.AdminEmail, "@") {
		return "Invalid WordPress install input", fmt.Errorf("invalid input")
	}
	if p.Root == "" {
		p.Root = filepath.Join(a.cfg.WebRoot, p.Domain, "htdocs")
	}
	_ = os.MkdirAll(p.Root, 0755)
	dbName := safeDBName(strings.ReplaceAll(p.Domain, ".", "_"))
	dbUser := dbName
	if len(dbUser) > 32 {
		dbUser = dbUser[:32]
	}
	dbPass := randomToken()[:24]
	sql := fmt.Sprintf("CREATE DATABASE IF NOT EXISTS %s CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci; CREATE USER IF NOT EXISTS %s@'localhost' IDENTIFIED BY %s; ALTER USER %s@'localhost' IDENTIFIED BY %s; GRANT ALL PRIVILEGES ON %s.* TO %s@'localhost'; FLUSH PRIVILEGES;",
		sqlIdent(dbName), sqlString(dbUser), sqlString(dbPass), sqlString(dbUser), sqlString(dbPass), sqlIdent(dbName), sqlString(dbUser))
	logs := []string{"Create document root..."}
	if err := exec.Command("mysql", "-e", sql).Run(); err != nil {
		a.audit("wordpress.install", "failure", p.Domain)
		return "Database setup failed: " + err.Error(), err
	}
	a.rememberCred(dbUser, dbPass, p.Domain)
	_ = a.attachDBToSite(p.Domain, dbName)
	logs = append(logs, "Database ready: "+dbName)
	steps := [][]string{
		{"wp", "core", "download", "--path=" + p.Root, "--force", "--allow-root"},
		{"wp", "config", "create", "--path=" + p.Root, "--dbname=" + dbName, "--dbuser=" + dbUser, "--dbpass=" + dbPass, "--dbhost=localhost", "--force", "--allow-root"},
		{"wp", "core", "install", "--path=" + p.Root, "--url=http://" + p.Domain, "--title=" + p.Title, "--admin_user=" + p.AdminUser, "--admin_password=" + p.AdminPass, "--admin_email=" + p.AdminEmail, "--skip-email", "--allow-root"},
	}
	for _, step := range steps {
		out, err := exec.Command(step[0], step[1:]...).CombinedOutput()
		logs = append(logs, strings.Join(step, " "))
		if len(out) > 0 {
			logs = append(logs, string(out))
		}
		if err != nil {
			a.audit("wordpress.install", "failure", p.Domain)
			return strings.Join(append(logs, "Failed: "+err.Error()), "\n"), err
		}
	}
	a.writeNginxSite(p.Domain, p.Root, false, "8.4")
	_ = exec.Command("chown", "-R", "www-data:www-data", filepath.Join(a.cfg.WebRoot, p.Domain)).Run()
	a.audit("wordpress.install", "success", p.Domain)
	logs = append(logs, "WordPress installed for "+p.Domain)
	return strings.Join(logs, "\n"), nil
}

func (a *App) wordpressInstall(w http.ResponseWriter, r *http.Request) {
	out, _ := a.installWordPress(WPInstallParams{
		Domain:     strings.ToLower(strings.TrimSpace(r.FormValue("domain"))),
		Title:      strings.TrimSpace(r.FormValue("title")),
		AdminUser:  safeName(r.FormValue("admin_user")),
		AdminPass:  strings.TrimSpace(r.FormValue("admin_pass")),
		AdminEmail: strings.TrimSpace(r.FormValue("admin_email")),
	})
	http.Redirect(w, r, "/wordpress?output="+template.URLQueryEscaper(out), http.StatusSeeOther)
}
