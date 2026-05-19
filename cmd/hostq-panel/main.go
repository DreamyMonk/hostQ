package main

import (
	"archive/tar"
	"compress/gzip"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
)

type Config struct {
	Addr          string
	DataDir       string
	WebRoot       string
	NginxSitesDir string
	JWTSecret     string
}

type Account struct {
	Username     string `json:"username"`
	PasswordHash string `json:"passwordHash"`
	Role         string `json:"role"`
}

type Site struct {
	Domain  string
	Root    string
	Enabled bool
	SSL     bool
	Cache   bool
}

type FileItem struct {
	Name string
	Kind string
	Path string
}

type DatabaseInfo struct {
	Name string
}

type CertInfo struct {
	Domain string
	Expiry string
	Days   int
	Status string
}

type Service struct {
	ID      string
	Name    string
	Systemd string
	Status  string
}

type App struct {
	cfg Config
	tpl *template.Template
}

var domainRe = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)

func main() {
	app := &App{
		cfg: Config{
			Addr:          env("HOSTQ_GO_ADDR", "127.0.0.1:8091"),
			DataDir:       env("HOSTQ_DATA_DIR", "/etc/hostq"),
			WebRoot:       env("WEB_ROOT", "/var/www"),
			NginxSitesDir: env("HOSTQ_NGINX_AVAILABLE", "/etc/nginx/sites-available"),
			JWTSecret:     env("JWT_SECRET", "change_this_hostq_go_secret"),
		},
	}
	if len(os.Args) > 1 && os.Args[1] == "init-admin" {
		if err := app.initAdmin(); err != nil {
			log.Fatal(err)
		}
		return
	}
	app.tpl = template.Must(template.New("hostq").Funcs(template.FuncMap{
		"now": time.Now,
	}).Parse(layoutTemplate))

	mux := http.NewServeMux()
	mux.HandleFunc("/", app.requireAuth(app.dashboard))
	mux.HandleFunc("/login", app.login)
	mux.HandleFunc("/logout", app.logout)
	mux.HandleFunc("/sites", app.requireAuth(app.sites))
	mux.HandleFunc("/site-action", app.requireAuth(app.siteAction))
	mux.HandleFunc("/files", app.requireAuth(app.files))
	mux.HandleFunc("/databases", app.requireAuth(app.databases))
	mux.HandleFunc("/ssl", app.requireAuth(app.ssl))
	mux.HandleFunc("/services", app.requireAuth(app.services))
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte("ok")) })

	log.Printf("hostQ Go panel listening on http://%s", app.cfg.Addr)
	log.Fatal(http.ListenAndServe(app.cfg.Addr, securityHeaders(mux)))
}

func env(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}

func (a *App) initAdmin() error {
	username := env("HOSTQ_ADMIN_USER", "admin")
	password := env("HOSTQ_ADMIN_PASS", "")
	if password == "" {
		password = randomToken()[:20]
	}
	if err := os.MkdirAll(a.cfg.DataDir, 0700); err != nil {
		return err
	}
	adminPath := filepath.Join(a.cfg.DataDir, "admin.json")
	if _, err := os.Stat(adminPath); err == nil {
		fmt.Println("Existing admin account found; not regenerating credentials.")
		return nil
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	if err != nil {
		return err
	}
	account := map[string]any{
		"username":     username,
		"passwordHash": string(hash),
		"role":         "admin",
		"otpEnabled":   false,
		"createdAt":    time.Now().Format(time.RFC3339),
	}
	data, _ := json.MarshalIndent(account, "", "  ")
	if err := os.WriteFile(adminPath, data, 0600); err != nil {
		return err
	}
	fmt.Println("Initial hostQ admin login:")
	fmt.Println("  Username:", username)
	fmt.Println("  Password:", password)
	return nil
}

func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "same-origin")
		w.Header().Set("Content-Security-Policy", "default-src 'self'; style-src 'self' 'unsafe-inline'; frame-ancestors 'none'; base-uri 'self'; form-action 'self'")
		next.ServeHTTP(w, r)
	})
}

