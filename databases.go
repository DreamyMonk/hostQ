package main

import (
	"fmt"
	"net/http"
	"os/exec"
	"regexp"
	"sort"
	"strings"
)

func (a *App) databases(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		a.databaseAction(w, r)
		return
	}
	a.render(w, "databases", map[string]any{
		"Title":     "Databases",
		"Databases": a.listDatabases(),
		"Created":   r.URL.Query().Get("created"),
		"User":      r.URL.Query().Get("user"),
		"Password":  r.URL.Query().Get("password"),
		"Site":      strings.TrimSpace(r.URL.Query().Get("site")),
	})
}

func (a *App) listDatabases() []DatabaseInfo {
	out, _ := exec.Command("mysql", "-N", "-B", "-e", "SHOW DATABASES").Output()
	dbs := []DatabaseInfo{}
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || line == "information_schema" || line == "performance_schema" || line == "mysql" || line == "sys" {
			continue
		}
		dbs = append(dbs, DatabaseInfo{Name: line})
	}
	sort.Slice(dbs, func(i, j int) bool { return dbs[i].Name < dbs[j].Name })
	return dbs
}

func safeDBName(name string) string {
	name = strings.ToLower(strings.TrimSpace(name))
	name = regexp.MustCompile(`[^a-z0-9_]+`).ReplaceAllString(name, "_")
	name = strings.Trim(name, "_")
	if len(name) > 48 {
		name = name[:48]
	}
	if name == "" {
		return ""
	}
	if strings.HasPrefix(name, "hostq_") {
		return name
	}
	return "hostq_" + name
}

func sqlIdent(name string) string {
	return "`" + strings.ReplaceAll(name, "`", "``") + "`"
}

func sqlString(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "''") + "'"
}

func (a *App) databaseAction(w http.ResponseWriter, r *http.Request) {
	action := r.FormValue("action")
	name := safeDBName(r.FormValue("name"))
	switch action {
	case "create":
		if name != "" {
			password := randomToken()[:24]
			user := name
			if len(user) > 32 {
				user = user[:32]
			}
			sql := fmt.Sprintf("CREATE DATABASE IF NOT EXISTS %s CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci; CREATE USER IF NOT EXISTS %s@'localhost' IDENTIFIED BY %s; ALTER USER %s@'localhost' IDENTIFIED BY %s; GRANT ALL PRIVILEGES ON %s.* TO %s@'localhost'; FLUSH PRIVILEGES;",
				sqlIdent(name), sqlString(user), sqlString(password), sqlString(user), sqlString(password), sqlIdent(name), sqlString(user))
			if err := exec.Command("mysql", "-e", sql).Run(); err == nil {
				a.audit("database.create", "success", name)
				http.Redirect(w, r, "/databases?created="+name+"&user="+user+"&password="+password, http.StatusSeeOther)
				return
			}
			a.audit("database.create", "failure", name)
		}
	case "delete":
		target := safeDBName(r.FormValue("target"))
		if target != "" {
			user := target
			if len(user) > 32 {
				user = user[:32]
			}
			sql := fmt.Sprintf("DROP DATABASE IF EXISTS %s; DROP USER IF EXISTS %s@'localhost'; FLUSH PRIVILEGES;", sqlIdent(target), sqlString(user))
			status := "failure"
			if err := exec.Command("mysql", "-e", sql).Run(); err == nil {
				status = "success"
			}
			a.audit("database.delete", status, target)
		}
	}
	http.Redirect(w, r, "/databases", http.StatusSeeOther)
}
