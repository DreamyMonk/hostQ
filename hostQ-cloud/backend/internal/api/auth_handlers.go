// Package api owns HTTP handlers. One file per resource group; this file is
// /api/auth/*.
package api

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/DreamyMonk/hostQ-cloud/backend/internal/auth"
	"github.com/DreamyMonk/hostQ-cloud/backend/internal/middleware"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type AuthHandlers struct {
	Pool *pgxpool.Pool
	Auth *auth.Service
	// CookieSecure flips Secure=true on refresh cookie in prod.
	CookieSecure bool
}

type loginReq struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type loginResp struct {
	AccessToken string `json:"accessToken"`
	User        struct {
		ID    uuid.UUID `json:"id"`
		Email string    `json:"email"`
		Role  string    `json:"role"`
		Name  string    `json:"displayName"`
	} `json:"user"`
}

func (h *AuthHandlers) Login(w http.ResponseWriter, r *http.Request) {
	var req loginReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, "bad json")
		return
	}
	req.Email = strings.TrimSpace(strings.ToLower(req.Email))
	if req.Email == "" || req.Password == "" {
		writeErr(w, http.StatusBadRequest, "email and password required")
		return
	}
	var (
		uid    uuid.UUID
		hash   string
		role   string
		name   *string
		locked bool
	)
	err := h.Pool.QueryRow(r.Context(), `
		SELECT id, password_hash, role, display_name, (locked_until IS NOT NULL AND locked_until > now())
		FROM users WHERE email = $1
	`, req.Email).Scan(&uid, &hash, &role, &name, &locked)
	if err != nil || locked {
		writeErr(w, http.StatusUnauthorized, "invalid credentials")
		return
	}
	if !auth.VerifyPassword(hash, req.Password) {
		_, _ = h.Pool.Exec(r.Context(), `UPDATE users SET failed_logins = failed_logins + 1,
			locked_until = CASE WHEN failed_logins + 1 >= 5 THEN now() + interval '15 minutes' ELSE NULL END
			WHERE id = $1`, uid)
		writeAudit(r, h.Pool, uid, role, "auth.login", req.Email, "failure")
		writeErr(w, http.StatusUnauthorized, "invalid credentials")
		return
	}
	_, _ = h.Pool.Exec(r.Context(), `UPDATE users SET failed_logins = 0, locked_until = NULL,
		last_login_at = now(), last_login_ip = NULLIF($1,'')::inet WHERE id = $2`,
		clientIP(r), uid)

	access, err := h.Auth.IssueAccess(uid, role)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "token issue")
		return
	}
	refresh, err := h.Auth.IssueRefresh(r.Context(), uid, r.UserAgent(), clientIP(r))
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "session issue")
		return
	}
	auth.SetRefreshCookie(w, refresh, h.CookieSecure)
	writeAudit(r, h.Pool, uid, role, "auth.login", req.Email, "success")

	resp := loginResp{AccessToken: access}
	resp.User.ID = uid
	resp.User.Email = req.Email
	resp.User.Role = role
	if name != nil {
		resp.User.Name = *name
	}
	writeJSON(w, http.StatusOK, resp)
}

func (h *AuthHandlers) Refresh(w http.ResponseWriter, r *http.Request) {
	c, err := r.Cookie(auth.RefreshCookie)
	if err != nil || c.Value == "" {
		writeErr(w, http.StatusUnauthorized, "no refresh cookie")
		return
	}
	uid, role, newRefresh, err := h.Auth.Rotate(r.Context(), c.Value, r.UserAgent(), clientIP(r))
	if err != nil {
		auth.ClearRefreshCookie(w, h.CookieSecure)
		writeErr(w, http.StatusUnauthorized, "invalid refresh")
		return
	}
	access, err := h.Auth.IssueAccess(uid, role)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "token issue")
		return
	}
	auth.SetRefreshCookie(w, newRefresh, h.CookieSecure)
	writeJSON(w, http.StatusOK, map[string]string{"accessToken": access, "role": role})
}

func (h *AuthHandlers) Logout(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie(auth.RefreshCookie); err == nil && c.Value != "" {
		_ = h.Auth.Revoke(r.Context(), c.Value)
	}
	auth.ClearRefreshCookie(w, h.CookieSecure)
	w.WriteHeader(http.StatusNoContent)
}

// Me returns the currently authenticated user. Useful for the frontend's
// initial bootstrap after parsing the access token.
func (h *AuthHandlers) Me(w http.ResponseWriter, r *http.Request) {
	uid, _ := middleware.UserID(r)
	var (
		email string
		role  string
		name  *string
	)
	err := h.Pool.QueryRow(r.Context(), `
		SELECT email, role, display_name FROM users WHERE id = $1
	`, uid).Scan(&email, &role, &name)
	if err != nil {
		writeErr(w, http.StatusNotFound, "user not found")
		return
	}
	resp := map[string]any{"id": uid, "email": email, "role": role}
	if name != nil {
		resp["displayName"] = *name
	}
	writeJSON(w, http.StatusOK, resp)
}