func (a *App) readAccount() (*Account, error) {
	data, err := os.ReadFile(filepath.Join(a.cfg.DataDir, "admin.json"))
	if err != nil {
		return nil, err
	}
	var account Account
	if err := json.Unmarshal(data, &account); err != nil {
		return nil, err
	}
	if account.Role == "" {
		account.Role = "admin"
	}
	return &account, nil
}

func randomToken() string {
	buf := make([]byte, 32)
	_, _ = rand.Read(buf)
	return base64.RawURLEncoding.EncodeToString(buf)
}

func (a *App) sign(value string) string {
	mac := hmac.New(sha256.New, []byte(a.cfg.JWTSecret))
	mac.Write([]byte(value))
	return hex.EncodeToString(mac.Sum(nil))
}

func (a *App) sessionCookie(username string) *http.Cookie {
	payload := fmt.Sprintf("%s:%d:%s", username, time.Now().Add(24*time.Hour).Unix(), randomToken())
	return &http.Cookie{
		Name:     "hostq_go_session",
		Value:    base64.RawURLEncoding.EncodeToString([]byte(payload)) + "." + a.sign(payload),
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   86400,
	}
}

func (a *App) verifySession(r *http.Request) bool {
	cookie, err := r.Cookie("hostq_go_session")
	if err != nil {
		return false
	}
	parts := strings.Split(cookie.Value, ".")
	if len(parts) != 2 {
		return false
	}
	raw, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return false
	}
	payload := string(raw)
	if subtle.ConstantTimeCompare([]byte(a.sign(payload)), []byte(parts[1])) != 1 {
		return false
	}
	fields := strings.Split(payload, ":")
	if len(fields) < 2 {
		return false
	}
	expires, err := strconv.ParseInt(fields[1], 10, 64)
	return err == nil && time.Now().Unix() < expires
}

func (a *App) requireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !a.verifySession(r) {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}
		next(w, r)
	}
}

func (a *App) login(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		a.render(w, "login", map[string]any{"Title": "Sign in"})
		return
	}
	account, err := a.readAccount()
	if err != nil {
		a.render(w, "login", map[string]any{"Title": "Sign in", "Error": "Admin account is not initialized. Run setup.sh first."})
		return
	}
	username := r.FormValue("username")
	password := r.FormValue("password")
	if username != account.Username || bcrypt.CompareHashAndPassword([]byte(account.PasswordHash), []byte(password)) != nil {
		a.audit("auth.login", "failure", username)
		a.render(w, "login", map[string]any{"Title": "Sign in", "Error": "Invalid username or password"})
		return
	}
	http.SetCookie(w, a.sessionCookie(username))
	a.audit("auth.login", "success", username)
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (a *App) logout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{Name: "hostq_go_session", Value: "", Path: "/", MaxAge: -1})
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

func (a *App) dashboard(w http.ResponseWriter, _ *http.Request) {
	sites := a.listSites()
	services := a.listServices()
	a.render(w, "dashboard", map[string]any{"Title": "Dashboard", "Sites": sites, "Services": services})
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
			_ = os.WriteFile(index, []byte("<h1>"+template.HTMLEscapeString(domain)+"</h1><p>Managed by hostQ Go</p>"), 0644)
		}
		a.writeNginxSite(domain, root, false)
		a.audit("site.create", "success", domain)
		http.Redirect(w, r, "/sites", http.StatusSeeOther)
		return
	}
	a.render(w, "sites", map[string]any{"Title": "Sites", "Sites": a.listSites()})
}

func (a *App) files(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		a.fileAction(w, r)
		return
	}
	reqPath := filepath.Clean("/" + strings.TrimPrefix(r.URL.Query().Get("path"), "/"))
	full := a.safeWebPath(reqPath)
	entries, _ := os.ReadDir(full)
	items := []FileItem{}
	for _, entry := range entries {
		if blockedFileName(entry.Name()) {
			continue
		}
		kind := "file"
		if entry.IsDir() {
			kind = "dir"
		}
		items = append(items, FileItem{Name: entry.Name(), Kind: kind, Path: filepath.Join(reqPath, entry.Name())})
	}
	a.render(w, "files", map[string]any{"Title": "Files", "Path": reqPath, "Items": items})
}

