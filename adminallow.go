package main

import (
	"encoding/json"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// AdminAllowlist gates the /admin scope by IP/CIDR. When Enabled is false
// (or the file is missing), the gate is a no-op so a fresh install never
// locks anyone out. When Enabled is true, requests to admin routes from an
// IP not in Entries get 403. Loopback (127.0.0.1, ::1) is *always* allowed —
// the operator can always recover via the panel direct port on the box.
type AdminAllowlist struct {
	Enabled bool                  `json:"enabled"`
	Entries []AdminAllowlistEntry `json:"entries"`
	Updated string                `json:"updated"`
}

type AdminAllowlistEntry struct {
	CIDR    string `json:"cidr"`
	Note    string `json:"note,omitempty"`
	Added   string `json:"added"`
}

const adminAllowlistFile = "admin-allowlist.json"

// adminScopePrefixes is the set of routes the allowlist gates. Kept in sync
// with the IsAdmin switch in render() so the gate matches what the sidebar
// considers "admin".
var adminScopePrefixes = []string{
	"/services",
	"/security",
	"/malfix",
	"/account",
	"/audit",
	"/php",
	"/redis",
	"/firewall",
}

func (a *App) loadAdminAllowlist() AdminAllowlist {
	var s AdminAllowlist
	data, err := os.ReadFile(filepath.Join(a.cfg.DataDir, adminAllowlistFile))
	if err != nil {
		return s
	}
	_ = json.Unmarshal(data, &s)
	return s
}

func (a *App) saveAdminAllowlist(s AdminAllowlist) error {
	if err := os.MkdirAll(a.cfg.DataDir, 0700); err != nil {
		return err
	}
	s.Updated = time.Now().Format(time.RFC3339)
	data, _ := json.MarshalIndent(s, "", "  ")
	return os.WriteFile(filepath.Join(a.cfg.DataDir, adminAllowlistFile), data, 0600)
}

// clientIP returns the best-guess source IP for a request. The panel runs
// behind nginx with X-Forwarded-For set, so prefer the leftmost entry there
// (the original client) and fall back to RemoteAddr when direct.
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

// ipMatchesEntry returns true if ip is inside cidr, where cidr may be a bare
// IP ("203.0.113.4"), a CIDR ("203.0.113.0/24"), or an IPv6 form. Bare IPs
// are matched as /32 (or /128 for v6).
func ipMatchesEntry(ip, cidr string) bool {
	parsed := net.ParseIP(ip)
	if parsed == nil {
		return false
	}
	if !strings.Contains(cidr, "/") {
		other := net.ParseIP(cidr)
		return other != nil && other.Equal(parsed)
	}
	_, network, err := net.ParseCIDR(cidr)
	if err != nil {
		return false
	}
	return network.Contains(parsed)
}

func (a *App) ipAllowed(r *http.Request) bool {
	list := a.loadAdminAllowlist()
	if !list.Enabled || len(list.Entries) == 0 {
		return true
	}
	ip := clientIP(r)
	// Loopback always allowed — last-resort recovery.
	if ip == "127.0.0.1" || ip == "::1" {
		return true
	}
	for _, e := range list.Entries {
		if ipMatchesEntry(ip, e.CIDR) {
			return true
		}
	}
	return false
}

// requireAdminAllow wraps requireAuth-protected admin handlers with the
// allowlist gate. Non-admin scope routes are unaffected.
func (a *App) requireAdminAllow(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !a.ipAllowed(r) {
			a.audit("admin.blocked", "failure", clientIP(r)+" "+r.URL.Path)
			http.Error(w, "Admin area is restricted. Your IP is not on the allowlist.\n\nIf you are the operator, sign in via the panel direct port from localhost (127.0.0.1) and remove or update the allowlist on /account.", http.StatusForbidden)
			return
		}
		next(w, r)
	}
}

// allowlistAction handles add/remove/toggle from the Account page.
func (a *App) allowlistAction(r *http.Request) string {
	list := a.loadAdminAllowlist()
	switch r.FormValue("action") {
	case "allow-add":
		cidr := strings.TrimSpace(r.FormValue("cidr"))
		note := strings.TrimSpace(r.FormValue("note"))
		if cidr == "" {
			return "Provide an IP or CIDR (eg 203.0.113.5 or 203.0.113.0/24)."
		}
		// Validate.
		if strings.Contains(cidr, "/") {
			if _, _, err := net.ParseCIDR(cidr); err != nil {
				return "Invalid CIDR: " + err.Error()
			}
		} else if net.ParseIP(cidr) == nil {
			return "Invalid IP address."
		}
		for _, e := range list.Entries {
			if e.CIDR == cidr {
				return "Entry already in the allowlist."
			}
		}
		list.Entries = append(list.Entries, AdminAllowlistEntry{
			CIDR: cidr, Note: note, Added: time.Now().Format(time.RFC3339),
		})
		if err := a.saveAdminAllowlist(list); err != nil {
			return "Save failed: " + err.Error()
		}
		a.audit("admin.allowlist-add", "success", cidr)
		return "Added " + cidr + " to allowlist."
	case "allow-remove":
		cidr := r.FormValue("cidr")
		kept := list.Entries[:0]
		for _, e := range list.Entries {
			if e.CIDR != cidr {
				kept = append(kept, e)
			}
		}
		list.Entries = kept
		if err := a.saveAdminAllowlist(list); err != nil {
			return "Save failed: " + err.Error()
		}
		a.audit("admin.allowlist-remove", "success", cidr)
		return "Removed " + cidr + " from allowlist."
	case "allow-toggle":
		// Safety: refuse to enable an empty list because that would
		// instantly lock the operator out of /account (only loopback
		// would still work). Force them to add at least one entry first.
		want := r.FormValue("enabled") == "1"
		if want && len(list.Entries) == 0 {
			return "Add at least one IP/CIDR before enabling the allowlist."
		}
		list.Enabled = want
		if err := a.saveAdminAllowlist(list); err != nil {
			return "Save failed: " + err.Error()
		}
		if want {
			a.audit("admin.allowlist-enable", "success", "")
			return "Allowlist enabled."
		}
		a.audit("admin.allowlist-disable", "success", "")
		return "Allowlist disabled."
	}
	return ""
}
