package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// hostQ runs Apache as an optional per-site backend behind Nginx — never as
// the public-facing server. Nginx keeps port 80/443 (TLS, FastCGI cache, the
// panel proxy); Apache listens only on 127.0.0.1:8080 and handles PHP + the
// site's .htaccess for the domains the operator opts in. This gives sites
// that rely on Apache rewrite rules a real Apache without giving up any of the
// Nginx front-door machinery.
//
// Per-site choice is persisted at /etc/hostq/sites/<domain>.backend and read
// back inside writeNginxSite, mirroring how aliases / php.ini / extra-nginx
// already flow — so no writeNginxSite caller signature has to change.

const apacheBackendAddr = "127.0.0.1:8080"

// siteBackendPath is where a site's chosen web backend ("nginx" or "apache")
// is stored. Absent file == the Nginx default.
func (a *App) siteBackendPath(domain string) string {
	if !domainRe.MatchString(domain) {
		return ""
	}
	return filepath.Join(a.cfg.DataDir, "sites", domain+".backend")
}

// siteBackend returns "apache" when the operator has switched this site to the
// Apache backend, otherwise "nginx".
func (a *App) siteBackend(domain string) string {
	p := a.siteBackendPath(domain)
	if p == "" {
		return "nginx"
	}
	data, err := os.ReadFile(p)
	if err != nil {
		return "nginx"
	}
	if strings.TrimSpace(string(data)) == "apache" {
		return "apache"
	}
	return "nginx"
}

// setSiteBackend persists the site's backend choice. "nginx" removes the file
// so the default is represented by absence.
func (a *App) setSiteBackend(domain, backend string) error {
	p := a.siteBackendPath(domain)
	if p == "" {
		return fmt.Errorf("invalid domain")
	}
	if backend != "apache" {
		_ = os.Remove(p)
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(p), 0700); err != nil {
		return err
	}
	return os.WriteFile(p, []byte("apache"), 0644)
}

// apacheInstalled reports whether the apache2 package is on disk.
func apacheInstalled() bool { return isAptInstalled("apache2") }

// ensureApacheHybrid configures a freshly-installed Apache to coexist with
// Nginx: it binds Apache to 127.0.0.1:8080 only, enables the modules the
// FastCGI-proxy vhosts need, drops Apache's own default sites (which would
// otherwise grab :80 and clash with Nginx), and restarts the service. Safe to
// call repeatedly — every step is idempotent.
func (a *App) ensureApacheHybrid() error {
	if !apacheInstalled() {
		return fmt.Errorf("apache2 is not installed")
	}
	// Bind Apache to the loopback backend port only. Overwriting ports.conf is
	// safe because hostQ owns Apache's listener config in the hybrid model.
	ports := "# hostQ managed — Apache runs as a loopback backend behind Nginx.\nListen " + apacheBackendAddr + "\n"
	if err := os.WriteFile("/etc/apache2/ports.conf", []byte(ports), 0644); err != nil {
		return err
	}
	// Modules required for the proxy-to-php-fpm vhosts + .htaccess rewrites.
	_ = exec.Command("a2enmod", "proxy", "proxy_fcgi", "setenvif", "rewrite", "headers").Run()
	// Apache's stock sites listen on *:80 and would fight Nginx.
	_ = exec.Command("a2dissite", "000-default", "default-ssl").Run()
	if err := exec.Command("apache2ctl", "configtest").Run(); err != nil {
		// configtest is best-effort noise on some distros; don't hard-fail here.
		_ = err
	}
	if out, err := exec.Command("systemctl", "restart", "apache2").CombinedOutput(); err != nil {
		return fmt.Errorf("apache restart failed: %s", tail(string(out), 200))
	}
	_ = exec.Command("systemctl", "enable", "apache2").Run()
	return nil
}

// writeApacheSite renders the loopback VirtualHost for a domain and enables it.
// PHP is handed to the same php-fpm socket Nginx would have used, so switching
// backends never changes the PHP runtime — only who interprets .htaccess.
func (a *App) writeApacheSite(domain, root, phpVersion string) error {
	if !domainRe.MatchString(domain) {
		return fmt.Errorf("invalid domain")
	}
	if !phpVersionRe.MatchString(phpVersion) {
		phpVersion = "8.4"
	}
	serverAlias := "www." + domain
	if aliases := a.loadAliases(domain); len(aliases) > 0 {
		serverAlias += " " + strings.Join(aliases, " ")
	}
	conf := fmt.Sprintf(`# hostQ managed - %s (Apache backend behind Nginx)
<VirtualHost %s>
    ServerName %s
    ServerAlias %s
    DocumentRoot %s
    DirectoryIndex index.php index.html

    <Directory %s>
        Options -Indexes +FollowSymLinks
        AllowOverride All
        Require all granted
    </Directory>

    <FilesMatch \.php$>
        SetHandler "proxy:unix:/run/php/php%s-fpm.sock|fcgi://localhost"
    </FilesMatch>

    ErrorLog ${APACHE_LOG_DIR}/%s.error.log
    CustomLog ${APACHE_LOG_DIR}/%s.access.log combined
</VirtualHost>
`, domain, apacheBackendAddr, domain, serverAlias, root, root, phpVersion, domain, domain)

	dir := "/etc/apache2/sites-available"
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(dir, domain+".conf"), []byte(conf), 0644); err != nil {
		return err
	}
	_ = exec.Command("a2ensite", domain+".conf").Run()
	if out, err := exec.Command("apache2ctl", "configtest").CombinedOutput(); err != nil {
		return fmt.Errorf("apache config invalid: %s", tail(string(out), 200))
	}
	_ = exec.Command("systemctl", "reload", "apache2").Run()
	return nil
}

// removeApacheSite disables and deletes a domain's Apache vhost. Called when a
// site is switched back to the Nginx backend or deleted.
func (a *App) removeApacheSite(domain string) {
	if !domainRe.MatchString(domain) {
		return
	}
	_ = exec.Command("a2dissite", domain+".conf").Run()
	_ = os.Remove(filepath.Join("/etc/apache2/sites-available", domain+".conf"))
	if apacheInstalled() {
		_ = exec.Command("systemctl", "reload", "apache2").Run()
	}
}
