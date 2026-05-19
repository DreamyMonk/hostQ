# hostQ

hostQ is a lightweight self-hosted hosting control panel for small VPS servers. It is built with Next.js 16 and manages common hosting tasks such as sites, Nginx vhosts, PHP-FPM versions, MariaDB databases, WordPress, files, SSL, FTP, backups, and updates.

## Supported OS

Production targets:

- Ubuntu 22.04 LTS
- Ubuntu 24.04 LTS
- Debian 12

Minimum VPS:

- 1 vCPU
- 1 GB RAM
- 10 GB disk

Recommended VPS:

- 2 vCPU
- 2 GB RAM
- 20 GB+ disk

Local development works on Windows, macOS, and Linux. Real hosting actions require a Linux VPS with root access.

## Features

| Area | Features |
|---|---|
| Auth | SSH-generated admin credentials, hashed passwords, JWT sessions, optional TOTP 2FA, login rate limits |
| Security | CSRF protection, secure headers, HTTPS-only production mode, audit hash chain, session revocation |
| Sites | Add sites, manage vhosts, enable/disable, soft-delete, backups, restore dry-run, per-site users |
| Site Safety | File permission scan, sanitize action, secret-file quarantine, WordPress database detection |
| Files | File manager locked to `/var/www` on Linux, safe filename normalization, blocked secret files |
| WordPress | WP-CLI install, WordPress discovery, database setup, soft-delete |
| Databases | MariaDB/MySQL databases, users, grants, drops, phpMyAdmin link |
| PHP | PHP-FPM 8.2, 8.3, 8.4, and 8.5 management |
| Cache | PHP OPcache by default, optional Redis service, optional per-site Nginx FastCGI cache |
| SSL | Let's Encrypt via Certbot, renew/delete, manual PEM upload |
| Services | Nginx, Apache, MariaDB, PHP-FPM, Certbot, WP-CLI, Pure-FTPd, phpMyAdmin |
| Updates | Web updater and SSH CLI updater using GitHub releases |

## VPS Deployment

Run as root on a fresh Ubuntu/Debian VPS:

```bash
git clone https://github.com/DreamyMonk/hostQ.git /opt/hostq
cd /opt/hostq
bash setup.sh
```

The setup script installs:

- Node.js 20
- Nginx
- MariaDB
- PHP-FPM 8.2, 8.3, 8.4, 8.5 where available
- Certbot
- WP-CLI
- Pure-FTPd
- phpMyAdmin
- PM2
- hostQ privileged helper
- `hostq-update` SSH updater

At the end of setup, SSH prints the first admin login:

```text
Initial hostQ admin login:
  Username: admin
  Password: <generated-password>
```

Save the password immediately. It is shown only once.

After login:

1. Open **Admin -> Security**.
2. Change the generated password.
3. Start and verify 2FA if you want TOTP enabled.
4. Fix all red/yellow production readiness checks before exposing the panel publicly.

By default Nginx exposes the panel on:

```text
http://SERVER_IP
http://SERVER_IP:8090
```

Use a domain and HTTPS for production. Port `8090` is intended as a direct setup/admin access port.
The installer enables `HOSTQ_ALLOW_INSECURE_HTTP=true` so first login works on the direct IP setup port. After your domain and SSL are working, set it to `false` in `/opt/hostq/.env.local` and restart the panel.

## Optional Infrastructure Deployment

The primary install path is the cPanel/Plesk-style VPS installer above: clone the repo and run `setup.sh`.

Terraform and Ansible are optional automation helpers for providers or power users who want repeatable VPS provisioning:

```bash
cd infra/terraform/vultr
cp terraform.tfvars.example terraform.tfvars
terraform init
terraform apply

cd ../../ansible
cp inventory.ini.example inventory.ini
ansible-playbook -i inventory.ini playbook.yml
```

Terraform provisions the Vultr VPS, firewall rules, SSH key, and optional DNS records. Ansible hardens the server, creates swap for 1GB VPS plans, installs hostQ, and can update to a tagged release. You do not need Terraform or Ansible to use hostQ.

See [infra/README.md](./infra/README.md).

## Lightweight Go Runtime Preview

The current production panel remains the Next.js app while the lightweight Go runtime is being ported module by module. The Go panel is designed for lower RAM usage on 1GB VPS plans and uses server-rendered HTML instead of React.

Install the Go preview beside the existing panel:

```bash
cd /opt/hostq
sudo bash scripts/install-go-panel.sh
```

It runs as:

