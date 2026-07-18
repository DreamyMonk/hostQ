package main

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"strconv"
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
	// Maintenance / recovery CLI (from the 2026-07-18 incident report). These
	// exit the process; the mutating ones run under a cross-process config lock.
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "doctor":
			// Detect zero-byte/missing managed vhosts, restore from the backup
			// store, validate, and reload/start nginx only if `nginx -t` passes.
			var res DoctorResult
			app.withConfigLock(func() {
				app.cleanupStaleTemps()
				app.seedConfBackups()
				res = app.runDoctor()
			})
			printDoctorResult(res)
			if !res.NginxOK {
				os.Exit(1)
			}
			return
		case "repair":
			// `hostq repair` / `hostq repair nginx`: regenerate every vhost from
			// /etc/hostq/sites/*.json metadata, restore any legacy config from
			// backup, validate, and reload.
			app.backfillSiteMeta()
			var res DoctorResult
			app.withConfigLock(func() {
				app.seedConfBackups()
				res = app.runRepair()
			})
			printDoctorResult(res)
			if !res.NginxOK {
				os.Exit(1)
			}
			return
		case "rebuild":
			// Regenerate every vhost from metadata (idempotent).
			app.backfillSiteMeta()
			var rs []RebuildResult
			app.withConfigLock(func() { rs = app.runRebuild() })
			printRebuildResults(rs)
			return
		case "validate":
			// Read-only: nginx -t + php-fpm config test + empty-config scan.
			v := app.runValidate()
			printValidateResult(v)
			if !v.AllOK {
				os.Exit(1)
			}
			return
		case "status":
			s := app.runStatus()
			printStatus(s)
			if !s.Healthy {
				os.Exit(1)
			}
			return
		case "backup":
			// Snapshot every managed vhost into the store as a fresh revision.
			var saved []string
			app.withConfigLock(func() { saved = app.runBackup() })
			printBackupResult(saved)
			return
		case "restore":
			app.runRestoreCLI(os.Args[2:])
			return
		case "deploy-log":
			n := 50
			if len(os.Args) >= 3 {
				if v, err := strconv.Atoi(os.Args[2]); err == nil && v > 0 {
					n = v
				}
			}
			lines := app.tailDeployLog(n)
			if len(lines) == 0 {
				fmt.Println("deploy log is empty (" + app.deployLogPath() + ")")
				return
			}
			for _, l := range lines {
				fmt.Println(l)
			}
			return
		}
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
	// Admin-scope routes wrap requireAuth with requireAdminAllow so the
	// optional IP allowlist (managed on /account) also gates them.
	mux.HandleFunc("/cron", app.requireAuth(app.requireAdminAllow(app.cron)))
	mux.HandleFunc("/services", app.requireAuth(app.requireAdminAllow(app.services)))
	mux.HandleFunc("/account", app.requireAuth(app.requireAdminAllow(app.account)))
	mux.HandleFunc("/audit", app.requireAuth(app.requireAdminAllow(app.auditLog)))
	mux.HandleFunc("/redis", app.requireAuth(app.requireAdminAllow(app.redis)))
	mux.HandleFunc("/security", app.requireAuth(app.requireAdminAllow(app.security)))
	mux.HandleFunc("/malfix", app.requireAuth(app.requireAdminAllow(app.malfix)))
	mux.HandleFunc("/firewall", app.requireAuth(app.requireAdminAllow(app.firewall)))
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

	// Best-effort: write the pma snippet + default vhost if missing. If
	// PHP-FPM or phpMyAdmin aren't on the box yet this no-ops with an err
	// the operator can re-trigger from the panel later. We deliberately do
	// not fail the panel boot on this.
	if err := app.ensurePMADefaultVhost(); err != nil {
		log.Printf("pma setup skipped: %v", err)
	} else {
		log.Printf("pma setup verified: snippet + default vhost in place")
	}

	// Boot-time nginx safety (crash recovery + self-heal):
	//   1. sweep stranded atomic-write temp files from a mid-write crash,
	//   2. backfill site metadata so rebuild/repair works for existing sites,
	//   3. snapshot any healthy managed vhosts that lack a good-config backup,
	//   4. restore + reload any vhost that is currently zero-byte/missing.
	// Best-effort — a wiped sites-available must not stop the panel serving.
	app.cleanupStaleTemps()
	app.backfillSiteMeta()
	app.seedConfBackups()
	app.nginxStartupHeal()

	log.Printf("hostQ panel listening on http://%s", app.cfg.Addr)
	log.Fatal(http.ListenAndServe(app.cfg.Addr, gzipMiddleware(securityHeaders(mux))))
}

func env(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}
