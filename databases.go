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
	siteFilter := strings.TrimSpace(r.URL.Query().Get("site"))
	a.render(w, "databases", map[string]any{
		"Title":     "Databases",
		"Databases": a.listDatabasesForSite(siteFilter),
		"Created":   r.URL.Query().Get("created"),
		"User":      r.URL.Query().Get("user"),
		"Password":  r.URL.Query().Get("password"),
		"Site":      siteFilter,
	})
}

// listDatabases returns all panel-managed databases.
func (a *App) listDatabases() []DatabaseInfo {
	out, _ := exec.Command("mysql", "-N", "-B", "-e", "SHOW DATABASES").Output()
	dbs := []DatabaseInfo{}
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || line == "information_schema" || line == "performance_schema" || line == "mysql" || line == "sys" {
			continue
		}
		dbs = append(dbs, DatabaseInfo{Name: line, Users: a.listDBUsers(line)})
	}
	sort.Slice(dbs, func(i, j int) bool { return dbs[i].Name < dbs[j].Name })
	return dbs
}

// listDatabasesForSite filters databases to those that match the site's
// auto-generated prefix when site is non-empty. Pattern: hostq_<domain_underscored>.
func (a *App) listDatabasesForSite(site string) []DatabaseInfo {
	all := a.listDatabases()
	if site == "" {
		return all
	}
	prefix := dbPrefixForSite(site)
	matched := []DatabaseInfo{}
	for _, db := range all {
		if db.Name == prefix || strings.HasPrefix(db.Name, prefix+"_") {
			matched = append(matched, db)
		}
	}
	return matched
}

func dbPrefixForSite(site string) string {
	return safeDBName(strings.ReplaceAll(site, ".", "_"))
}

func (a *App) listDBUsers(db string) []DBUser {
	if db == "" {
		return nil
	}
	sql := fmt.Sprintf("SELECT DISTINCT User, Host FROM mysql.db WHERE Db=%s", sqlString(db))
	out, err := exec.Command("mysql", "-N", "-B", "-e", sql).Output()
	if err != nil {
		return nil
	}
	users := []DBUser{}
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		users = append(users, DBUser{Login: fields[0], Host: fields[1]})
	}
	return users
}

var dbUserRe = regexp.MustCompile(`^[a-zA-Z0-9_]{1,32}$`)

func safeDBUser(name string) string {
	name = strings.ToLower(strings.TrimSpace(name))
	name = regexp.MustCompile(`[^a-z0-9_]+`).ReplaceAllString(name, "_")
	name = strings.Trim(name, "_")
	if len(name) > 32 {
		name = name[:32]
	}
	if !dbUserRe.MatchString(name) {
		return ""
	}
	return name
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
	site := strings.TrimSpace(r.FormValue("site"))
	redirectURL := "/databases"
	if site != "" {
		redirectURL = "/site?domain=" + site + "&tab=database"
	}

	switch action {
	case "create":
		name := safeDBName(r.FormValue("name"))
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
				http.Redirect(w, r, redirectURL+"&created="+name+"&user="+user+"&password="+password, http.StatusSeeOther)
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
	case "user-create":
		db := safeDBName(r.FormValue("target"))
		user := safeDBUser(r.FormValue("user"))
		pass := strings.TrimSpace(r.FormValue("password"))
		if pass == "" {
			pass = randomToken()[:20]
		}
		if db != "" && user != "" && len(pass) >= 8 {
			sql := fmt.Sprintf("CREATE USER IF NOT EXISTS %s@'localhost' IDENTIFIED BY %s; ALTER USER %s@'localhost' IDENTIFIED BY %s; GRANT ALL PRIVILEGES ON %s.* TO %s@'localhost'; FLUSH PRIVILEGES;",
				sqlString(user), sqlString(pass), sqlString(user), sqlString(pass), sqlIdent(db), sqlString(user))
			status := "failure"
			if err := exec.Command("mysql", "-e", sql).Run(); err == nil {
				status = "success"
			}
			a.audit("database.user-create", status, db+"/"+user)
			http.Redirect(w, r, redirectURL+"&dbuser="+user+"&dbpass="+pass+"&db="+db, http.StatusSeeOther)
			return
		}
	case "user-password":
		user := safeDBUser(r.FormValue("user"))
		pass := strings.TrimSpace(r.FormValue("password"))
		if user != "" && len(pass) >= 8 {
			sql := fmt.Sprintf("ALTER USER %s@'localhost' IDENTIFIED BY %s; FLUSH PRIVILEGES;", sqlString(user), sqlString(pass))
			status := "failure"
			if err := exec.Command("mysql", "-e", sql).Run(); err == nil {
				status = "success"
			}
			a.audit("database.user-password", status, user)
			http.Redirect(w, r, redirectURL+"&dbuser="+user+"&dbpass="+pass, http.StatusSeeOther)
			return
		}
	case "user-delete":
		user := safeDBUser(r.FormValue("user"))
		db := safeDBName(r.FormValue("target"))
		if user != "" {
			sql := ""
			if db != "" {
				sql = fmt.Sprintf("REVOKE ALL PRIVILEGES ON %s.* FROM %s@'localhost'; ", sqlIdent(db), sqlString(user))
			}
			sql += fmt.Sprintf("DROP USER IF EXISTS %s@'localhost'; FLUSH PRIVILEGES;", sqlString(user))
			status := "failure"
			if err := exec.Command("mysql", "-e", sql).Run(); err == nil {
				status = "success"
			}
			a.audit("database.user-delete", status, user)
		}
	case "user-grant":
		user := safeDBUser(r.FormValue("user"))
		db := safeDBName(r.FormValue("target"))
		if user != "" && db != "" {
			sql := fmt.Sprintf("GRANT ALL PRIVILEGES ON %s.* TO %s@'localhost'; FLUSH PRIVILEGES;", sqlIdent(db), sqlString(user))
			status := "failure"
			if err := exec.Command("mysql", "-e", sql).Run(); err == nil {
				status = "success"
			}
			a.audit("database.user-grant", status, db+"/"+user)
		}
	}
	http.Redirect(w, r, redirectURL, http.StatusSeeOther)
}
