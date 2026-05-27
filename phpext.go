package main

import (
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// commonPHPExtensions is the curated set surfaced on the panel for one-click
// install even when the apt package isn't on disk yet. Each one is available
// on Ubuntu/Debian as php<ver>-<name>.
var commonPHPExtensions = []string{
	"bcmath", "bz2", "curl", "gd", "gmp",
	"imagick", "imap", "intl",
	"ldap", "mbstring", "memcached", "mongodb", "mysql",
	"opcache", "pgsql", "redis", "soap", "sqlite3",
	"tidy", "xdebug", "xml", "xsl", "zip",
}

var phpExtRe = regexp.MustCompile(`^[a-z0-9_-]{1,40}$`)

// phpExtensions enumerates every PHP extension the panel knows about for the
// given PHP version: installed-and-enabled, installed-but-disabled, and the
// curated common set we expose for install. The list is sorted by name.
func phpExtensions(ver string) []PHPExtension {
	if !phpVersionRe.MatchString(ver) {
		return nil
	}
	seen := map[string]*PHPExtension{}

	// Modules with an .ini in mods-available are installed via apt; the file
	// is owned by the php<ver>-<name> package on Debian/Ubuntu.
	modsDir := filepath.Join("/etc/php", ver, "mods-available")
	if entries, err := os.ReadDir(modsDir); err == nil {
		for _, e := range entries {
			if e.IsDir() || !strings.HasSuffix(e.Name(), ".ini") {
				continue
			}
			name := strings.TrimSuffix(e.Name(), ".ini")
			if !phpExtRe.MatchString(name) {
				continue
			}
			seen[name] = &PHPExtension{
				Name:      name,
				Apt:       "php" + ver + "-" + name,
				Installed: true,
			}
		}
	}

	// Symlinks present in fpm/conf.d/ → enabled for the FPM SAPI. File names
	// look like "20-gd.ini"; strip the priority prefix.
	fpmDir := filepath.Join("/etc/php", ver, "fpm", "conf.d")
	if entries, err := os.ReadDir(fpmDir); err == nil {
		for _, e := range entries {
			name := strings.TrimSuffix(e.Name(), ".ini")
			if i := strings.Index(name, "-"); i >= 0 && i <= 3 {
				name = name[i+1:]
			}
			if ext, ok := seen[name]; ok {
				ext.Enabled = true
			}
		}
	}

	// `php -m` lists what's actually loaded by the CLI; close enough to FPM
	// for showing a "Loaded" badge on each row.
	if out, err := exec.Command("php"+ver, "-m").Output(); err == nil {
		for _, line := range strings.Split(string(out), "\n") {
			line = strings.ToLower(strings.TrimSpace(line))
			if line == "" || strings.HasPrefix(line, "[") {
				continue
			}
			if ext, ok := seen[line]; ok {
				ext.Loaded = true
			}
		}
	}

	for _, name := range commonPHPExtensions {
		if ext, ok := seen[name]; ok {
			ext.Common = true
			continue
		}
		seen[name] = &PHPExtension{
			Name:   name,
			Apt:    "php" + ver + "-" + name,
			Common: true,
		}
	}

	out := make([]PHPExtension, 0, len(seen))
	for _, ext := range seen {
		out = append(out, *ext)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

// sitePhpExt handles the per-site PHP extension form on the site's PHP tab.
// Two actions:
//
//	apply     — reconciles every extension to the checked/unchecked state
//	uninstall — apt purge of a single extension
//
// All work targets the PHP version the site currently uses. Because PHP
// modules are global per FPM pool/version, the change affects every site
// sharing that PHP version — the UI calls this out explicitly.
func (a *App) sitePhpExt(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/sites", http.StatusSeeOther)
		return
	}
	domain := strings.ToLower(strings.TrimSpace(r.FormValue("domain")))
	site, ok := a.findSite(domain)
	if !ok {
		http.Redirect(w, r, "/sites", http.StatusSeeOther)
		return
	}
	ver := site.PHPVersion
	if !phpVersionRe.MatchString(ver) {
		http.Redirect(w, r, "/site?domain="+domain+"&tab=php", http.StatusSeeOther)
		return
	}
	action := r.FormValue("action")
	output := ""
	switch action {
	case "apply":
		_ = r.ParseForm()
		desired := map[string]bool{}
		for k := range r.PostForm {
			if strings.HasPrefix(k, "ext_") {
				name := strings.TrimPrefix(k, "ext_")
				if phpExtRe.MatchString(name) {
					desired[name] = true
				}
			}
		}
		installed, enabled, disabled, errs := 0, 0, 0, 0
		for _, ext := range phpExtensions(ver) {
			want := desired[ext.Name]
			switch {
			case want && !ext.Installed:
				if _, err := aptInstall("php" + ver + "-" + ext.Name); err != nil {
					errs++
					continue
				}
				installed++
				if err := exec.Command("phpenmod", "-v", ver, ext.Name).Run(); err == nil {
					enabled++
				}
			case want && ext.Installed && !ext.Enabled:
				if err := exec.Command("phpenmod", "-v", ver, ext.Name).Run(); err == nil {
					enabled++
				}
			case !want && ext.Enabled:
				if err := exec.Command("phpdismod", "-v", ver, ext.Name).Run(); err == nil {
					disabled++
				}
			}
		}
		_ = exec.Command("systemctl", "reload", "php"+ver+"-fpm").Run()
		a.cache.invalidate("php", "services")
		output = fmt.Sprintf("PHP %s: installed %d, enabled %d, disabled %d", ver, installed, enabled, disabled)
		if errs > 0 {
			output += fmt.Sprintf(", %d failed (see journalctl)", errs)
		}
		a.audit("php.ext-apply", "success", domain+"/"+ver)
	case "uninstall":
		name := r.FormValue("ext")
		if !phpExtRe.MatchString(name) {
			output = "Invalid extension name"
			break
		}
		out, err := aptRemove("php" + ver + "-" + name)
		if err != nil {
			output = "Uninstall failed: " + tail(string(out), 200)
			a.audit("php.ext-uninstall", "failure", domain+"/"+name)
			break
		}
		_ = exec.Command("systemctl", "reload", "php"+ver+"-fpm").Run()
		a.cache.invalidate("php", "services")
		output = "Uninstalled php" + ver + "-" + name
		a.audit("php.ext-uninstall", "success", domain+"/"+name)
	default:
		output = "Unknown action"
	}
	http.Redirect(w, r, "/site?domain="+domain+"&tab=php&output="+queryEscape(output), http.StatusSeeOther)
}
