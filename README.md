# hostQ

hostQ is a lightweight Go hosting control panel for small VPS servers. It runs as one `hostq-panel` systemd service behind Nginx.

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
| Runtime | Single Go HTTP server, systemd service, Nginx reverse proxy |
| Auth | SSH-generated admin credentials, bcrypt password hash, signed HTTP-only sessions |
| Sites | Add PHP sites, Nginx vhosts, enable/disable, soft-delete, backups, FastCGI cache toggle |
| WordPress | WP-CLI install and discovery |
| Files | File manager locked to `/var/www`, create folder/file, chmod, soft-delete, secret blocking |
| Databases | MariaDB/MySQL inventory, create generated user/password, delete |
| PHP | PHP-FPM 8.2, 8.3, 8.4, 8.5 service status and per-site switch |
| SSL | Let's Encrypt install/renew/delete and stale Nginx SSL repair |
| Services | Narrow allowlist for Nginx, MariaDB, Redis, PHP-FPM, Pure-FTPd |
| Updates | SSH updater that downloads a GitHub tag, backs up, rebuilds Go, restarts systemd |

## VPS Deployment

Run as root on a fresh Ubuntu/Debian VPS:

```bash
git clone https://github.com/DreamyMonk/hostQ.git /opt/hostq
cd /opt/hostq
bash setup.sh
```

The setup script installs:

- Go
- Nginx
- MariaDB
- PHP-FPM 8.2, 8.3, 8.4, 8.5 where available
- Certbot
- WP-CLI
- Pure-FTPd
- phpMyAdmin
- `hostq-panel.service`
- `hostq-update`

At the end of setup, SSH prints the first admin login:

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

## Updating

```bash
sudo hostq-update
sudo hostq-update v0.3.0
```

Updates create a backup first:

```text
/var/backups/hostq/panel-*.tar.gz
```

## Manual Go Reinstall

```bash
cd /opt/hostq
sudo bash scripts/install-go-panel.sh
sudo systemctl restart nginx hostq-panel
```

## Useful VPS Commands

```bash
sudo systemctl status hostq-panel --no-pager -l
sudo journalctl -u hostq-panel -f
sudo nginx -t
sudo ss -ltnp | grep -E ':80|:443|:8090|:8091'
curl -I http://127.0.0.1:8091
```

## Cloudflare

After origin SSL is installed, set Cloudflare SSL/TLS mode to **Full** or **Full strict**. If Cloudflare shows 502, check that Nginx proxies the panel vhost to:

```nginx
proxy_pass http://127.0.0.1:8091;
```

## Local Development

```bash
go run ./cmd/hostq-panel
```

Validation:

```bash
go test ./cmd/hostq-panel
go build ./cmd/hostq-panel
```