func (a *App) safeWebPath(reqPath string) string {
	clean := filepath.Clean("/" + strings.TrimPrefix(reqPath, "/"))
	full := filepath.Join(a.cfg.WebRoot, clean)
	root := filepath.Clean(a.cfg.WebRoot)
	if full != root && !strings.HasPrefix(full, root+string(os.PathSeparator)) {
		return root
	}
	return full
}

func safeName(name string) string {
	name = strings.TrimSpace(name)
	name = strings.ReplaceAll(name, "/", "-")
	name = strings.ReplaceAll(name, "\\", "-")
	name = regexp.MustCompile(`[^a-zA-Z0-9._ -]+`).ReplaceAllString(name, "-")
	return strings.Trim(name, ". ")
}

func blockedFileName(name string) bool {
	lower := strings.ToLower(name)
	return strings.HasPrefix(lower, ".env") ||
		strings.Contains(lower, ".key") ||
		strings.Contains(lower, "id_rsa") ||
		strings.Contains(lower, "id_ed25519") ||
		strings.HasSuffix(lower, ".pem") ||
		strings.HasSuffix(lower, ".p12") ||
		strings.HasSuffix(lower, ".pfx")
}

func (a *App) canMutateWebPath(path string) bool {
	root := filepath.Clean(a.cfg.WebRoot)
	clean := filepath.Clean(path)
	if clean == root || !strings.HasPrefix(clean, root+string(os.PathSeparator)) {
		return false
	}
	return !blockedFileName(filepath.Base(clean))
}

func (a *App) fileAction(w http.ResponseWriter, r *http.Request) {
	basePath := r.FormValue("path")
	full := a.safeWebPath(basePath)
	action := r.FormValue("action")
	name := safeName(r.FormValue("name"))
	target := filepath.Join(full, name)
	if name == "" && (action == "mkdir" || action == "touch") {
		http.Redirect(w, r, "/files?path="+basePath, http.StatusSeeOther)
		return
	}
	switch action {
	case "mkdir":
		if !a.canMutateWebPath(target) {
			break
		}
		_ = os.MkdirAll(target, 0755)
		a.audit("file.mkdir", "success", target)
	case "touch":
		if !a.canMutateWebPath(target) {
			break
		}
		f, err := os.OpenFile(target, os.O_CREATE|os.O_EXCL, 0644)
		if err == nil {
			_ = f.Close()
		}
		a.audit("file.create", "success", target)
	case "delete":
		deletePath := a.safeWebPath(r.FormValue("target"))
		if !a.canMutateWebPath(deletePath) {
			break
		}
		trash := filepath.Join(a.cfg.WebRoot, ".hostq-trash")
		_ = os.MkdirAll(trash, 0700)
		_ = os.Rename(deletePath, filepath.Join(trash, fmt.Sprintf("%d-%s", time.Now().Unix(), filepath.Base(deletePath))))
		a.audit("file.soft_delete", "success", deletePath)
	case "chmod":
		chmodPath := a.safeWebPath(r.FormValue("target"))
		if !a.canMutateWebPath(chmodPath) {
			break
		}
		mode := r.FormValue("mode")
		if regexp.MustCompile(`^[0-7]{3,4}$`).MatchString(mode) {
			parsed, _ := strconv.ParseUint(mode, 8, 32)
			_ = os.Chmod(chmodPath, os.FileMode(parsed))
			a.audit("file.chmod", "success", chmodPath)
		}
	}
	http.Redirect(w, r, "/files?path="+basePath, http.StatusSeeOther)
}

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
	a.writeNginxSite(site.Domain, site.Root, site.Cache)
	return fmt.Sprintf("Removed stale SSL references from %s. Backup: %s\n", configPath, backupPath)
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}

