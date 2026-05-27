package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
)

// panelHostState tracks what hostname the panel is currently bound to via the
// auto-managed nginx proxy vhost. Persisted at /etc/hostq/panel-host.json so
// the Account page can show the current setting and certbot status across
// restarts.
type PanelHostState struct {
	Hostname string `json:"hostname"`
	SSL      bool   `json:"ssl"`
	Email    string `json:"email"`
	Updated  string `json:"updated"`
}

const (
	panelVhostPath     = "/etc/nginx/sites-available/hostq-panel"
	panelVhostLink     = "/etc/nginx/sites-enabled/hostq-panel"
	panelHostStateFile = "panel-host.json"
)

func (a *App) loadPanelHostState() PanelHostState {
	var s PanelHostState
	data, err := os.ReadFile(filepath.Join(a.cfg.DataDir, panelHostStateFile))
	if err != nil {
		return s
	}
	_ = json.Unmarshal(data, &s)
	return s
}

func (a *App) savePanelHostState(s PanelHostState) error {
	if err := os.MkdirAll(a.cfg.DataDir, 0700); err != nil {
		return err
	}
	data, _ := json.MarshalIndent(s, "", "  ")
	return os.WriteFile(filepath.Join(a.cfg.DataDir, panelHostStateFile), data, 0600)
}

// writePanelProxyVhost emits the nginx vhost that proxies <hostname> to the
// panel on 127.0.0.1:8090 and exposes /phpmyadmin/ via the managed snippet.
// If a Let's Encrypt cert already exists for the hostname, the file emits
// both a port 80 → 443 redirect block and a full HTTPS server block with
// the pma include in it. Otherwise it's HTTP-only.
//
// We deliberately overwrite anything certbot --nginx added — certbot's
// auto-modify creates a fresh server block without our `include
// snippets/hostq-pma.conf;` line, so /phpmyadmin/ would 404 on the
// HTTPS side. The panel keeps re-emitting in our own shape every time
// the setup is touched (or re-run after certbot), so both protocol
// blocks always carry the include.
func (a *App) writePanelProxyVhost(hostname string) error {
	if !domainRe.MatchString(hostname) {
		return fmt.Errorf("invalid hostname")
	}
	pmaInclude := ""
	if _, err := os.Stat("/etc/nginx/snippets/hostq-pma.conf"); err == nil {
		pmaInclude = "    include snippets/hostq-pma.conf;\n"
	}
	// Shared server body — pma include + proxy_pass location.
	body := fmt.Sprintf(`    access_log /var/log/nginx/hostq-panel.access.log combined;

    client_max_body_size 512m;
%s
    location / {
        proxy_pass http://127.0.0.1:8090;
        proxy_http_version 1.1;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_read_timeout 600s;
        proxy_send_timeout 600s;
        proxy_buffering off;
    }
`, pmaInclude)

	hasSSL := a.certExists(hostname)
	var content string
	if hasSSL {
		sslIncludes := ""
		if _, err := os.Stat("/etc/letsencrypt/options-ssl-nginx.conf"); err == nil {
			sslIncludes += "    include /etc/letsencrypt/options-ssl-nginx.conf;\n"
		}
		if _, err := os.Stat("/etc/letsencrypt/ssl-dhparams.pem"); err == nil {
			sslIncludes += "    ssl_dhparam /etc/letsencrypt/ssl-dhparams.pem;\n"
		}
		content = fmt.Sprintf(`# hostQ panel proxy — auto-managed. Bind the panel UI to %s
# and expose /phpmyadmin/ on the same origin so SSO works.
server {
    listen 80;
    listen [::]:80;
    server_name %s;
    return 301 https://$host$request_uri;
}
server {
    listen 443 ssl http2;
    listen [::]:443 ssl http2;
    server_name %s;
    ssl_certificate /etc/letsencrypt/live/%s/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/%s/privkey.pem;
%s%s}
`, hostname, hostname, hostname, hostname, hostname, sslIncludes, body)
	} else {
		content = fmt.Sprintf(`# hostQ panel proxy — auto-managed. Bind the panel UI to %s
# and expose /phpmyadmin/ on the same origin so SSO works.
server {
    listen 80;
    listen [::]:80;
    server_name %s;
%s}
`, hostname, hostname, body)
	}
	if err := os.WriteFile(panelVhostPath, []byte(content), 0644); err != nil {
		return err
	}
	if _, err := os.Lstat(panelVhostLink); err != nil {
		if err := os.Symlink(panelVhostPath, panelVhostLink); err != nil {
			return err
		}
	}
	return nil
}

