// Package config loads environment-backed runtime config. Keep this small and
// boring; every other package reads from the returned Config struct rather than
// touching os.Getenv directly so wiring stays testable.
package config

import (
	"fmt"
	"os"
	"strings"
)

type Config struct {
	Addr           string
	DBURL          string
	RedisURL       string
	JWTSecret      string
	NginxSitesDir  string
	ApacheSitesDir string
	WebRoot        string
	FrontendOrigin string
}

// Load reads required vars, returns a helpful error if any are missing.
func Load() (*Config, error) {
	c := &Config{
		Addr:           env("HOSTQ_ADDR", "127.0.0.1:8091"),
		DBURL:          env("HOSTQ_DB_URL", ""),
		RedisURL:       env("HOSTQ_REDIS_URL", "redis://127.0.0.1:6379/0"),
		JWTSecret:      env("HOSTQ_JWT_SECRET", ""),
		NginxSitesDir:  env("HOSTQ_NGINX_SITES", "/etc/nginx/sites-available"),
		ApacheSitesDir: env("HOSTQ_APACHE_SITES", "/etc/apache2/sites-available"),
		WebRoot:        env("HOSTQ_WEB_ROOT", "/var/www"),
		FrontendOrigin: env("HOSTQ_FRONTEND_ORIGIN", "http://localhost:3000"),
	}
	var missing []string
	if c.DBURL == "" {
		missing = append(missing, "HOSTQ_DB_URL")
	}
	if c.JWTSecret == "" || len(c.JWTSecret) < 32 {
		missing = append(missing, "HOSTQ_JWT_SECRET (at least 32 chars)")
	}
	if len(missing) > 0 {
		return nil, fmt.Errorf("missing required env: %s", strings.Join(missing, ", "))
	}
	return c, nil
}

func env(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}