func (a *App) listCertificates() []CertInfo {
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

func (a *App) services(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		id := r.FormValue("id")
		action := r.FormValue("action")
		if allowedServiceAction(id, action) {
			_ = exec.Command("systemctl", action, serviceMap()[id]).Run()
			a.audit("service."+action, "success", id)
		}
		http.Redirect(w, r, "/services", http.StatusSeeOther)
		return
	}
	a.render(w, "services", map[string]any{"Title": "Services", "Services": a.listServices()})
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
		a.writeNginxSite(site.Domain, site.Root, true)
	case "cache-off":
		a.writeNginxSite(site.Domain, site.Root, false)
	case "permissions":
		_ = exec.Command("chown", "-R", "www-data:www-data", filepath.Join(a.cfg.WebRoot, domain)).Run()
		_ = exec.Command("find", filepath.Join(a.cfg.WebRoot, domain), "-type", "d", "-exec", "chmod", "755", "{}", ";").Run()
		_ = exec.Command("find", filepath.Join(a.cfg.WebRoot, domain), "-type", "f", "-exec", "chmod", "644", "{}", ";").Run()
	case "backup":
		_ = a.backupSite(site)
	case "delete":
		_ = os.Remove(filepath.Join("/etc/nginx/sites-enabled", domain))
		_ = os.Remove(filepath.Join(a.cfg.NginxSitesDir, domain))
		siteBase := filepath.Dir(site.Root)
		if a.canMutateWebPath(siteBase) {
			trash := filepath.Join(a.cfg.WebRoot, ".hostq-trash")
			_ = os.MkdirAll(trash, 0700)
			_ = os.Rename(siteBase, filepath.Join(trash, fmt.Sprintf("%d-%s", time.Now().Unix(), filepath.Base(siteBase))))
		}
		_ = exec.Command("systemctl", "reload", "nginx").Run()
	}
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

func (a *App) backupSite(site Site) error {
	backupDir := "/var/backups/hostq"
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		return err
	}
	target := filepath.Join(backupDir, fmt.Sprintf("%s-%s.tar.gz", site.Domain, time.Now().Format("2006-01-02-150405")))
	file, err := os.Create(target)
	if err != nil {
		return err
	}
	defer file.Close()
	gz := gzip.NewWriter(file)
	defer gz.Close()
	tw := tar.NewWriter(gz)
	defer tw.Close()
	return filepath.Walk(site.Root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		header, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return nil
		}
		rel, _ := filepath.Rel(site.Root, path)
		header.Name = rel
		if err := tw.WriteHeader(header); err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		src, err := os.Open(path)
		if err != nil {
			return nil
		}
		defer src.Close()
		_, err = io.Copy(tw, src)
		return err
	})
}

func serviceMap() map[string]string {
	return map[string]string{
		"nginx": "nginx", "mariadb": "mariadb", "redis": "redis-server",
		"php84": "php8.4-fpm", "pureftpd": "pure-ftpd",
	}
}

func allowedServiceAction(id, action string) bool {
	if _, ok := serviceMap()[id]; !ok {
		return false
	}
	return action == "start" || action == "stop" || action == "restart"
}

