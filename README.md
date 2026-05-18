# HostPanel - Hosting Control Panel

A single-user self-hosted hosting panel for a small VPS. It is built with Next.js 16, TypeScript, Tailwind CSS, and server-side Linux commands for real hosting operations.

## Supported OS and Server Size

Recommended production targets:

- Ubuntu 22.04 LTS
- Ubuntu 24.04 LTS
- Debian 12

Minimum VPS size:

- 1 CPU core
- 1 GB RAM
- 10 GB disk

Recommended VPS size:

- 2 CPU cores
- 2 GB RAM
- 20 GB+ disk

Local development works on Windows, macOS, and Linux, but real hosting actions such as Nginx vhosts, PHP-FPM switching, Certbot, Pure-FTPd, MariaDB, and file ownership changes require a Linux VPS with root access.

## Features

| Feature | Description |
|---|---|
| Auth | Single-admin JWT login with a cookie session |
| Dashboard | CPU, RAM, disk, uptime, service status |
| Sites | Add, disable, delete, and manage domain/subdomain sites |
| Site types | HTML/CSS static sites, PHP sites, and WordPress-ready folders |
| WordPress | One-click install with WP-CLI plus admin, files, and delete actions |
| PHP Manager | Manage currently supported PHP 8.2, 8.3, 8.4, and 8.5 FPM branches |
| Database Manager | MariaDB/MySQL databases, users, grants, drops, and phpMyAdmin links |
| File Manager | Browse, upload, edit, create, rename, and delete files inside a configured root |
| SSL Manager | Let's Encrypt via Certbot, renewal, delete, and manual PEM certificate upload |
| FTP | Pure-FTPd service install/start/stop controls |
| Services | Nginx, MariaDB, PHP-FPM, Certbot, WP-CLI, phpMyAdmin, and Pure-FTPd management |

## Local Development

```bash
npm install
npm run dev
```

Open [http://localhost:3000](http://localhost:3000).

Default local login from `.env.local`:

```text
admin / admin123
```

Change this before any real deployment.

## VPS Deployment

On Ubuntu/Debian as root:

```bash
git clone https://github.com/DreamyMonk/hostQ.git /opt/hosting-panel
cd /opt/hosting-panel
bash setup.sh
```

The installer chooses a lightweight Nginx-first LEMP stack for 1GB RAM / 1 CPU servers. Apache is available from the Services page, but Nginx + PHP-FPM is the default because it uses less memory.

Installed stack:

- Node.js 20
- Nginx 1.24+ where available
- MariaDB 10.11 where available
- PHP-FPM 8.2, 8.3, 8.4, 8.5
- Certbot
- WP-CLI
- Pure-FTPd
- phpMyAdmin 5.2
- PM2 with a 384 MB memory restart limit

## How to Use

1. Deploy with `bash setup.sh`.
2. Open the panel URL shown at the end of setup.
3. Login with the temporary default credentials.
4. Edit `/opt/hosting-panel/.env.local` and change `PANEL_USERNAME`, `PANEL_PASSWORD`, and `JWT_SECRET`.
5. Restart the panel with `pm2 restart hosting-panel`.
6. Use Services to confirm Nginx, MariaDB, PHP-FPM, Certbot, Pure-FTPd, WP-CLI, and phpMyAdmin are installed.
7. Add a site from Domain Manager, choosing HTML/CSS, PHP, or WordPress-ready.
8. Use SSL Manager for Let&apos;s Encrypt or manual PEM certificate upload.
9. Use File Manager or FTP to upload site files.

## Configuration

Copy and edit `.env.example` to `.env.local`:

```env
PANEL_USERNAME=admin
PANEL_PASSWORD=your_secure_password
JWT_SECRET=random_256bit_string
DB_HOST=localhost
DB_ROOT_USER=root
DB_ROOT_PASSWORD=mysql_root_pass
PHPMYADMIN_URL=http://localhost/phpmyadmin
FILE_MANAGER_ROOT=/var/www
WEB_ROOT=/var/www/html
PANEL_URL=https://panel.yourdomain.com
```

## Verification

```bash
npm run lint
npm run build
```

## Security Notes

- Change default credentials immediately.
- Use HTTPS for the panel itself.
- Keep `FILE_MANAGER_ROOT` restricted to the smallest useful directory.
- Do not expose MariaDB/MySQL publicly.
- Run `mysql_secure_installation` after deployment.
