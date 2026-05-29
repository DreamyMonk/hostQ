// Command seed creates the first superadmin if none exists. Re-runnable.
//
//	go run ./cmd/seed
//	HOSTQ_ADMIN_EMAIL=ops@example.com HOSTQ_ADMIN_PASSWORD=secret go run ./cmd/seed
package main

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/DreamyMonk/hostQ-cloud/backend/internal/auth"
	"github.com/DreamyMonk/hostQ-cloud/backend/internal/config"
	"github.com/DreamyMonk/hostQ-cloud/backend/internal/db"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	pool, err := db.Open(ctx, cfg.DBURL)
	if err != nil {
		log.Fatalf("db: %v", err)
	}
	defer pool.Close()

	var exists bool
	if err := pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM users WHERE role = 'superadmin')`).Scan(&exists); err != nil {
		log.Fatalf("query: %v", err)
	}
	if exists {
		fmt.Println("superadmin already exists — not seeding.")
		return
	}

	email := env("HOSTQ_ADMIN_EMAIL", "admin@hostq.local")
	pw := env("HOSTQ_ADMIN_PASSWORD", "")
	if pw == "" {
		buf := make([]byte, 16)
		_, _ = rand.Read(buf)
		pw = base64.RawURLEncoding.EncodeToString(buf)
	}
	hash, err := auth.HashPassword(pw)
	if err != nil {
		log.Fatalf("hash: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO users (email, password_hash, role, display_name)
		VALUES ($1, $2, 'superadmin', 'Operator')
	`, email, hash); err != nil {
		log.Fatalf("insert: %v", err)
	}
	fmt.Println("Initial hostQ-cloud superadmin:")
	fmt.Println("  Email:    ", email)
	fmt.Println("  Password: ", pw)
}

func env(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}
