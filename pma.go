package main

import (
	"crypto/tls"
	"fmt"
	"html"
	"io"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"
)

// ensurePMASnippet writes /etc/nginx/snippets/hostq-pma.conf when missing.
// install.sh wrote this on first boot, but operators who install phpMyAdmin
// later (via the Services & Packages page or apt directly) won't get the
// snippet automatically — and without the snippet every /phpmyadmin/ request
// 404s.
//
// We pick the first available PHP-FPM socket (prefer 8.3 — phpMyAdmin's
// upstream test target), falling back through 8.2/8.4/8.5. If no FPM is
// installed yet, return an error so the caller can skip the rest of the
// setup.
func (a *App) ensurePMASnippet() error {
	if _, err := os.Stat("/usr/share/phpmyadmin/index.php"); err != nil {
		return fmt.Errorf("phpmyadmin package not installed")
	}
	const path = "/etc/nginx/snippets/hostq-pma.conf"
	sock := ""
	for _, v := range []string{"8.3", "8.2", "8.4", "8.5"} {
		s := "/run/php/php" + v + "-fpm.sock"
		if _, err := os.Stat(s); err == nil {
			sock = s
			break
		}
	}
	if sock == "" {
		return fmt.Errorf("no PHP-FPM socket under /run/php — install PHP first")
	}
	// If the file is already correct (points at the socket we'd pick), skip.
	// Otherwise rewrite — covers the case where install.sh wrote the snippet
	// against a PHP-FPM version that's since been removed or replaced.
	if data, err := os.ReadFile(path); err == nil && strings.Contains(string(data), "fastcgi_pass unix:"+sock+";") {
		return nil
	}
	content := `# hostQ phpMyAdmin alias. Sites include this from their server block
# so https://<domain>/phpmyadmin works without a separate vhost.
location ^~ /phpmyadmin/ {
    alias /usr/share/phpmyadmin/;
    index index.php;
    try_files $uri $uri/ /phpmyadmin/index.php?$args;
    location ~ ^/phpmyadmin/(.+\.php)$ {
        alias /usr/share/phpmyadmin/$1;
        include snippets/fastcgi-php.conf;
        fastcgi_pass unix:` + sock + `;
        fastcgi_param SCRIPT_FILENAME $request_filename;
        include fastcgi_params;
    }
    location ~* ^/phpmyadmin/(.+\.(jpg|jpeg|gif|css|png|js|ico|html|xml|txt|woff|woff2|svg|map))$ {
        alias /usr/share/phpmyadmin/$1;
        access_log off;
        expires 1d;
    }
}
location = /phpmyadmin { return 301 /phpmyadmin/; }
`
	_ = os.MkdirAll("/etc/nginx/snippets", 0755)
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return err
	}
	a.audit("pma.snippet", "success", "auto-created using "+sock)
	return nil
}

// ensurePMADefaultVhost writes /etc/nginx/sites-available/hostq-default the
// first time it's needed. Without an explicit default_server on :80, nginx
// picks an arbitrary hostQ-managed vhost — one that very likely has SSL and
// returns 301 to https — for any unmatched Host header (e.g. requests to the
// bare IP). That ricochets into a too-many-redirects loop the first time the
// user opens /phpmyadmin/ on the IP.
//
// The default vhost only exposes /phpmyadmin/ (via the panel-managed snippet)
// and returns 404 for everything else, so it can't accidentally serve a real
// site. Idempotent: if the file already exists, we just confirm the symlink.
func (a *App) ensurePMADefaultVhost() error {
	if err := a.ensurePMASnippet(); err != nil {
		return err
	}
	// Debian/Ubuntu's nginx ships an /etc/nginx/sites-enabled/default vhost
	// that also claims `listen 80 default_server` + `server_name _;`. When
	// both exist, nginx warns "conflicting server name '_'" and ignores
	// whichever loaded second. sites-enabled is read alphabetically, so the
	// stock "default" wins over "hostq-default" — and our snippet never gets
	// reached. Two-pronged fix below: disable the stock default if it's
	// still enabled, and symlink ours under a "00-" prefix so future
	// re-enables of the stock one still lose.
	if _, err := os.Lstat("/etc/nginx/sites-enabled/default"); err == nil {
		_ = os.Remove("/etc/nginx/sites-enabled/default")
		a.audit("pma.default-vhost", "success", "disabled stock /etc/nginx/sites-enabled/default")
	}
	const path = "/etc/nginx/sites-available/hostq-default"
	// "00-" prefix so sites-enabled loads us before any other vhost — that
	// guarantees we own the default_server flag on :80 even if a future
	// package install re-creates Debian's default link.
	const link = "/etc/nginx/sites-enabled/00-hostq-default"
	const oldLink = "/etc/nginx/sites-enabled/hostq-default"
	const content = `# hostQ default vhost — auto-created by the panel.
# Catches requests where the Host header doesn't match any per-site
# server_name (typically: bare IP access). Exposes /phpmyadmin/ via the
# managed snippet and 404s everything else so it can't serve a site.
server {
    listen 80 default_server;
    listen [::]:80 default_server;
    server_name _;
    access_log /var/log/nginx/hostq-default.access.log combined;

    include snippets/hostq-pma.conf;

    location / {
        return 404 "hostQ default vhost. Add a site or use a real domain.\n";
    }
}
`
	if _, err := os.Stat(path); err != nil {
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			return err
		}
		a.audit("pma.default-vhost", "success", "auto-created")
	}
	// Migrate from the old un-prefixed symlink to the new "00-" prefix.
	if _, err := os.Lstat(oldLink); err == nil {
		_ = os.Remove(oldLink)
	}
	if _, err := os.Lstat(link); err != nil {
		if err := os.Symlink(path, link); err != nil {
			return err
		}
	}
	if out, err := exec.Command("nginx", "-t").CombinedOutput(); err != nil {
		return fmt.Errorf("nginx -t failed: %s", strings.TrimSpace(string(out)))
	}
	_ = exec.Command("systemctl", "reload", "nginx").Run()
	return nil
}

