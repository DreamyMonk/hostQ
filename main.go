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
		"now": time.Now,
	}).Parse(layoutTemplate))

	mux := http.NewServeMux()
	mux.HandleFunc("/", app.requireAuth(app.dashboard))
	mux.HandleFunc("/login", app.login)
	mux.HandleFunc("/logout", app.logout)
	mux.HandleFunc("/sites", app.requireAuth(app.sites))
	mux.HandleFunc("/site", app.requireAuth(app.siteManager))
	mux.HandleFunc("/site-action", app.requireAuth(app.siteAction))
	mux.HandleFunc("/backups", app.requireAuth(app.backups))
	mux.HandleFunc("/files", app.requireAuth(app.files))
	mux.HandleFunc("/databases", app.requireAuth(app.databases))
	mux.HandleFunc("/wordpress", app.requireAuth(app.wordpress))
	mux.HandleFunc("/php", app.requireAuth(app.php))
	mux.HandleFunc("/ssl", app.requireAuth(app.ssl))
	mux.HandleFunc("/services", app.requireAuth(app.services))
	mux.HandleFunc("/cron", app.requireAuth(app.cron))
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte("ok")) })

	log.Printf("hostQ panel listening on http://%s", app.cfg.Addr)
	log.Fatal(http.ListenAndServe(app.cfg.Addr, securityHeaders(mux)))
}

func env(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}
