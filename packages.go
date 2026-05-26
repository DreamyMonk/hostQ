package main

import (
	"net/http"
	"os/exec"
	"sort"
	"strings"
)

// PackageDef describes an apt package that hostQ knows how to install/uninstall
// on demand. Kept narrow on purpose — only packages we genuinely manage.
type PackageDef struct {
	ID          string // short ID used in URLs
	Name        string // display name
	Apt         string // apt-get package name (sometimes a space-separated list)
	Service     string // systemd unit (empty if not a service)
	Description string
	Group       string // grouping for the UI
}

// PackageInfo is the runtime snapshot of a PackageDef.
type PackageInfo struct {
	PackageDef
	Installed bool
	Version   string
	Active    bool
}

func managedPackages() []PackageDef {
	return []PackageDef{
		{ID: "redis", Name: "Redis", Apt: "redis-server", Service: "redis-server", Description: "Optional in-memory cache for WordPress object cache plugins.", Group: "Cache"},
		{ID: "php82", Name: "PHP 8.2 FPM", Apt: "php8.2-fpm php8.2-cli php8.2-mysql php8.2-xml php8.2-mbstring php8.2-curl php8.2-zip php8.2-gd php8.2-intl", Service: "php8.2-fpm", Description: "PHP 8.2 FastCGI Process Manager.", Group: "PHP"},
		{ID: "php83", Name: "PHP 8.3 FPM", Apt: "php8.3-fpm php8.3-cli php8.3-mysql php8.3-xml php8.3-mbstring php8.3-curl php8.3-zip php8.3-gd php8.3-intl", Service: "php8.3-fpm", Description: "PHP 8.3 FastCGI Process Manager.", Group: "PHP"},
		{ID: "php84", Name: "PHP 8.4 FPM", Apt: "php8.4-fpm php8.4-cli php8.4-mysql php8.4-xml php8.4-mbstring php8.4-curl php8.4-zip php8.4-gd php8.4-intl", Service: "php8.4-fpm", Description: "PHP 8.4 FastCGI Process Manager.", Group: "PHP"},
		{ID: "php85", Name: "PHP 8.5 FPM", Apt: "php8.5-fpm php8.5-cli php8.5-mysql php8.5-xml php8.5-mbstring php8.5-curl php8.5-zip php8.5-gd php8.5-intl", Service: "php8.5-fpm", Description: "PHP 8.5 FastCGI Process Manager.", Group: "PHP"},
		{ID: "certbot", Name: "Certbot (Let's Encrypt)", Apt: "certbot python3-certbot-nginx", Description: "Free TLS certificates via ACME.", Group: "TLS"},
		{ID: "phpmyadmin", Name: "phpMyAdmin", Apt: "phpmyadmin", Description: "Web UI for MariaDB/MySQL. Auto-login from the panel is wired through this.", Group: "Database"},
		{ID: "pureftpd", Name: "Pure-FTPd", Apt: "pure-ftpd pure-ftpd-common", Service: "pure-ftpd", Description: "Lightweight FTP server.", Group: "Transfer"},
		{ID: "wpcli", Name: "WP-CLI", Apt: "", Description: "Already installed by install.sh from the official phar build.", Group: "Tooling"},
	}
}

// isAptInstalled returns true only when dpkg-query reports the package status
// as "install ok installed". Works for packages whose Apt field is a single
// name; multi-package definitions report installed when the first listed
// package is present (that's typically the umbrella one).
func isAptInstalled(pkg string) bool {
	if pkg == "" {
		return false
	}
	first := strings.Fields(pkg)[0]
	out, err := exec.Command("dpkg-query", "-W", "-f=${Status}", first).Output()
	if err != nil {
		return false
	}
	return strings.Contains(string(out), "install ok installed")
}

func aptPackageVersion(pkg string) string {
	if pkg == "" {
		return ""
	}
	first := strings.Fields(pkg)[0]
	out, err := exec.Command("dpkg-query", "-W", "-f=${Version}", first).Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func systemdActive(unit string) bool {
	if unit == "" {
		return false
	}
	out, err := exec.Command("systemctl", "is-active", unit).Output()
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(out)) == "active"
}

// aptInstall handles the special-case preseed for phpmyadmin (since the
// package is interactive by default and silently fails otherwise).
func aptInstall(apt string) ([]byte, error) {
	if apt == "" {
		return nil, nil
	}
	args := append([]string{"install", "-y", "-qq"}, strings.Fields(apt)...)
	cmd := exec.Command("apt-get", args...)
	cmd.Env = append(cmd.Env, "DEBIAN_FRONTEND=noninteractive")
	return cmd.CombinedOutput()
}