// pmaLogin signs the panel user into phpMyAdmin without an interactive login.
//
// Strategy:
//  1. Fetch /phpmyadmin/index.php server-side via local nginx with the user's
//     Host header so the right vhost serves it (carrying our hostq-pma.conf).
//  2. Extract the CSRF token field and the session cookie pma sets in its
//     login response.
//  3. Forward that session cookie to the user's browser via Set-Cookie on the
//     same host the panel is being accessed on — so the upcoming POST from
//     the browser to phpMyAdmin carries it (same-origin) and the token
//     matches the session it was issued for.
//  4. Render an auto-submitting form POSTing username + password + token to
//     the phpMyAdmin URL on the bare host (port stripped) so even a panel
//     hit on :8090 directs the browser to nginx on :80/:443.
//
// For SSO to work end-to-end the panel and pma have to share an origin (same
// host, same port). When the panel is being accessed on its setup-only :8090
// listener, the splash page still loads but the POST will be cross-origin and
// the forwarded cookies won't apply — the user lands on phpMyAdmin's manual
// login. The splash hints at the recommended fix in that case.
func (a *App) pmaLogin(w http.ResponseWriter, r *http.Request) {
	user := safeDBUser(r.URL.Query().Get("user"))
	db := safeDBName(r.URL.Query().Get("db"))
	domain := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("domain")))
	if user == "" {
		http.Error(w, "missing user", http.StatusBadRequest)
		return
	}
	cred, err := a.lookupCred(user)
	if err != nil {
		renderPMAFallback(w, user, db, domain)
		a.audit("pma.signon", "failure", user)
		return
	}

	// Bare host (strip any port like :8090) — pma sits behind nginx on the
	// site's normal :80/:443 listener.
	host := r.Host
	bareHost := host
	if i := strings.LastIndex(host, ":"); i > 0 {
		bareHost = host[:i]
	}
	if bareHost == "" {
		bareHost = "localhost"
	}
	directPort := host != bareHost

	// Choose scheme for the user's redirect: keep their current scheme when
	// they came in via TLS, else http. (PMA SSO across https↔http boundaries
	// would lose cookies, so we don't try to upgrade.)
	scheme := "http"
	if r.TLS != nil || strings.EqualFold(r.Header.Get("X-Forwarded-Proto"), "https") {
		scheme = "https"
	}
	pmaURL := scheme + "://" + bareHost + "/phpmyadmin/index.php"

	// Server-side fetch of the pma login page through local nginx to grab the
	// CSRF token + session cookie.
	token, cookies := fetchPMASession(bareHost)
	for _, c := range cookies {
		c.Path = "/"
		http.SetCookie(w, c)
	}

	a.audit("pma.signon", "success", user)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Referrer-Policy", "same-origin")

	tokenInput := ""
	if token != "" {
		tokenInput = `<input type="hidden" name="token" value="` + html.EscapeString(token) + `">`
	}
	dbInput := ""
	if db != "" {
		dbInput = `<input type="hidden" name="db" value="` + html.EscapeString(db) + `">`
	}
	hint := ""
	if directPort {
		hint = `<div class="muted" style="margin-top:14px;font-size:12.5px;max-width:380px;line-height:1.5">You are using the panel's direct setup port (` + html.EscapeString(host) + `). For phpMyAdmin single sign-on to work, access the panel through your nginx-proxied domain instead — the panel and phpMyAdmin need to share an origin.</div>`
	}

	fmt.Fprintf(w, `<!doctype html><html><head><meta charset="utf-8"><title>Signing in to phpMyAdmin…</title>
<style>body{margin:0;display:grid;place-items:center;height:100vh;font-family:Inter,system-ui,sans-serif;background:#0b1220;color:#e2e8f0;padding:24px;text-align:center}
.box{max-width:520px}.spinner{width:32px;height:32px;border:3px solid #1e293b;border-top-color:#3b82f6;border-radius:50%%;margin:0 auto 14px;animation:s 1s linear infinite}
@keyframes s{to{transform:rotate(360deg)}}
.muted{color:#94a3b8;font-size:13px;margin-top:6px}
button.link{background:none;border:none;color:#60a5fa;text-decoration:underline;cursor:pointer;font:inherit}</style></head>
<body><div class="box"><div class="spinner"></div><div>Signing you in to phpMyAdmin…</div><div class="muted">If nothing happens, <button class="link" onclick="document.forms.f.submit()">click here</button>.</div>
<form id="f" name="f" method="post" action="%s" autocomplete="off">
  <input type="hidden" name="pma_username" value="%s">
  <input type="hidden" name="pma_password" value="%s">
  <input type="hidden" name="server" value="1">
  %s
  %s
</form>
%s
</div>
<script>setTimeout(function(){document.forms.f.submit();}, 80);</script>
</body></html>`,
		html.EscapeString(pmaURL),
		html.EscapeString(cred.User),
		html.EscapeString(cred.Password),
		tokenInput,
		dbInput,
		hint,
	)
}

