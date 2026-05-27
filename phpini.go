package main

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Per-site PHP value overrides — persisted at
// /etc/hostq/sites/<domain>.php.ini. writeNginxSite collects every non-
// blank, non-comment line and emits one fastcgi_param PHP_VALUE inside the
// site's .php location, so changes apply on the next Nginx reload without
// editing the system-wide php.ini.

// managedPHPKeys is the curated set the friendly form exposes. The order
// is what gets rendered in the UI.
var managedPHPKeys = []struct {
	Key   string
	Label string
	Hint  string
}{
	{"upload_max_filesize", "Upload max filesize", "Max size of a single uploaded file. Example: 64M"},
	{"post_max_size", "POST max size", "Max size of an HTTP POST body (form + uploads). Set ≥ upload_max_filesize. Example: 64M"},
	{"memory_limit", "Memory limit", "Per-request memory budget. Example: 256M"},
	{"max_execution_time", "Max execution time", "Seconds before PHP kills a script. Example: 300"},
	{"max_input_time", "Max input time", "Seconds to spend parsing the request body. Example: 60"},
	{"max_input_vars", "Max input vars", "Hard cap on POST/GET variables. Example: 3000"},
	{"date.timezone", "Default timezone", "IANA name. Example: UTC, Europe/Berlin, Asia/Kolkata"},
	{"display_errors", "Display errors", "On (dev) / Off (prod)"},
	{"file_uploads", "File uploads", "On / Off"},
	{"allow_url_fopen", "allow_url_fopen", "On / Off — required by some plugins, security risk if not needed"},
}

func (a *App) sitePHPIniPath(domain string) string {
	return filepath.Join(a.cfg.DataDir, "sites", domain+".php.ini")
}

func (a *App) loadSitePHPIni(domain string) string {
	data, _ := os.ReadFile(a.sitePHPIniPath(domain))
	return string(data)
}

func (a *App) saveSitePHPIni(domain, content string) error {
	p := a.sitePHPIniPath(domain)
	if strings.TrimSpace(content) == "" {
		_ = os.Remove(p)
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(p), 0700); err != nil {
		return err
	}
	return os.WriteFile(p, []byte(content), 0644)
}

// parseSitePHPIni returns a key→value map. Comments and section headers
// are skipped; values are stripped of any wrapping quotes.
func parseSitePHPIni(content string) map[string]string {
	out := map[string]string{}
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, ";") || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "[") {
			continue
		}
		i := strings.Index(line, "=")
		if i <= 0 {
			continue
		}
		k := strings.TrimSpace(line[:i])
		v := strings.TrimSpace(line[i+1:])
		v = strings.Trim(v, `"'`)
		out[k] = v
	}
	return out
}

// buildSitePHPIniFromForm merges the managed form values with the existing
// raw INI, overwriting just the managed keys. Anything else the user typed
// in via the full editor is preserved.
func buildSitePHPIniFromForm(existing string, form map[string]string) string {
	current := parseSitePHPIni(existing)
	for _, m := range managedPHPKeys {
		v := strings.TrimSpace(form[m.Key])
		if v == "" {
			delete(current, m.Key)
		} else {
			current[m.Key] = v
		}
	}
	keys := make([]string, 0, len(current))
	for k := range current {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var b strings.Builder
	b.WriteString("; hostQ — per-site PHP overrides for this domain.\n")
	b.WriteString("; Applied via fastcgi_param PHP_VALUE in the site's nginx vhost.\n")
	for _, k := range keys {
		b.WriteString(k)
		b.WriteString("=")
		b.WriteString(current[k])
		b.WriteString("\n")
	}
	return b.String()
}

// renderPHPValueBlock returns the contents of `fastcgi_param PHP_VALUE "..."`
// (with the opening indent + leading newline) to splice into the .php
// location. Empty string means no override file exists.
func renderPHPValueBlock(rawIni string) string {
	kept := []string{}
	for _, line := range strings.Split(rawIni, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, ";") || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "[") {
			continue
		}
		kept = append(kept, line)
	}
	if len(kept) == 0 {
		return ""
	}
	return "\n        fastcgi_param PHP_VALUE \"" + strings.Join(kept, "\n") + "\";"
}
