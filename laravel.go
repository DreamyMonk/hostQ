package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Laravel support mirrors the one-shot WordPress installer: scaffold the app
// with Composer, provision a dedicated MySQL database + user, wire the
// credentials into .env, generate the app key, run the initial migrations and
// fix filesystem permissions. The web docroot is the app's public/ directory,
// and the existing Nginx front-controller (try_files … /index.php?$query_string)
// serves it without any extra config.

// composerAvailable reports whether the composer binary is on PATH.
func composerAvailable() bool {
	return exec.Command("composer", "--version").Run() == nil
}

// LaravelInstallParams collects what installLaravel needs. ProjectDir is the
// directory the Laravel app is created in; the web docroot becomes
// ProjectDir/public.
type LaravelInstallParams struct {
	Domain     string
	ProjectDir string // e.g. /var/www/<domain>/htdocs — app root, public/ lives under it
	PHPVersion string
}

// installLaravel scaffolds a Laravel app and returns the combined log so the
// caller can surface it on failure.
func (a *App) installLaravel(p LaravelInstallParams) (string, error) {
	if !domainRe.MatchString(p.Domain) {
		return "Invalid domain", fmt.Errorf("invalid domain")
	}
	if !composerAvailable() {
		return "Composer is not installed. Install it from Services & Packages first.", fmt.Errorf("composer missing")
	}
	if !phpVersionRe.MatchString(p.PHPVersion) {
		p.PHPVersion = "8.4"
	}
	if p.ProjectDir == "" {
		p.ProjectDir = filepath.Join(a.cfg.WebRoot, p.Domain, "htdocs")
	}
	if !a.canMutateWebPath(p.ProjectDir) {
		return "Refusing to scaffold outside the web root", fmt.Errorf("path outside web root")
	}
	_ = os.MkdirAll(p.ProjectDir, 0755)

	// Composer refuses to create-project into a non-empty directory. The blank
	// site shell may have dropped a placeholder index.html there — clear it.
	if entries, _ := os.ReadDir(p.ProjectDir); len(entries) > 0 {
		for _, e := range entries {
			_ = os.RemoveAll(filepath.Join(p.ProjectDir, e.Name()))
		}
	}

	logs := []string{"Scaffolding Laravel with Composer (this can take a minute)…"}
	env := append(os.Environ(), "COMPOSER_ALLOW_SUPERUSER=1", "COMPOSER_NO_INTERACTION=1", "HOME=/root")

	out, err := runIn(filepath.Dir(p.ProjectDir), env, "composer", "create-project", "laravel/laravel", filepath.Base(p.ProjectDir), "--prefer-dist", "--no-interaction")
	logs = append(logs, "composer create-project laravel/laravel", out)
	if err != nil {
		a.audit("laravel.install", "failure", p.Domain)
		return strings.Join(append(logs, "Failed: "+err.Error()), "\n"), err
	}

	// Dedicated database + user, same scheme the WordPress installer uses.
	dbName := safeDBName(strings.ReplaceAll(p.Domain, ".", "_"))
	dbUser := dbName
	if len(dbUser) > 32 {
		dbUser = dbUser[:32]
	}
	dbPass := randomToken()[:24]
	sql := fmt.Sprintf("CREATE DATABASE IF NOT EXISTS %s CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci; CREATE USER IF NOT EXISTS %s@'localhost' IDENTIFIED BY %s; ALTER USER %s@'localhost' IDENTIFIED BY %s; GRANT ALL PRIVILEGES ON %s.* TO %s@'localhost'; FLUSH PRIVILEGES;",
		sqlIdent(dbName), sqlString(dbUser), sqlString(dbPass), sqlString(dbUser), sqlString(dbPass), sqlIdent(dbName), sqlString(dbUser))
	if err := exec.Command("mysql", "-e", sql).Run(); err != nil {
		a.audit("laravel.install", "failure", p.Domain)
		return strings.Join(append(logs, "Database setup failed: "+err.Error()), "\n"), err
	}
	a.rememberCred(dbUser, dbPass, p.Domain)
	_ = a.attachDBToSite(p.Domain, dbName)
	logs = append(logs, "Database ready: "+dbName)

	// Wire DB creds + app URL into .env (created from .env.example by Composer).
	if err := setLaravelEnv(filepath.Join(p.ProjectDir, ".env"), map[string]string{
		"APP_URL":       "http://" + p.Domain,
		"DB_CONNECTION": "mysql",
		"DB_HOST":       "127.0.0.1",
		"DB_PORT":       "3306",
		"DB_DATABASE":   dbName,
		"DB_USERNAME":   dbUser,
		"DB_PASSWORD":   dbPass,
	}); err != nil {
		logs = append(logs, "Warning: could not update .env: "+err.Error())
	} else {
		logs = append(logs, "Wrote database credentials to .env")
	}

	// key:generate then the initial migrations. Migrations are best-effort —
	// a fresh Laravel app has none that can fail on a clean DB, but we don't
	// want a migration hiccup to fail the whole scaffold.
	if out, err := runIn(p.ProjectDir, env, "php", "artisan", "key:generate", "--force"); err != nil {
		logs = append(logs, "key:generate failed: "+err.Error(), out)
	} else {
		logs = append(logs, "Application key generated")
	}
	if out, err := runIn(p.ProjectDir, env, "php", "artisan", "migrate", "--force"); err != nil {
		logs = append(logs, "migrate skipped: "+strings.TrimSpace(out))
	} else {
		logs = append(logs, "Database migrated")
	}

	// Ownership + Laravel's writable dirs (storage, bootstrap/cache).
	_ = exec.Command("chown", "-R", "www-data:www-data", filepath.Join(a.cfg.WebRoot, p.Domain)).Run()
	_ = exec.Command("chmod", "-R", "ug+rwX", filepath.Join(p.ProjectDir, "storage"), filepath.Join(p.ProjectDir, "bootstrap", "cache")).Run()

	// Point the vhost at the app's public/ directory.
	public := filepath.Join(p.ProjectDir, "public")
	a.writeNginxSite(p.Domain, public, false, p.PHPVersion)
	a.audit("laravel.install", "success", p.Domain)
	logs = append(logs, "Laravel installed for "+p.Domain+" (docroot: "+public+")")
	return strings.Join(logs, "\n"), nil
}

