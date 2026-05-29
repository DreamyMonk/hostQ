// Command api is the hostQ-cloud REST server.
//
// Routes:
//
//	POST   /api/auth/login        public
//	POST   /api/auth/refresh      public (reads refresh cookie)
//	POST   /api/auth/logout       authed
//	GET    /api/auth/me           authed
//
//	GET    /api/sites             authed — own (tenant) or all (admin)
//	POST   /api/sites             authed
//	DELETE /api/sites/{id}        authed (owner or admin)
//
//	GET    /api/healthz           public
//
// Wire up an admin via `api seed-admin` first (one-shot subcommand).
package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/DreamyMonk/hostQ-cloud/backend/internal/api"
	"github.com/DreamyMonk/hostQ-cloud/backend/internal/auth"
	"github.com/DreamyMonk/hostQ-cloud/backend/internal/config"
	"github.com/DreamyMonk/hostQ-cloud/backend/internal/db"
	mw "github.com/DreamyMonk/hostQ-cloud/backend/internal/middleware"
	"github.com/DreamyMonk/hostQ-cloud/backend/internal/sites"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	pool, err := db.Open(ctx, cfg.DBURL)
	if err != nil {
		log.Fatalf("db: %v", err)
	}
	defer pool.Close()

	authSvc := auth.New(pool, cfg.JWTSecret)
	vhost := sites.New(cfg.NginxSitesDir, cfg.ApacheSitesDir, cfg.WebRoot)

	authH := &api.AuthHandlers{Pool: pool, Auth: authSvc, CookieSecure: !isDev(cfg)}
	siteH := &api.SiteHandlers{Pool: pool, Vhost: vhost}

	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(60 * time.Second))
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{cfg.FrontendOrigin},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Authorization", "Content-Type"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	r.Get("/api/healthz", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"ok":true}`))
	})

	r.Post("/api/auth/login", authH.Login)
	r.Post("/api/auth/refresh", authH.Refresh)

	r.Group(func(r chi.Router) {
		r.Use(mw.RequireAuth(authSvc))
		r.Post("/api/auth/logout", authH.Logout)
		r.Get("/api/auth/me", authH.Me)

		// Sites: authed — handler discriminates own vs all by role.
		r.Get("/api/sites", siteH.List)
		r.Post("/api/sites", siteH.Create)
		r.Delete("/api/sites/{id}", siteH.Delete)
	})

	srv := &http.Server{
		Addr:              cfg.Addr,
		Handler:           r,
		ReadHeaderTimeout: 15 * time.Second,
	}

	go func() {
		log.Printf("hostQ-cloud API listening on http://%s", cfg.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %v", err)
		}
	}()

	<-ctx.Done()
	log.Println("shutting down…")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = srv.Shutdown(shutdownCtx)
}

func isDev(cfg *config.Config) bool {
	return strings.HasPrefix(cfg.FrontendOrigin, "http://")
}
