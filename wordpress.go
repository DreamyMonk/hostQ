package main

import (
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
	a.render(w, "wordpress", map[string]any{
		"Title":    "WordPress",
		"Installs": a.listWordPress(),
		"Output":   r.URL.Query().Get("output"),
		"Site":     strings.TrimSpace(r.URL.Query().Get("site")),
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
		installs = append(installs, WordPressInfo{Domain: domain, Path: root, Status: "installed"})
		return nil
	})
	sort.Slice(installs, func(i, j int) bool { return installs[i].Domain < installs[j].Domain })
	return installs
}

func (a *App) wordpressAction(w http.ResponseWriter, r *http.Request) {
	domain := strings.ToLower(strings.TrimSpace(r.FormValue("domain")))
	title := strings.TrimSpace(r.FormValue("title"))
	adminUser := safeName(r.FormValue("admin_user"))
	adminPass := strings.TrimSpace(r.FormValue("admin_pass"))
	adminEmail := strings.TrimSpace(r.FormValue("admin_email"))
	if !domainRe.MatchString(domain) || title == "" || adminUser == "" || adminPass == "" || !strings.Contains(adminEmail, "@") {
		http.Redirect(w, r, "/wordpress?output="+template.URLQueryEscaper("Invalid WordPress install input"), http.StatusSeeOther)
		return
	}
	root := filepath.Join(a.cfg.WebRoot, domain, "htdocs")
	_ = os.MkdirAll(root, 0755)
	dbName := safeDBName(strings.ReplaceAll(domain, ".", "_"))
	dbUser := dbName
	if len(dbUser) > 32 {
		dbUser = dbUser[:32]
	}
	dbPass := randomToken()[:24]
	sql := fmt.Sprintf("CREATE DATABASE IF NOT EXISTS %s CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci; CREATE USER IF NOT EXISTS %s@'localhost' IDENTIFIED BY %s; ALTER USER %s@'localhost' IDENTIFIED BY %s; GRANT ALL PRIVILEGES ON %s.* TO %s@'localhost'; FLUSH PRIVILEGES;",
		sqlIdent(dbName), sqlString(dbUser), sqlString(dbPass), sqlString(dbUser), sqlString(dbPass), sqlIdent(dbName), sqlString(dbUser))
	logs := []string{"Create document root..."}
	if err := exec.Command("mysql", "-e", sql).Run(); err != nil {
		logs = append(logs, "Database setup failed: "+err.Error())
		a.audit("wordpress.install", "failure", domain)
		http.Redirect(w, r, "/wordpress?output="+template.URLQueryEscaper(strings.Join(logs, "\n")), http.StatusSeeOther)
		return
	}
	logs = append(logs, "Database ready: "+dbName)
	steps := [][]string{
		{"wp", "core", "download", "--path=" + root, "--force", "--allow-root"},
		{"wp", "config", "create", "--path=" + root, "--dbname=" + dbName, "--dbuser=" + dbUser, "--dbpass=" + dbPass, "--dbhost=localhost", "--force", "--allow-root"},
		{"wp", "core", "install", "--path=" + root, "--url=http://" + domain, "--title=" + title, "--admin_user=" + adminUser, "--admin_password=" + adminPass, "--admin_email=" + adminEmail, "--skip-email", "--allow-root"},
	}
	for _, step := range steps {
		out, err := exec.Command(step[0], step[1:]...).CombinedOutput()
		logs = append(logs, strings.Join(step, " "))
		if len(out) > 0 {
			logs = append(logs, string(out))
		}
		if err != nil {
			logs = append(logs, "Failed: "+err.Error())
			a.audit("wordpress.install", "failure", domain)
			http.Redirect(w, r, "/wordpress?output="+template.URLQueryEscaper(strings.Join(logs, "\n")), http.StatusSeeOther)
			return
		}
	}
	a.writeNginxSite(domain, root, false, "8.4")
	_ = exec.Command("chown", "-R", "www-data:www-data", filepath.Join(a.cfg.WebRoot, domain)).Run()
	a.audit("wordpress.install", "success", domain)
	logs = append(logs, "WordPress installed for "+domain)
	http.Redirect(w, r, "/wordpress?output="+template.URLQueryEscaper(strings.Join(logs, "\n")), http.StatusSeeOther)
}
