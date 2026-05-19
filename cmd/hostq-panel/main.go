package main

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"html/template"
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
	app.tpl = template.Must(template.New("hostq").Funcs(template.FuncMap{
		"now": time.Now,
	}).Parse(layoutTemplate))

	mux := http.NewServeMux()
	mux.HandleFunc("/", app.requireAuth(app.dashboard))
	mux.HandleFunc("/login", app.login)
	mux.HandleFunc("/logout", app.logout)
	mux.HandleFunc("/sites", app.requireAuth(app.sites))
	mux.HandleFunc("/files", app.requireAuth(app.files))
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
	reqPath := filepath.Clean("/" + strings.TrimPrefix(r.URL.Query().Get("path"), "/"))
	full := filepath.Join(a.cfg.WebRoot, reqPath)
	if !strings.HasPrefix(full, filepath.Clean(a.cfg.WebRoot)) {
		full = a.cfg.WebRoot
	}
	entries, _ := os.ReadDir(full)
	type item struct{ Name, Kind, Path string }
	items := []item{}
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), ".env") || strings.Contains(entry.Name(), ".key") {
			continue
		}
		kind := "file"
		if entry.IsDir() {
			kind = "dir"
		}
		items = append(items, item{Name: entry.Name(), Kind: kind, Path: filepath.Join(reqPath, entry.Name())})
	}
	a.render(w, "files", map[string]any{"Title": "Files", "Path": reqPath, "Items": items})
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
			SSL: strings.Contains(text, "ssl_certificate"),
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
        include fastcgi_params;
    }
}
`, domain, map[bool]string{true: "on", false: "off"}[cache], domain, domain, root)
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
body{margin:0;font-family:Inter,system-ui,Segoe UI,sans-serif;background:#f5f7fb;color:#111827}a{color:inherit;text-decoration:none}.shell{display:grid;grid-template-columns:230px 1fr;min-height:100vh}.side{background:#fff;border-right:1px solid #e5e7eb;padding:18px}.brand{font-size:22px;font-weight:850;margin-bottom:24px}.nav a{display:block;padding:10px 12px;border-radius:8px;margin-bottom:6px;color:#475467}.nav a:hover{background:#eef4ff;color:#2563eb}.main{padding:24px}.top{display:flex;justify-content:space-between;align-items:center;margin-bottom:20px}.card{background:#fff;border:1px solid #e5e7eb;border-radius:8px;padding:16px;margin-bottom:14px}.grid{display:grid;grid-template-columns:repeat(auto-fit,minmax(240px,1fr));gap:12px}.btn{border:1px solid #cbd5e1;border-radius:7px;background:#fff;padding:8px 11px;font-weight:700;cursor:pointer}.primary{background:#2563eb;color:#fff;border-color:#2563eb}.danger{color:#dc2626}.input{width:100%;box-sizing:border-box;border:1px solid #cbd5e1;border-radius:7px;padding:10px}.muted{color:#667085;font-size:13px}.badge{display:inline-block;border:1px solid #cbd5e1;border-radius:999px;padding:3px 8px;font-size:12px}.ok{color:#15803d}.bad{color:#dc2626}table{width:100%;border-collapse:collapse;background:#fff}td,th{border-bottom:1px solid #e5e7eb;padding:10px;text-align:left;font-size:13px}.login{max-width:380px;margin:12vh auto}
</style></head><body>{{if eq .View "login"}}{{template "login" .}}{{else}}<div class="shell"><aside class="side"><div class="brand">hostQ Go</div><nav class="nav"><a href="/">Dashboard</a><a href="/sites">Sites</a><a href="/files">Files</a><a href="/services">Services</a><a href="/logout">Logout</a></nav></aside><main class="main">{{if eq .View "dashboard"}}{{template "dashboard" .}}{{else if eq .View "sites"}}{{template "sites" .}}{{else if eq .View "files"}}{{template "files" .}}{{else if eq .View "services"}}{{template "services" .}}{{end}}</main></div>{{end}}</body></html>
{{end}}
{{define "login"}}<div class="login card"><h1>hostQ Go</h1><p class="muted">Lightweight control panel preview</p>{{if .Error}}<div class="card bad">{{.Error}}</div>{{end}}<form method="post"><p><input class="input" name="username" placeholder="Username" autocomplete="username"></p><p><input class="input" name="password" type="password" placeholder="Password" autocomplete="current-password"></p><button class="btn primary" type="submit">Sign in</button></form></div>{{end}}
{{define "dashboard"}}<div class="top"><div><h1>Dashboard</h1><p class="muted">Go runtime preview for low-memory VPS deployments</p></div><span class="badge">{{now.Format "Jan 02 15:04"}}</span></div><div class="grid"><div class="card"><h2>{{len .Sites}}</h2><p class="muted">Sites</p></div><div class="card"><h2>{{len .Services}}</h2><p class="muted">Services tracked</p></div></div>{{end}}
{{define "sites"}}<div class="top"><h1>Sites</h1></div>{{if .Error}}<div class="card bad">{{.Error}}</div>{{end}}<div class="card"><form method="post"><div class="grid"><input class="input" name="domain" placeholder="example.com"><button class="btn primary">Add PHP Site</button></div></form></div><table><tr><th>Domain</th><th>Root</th><th>Status</th><th>Cache</th></tr>{{range .Sites}}<tr><td>{{.Domain}}</td><td class="muted">{{.Root}}</td><td>{{if .Enabled}}<span class="ok">enabled</span>{{else}}<span class="bad">disabled</span>{{end}}</td><td>{{if .Cache}}on{{else}}off{{end}}</td></tr>{{end}}</table>{{end}}
{{define "files"}}<div class="top"><h1>Files</h1><span class="badge">{{.Path}}</span></div><table><tr><th>Name</th><th>Type</th></tr>{{range .Items}}<tr><td>{{if eq .Kind "dir"}}<a href="/files?path={{.Path}}">{{.Name}}</a>{{else}}{{.Name}}{{end}}</td><td>{{.Kind}}</td></tr>{{end}}</table>{{end}}
{{define "services"}}<div class="top"><h1>Services</h1></div><table><tr><th>Name</th><th>Status</th><th>Actions</th></tr>{{range .Services}}<tr><td>{{.Name}}</td><td>{{.Status}}</td><td><form method="post" style="display:flex;gap:6px"><input type="hidden" name="id" value="{{.ID}}"><button class="btn" name="action" value="restart">Restart</button><button class="btn" name="action" value="start">Start</button><button class="btn danger" name="action" value="stop">Stop</button></form></td></tr>{{end}}</table>{{end}}
`