func (a *App) listServices() []Service {
	labels := map[string]string{"nginx": "Nginx", "mariadb": "MariaDB", "redis": "Redis", "php84": "PHP 8.4-FPM", "pureftpd": "Pure-FTPd"}
	out := []Service{}
	for id, systemd := range serviceMap() {
		status := "unknown"
		if data, err := exec.Command("systemctl", "is-active", systemd).Output(); err == nil {
			status = strings.TrimSpace(string(data))
		}
		out = append(out, Service{ID: id, Name: labels[id], Systemd: systemd, Status: status})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

func (a *App) listSites() []Site {
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
		if domain == "" {
			domain = entry.Name()
		}
		_, enabledErr := os.Stat(filepath.Join("/etc/nginx/sites-enabled", entry.Name()))
		sites = append(sites, Site{
			Domain: domain, Root: root, Enabled: enabledErr == nil,
			SSL:   strings.Contains(text, "ssl_certificate") && a.certExists(domain),
			Cache: strings.Contains(text, "hostQ fastcgi cache: on"),
		})
	}
	sort.Slice(sites, func(i, j int) bool { return sites[i].Domain < sites[j].Domain })
	return sites
}

func firstMatch(text, expr string) string {
	match := regexp.MustCompile(expr).FindStringSubmatch(text)
	if len(match) > 1 {
		return strings.TrimSpace(match[1])
	}
	return ""
}

func (a *App) writeNginxSite(domain, root string, cache bool) {
	cacheBlock := ""
	if cache {
		cacheBlock = `
        fastcgi_cache HOSTQ_FASTCGI;
        fastcgi_cache_valid 200 301 302 10m;
        add_header X-hostQ-Cache $upstream_cache_status always;`
	}
	conf := fmt.Sprintf(`# hostQ managed - %s
# hostQ fastcgi cache: %s
server {
    listen 80;
    server_name %s www.%s;
    root %s;
    index index.php index.html;
    location / { try_files $uri $uri/ /index.php?$query_string; }
    location ~ \.php$ {
        include snippets/fastcgi-php.conf;
        fastcgi_pass unix:/run/php/php8.4-fpm.sock;
        fastcgi_param SCRIPT_FILENAME $realpath_root$fastcgi_script_name;
        include fastcgi_params;%s
    }
}
`, domain, map[bool]string{true: "on", false: "off"}[cache], domain, domain, root, cacheBlock)
	_ = os.WriteFile(filepath.Join(a.cfg.NginxSitesDir, domain), []byte(conf), 0644)
	_ = os.Remove(filepath.Join("/etc/nginx/sites-enabled", domain))
	_ = os.Symlink(filepath.Join(a.cfg.NginxSitesDir, domain), filepath.Join("/etc/nginx/sites-enabled", domain))
	_ = exec.Command("nginx", "-t").Run()
	_ = exec.Command("systemctl", "reload", "nginx").Run()
}

func (a *App) audit(action, status, target string) {
	_ = os.MkdirAll(a.cfg.DataDir, 0700)
	file := filepath.Join(a.cfg.DataDir, "audit-go.log")
	line, _ := json.Marshal(map[string]string{"ts": time.Now().Format(time.RFC3339), "action": action, "status": status, "target": target})
	f, err := os.OpenFile(file, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
	if err == nil {
		defer f.Close()
		_, _ = f.Write(append(line, '\n'))
	}
}

func (a *App) render(w http.ResponseWriter, view string, data map[string]any) {
	data["View"] = view
	if err := a.tpl.ExecuteTemplate(w, "layout", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

const layoutTemplate = `
{{define "layout"}}
<!doctype html><html><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1">
<title>hostQ Go - {{.Title}}</title>
<style>
body{margin:0;font-family:Inter,system-ui,Segoe UI,sans-serif;background:#f5f7fb;color:#111827}a{color:inherit;text-decoration:none}.shell{display:grid;grid-template-columns:230px 1fr;min-height:100vh}.side{background:#fff;border-right:1px solid #e5e7eb;padding:18px}.brand{font-size:22px;font-weight:850;margin-bottom:24px}.nav a{display:block;padding:10px 12px;border-radius:8px;margin-bottom:6px;color:#475467}.nav a:hover{background:#eef4ff;color:#2563eb}.main{padding:24px}.top{display:flex;justify-content:space-between;align-items:center;margin-bottom:20px}.card{background:#fff;border:1px solid #e5e7eb;border-radius:8px;padding:16px;margin-bottom:14px}.grid{display:grid;grid-template-columns:repeat(auto-fit,minmax(240px,1fr));gap:12px}.btn{border:1px solid #cbd5e1;border-radius:7px;background:#fff;padding:8px 11px;font-weight:700;cursor:pointer}.primary{background:#2563eb;color:#fff;border-color:#2563eb}.danger{color:#dc2626}.input{width:100%;box-sizing:border-box;border:1px solid #cbd5e1;border-radius:7px;padding:10px}.muted{color:#667085;font-size:13px}.badge{display:inline-block;border:1px solid #cbd5e1;border-radius:999px;padding:3px 8px;font-size:12px}.ok{color:#15803d}.bad{color:#dc2626}.actions{display:flex;gap:6px;flex-wrap:wrap}.inline{display:inline-flex;gap:6px;align-items:center}.mini{padding:6px 8px;font-size:12px}.mono{font-family:ui-monospace,SFMono-Regular,Consolas,monospace}table{width:100%;border-collapse:collapse;background:#fff}td,th{border-bottom:1px solid #e5e7eb;padding:10px;text-align:left;font-size:13px}.login{max-width:380px;margin:12vh auto}
</style></head><body>{{if eq .View "login"}}{{template "login" .}}{{else}}<div class="shell"><aside class="side"><div class="brand">hostQ Go</div><nav class="nav"><a href="/">Dashboard</a><a href="/sites">Sites</a><a href="/files">Files</a><a href="/databases">Databases</a><a href="/ssl">SSL</a><a href="/services">Services</a><a href="/logout">Logout</a></nav></aside><main class="main">{{if eq .View "dashboard"}}{{template "dashboard" .}}{{else if eq .View "sites"}}{{template "sites" .}}{{else if eq .View "files"}}{{template "files" .}}{{else if eq .View "databases"}}{{template "databases" .}}{{else if eq .View "ssl"}}{{template "ssl" .}}{{else if eq .View "services"}}{{template "services" .}}{{end}}</main></div>{{end}}</body></html>
{{end}}
{{define "login"}}<div class="login card"><h1>hostQ Go</h1><p class="muted">Lightweight control panel preview</p>{{if .Error}}<div class="card bad">{{.Error}}</div>{{end}}<form method="post"><p><input class="input" name="username" placeholder="Username" autocomplete="username"></p><p><input class="input" name="password" type="password" placeholder="Password" autocomplete="current-password"></p><button class="btn primary" type="submit">Sign in</button></form></div>{{end}}
{{define "dashboard"}}<div class="top"><div><h1>Dashboard</h1><p class="muted">Go runtime preview for low-memory VPS deployments</p></div><span class="badge">{{now.Format "Jan 02 15:04"}}</span></div><div class="grid"><div class="card"><h2>{{len .Sites}}</h2><p class="muted">Sites</p></div><div class="card"><h2>{{len .Services}}</h2><p class="muted">Services tracked</p></div></div>{{end}}
{{define "sites"}}<div class="top"><h1>Sites</h1></div>{{if .Error}}<div class="card bad">{{.Error}}</div>{{end}}<div class="card"><form method="post"><div class="grid"><input class="input" name="domain" placeholder="example.com"><button class="btn primary">Add PHP Site</button></div><p class="muted">Each site uses /var/www/domain/htdocs so website files stay separate from logs, backups, and private data.</p></form></div><table><tr><th>Domain</th><th>Root</th><th>Status</th><th>Cache</th><th>Actions</th></tr>{{range .Sites}}<tr><td>{{.Domain}}</td><td class="muted mono">{{.Root}}</td><td>{{if .Enabled}}<span class="ok">enabled</span>{{else}}<span class="bad">disabled</span>{{end}}</td><td>{{if .Cache}}on{{else}}off{{end}}</td><td><div class="actions"><form method="post" action="/site-action"><input type="hidden" name="domain" value="{{.Domain}}">{{if .Enabled}}<button class="btn mini" name="action" value="disable">Disable</button>{{else}}<button class="btn mini" name="action" value="enable">Enable</button>{{end}}</form><form method="post" action="/site-action"><input type="hidden" name="domain" value="{{.Domain}}">{{if .Cache}}<button class="btn mini" name="action" value="cache-off">Cache off</button>{{else}}<button class="btn mini" name="action" value="cache-on">Cache on</button>{{end}}</form><form method="post" action="/site-action"><input type="hidden" name="domain" value="{{.Domain}}"><button class="btn mini" name="action" value="permissions">Fix permissions</button></form><form method="post" action="/site-action"><input type="hidden" name="domain" value="{{.Domain}}"><button class="btn mini" name="action" value="backup">Backup</button></form><form method="post" action="/site-action"><input type="hidden" name="domain" value="{{.Domain}}"><button class="btn mini danger" name="action" value="delete">Soft delete</button></form></div></td></tr>{{end}}</table>{{end}}
{{define "files"}}<div class="top"><h1>Files</h1><span class="badge mono">{{.Path}}</span></div><div class="card"><form class="grid" method="post"><input type="hidden" name="path" value="{{.Path}}"><input class="input" name="name" placeholder="folder-or-file-name"><div class="actions"><button class="btn primary" name="action" value="mkdir">New folder</button><button class="btn" name="action" value="touch">New file</button></div></form><p class="muted">Secret files such as .env, private keys, and certificate bundles are hidden and blocked by default.</p></div><table><tr><th>Name</th><th>Type</th><th>Actions</th></tr>{{range .Items}}<tr><td>{{if eq .Kind "dir"}}<a href="/files?path={{.Path}}">{{.Name}}</a>{{else}}{{.Name}}{{end}}</td><td>{{.Kind}}</td><td><div class="actions"><form method="post"><input type="hidden" name="path" value="{{$.Path}}"><input type="hidden" name="target" value="{{.Path}}"><input class="input mini mono" name="mode" placeholder="755" style="width:80px"><button class="btn mini" name="action" value="chmod">Chmod</button></form><form method="post"><input type="hidden" name="path" value="{{$.Path}}"><input type="hidden" name="target" value="{{.Path}}"><button class="btn mini danger" name="action" value="delete">Soft delete</button></form></div></td></tr>{{end}}</table>{{end}}
{{define "databases"}}<div class="top"><h1>Databases</h1><span class="badge">MariaDB/MySQL</span></div>{{if .Created}}<div class="card ok"><b>Database created:</b> <span class="mono">{{.Created}}</span><br><b>User:</b> <span class="mono">{{.User}}</span><br><b>Password:</b> <span class="mono">{{.Password}}</span><p class="muted">Save this password now. It is shown only once.</p></div>{{end}}<div class="card"><form method="post" class="grid"><input class="input" name="name" placeholder="project_name"><button class="btn primary" name="action" value="create">Create database</button></form></div><table><tr><th>Database</th><th>Actions</th></tr>{{range .Databases}}<tr><td class="mono">{{.Name}}</td><td><form method="post"><input type="hidden" name="target" value="{{.Name}}"><button class="btn mini danger" name="action" value="delete">Delete database</button></form></td></tr>{{else}}<tr><td class="muted" colspan="2">No user databases found, or mysql CLI is not available to the panel user.</td></tr>{{end}}</table>{{end}}
{{define "ssl"}}<div class="top"><h1>SSL</h1><span class="badge">Let's Encrypt</span></div>{{if .Output}}<pre class="card mono" style="white-space:pre-wrap">{{.Output}}</pre>{{end}}<div class="card"><form method="post" class="grid"><input class="input" name="domain" placeholder="example.com"><input class="input" name="email" placeholder="admin@example.com"><button class="btn primary" name="action" value="issue">Install SSL</button><button class="btn" name="action" value="renew">Renew</button><button class="btn danger" name="action" value="delete">Delete cert</button></form><p class="muted">Install SSL repairs stale Nginx certificate references before running certbot, so missing /etc/letsencrypt files do not block reinstall.</p></div><table><tr><th>Certificate</th><th>Expiry</th><th>Status</th><th>Days left</th></tr>{{range .Certificates}}<tr><td class="mono">{{.Domain}}</td><td>{{.Expiry}}</td><td>{{.Status}}</td><td>{{.Days}}d</td></tr>{{else}}<tr><td class="muted" colspan="4">No certificates found, or certbot is not installed.</td></tr>{{end}}</table>{{end}}
{{define "services"}}<div class="top"><h1>Services</h1></div><table><tr><th>Name</th><th>Status</th><th>Actions</th></tr>{{range .Services}}<tr><td>{{.Name}}</td><td>{{.Status}}</td><td><form method="post" style="display:flex;gap:6px"><input type="hidden" name="id" value="{{.ID}}"><button class="btn" name="action" value="restart">Restart</button><button class="btn" name="action" value="start">Start</button><button class="btn danger" name="action" value="stop">Stop</button></form></td></tr>{{end}}</table>{{end}}
`