```text
systemd service: hostq-panel
backend: 127.0.0.1:8091
preview URL: http://SERVER_IP:8092
```

The Go preview currently includes:

- Login using the existing `/etc/hostq/admin.json`
- Dashboard
- Sites list, PHP site creation under `/var/www/domain/htdocs`, enable/disable, cache toggle, permissions repair, backup, and soft-delete
- File manager locked to `/var/www` with create folder/file, chmod, and soft-delete actions
- MariaDB/MySQL database inventory, create, generated user/password, and delete
- Let's Encrypt SSL inventory, install, renew, delete, and stale Nginx SSL repair
- Service start/stop/restart for a narrow allowlist
- Go audit log at `/etc/hostq/audit-go.log`

The migration target is:

```text
Go panel + systemd + SQLite/JSON metadata + hostq-helper
```

The Next.js panel will stay available until the Go panel reaches feature parity.

## Updating

From the panel:

- Go to **Admin -> Security -> Updates**
- Click **Check**
- Click **Update** when a release is available

From SSH:

```bash
sudo hostq-update
```

Update to a specific release:

```bash
sudo hostq-update v0.2.3
```

Updates create a backup first:

```text
/var/backups/hostq/panel-*.tar.gz
```

Release tags are expected to look like:

```text
v0.2.3
```

## Local Development

```bash
npm install
npm run dev -- --port 8090
```

Open:

[http://localhost:8090](http://localhost:8090)

If no admin account exists locally, the browser fallback setup can create one. Production installs should use `setup.sh`, which generates the admin account over SSH.

## Configuration

Copy `.env.example` to `.env.local` and edit:

```env
HOSTQ_DATA_DIR=/etc/hostq
JWT_SECRET=random_256bit_string
JWT_EXPIRY=24h
SESSION_IDLE_TIMEOUT_MINUTES=30

HOSTQ_REQUIRE_HELPER=false
HOSTQ_HELPER=/usr/local/sbin/hostq-helper

DB_HOST=localhost
DB_ROOT_USER=root
DB_ROOT_PASSWORD=mysql_root_pass

PHPMYADMIN_URL=http://localhost/phpmyadmin
FILE_MANAGER_ROOT=/var/www
WEB_ROOT=/var/www
PANEL_DOMAIN=panel.yourdomain.com
PANEL_URL=https://panel.yourdomain.com
PANEL_PUBLIC_PORT=8090
HOSTQ_ALLOW_INSECURE_HTTP=true
NODE_OPTIONS=--max-old-space-size=384
```

When helper coverage is complete on your server, enable:

```env
HOSTQ_REQUIRE_HELPER=true
```

## UI Modes

hostQ has two main modes:

- **User mode:** sites, files, WordPress, SSL, and database shortcuts.
- **Admin mode:** server overview, domains, PHP, services, security, updates, sessions, and audit logs.

Each site has a **Manage Site** panel with:

- Open site
- Files
- SSL
- WordPress
- Database shortcut
- PHP shortcut
- Enable/disable
- File permissions and database scan
- Optional Nginx FastCGI cache toggle
- Sanitize files and repair permissions
- Backup
- Soft-delete options
- Per-site users and roles

## Security Model

Implemented controls:

- CSRF tokens for mutating API requests
- HTTPS-only enforcement in production
- Admin panel host setting for `PANEL_DOMAIN`, `PANEL_URL`, and temporary direct-IP HTTP setup mode
- Secure, strict cookies
- Login rate limiting persisted on disk
- Optional TOTP 2FA
- Session list and session revoke API
- Per-site RBAC enforcement
- Audit log hash chain and rotation
- File manager restricted to `/var/www` on Linux
- Secret-like files blocked or quarantined
- Redis is optional and not required for 1GB VPS installs
- Restore dry-run and confirmation flow
- Privileged helper allowlist for sensitive tasks

Read more in:

[SECURITY_PRODUCTION.md](./SECURITY_PRODUCTION.md)

## Verification

Before release or deployment:

```bash
npm run lint
npm run security:test
npm run build
```

The production build may show a non-blocking Turbopack trace warning for runtime filesystem APIs such as site safety checks.

## GitHub Releases

Create a release from a pushed tag:

```bash
git tag -a v0.2.3 -m "hostQ v0.2.3"
git push origin v0.2.3
```

Then publish the release on GitHub:

[https://github.com/DreamyMonk/hostQ/releases/new](https://github.com/DreamyMonk/hostQ/releases/new)

The web and SSH updaters read GitHub releases from:

```text
DreamyMonk/hostQ
```