func renderPMAFallback(w http.ResponseWriter, user, db, domain string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	backLabel := "Back to site"
	backHref := "/site?domain=" + html.EscapeString(domain) + "&tab=database"
	if strings.TrimSpace(domain) == "" {
		backLabel = "Back to databases"
		backHref = "/databases"
	}
	fmt.Fprintf(w, `<!doctype html><html><body style="font-family:sans-serif;padding:40px;max-width:520px;margin:auto;line-height:1.55">
<h2>No saved password for <code>%s</code></h2>
<p>This database user wasn't created or reset through hostQ, so no password is remembered for auto-login.</p>
<p>Reset the user's password from the Databases page and then click phpMyAdmin again, or open phpMyAdmin manually below.</p>
<p><a href="/phpmyadmin/?db=%s" style="display:inline-block;background:#2563eb;color:#fff;padding:10px 18px;border-radius:8px;text-decoration:none">Open phpMyAdmin</a> &nbsp; <a href="%s">%s</a></p>
</body></html>`,
		html.EscapeString(user), html.EscapeString(db), backHref, html.EscapeString(backLabel))
}

var pmaTokenRe = regexp.MustCompile(`name=["']token["'][^>]*value=["']([^"']+)["']|value=["']([^"']+)["'][^>]*name=["']token["']`)

func extractPMAToken(body string) string {
	if m := pmaTokenRe.FindStringSubmatch(body); m != nil {
		if m[1] != "" {
			return m[1]
		}
		return m[2]
	}
	return ""
}

// fetchPMASession GETs the phpMyAdmin login page through local nginx (with the
// user's Host header so the right vhost answers), and returns the CSRF token
// + the Set-Cookie values pma issued. Tries http first, falls back to https
// with cert verification disabled (local loopback).
func fetchPMASession(hostHeader string) (string, []*http.Cookie) {
	tries := []struct {
		url       string
		transport *http.Transport
	}{
		{url: "http://127.0.0.1/phpmyadmin/index.php"},
		{url: "https://127.0.0.1/phpmyadmin/index.php",
			transport: &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}},
	}
	for _, t := range tries {
		client := &http.Client{
			Timeout: 8 * time.Second,
			CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
				return http.ErrUseLastResponse
			},
		}
		if t.transport != nil {
			client.Transport = t.transport
		}
		req, err := http.NewRequest(http.MethodGet, t.url, nil)
		if err != nil {
			continue
		}
		req.Host = hostHeader
		req.Header.Set("User-Agent", "hostQ-pma-signon/1")
		req.Header.Set("Accept", "text/html")
		resp, err := client.Do(req)
		if err != nil {
			continue
		}
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
		resp.Body.Close()
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			continue
		}
		token := extractPMAToken(string(body))
		if token != "" || len(resp.Cookies()) > 0 {
			return token, resp.Cookies()
		}
	}
	return "", nil
}