// setupPanelHostname writes the vhost, optionally runs certbot for SSL, and
// persists the result. Errors are returned as the message so the caller can
// surface them via the toast on /account.
func (a *App) setupPanelHostname(hostname, email string, withSSL bool) (string, error) {
	hostname = strings.ToLower(strings.TrimSpace(hostname))
	hostname = strings.TrimPrefix(hostname, "www.")
	if !domainRe.MatchString(hostname) {
		return "Invalid hostname", fmt.Errorf("bad hostname")
	}
	if err := a.writePanelProxyVhost(hostname); err != nil {
		return "write vhost failed: " + err.Error(), err
	}
	if out, err := exec.Command("nginx", "-t").CombinedOutput(); err != nil {
		return "nginx -t failed: " + tail(string(out), 240), err
	}
	_ = exec.Command("systemctl", "reload", "nginx").Run()
	state := PanelHostState{Hostname: hostname, SSL: false, Email: email, Updated: time.Now().Format(time.RFC3339)}
	if withSSL {
		if email == "" || !strings.Contains(email, "@") {
			_ = a.savePanelHostState(state)
			return "Panel hostname set, but SSL needs a valid admin email — re-run with one filled in. HTTP works at http://" + hostname + "/ now.", nil
		}
		out, err := exec.Command("certbot", "--nginx", "-d", hostname, "-m", email, "--agree-tos", "--non-interactive", "--redirect").CombinedOutput()
		if err != nil {
			_ = a.savePanelHostState(state)
			a.audit("panel.host-ssl", "failure", hostname)
			return "Vhost created (HTTP only). Certbot failed: " + tail(string(out), 240), err
		}
		// Certbot's --nginx auto-edit creates a fresh server block on :443
		// without our `include snippets/hostq-pma.conf;` line, so /phpmyadmin/
		// would 404 on HTTPS while working on HTTP. Re-write the vhost in
		// our own shape now that the cert exists — writePanelProxyVhost
		// detects the cert and emits both 80-redirect + 443 blocks with the
		// pma include in both.
		_ = a.writePanelProxyVhost(hostname)
		if out2, err := exec.Command("nginx", "-t").CombinedOutput(); err != nil {
			return "Cert installed but post-certbot vhost rewrite failed: " + tail(string(out2), 240), err
		}
		_ = exec.Command("systemctl", "reload", "nginx").Run()
		state.SSL = true
		_ = a.savePanelHostState(state)
		a.audit("panel.host-ssl", "success", hostname)
		return "Panel hostname ready at https://" + hostname + "/ — SSL installed, phpMyAdmin wired up.", nil
	}
	_ = a.savePanelHostState(state)
	a.audit("panel.host-set", "success", hostname)
	return "Panel hostname ready at http://" + hostname + "/", nil
}

// removePanelHostname undoes everything setupPanelHostname did. The Let's
// Encrypt cert (if any) is kept on disk — operators can certbot delete it
// manually if they really want it gone.
func (a *App) removePanelHostname() (string, error) {
	_ = os.Remove(panelVhostLink)
	_ = os.Remove(panelVhostPath)
	_ = os.Remove(filepath.Join(a.cfg.DataDir, panelHostStateFile))
	if out, err := exec.Command("nginx", "-t").CombinedOutput(); err != nil {
		return "nginx -t failed after removal: " + tail(string(out), 240), err
	}
	_ = exec.Command("systemctl", "reload", "nginx").Run()
	a.audit("panel.host-remove", "success", "")
	return "Panel hostname removed. Direct :8090 access still works.", nil
}

func (a *App) account(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		a.accountAction(w, r)
		return
	}
	acc, _ := a.readAccount()
	a.render(w, "account", map[string]any{
		"Title":     "Account",
		"Account":   acc,
		"Output":    r.URL.Query().Get("output"),
		"PanelHost": a.loadPanelHostState(),
	})
}

func (a *App) accountAction(w http.ResponseWriter, r *http.Request) {
	output := ""
	switch r.FormValue("action") {
	case "panel-host-set":
		msg, _ := a.setupPanelHostname(
			r.FormValue("hostname"),
			strings.TrimSpace(r.FormValue("email")),
			r.FormValue("ssl") == "1",
		)
		http.Redirect(w, r, "/account?output="+queryEscape(msg), http.StatusSeeOther)
		return
	case "panel-host-remove":
		msg, _ := a.removePanelHostname()
		http.Redirect(w, r, "/account?output="+queryEscape(msg), http.StatusSeeOther)
		return
	}
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
