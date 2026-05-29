// Package auth handles password hashing, JWT issuance/verification, and
// session refresh-token rotation. Short access tokens (15m) ride in the
// Authorization header; long refresh tokens (30d) ride in a httpOnly cookie
// and are rotated on every /refresh call.
package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

const (
	AccessTokenTTL  = 15 * time.Minute
	RefreshTokenTTL = 30 * 24 * time.Hour
	RefreshCookie   = "hostq_refresh"
)

type Claims struct {
	UserID uuid.UUID `json:"uid"`
	Role   string    `json:"role"`
	jwt.RegisteredClaims
}

type Service struct {
	pool   *pgxpool.Pool
	secret []byte
}

func New(pool *pgxpool.Pool, secret string) *Service {
	return &Service{pool: pool, secret: []byte(secret)}
}

// HashPassword runs bcrypt at cost 12. Slow on purpose.
func HashPassword(pw string) (string, error) {
	h, err := bcrypt.GenerateFromPassword([]byte(pw), 12)
	return string(h), err
}

func VerifyPassword(hash, pw string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(pw)) == nil
}

// IssueAccess returns a signed JWT for the user with role embedded.
func (s *Service) IssueAccess(uid uuid.UUID, role string) (string, error) {
	now := time.Now()
	claims := Claims{
		UserID: uid,
		Role:   role,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(AccessTokenTTL)),
			Subject:   uid.String(),
		},
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(s.secret)
}

func (s *Service) ParseAccess(tok string) (*Claims, error) {
	parsed, err := jwt.ParseWithClaims(tok, &Claims{}, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("bad signing method")
		}
		return s.secret, nil
	})
	if err != nil {
		return nil, err
	}
	claims, ok := parsed.Claims.(*Claims)
	if !ok || !parsed.Valid {
		return nil, errors.New("invalid token")
	}
	return claims, nil
}

// IssueRefresh stores a fresh refresh token for the user, returns the raw
// value to set in the cookie. The DB only stores the SHA-256 of the value so
// a DB leak doesn't grant sessions.
func (s *Service) IssueRefresh(ctx context.Context, uid uuid.UUID, ua, ip string) (string, error) {
	raw := randomToken(32)
	digest := sha256.Sum256([]byte(raw))
	_, err := s.pool.Exec(ctx, `
		INSERT INTO sessions (user_id, refresh_hash, user_agent, ip, expires_at)
		VALUES ($1, $2, $3, NULLIF($4,'')::inet, now() + interval '30 days')
	`, uid, hex.EncodeToString(digest[:]), ua, ip)
	return raw, err
}

// Rotate swaps an existing refresh token for a new one. Old token is revoked.
// Returns (new uid, new raw refresh) or error.
func (s *Service) Rotate(ctx context.Context, raw, ua, ip string) (uuid.UUID, string, string, error) {
	digest := sha256.Sum256([]byte(raw))
	hashHex := hex.EncodeToString(digest[:])
	var uid uuid.UUID
	var role string
	var sessID uuid.UUID
	err := s.pool.QueryRow(ctx, `
		SELECT s.id, s.user_id, u.role
		FROM sessions s JOIN users u ON u.id = s.user_id
		WHERE s.refresh_hash = $1 AND s.revoked_at IS NULL AND s.expires_at > now()
	`, hashHex).Scan(&sessID, &uid, &role)
	if err != nil {
		return uuid.Nil, "", "", err
	}
	// Revoke the used token. New one minted below — refresh tokens are single-use.
	if _, err := s.pool.Exec(ctx, `UPDATE sessions SET revoked_at = now() WHERE id = $1`, sessID); err != nil {
		return uuid.Nil, "", "", err
	}
	newRaw, err := s.IssueRefresh(ctx, uid, ua, ip)
	if err != nil {
		return uuid.Nil, "", "", err
	}
	return uid, role, newRaw, nil
}

func (s *Service) Revoke(ctx context.Context, raw string) error {
	digest := sha256.Sum256([]byte(raw))
	_, err := s.pool.Exec(ctx, `UPDATE sessions SET revoked_at = now() WHERE refresh_hash = $1`,
		hex.EncodeToString(digest[:]))
	return err
}

func randomToken(n int) string {
	buf := make([]byte, n)
	_, _ = rand.Read(buf)
	return base64.RawURLEncoding.EncodeToString(buf)
}

// SetRefreshCookie writes the long-lived refresh cookie. httpOnly + SameSite
// strict so it never leaks to JS or third-party origins.
func SetRefreshCookie(w http.ResponseWriter, raw string, secure bool) {
	http.SetCookie(w, &http.Cookie{
		Name:     RefreshCookie,
		Value:    raw,
		Path:     "/",
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteStrictMode,
		Expires:  time.Now().Add(RefreshTokenTTL),
	})
}

func ClearRefreshCookie(w http.ResponseWriter, secure bool) {
	http.SetCookie(w, &http.Cookie{
		Name:     RefreshCookie,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   -1,
	})
}
