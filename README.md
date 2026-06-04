# hostQ

A lightweight hosting control panel for small VPS servers. One Go binary,
one systemd service, behind Nginx. No Node, no PHP runtime for the UI,
no SPA build step.

**Latest:** `v0.11.1` — see [Releases](https://github.com/DreamyMonk/hostQ/releases) for the full changelog.

## Requirements

- Ubuntu 22.04 / 24.04 LTS or Debian 12
- 1 vCPU, 1 GB RAM, 10 GB disk (minimum)
- Run as `root`

## Install

On a fresh VPS:

```bash
sudo git clone https://github.com/DreamyMonk/hostQ.git /opt/hostq && cd /opt/hostq && sudo bash install.sh
```

Non-interactive (CI / cloud-init / scripted setup):

```bash
sudo git clone https://github.com/DreamyMonk/hostQ.git /opt/hostq && cd /opt/hostq && sudo HOSTQ_ASSUME_YES=true bash install.sh
```

Pinned to a release tag:

```bash
sudo git clone --branch v0.11.1 --depth 1 https://github.com/DreamyMonk/hostQ.git /opt/hostq && cd /opt/hostq && sudo bash install.sh
```

The installer brings in Nginx, MariaDB, PHP-FPM 8.2–8.5, Certbot, WP-CLI,
Pure-FTPd, phpMyAdmin (debconf-preseeded), the `hostq-panel` systemd
service, the `hostq-update` CLI, and a cron job for automatic backups.

At the end of install you get a one-time admin password:

```text
Initial hostQ admin login:
  Username: admin
  Password: <generated-password>
```

The panel is then reachable at:

```text
http://<server-ip>          (via Nginx)
http://<server-ip>:8090     (direct setup port)
```

Use a real domain + HTTPS for anything beyond setup. Close or firewall
`:8090` once you have a domain working — it is intended for first-boot
access only.

## First login

1. Sign in with the printed credentials.
2. **Account** → set your own password.
3. **Sites** → add your first domain. Open the Site Manager, install SSL.
4. **Services & Packages** → confirm what you need is `active`, install
   anything you skipped during setup (Redis, an extra PHP version, etc.).

## What's included

| Area | Features |
|---|---|
| **Sites** | One per-site workspace tabbed across Overview, Database, WordPress, SSL, PHP, Files, Security, Backups |
| **Files** | Right-click context menu, breadcrumbs, checkbox + bulk actions (delete / chmod / move), file & folder upload with **live progress bar**, single-file download, recursive copy, secret-name blocking |
| **Databases** | Per-site MariaDB DB + multi-user management with reset/drop, phpMyAdmin one-click sign-on |
| **WordPress** | WP-CLI install, core update, cache flush, user list, password reset, site-URL search-replace, delete, and **Malfix** integrity check + repair |
| **Security** | Per-site malware scanner with curated rule set (webshells, eval-decode, RCE patterns, hidden iframes, PHP-in-uploads). Findings are quarantined (moved to `/var/backups/hostq/quarantine/`) or deleted |
| **SSL** | Let's Encrypt issue / renew / repair. SSL is **preserved** across PHP / cache / WordPress operations |
| **Services & Packages** | Unified page: install / uninstall apt packages and control their systemd units. Start-on-missing auto-installs |
| **Redis** | Optional cache manager: memory, hits, keys, flush, restart |
| **Backups** | Per-site manual zip + automatic daily/weekly/monthly with retention + restore (full / files-only / db-only) |
| **Cron** | Managed cron entries in `/etc/cron.d/hostq-user-jobs` |
| **UI** | Dark + light theme toggle, ⌘K command palette, toast notifications, modal-based forms, inline Lucide-style SVG icons (CSP-safe, no CDN) |
| **Admin** | Bcrypt password change, audit log of the last 200 actions |

## Updating

```bash
sudo hostq-update            # latest tag
sudo hostq-update v0.11.1    # pin a specific tag
```

The updater snapshots the previous install to
`/var/backups/hostq/panel-<timestamp>.tar.gz` before rebuilding.

Roll back if needed:

```bash
sudo systemctl stop hostq-panel
sudo tar -xzf /var/backups/hostq/panel-<timestamp>.tar.gz -C /
sudo systemctl start hostq-panel
```

## Reinstall scenarios

`git clone` refuses to write into a non-empty `/opt/hostq`. Pick the one
that matches your situation:

```bash
# Updater — preserves /etc/hostq + /var/www (recommended)
sudo hostq-update

# Update in place from the repo
cd /opt/hostq && sudo git pull && sudo bash install.sh

# Wipe and reinstall /opt/hostq only — keeps sites + DBs + credentials
sudo systemctl stop hostq-panel && sudo rm -rf /opt/hostq \
  && sudo git clone https://github.com/DreamyMonk/hostQ.git /opt/hostq \
  && cd /opt/hostq && sudo bash install.sh
```

`/etc/hostq` (admin account, audit log, backup policies, DB credentials,
scan reports) and `/var/www` (site files) live **outside** `/opt/hostq`,
so blowing away `/opt/hostq` never touches your sites or login.

## Troubleshooting

```bash
sudo systemctl status hostq-panel --no-pager -l
sudo journalctl -u hostq-panel -f
sudo nginx -t
sudo ss -ltnp | grep -E ':80|:443|:8090'
curl -I http://127.0.0.1:8090
```

## Cloudflare

After origin SSL is installed, set Cloudflare **SSL/TLS → Overview** to
**Full (strict)**. If you get a 521, the origin probably isn't listening
on `:443` yet — install the cert from the panel's SSL tab first
(temporarily set the DNS record to grey-cloud / DNS only while Certbot
runs, then flip back to orange).

If Cloudflare returns 502 once SSL is up, your reverse proxy needs to
point the panel vhost at the loopback panel:

```nginx
proxy_pass http://127.0.0.1:8090;
```

## Local development

```bash
# Run against the system paths (needs root + Nginx etc.)
go run .

# Or a sandbox without touching /var/www or /etc/hostq
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

### Code map

The panel is a flat Go module. Files are grouped by feature, no
sub-packages:

| Group | Files |
|---|---|
| Boot + auth | `main.go`, `auth.go`, `account.go`, `audit.go` |
| Sites + dashboard | `sites.go`, `stats.go`, `stats_other.go` |
| File manager | `files.go` |
| Databases + creds | `databases.go`, `creds.go` |
| WordPress + Malfix | `wordpress.go`, `malfix.go` |
| Security scanner | `scanner.go`, `security.go` |
| SSL + PHP + cron + backups | `ssl.go`, `php.go`, `cron.go`, `backups.go` |
| Services + packages | `services.go`, `packages.go`, `redis.go` |
| phpMyAdmin SSO | `pma.go` |
| HTTP plumbing | `gzip.go`, `cache.go` |
| Types + templates | `types.go`, `ui_templates.go` |

### Validation

```bash
go test ./...
go build .
GOOS=linux go build .
```

### Releasing

```bash
# Tag and push
git tag -a v0.X.Y -m "v0.X.Y: short summary"
git push origin v0.X.Y

# Roll out to a server
sudo hostq-update v0.X.Y
```

## License

See [LICENSE](LICENSE) if present in the repo.
