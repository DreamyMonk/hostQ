package api

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

// clientIP picks the leftmost X-Forwarded-For (the original client when
// behind nginx) or falls back to RemoteAddr's host part.
func clientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		if i := strings.IndexByte(xff, ','); i >= 0 {
			xff = xff[:i]
		}
		ip := strings.TrimSpace(xff)
		if ip != "" {
			return ip
		}
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

// writeAudit is best-effort. We log on a fire-and-forget goroutine so a slow
// audit insert never blocks the response.
func writeAudit(r *http.Request, pool *pgxpool.Pool, actor uuid.UUID, role, action, target, status string) {
	ip := clientIP(r)
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2_000_000_000) // 2s
		defer cancel()
		_, _ = pool.Exec(ctx, `
			INSERT INTO audit_log (actor_id, actor_role, actor_ip, action, target, status)
			VALUES ($1, $2, NULLIF($3,'')::inet, $4, $5, $6)
		`, actor, role, ip, action, target, status)
	}()
}
