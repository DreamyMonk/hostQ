# hostQ

hostQ is a lightweight hosting control panel for small VPS servers. It runs as one `hostq-panel` systemd service behind Nginx — no Node, no PHP runtime for the UI, no SPA build step. The entire control plane is one Go binary.

**Current release:** `v0.4.0` — redesigned UI, file manager with right-click context menu, account & audit pages.

## Supported OS

- Ubuntu 22.04 LTS
- Ubuntu 24.04 LTS
- Debian 12

Minimum VPS:

- 1 vCPU
- 1 GB RAM
- 10 GB disk

## Features

| Area | Features |
|---|---|
| Runtime | Single native HTTP server, systemd service, Nginx reverse proxy. CSP-locked UI with inline Lucide-style SVG icons — no external CDN |
| UI | Dark sidebar + light workspace, sticky topbar, card-based dashboard with load / memory / web-disk gauges, hostname & uptime |
| Auth | SSH-generated admin credentials, bcrypt password hash, signed HTTP-only sessions, in-panel password change |
| Sites | Add a site, then open one site manager for files, database, SSL, WordPress, PHP, FTP, cache, backups, and delete |
| Files | Per-site file manager under `/var/www/<domain>/htdocs` with **breadcrumbs**, **right-click context menu** (open, download, rename, chmod, copy, move, delete), **multi-file upload**, **single-file download**, recursive directory copy, size/mode/mtime columns, secret-name blocking |
| WordPress | WP-CLI install and discovery inside each site manager |
| Databases | Per-site MariaDB/MySQL database and generated user/password |
| PHP | PHP-FPM 8.2, 8.3, 8.4, 8.5 service status and per-site switch |
| SSL | Let's Encrypt install/renew/delete and stale Nginx SSL repair |
| Backups | Per-site manual zip backup, download, restore full/files/database, automatic daily/weekly/monthly retention |
| Cron | Cron Manager for hostQ-managed scheduled commands and the built-in backup runner |
| Services | Narrow allowlist for Nginx, MariaDB, Redis, PHP-FPM, Pure-FTPd |
| Admin | `/account` password change · `/audit` last-200 entries audit log |
| Updates | SSH updater that downloads a GitHub tag, backs up, rebuilds the panel, restarts systemd |

## VPS Deployment

Run as root on a fresh Ubuntu/Debian VPS. Two equivalent ways:

**A. Latest `main`** (preferred for now until v0.4.0 lands on the release channel):

```bash
sudo git clone https://github.com/DreamyMonk/hostQ.git /opt/hostq
cd /opt/hostq
sudo bash install.sh
```

**B. Pinned release tag** (reproducible):

```bash
sudo git clone --branch v0.4.0 --depth 1 https://github.com/DreamyMonk/hostQ.git /opt/hostq
cd /opt/hostq
sudo bash install.sh
```

To run the installer non-interactively (CI / cloud-init):

```bash
sudo HOSTQ_ASSUME_YES=true bash install.sh
```

The installer installs:

- Native build toolchain
- Nginx
- MariaDB
- PHP-FPM 8.2, 8.3, 8.4, 8.5 where available
- Certbot
- WP-CLI
- Pure-FTPd
- phpMyAdmin
- `hostq-panel.service`
- `hostq-update`
- `/etc/cron.d/hostq-backups` for automatic per-site backups

At the end of install, SSH prints the first admin login:

```text
Initial hostQ admin login:
  Username: admin
  Password: <generated-password>
```

Save the password immediately. It is shown only once.

By default Nginx exposes the panel on:

```text
http://SERVER_IP
http://SERVER_IP:8090
```

Use a real domain and HTTPS for production. Port `8090` is intended as direct setup access and should be restricted or closed after the panel domain works.

### First-login checklist

1. Sign in with the credentials printed by the installer.
2. Open **Account** → set your own password.
3. Open **Services** → confirm Nginx, MariaDB and the PHP-FPM you want are `active`.
4. Open **Sites** → add your first domain. Then open its Site Manager and install SSL.
5. Open **Audit Log** to verify the actions above were recorded.

## Updating

The packaged updater pulls a release tag, rebuilds the Go binary, and restarts the service.

```bash
# Update to the latest tag on GitHub
sudo hostq-update

# Or pin to a specific tag
sudo hostq-update v0.4.0
```

The updater always snapshots the current install before rebuilding:

```text
/var/backups/hostq/panel-*.tar.gz
```

If something goes wrong, restore with:

```bash
sudo systemctl stop hostq-panel
sudo tar -xzf /var/backups/hostq/panel-<timestamp>.tar.gz -C /
sudo systemctl start hostq-panel
```

## Manual Reinstall

```bash
cd /opt/hostq
sudo bash install.sh
sudo systemctl restart nginx hostq-panel
```

## Useful VPS Commands

```bash
sudo systemctl status hostq-panel --no-pager -l
sudo journalctl -u hostq-panel -f
sudo nginx -t
sudo ss -ltnp | grep -E ':80|:443|:8090'
curl -I http://127.0.0.1:8090
```

## Cloudflare

After origin SSL is installed, set Cloudflare SSL/TLS mode to **Full** or **Full strict**. If Cloudflare shows 502, check that Nginx proxies the panel vhost to:

```nginx
proxy_pass http://127.0.0.1:8090;
```

## Local Development

```bash
# Run the panel against /var/www and /etc/hostq locally
go run .

# Or point it at a sandbox without touching system paths
HOSTQ_ADDR=127.0.0.1:8090 \
HOSTQ_DATA_DIR=./.devdata \
WEB_ROOT=./.devweb \
HOSTQ_NGINX_AVAILABLE=./.devnginx \
go run . init-admin

HOSTQ_ADDR=127.0.0.1:8090 \
HOSTQ_DATA_DIR=./.devdata \
WEB_ROOT=./.devweb \
HOSTQ_NGINX_AVAILABLE=./.devnginx \
go run .
```

The panel is organized as a small modular service:

| File | Responsibility |
|---|---|
| `main.go` | Routes, template func map, server bootstrap |
| `auth.go` | Admin account, bcrypt, signed session cookies |
| `account.go` | `/account` (change password) and `/audit` (audit log viewer) |
| `sites.go` | Sites list/create/delete, Nginx vhost writer, dashboard |
| `files.go` | File manager: browse, mkdir, touch, rename, chmod, copy, move, delete, upload, download |
| `databases.go` | MariaDB create/drop, generated users |
| `wordpress.go` | WP-CLI install + install discovery |
| `php.go` | PHP-FPM status + per-site version switch |
| `ssl.go` | Certbot issue/renew/delete + stale Nginx SSL repair |
| `services.go` | Allow-listed `systemctl` operations |
| `cron.go` | `/etc/cron.d/hostq-user-jobs` reader/writer |
| `backups.go` | Per-site zip backups + scheduled retention |
| `stats.go` / `stats_other.go` | Linux system stats (load / mem / disk) with non-Linux stub |
| `ui_templates.go` | All HTML / CSS / inline JS / Lucide-style SVG icon set |

### Releasing

```bash
# Make sure tests/build pass
go test ./...
go build .

# Tag and push (annotated)
git tag -a v0.X.Y -m "v0.X.Y: short summary"
git push origin v0.X.Y

# Then run on a VPS
sudo hostq-update v0.X.Y
```

Validation:

```bash
go test ./...
go build .
GOOS=linux go build .   # confirm Linux cross-build
```
