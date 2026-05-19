package main

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
)

func (a *App) account(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		a.accountAction(w, r)
		return
	}
	acc, _ := a.readAccount()
	a.render(w, "account", map[string]any{
		"Title":   "Account",
		"Account": acc,
		"Output":  r.URL.Query().Get("output"),
	})
}

func (a *App) accountAction(w http.ResponseWriter, r *http.Request) {
	output := ""
	acc, err := a.readAccount()
	if err != nil {
		http.Redirect(w, r, "/account?output="+queryEscape("Cannot load account"), http.StatusSeeOther)
		return
	}
	current := r.FormValue("current")
	next := r.FormValue("next")
	confirm := r.FormValue("confirm")
	if next == "" || len(next) < 10 {
		output = "Use a password with at least 10 characters."
	} else if next != confirm {
		output = "New password and confirmation do not match."
	} else if bcrypt.CompareHashAndPassword([]byte(acc.PasswordHash), []byte(current)) != nil {
		output = "Current password is incorrect."
		a.audit("account.changepw", "failure", acc.Username)
	} else {
		hash, err := bcrypt.GenerateFromPassword([]byte(next), 12)
		if err == nil {
			payload := map[string]any{
				"username":     acc.Username,
				"passwordHash": string(hash),
				"role":         acc.Role,
				"updatedAt":    time.Now().Format(time.RFC3339),
			}
			data, _ := json.MarshalIndent(payload, "", "  ")
			if err := os.WriteFile(filepath.Join(a.cfg.DataDir, "admin.json"), data, 0600); err == nil {
				output = "Password updated."
				a.audit("account.changepw", "success", acc.Username)
			} else {
				output = "Failed to write: " + err.Error()
			}
		} else {
			output = "Hash failed: " + err.Error()
		}
	}
	http.Redirect(w, r, "/account?output="+queryEscape(output), http.StatusSeeOther)
}

func (a *App) auditLog(w http.ResponseWriter, r *http.Request) {
	data, _ := os.ReadFile(filepath.Join(a.cfg.DataDir, "audit.log"))
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	entries := []AuditEntry{}
	// Read latest 200 entries, newest first
	start := 0
	if len(lines) > 200 {
		start = len(lines) - 200
	}
	for i := len(lines) - 1; i >= start; i-- {
		line := strings.TrimSpace(lines[i])
		if line == "" {
			continue
		}
		var row map[string]string
		if err := json.Unmarshal([]byte(line), &row); err != nil {
			continue
		}
		entries = append(entries, AuditEntry{
			Timestamp: row["ts"], Action: row["action"],
			Status: row["status"], Target: row["target"],
		})
	}
	a.render(w, "audit", map[string]any{"Title": "Audit Log", "Entries": entries})
}
