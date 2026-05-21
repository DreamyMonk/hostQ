package main

import (
	"fmt"
	"html"
	"net/http"
	"strings"
)

// pmaLogin emits a tiny self-submitting HTML form that POSTs the saved
// credentials for the requested DB user to /phpmyadmin/index.php. phpMyAdmin
// uses cookie auth by default on Debian/Ubuntu and accepts username/password
// via POST on its index page, so the browser ends up signed in.
//
// Because the form is rendered same-origin (panel + pma sit behind the same
// nginx vhost) the resulting cookie is set on the right host. The page is a
// throwaway "logging in…" splash — the user never sees a login form.
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
		// No remembered password. Fall back to a friendly page with manual
		// "click here to open phpMyAdmin and sign in by hand".
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprintf(w, `<!doctype html><html><body style="font-family:sans-serif;padding:40px;max-width:520px;margin:auto;line-height:1.55">
<h2>No saved password for <code>%s</code></h2>
<p>This database user wasn't created or reset through hostQ, so no password is remembered for auto-login.</p>
<p>Reset the user's password from <strong>Site → Database</strong> and then click phpMyAdmin again, or open phpMyAdmin manually below.</p>
<p><a href="/phpmyadmin/?db=%s" style="display:inline-block;background:#2563eb;color:#fff;padding:10px 18px;border-radius:8px;text-decoration:none">Open phpMyAdmin</a> &nbsp; <a href="/site?domain=%s&tab=database">Back to site</a></p>
</body></html>`, html.EscapeString(user), html.EscapeString(db), html.EscapeString(domain))
		a.audit("pma.signon", "failure", user)
		return
	}
	a.audit("pma.signon", "success", user)
	// no caching this redirect splash
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Referrer-Policy", "same-origin")
	// We escape user/db/password into form values; pma_password contains the
	// real password and is submitted as POST body (not URL) so it doesn't end
	// up in server logs.
	target := "/phpmyadmin/index.php"
	dbParam := ""
	if db != "" {
		dbParam = `<input type="hidden" name="db" value="` + html.EscapeString(db) + `">`
	}
	fmt.Fprintf(w, `<!doctype html><html><head><meta charset="utf-8"><title>Signing in to phpMyAdmin…</title>
<style>body{margin:0;display:grid;place-items:center;height:100vh;font-family:Inter,system-ui,sans-serif;background:#0b1220;color:#e2e8f0}
.box{text-align:center}.spinner{width:32px;height:32px;border:3px solid #1e293b;border-top-color:#3b82f6;border-radius:50%%;margin:0 auto 14px;animation:s 1s linear infinite}
@keyframes s{to{transform:rotate(360deg)}}
.muted{color:#94a3b8;font-size:13px;margin-top:6px}</style></head>
<body><div class="box"><div class="spinner"></div><div>Signing you in to phpMyAdmin…</div><div class="muted">If nothing happens, <button onclick="document.forms.f.submit()" style="background:none;border:none;color:#60a5fa;text-decoration:underline;cursor:pointer;font:inherit">click here</button>.</div>
<form id="f" name="f" method="post" action="%s" autocomplete="off">
  <input type="hidden" name="pma_username" value="%s">
  <input type="hidden" name="pma_password" value="%s">
  <input type="hidden" name="server" value="1">
  <input type="hidden" name="target" value="index.php">
  %s
</form></div>
<script>document.forms.f.submit();</script>
</body></html>`,
		target,
		html.EscapeString(cred.User),
		html.EscapeString(cred.Password),
		dbParam,
	)
}