// runIn runs a command in dir with the given environment and returns combined
// output. env may be nil to inherit the process environment.
func runIn(dir string, env []string, name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	if env != nil {
		cmd.Env = env
	}
	out, err := cmd.CombinedOutput()
	return string(out), err
}

// setLaravelEnv updates (or appends) KEY=value lines in a Laravel .env file.
// Values containing spaces or special characters are wrapped in double quotes,
// matching how Laravel writes them.
func setLaravelEnv(path string, kv map[string]string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	lines := strings.Split(string(data), "\n")
	seen := map[string]bool{}
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		eq := strings.Index(trimmed, "=")
		if eq <= 0 {
			continue
		}
		key := strings.TrimSpace(trimmed[:eq])
		if v, ok := kv[key]; ok {
			lines[i] = key + "=" + envQuote(v)
			seen[key] = true
		}
	}
	for key, v := range kv {
		if !seen[key] {
			lines = append(lines, key+"="+envQuote(v))
		}
	}
	return os.WriteFile(path, []byte(strings.Join(lines, "\n")), 0640)
}

// envQuote wraps a value in double quotes when it contains characters that
// would otherwise break dotenv parsing.
func envQuote(v string) string {
	if v == "" {
		return v
	}
	if strings.ContainsAny(v, " \t#\"'$") {
		return `"` + strings.ReplaceAll(v, `"`, `\"`) + `"`
	}
	return v
}

// listLaravel scans the web root for Laravel apps (an `artisan` file at the
// project root). Used to surface "Laravel detected" on a site's overview.
func (a *App) listLaravel() []LaravelInfo {
	installs := []LaravelInfo{}
	_ = filepath.Walk(a.cfg.WebRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil || info == nil {
			return nil
		}
		if info.IsDir() {
			// Don't descend into vendor trees — huge and never a project root.
			if info.Name() == "vendor" || info.Name() == "node_modules" {
				return filepath.SkipDir
			}
			return nil
		}
		if info.Name() != "artisan" {
			return nil
		}
		root := filepath.Dir(path)
		// Confirm it's really a Laravel app root, not a stray file.
		if _, e := os.Stat(filepath.Join(root, "composer.json")); e != nil {
			return nil
		}
		rel, _ := filepath.Rel(a.cfg.WebRoot, root)
		domain := rel
		if parts := strings.Split(rel, string(os.PathSeparator)); len(parts) > 0 {
			domain = parts[0]
		}
		installs = append(installs, LaravelInfo{
			Domain:  domain,
			Path:    root,
			Public:  filepath.Join(root, "public"),
			Version: laravelVersion(root),
		})
		return nil
	})
	return installs
}

// laravelVersion resolves the installed laravel/framework version via the
// app's artisan, falling back to empty when it can't be read.
func laravelVersion(root string) string {
	cmd := exec.Command("php", "artisan", "--version")
	cmd.Dir = root
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	// Output looks like "Laravel Framework 11.9.0".
	fields := strings.Fields(strings.TrimSpace(string(out)))
	if len(fields) > 0 {
		return fields[len(fields)-1]
	}
	return ""
}
