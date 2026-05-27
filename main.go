package main

import (
	"html/template"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

func main() {
	app := &App{
		cfg: Config{
			Addr:          env("HOSTQ_ADDR", "127.0.0.1:8090"),
			DataDir:       env("HOSTQ_DATA_DIR", "/etc/hostq"),
			WebRoot:       env("WEB_ROOT", "/var/www"),
			NginxSitesDir: env("HOSTQ_NGINX_AVAILABLE", "/etc/nginx/sites-available"),
			JWTSecret:     env("JWT_SECRET", "change_this_hostq_secret"),
		},
		cache: newMemCache(),
	}
	if len(os.Args) > 1 && os.Args[1] == "init-admin" {
		if err := app.initAdmin(); err != nil {
			log.Fatal(err)
		}
		return
	}
	if len(os.Args) > 1 && os.Args[1] == "run-backups" {
		if err := app.runScheduledBackups(); err != nil {
			log.Fatal(err)
		}
		return
	}
	app.tpl = template.Must(template.New("hostq").Funcs(template.FuncMap{
		"now":  time.Now,
		"icon": icon,
		"hasPrefix": func(s, prefix string) bool {
			return strings.HasPrefix(s, prefix)
		},
		"humanBytes": humanSize,
	}).Parse(layoutTemplate))

	mux := http.NewServeMux()
	mux.HandleFunc("/", app.requireAuth(app.dashboard))
	mux.HandleFunc("/login", app.login)
	mux.HandleFunc("/logout", app.logout)
	mux.HandleFunc("/sites", app.requireAuth(app.sites))
	mux.HandleFunc("/sites/add", app.requireAuth(app.siteAdd))
	mux.HandleFunc("/site", app.requireAuth(app.siteManager))
	mux.HandleFunc("/site-action", app.requireAuth(app.siteAction))
	mux.HandleFunc("/site-nginx", app.requireAuth(app.siteNginx))
	mux.HandleFunc("/site-php-ext", app.requireAuth(app.sitePhpExt))
	mux.HandleFunc("/backups", app.requireAuth(app.backups))
	mux.HandleFunc("/files", app.requireAuth(app.files))
	mux.HandleFunc("/file-edit", app.requireAuth(app.fileEdit))
	mux.HandleFunc("/api/dirs", app.requireAuth(app.apiDirs))
	mux.HandleFunc("/databases", app.requireAuth(app.databases))
	mux.HandleFunc("/wordpress", app.requireAuth(app.wordpress))
	mux.HandleFunc("/php", app.requireAuth(app.php))
	mux.HandleFunc("/ssl", app.requireAuth(app.ssl))
	mux.HandleFunc("/services", app.requireAuth(app.services))
	mux.HandleFunc("/cron", app.requireAuth(app.cron))
	mux.HandleFunc("/account", app.requireAuth(app.account))
	mux.HandleFunc("/audit", app.requireAuth(app.auditLog))
	mux.HandleFunc("/redis", app.requireAuth(app.redis))
	mux.HandleFunc("/security", app.requireAuth(app.security))
	mux.HandleFunc("/malfix", app.requireAuth(app.malfix))
	mux.HandleFunc("/pma-login", app.requireAuth(app.pmaLogin))
	// When the user types /phpmyadmin/ on the panel host (typically :8090
	// for direct setup access), Go's catch-all "/" route would render the
	// panel dashboard. Redirect to the bare-host URL so nginx serves
	// phpMyAdmin from /usr/share/phpmyadmin via the hostq-pma snippet.
	// Also auto-installs the default :80 vhost the first time a redirect
	// fires — without it, bare-IP /phpmyadmin/ loops because nginx falls
	// to an arbitrary hostQ-managed vhost that 301s to https.
	pmaRedirect := func(w http.ResponseWriter, r *http.Request) {
		_ = app.ensurePMADefaultVhost()
		host := r.Host
		if i := strings.LastIndex(host, ":"); i > 0 {
			host = host[:i]
		}
		scheme := "http"
		if r.TLS != nil || strings.EqualFold(r.Header.Get("X-Forwarded-Proto"), "https") {
			scheme = "https"
		}
		target := scheme + "://" + host + "/phpmyadmin/"
		if q := r.URL.RawQuery; q != "" {
			target += "?" + q
		}
		http.Redirect(w, r, target, http.StatusFound)
	}
	mux.HandleFunc("/phpmyadmin", pmaRedirect)
	mux.HandleFunc("/phpmyadmin/", pmaRedirect)
	// /packages was merged into /services in v0.11. Keep a redirect for
	// bookmarks and old release notes.
	mux.HandleFunc("/packages", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/services", http.StatusMovedPermanently)
	})
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte("ok")) })

	log.Printf("hostQ panel listening on http://%s", app.cfg.Addr)
	log.Fatal(http.ListenAndServe(app.cfg.Addr, gzipMiddleware(securityHeaders(mux))))
}

func env(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}
