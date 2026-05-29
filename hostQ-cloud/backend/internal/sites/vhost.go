// Package sites emits the dual nginx+apache vhosts for hostQ-cloud. The
// model: nginx terminates TLS on :443, serves static, proxies everything
// else to Apache on 127.0.0.1:8080 which runs PHP via mod_php so .htaccess
// works for the long tail of legacy apps.
package sites

import (
	"fmt"
	"os"
	"path/filepath"
)

type VhostWriter struct {
	NginxDir  string
	ApacheDir string
	WebRoot   string
}

func New(nginxDir, apacheDir, webRoot string) *VhostWriter {
	return &VhostWriter{NginxDir: nginxDir, ApacheDir: apacheDir, WebRoot: webRoot}
}

func (v *VhostWriter) DocrootFor(domain string) string {
	return filepath.Join(v.WebRoot, domain, "public")
}

// Write emits nginx + apache vhost files. Returns their paths so they can be
// cached on the sites row for cleanup later.
func (v *VhostWriter) Write(domain, docroot, phpVersion string) (string, string, error) {
	if err := os.MkdirAll(docroot, 0755); err != nil {
		return "", "", fmt.Errorf("mkdir docroot: %w", err)
	}
	// drop an index.html landing page so the user can verify the vhost is live
	indexHTML := filepath.Join(docroot, "index.html")
	if _, err := os.Stat(indexHTML); os.IsNotExist(err) {
		_ = os.WriteFile(indexHTML, []byte(landingPage(domain)), 0644)
	}

	nginxPath := filepath.Join(v.NginxDir, domain+".conf")
	apachePath := filepath.Join(v.ApacheDir, domain+".conf")

	if err := os.WriteFile(nginxPath, []byte(nginxTemplate(domain)), 0644); err != nil {
		return "", "", fmt.Errorf("write nginx: %w", err)
	}
	if err := os.WriteFile(apachePath, []byte(apacheTemplate(domain, docroot, phpVersion)), 0644); err != nil {
		return "", "", fmt.Errorf("write apache: %w", err)
	}
	// Best-effort symlink into sites-enabled.
	_ = os.Symlink(nginxPath, filepath.Join(filepath.Dir(v.NginxDir), "sites-enabled", domain+".conf"))
	_ = os.Symlink(apachePath, filepath.Join(filepath.Dir(v.ApacheDir), "sites-enabled", domain+".conf"))
	return nginxPath, apachePath, nil
}

func (v *VhostWriter) Remove(domain string) error {
	_ = os.Remove(filepath.Join(filepath.Dir(v.NginxDir), "sites-enabled", domain+".conf"))
	_ = os.Remove(filepath.Join(filepath.Dir(v.ApacheDir), "sites-enabled", domain+".conf"))
	_ = os.Remove(filepath.Join(v.NginxDir, domain+".conf"))
	_ = os.Remove(filepath.Join(v.ApacheDir, domain+".conf"))
	return nil
}

func nginxTemplate(domain string) string {
	return fmt.Sprintf(`# hostQ-cloud — managed nginx front for %s
# Static is served by nginx. Dynamic .php (and anything without a static
# extension) is reverse-proxied to Apache on 127.0.0.1:8080 which runs
# mod_php so .htaccess / classic LAMP apps work.
server {
    listen 80;
    listen [::]:80;
    server_name %s www.%s;
    access_log /var/log/nginx/%s.access.log combined;
    error_log  /var/log/nginx/%s.error.log warn;

    client_max_body_size 512m;

    # Static fast-path: serve directly from disk, skip Apache entirely.
    location ~* \.(css|js|map|png|jpg|jpeg|gif|webp|svg|ico|woff2?|ttf|eot|otf|mp4|webm)$ {
        root /var/www/%s/public;
        try_files $uri @apache;
        expires 30d;
        access_log off;
        add_header Cache-Control "public, immutable";
    }

    # Everything else → Apache.
    location / {
        proxy_pass http://127.0.0.1:8080;
        proxy_http_version 1.1;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_read_timeout 600s;
        proxy_send_timeout 600s;
    }

    location @apache {
        proxy_pass http://127.0.0.1:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
`, domain, domain, domain, domain, domain, domain)
}

func apacheTemplate(domain, docroot, phpVersion string) string {
	return fmt.Sprintf(`# hostQ-cloud — managed Apache back for %s
# Listens only on 127.0.0.1:8080 — nginx is the public face.
<VirtualHost 127.0.0.1:8080>
    ServerName %s
    ServerAlias www.%s
    DocumentRoot %s

    <Directory %s>
        Options -Indexes +FollowSymLinks
        AllowOverride All
        Require all granted
    </Directory>

    # PHP %s via mod_php (libapache2-mod-php%s).
    <FilesMatch \.php$>
        SetHandler application/x-httpd-php
    </FilesMatch>

    # Respect the X-Forwarded-Proto header from nginx so $_SERVER['HTTPS']
    # is correct under PHP and apps that check it (WordPress, Laravel)
    # generate https:// URLs instead of http://.
    SetEnvIf X-Forwarded-Proto https HTTPS=on

    ErrorLog ${APACHE_LOG_DIR}/%s.error.log
    CustomLog ${APACHE_LOG_DIR}/%s.access.log combined
</VirtualHost>
`, domain, domain, domain, docroot, docroot, phpVersion, phpVersion, domain, domain)
}

func landingPage(domain string) string {
	return fmt.Sprintf(`<!doctype html>
<html><head><meta charset="utf-8"><title>%s</title>
<style>body{font-family:system-ui,sans-serif;display:grid;place-items:center;min-height:100vh;margin:0;background:#0b1220;color:#e6eaf2}
.card{background:#111a2e;border:1px solid #1f2a44;padding:40px 50px;border-radius:14px;text-align:center;max-width:480px}
h1{margin:0 0 6px;font-size:20px}
p{margin:0;color:#94a3b8;font-size:14px}</style></head>
<body><div class="card"><h1>%s is live</h1><p>Site provisioned by hostQ-cloud. Upload your content to replace this page.</p></div></body></html>
`, domain, domain)
}
