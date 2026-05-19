# hostQ Go Production Hardening

hostQ runs as one Go service behind Nginx.

## Supported Base OS

- Ubuntu 22.04 LTS or 24.04 LTS
- Debian 12
- 1 GB RAM / 1 vCPU minimum for small sites

## Network

- Allow only `22/tcp`, `80/tcp`, `443/tcp`, and FTP ports you intentionally use.
- `setup.sh` also opens `PANEL_PUBLIC_PORT`; the default is `8090`.
- Close or restrict `8090` after HTTPS/domain access is working.
- Nginx should proxy panel traffic to `127.0.0.1:8091`.
- Use Cloudflare SSL mode **Full** or **Full strict** after origin SSL is installed.

## Accounts And Sessions

- `setup.sh` generates the first admin username/password over SSH.
- Credentials are stored in `/etc/hostq/admin.json` with a bcrypt password hash.
- The panel uses signed, HTTP-only, same-site cookies.
- Change the generated password after first login.

## Files And Secrets

- File manager paths are locked under `/var/www`.
- `.env`, private keys, PEM/P12/PFX files, and common SSH key names are blocked by default.
- Deletes are soft deletes into `.hostq-trash`.

## Services

- The panel controls only a narrow service allowlist.
- No generic shell endpoint is exposed by the Go panel.

## Backups And Updates

- Site backups are written under `/var/backups/hostq`.
- `hostq-update` downloads GitHub tag tarballs, creates a backup, rebuilds the Go binary, and restarts `hostq-panel`.

```bash
sudo hostq-update
sudo hostq-update v0.3.0
```

## Validation

Before publishing a release:

```bash
go test ./cmd/hostq-panel
go build ./cmd/hostq-panel
```
