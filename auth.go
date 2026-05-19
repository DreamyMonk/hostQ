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
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
)

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
		w.Header().Set("Content-Security-Policy", "default-src 'self'; img-src 'self' data:; style-src 'self' 'unsafe-inline'; script-src 'self' 'unsafe-inline'; frame-ancestors 'none'; base-uri 'self'; form-action 'self'")
		// Pages are session-authenticated and full of mutable state; never cache.
		w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate")
		w.Header().Set("Pragma", "no-cache")
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
		Name:     "hostq_session",
		Value:    base64.RawURLEncoding.EncodeToString([]byte(payload)) + "." + a.sign(payload),
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   86400,
	}
}

func (a *App) verifySession(r *http.Request) bool {
	cookie, err := r.Cookie("hostq_session")
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
		a.render(w, "login", map[string]any{"Title": "Sign in", "Error": "Admin account is not initialized. Run install.sh first."})
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
	http.SetCookie(w, &http.Cookie{Name: "hostq_session", Value: "", Path: "/", MaxAge: -1})
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}