func aptRemove(apt string) ([]byte, error) {
	if apt == "" {
		return nil, nil
	}
	args := append([]string{"purge", "-y", "-qq"}, strings.Fields(apt)...)
	cmd := exec.Command("apt-get", args...)
	cmd.Env = append(cmd.Env, "DEBIAN_FRONTEND=noninteractive")
	return cmd.CombinedOutput()
}

func (a *App) listPackagesInfo() []PackageInfo {
	defs := managedPackages()
	out := make([]PackageInfo, 0, len(defs))
	for _, d := range defs {
		info := PackageInfo{PackageDef: d}
		if d.ID == "wpcli" {
			info.Installed = exec.Command("wp", "--version").Run() == nil
			info.Active = info.Installed
		} else {
			info.Installed = isAptInstalled(d.Apt)
			if info.Installed {
				info.Version = aptPackageVersion(d.Apt)
			}
			info.Active = systemdActive(d.Service)
		}
		out = append(out, info)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Group != out[j].Group {
			return out[i].Group < out[j].Group
		}
		return out[i].Name < out[j].Name
	})
	return out
}

func (a *App) findPackageDef(id string) (PackageDef, bool) {
	for _, d := range managedPackages() {
		if d.ID == id {
			return d, true
		}
	}
	return PackageDef{}, false
}

func (a *App) packages(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		a.packagesAction(w, r)
		return
	}
	a.render(w, "packages", map[string]any{
		"Title":    "Packages",
		"Packages": a.listPackagesInfo(),
	})
}

func (a *App) packagesAction(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimSpace(r.FormValue("id"))
	action := r.FormValue("action")
	def, ok := a.findPackageDef(id)
	output := ""
	if !ok {
		http.Redirect(w, r, "/packages?output="+queryEscape("Unknown package id"), http.StatusSeeOther)
		return
	}
	if def.Apt == "" {
		http.Redirect(w, r, "/packages?output="+queryEscape(def.Name+" is not managed by apt"), http.StatusSeeOther)
		return
	}
	switch action {
	case "install":
		// phpmyadmin needs preseed
		if def.ID == "phpmyadmin" {
			pmaPreseed()
		}
		out, err := aptInstall(def.Apt)
		if err != nil {
			output = "Install failed: " + tail(string(out), 240)
			a.audit("packages.install", "failure", id)
		} else {
			output = def.Name + " installed."
			a.audit("packages.install", "success", id)
			// Enable + start if it has a unit
			if def.Service != "" {
				_ = exec.Command("systemctl", "enable", "--now", def.Service).Run()
			}
			a.cache.invalidate("services", "php")
		}
	case "uninstall":
		out, err := aptRemove(def.Apt)
		if err != nil {
			output = "Uninstall failed: " + tail(string(out), 240)
			a.audit("packages.uninstall", "failure", id)
		} else {
			output = def.Name + " uninstalled."
			a.audit("packages.uninstall", "success", id)
			a.cache.invalidate("services", "php")
		}
	default:
		output = "Unknown action"
	}
	http.Redirect(w, r, "/packages?output="+queryEscape(output), http.StatusSeeOther)
}

// pmaPreseed writes the debconf answers phpmyadmin needs to install
// non-interactively. Uses a random password since we don't use the
// dbconfig-managed pma user for anything ourselves.
func pmaPreseed() {
	pass := randomToken()[:24]
	script := `phpmyadmin phpmyadmin/dbconfig-install boolean true
phpmyadmin phpmyadmin/app-password-confirm password ` + pass + `
phpmyadmin phpmyadmin/mysql/admin-pass password ` + pass + `
phpmyadmin phpmyadmin/mysql/app-pass password ` + pass + `
phpmyadmin phpmyadmin/password-confirm password ` + pass + `
phpmyadmin phpmyadmin/setup-password password ` + pass + `
phpmyadmin phpmyadmin/reconfigure-webserver multiselect
phpmyadmin phpmyadmin/mysql/admin-user string root
`
	cmd := exec.Command("debconf-set-selections")
	cmd.Stdin = strings.NewReader(script)
	_ = cmd.Run()
}

func tail(s string, n int) string {
	s = strings.TrimSpace(s)
	if len(s) <= n {
		return s
	}
	return "…" + s[len(s)-n:]
}
