#!/bin/bash
# hostQ - VPS setup script
# Target: Ubuntu 22.04/24.04 or Debian 12, run as root.

set -euo pipefail

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

log() { echo -e "${GREEN}[ok]${NC} $1"; }
warn() { echo -e "${YELLOW}[warn]${NC} $1"; }
error() { echo -e "${RED}[error]${NC} $1"; exit 1; }
header() { echo -e "\n${BLUE}=== $1 ===${NC}"; }

if [[ $EUID -ne 0 ]]; then error "Run as root: sudo bash setup.sh"; fi

header "hostQ VPS Setup"
echo "This script installs a lightweight LEMP hosting stack:"
echo "  - Node.js 20 LTS"
echo "  - Nginx 1.24+ where available"
echo "  - MariaDB 10.11 where available"
echo "  - PHP 8.2, 8.3, 8.4, 8.5 FPM only"
echo "  - Certbot, WP-CLI, Pure-FTPd, phpMyAdmin 5.2"
echo "  - PM2 with a 384 MB memory restart limit"
echo ""
read -r -p "Continue? [y/N] " confirm
[[ "$confirm" =~ ^[Yy]$ ]] || exit 0

export DEBIAN_FRONTEND=noninteractive

header "Updating system"
apt-get update -qq
apt-get upgrade -y -qq
apt-get install -y -qq ca-certificates curl gnupg lsb-release software-properties-common unzip rsync
log "System updated"

header "Installing Node.js 20"
if ! command -v node >/dev/null 2>&1; then
  curl -fsSL https://deb.nodesource.com/setup_20.x | bash -
  apt-get install -y -qq nodejs
fi
log "Node.js $(node -v) installed"

header "Installing Nginx"
apt-get install -y -qq nginx
systemctl enable nginx
systemctl start nginx
log "Nginx $(nginx -v 2>&1 | sed 's#nginx version: ##') installed"

header "Installing MariaDB"
apt-get install -y -qq mariadb-server mariadb-client
systemctl enable mariadb
systemctl start mariadb
log "$(mysql --version | awk '{print $1, $3, $5}') installed"

header "Installing supported PHP versions"
add-apt-repository -y ppa:ondrej/php >/dev/null 2>&1 || true
apt-get update -qq
for VER in 8.2 8.3 8.4 8.5; do
  if apt-get install -y -qq \
    php${VER} php${VER}-fpm php${VER}-cli php${VER}-common \
    php${VER}-mysql php${VER}-curl php${VER}-gd php${VER}-mbstring \
    php${VER}-xml php${VER}-zip php${VER}-bcmath php${VER}-intl >/dev/null 2>&1; then
    systemctl enable php${VER}-fpm >/dev/null 2>&1 || true
    systemctl start php${VER}-fpm >/dev/null 2>&1 || true
    log "PHP ${VER} installed"
  else
    warn "PHP ${VER} not available for this distro/repository"
  fi
done
update-alternatives --set php /usr/bin/php8.4 >/dev/null 2>&1 || true

header "Installing hosting tools"
apt-get install -y -qq certbot python3-certbot-nginx python3-certbot-apache pure-ftpd pure-ftpd-common
systemctl enable pure-ftpd >/dev/null 2>&1 || true
systemctl start pure-ftpd >/dev/null 2>&1 || true

curl -fsSL -o /usr/local/bin/wp https://raw.githubusercontent.com/wp-cli/builds/gh-pages/phar/wp-cli.phar
chmod +x /usr/local/bin/wp

apt-get install -y -qq phpmyadmin >/dev/null 2>&1 || warn "phpMyAdmin package install failed; install manually if needed"
npm install -g pm2 -q
pm2 startup systemd -u root --hp /root >/dev/null 2>&1 || true
log "Certbot, WP-CLI, Pure-FTPd, phpMyAdmin, and PM2 installed"

header "Setting up hostQ"
PANEL_DIR="/opt/hostq"
mkdir -p "$PANEL_DIR"

if [[ -f "./package.json" ]]; then
  rsync -a --delete --exclude node_modules --exclude .next ./ "$PANEL_DIR/"
  log "Copied panel files to $PANEL_DIR"
else
  warn "Run this script from the hostQ directory"
fi

if [[ ! -f "$PANEL_DIR/.env.local" ]]; then
  cp "$PANEL_DIR/.env.example" "$PANEL_DIR/.env.local"
  warn "Created .env.local"
fi

mkdir -p /etc/hostq
chmod 700 /etc/hostq
install -m 0750 -o root -g root "$PANEL_DIR/scripts/hostq-helper.mjs" /usr/local/sbin/hostq-helper
cat > /etc/sudoers.d/hostq-helper <<'EOF'
Defaults!/usr/local/sbin/hostq-helper !requiretty
root ALL=(root) NOPASSWD: /usr/local/sbin/hostq-helper
EOF
chmod 0440 /etc/sudoers.d/hostq-helper
log "Installed hostQ privileged helper allowlist"

cd "$PANEL_DIR"
npm ci

ADMIN_USER="admin"
ADMIN_PASS="$(openssl rand -base64 24 | tr -d '/+=' | cut -c1-20)"
if [[ ! -f /etc/hostq/admin.json ]]; then
  HOSTQ_ADMIN_USER="$ADMIN_USER" HOSTQ_ADMIN_PASS="$ADMIN_PASS" node <<'NODE'
const fs = require('fs');
const bcrypt = require('bcryptjs');
const username = process.env.HOSTQ_ADMIN_USER;
const password = process.env.HOSTQ_ADMIN_PASS;
fs.mkdirSync('/etc/hostq', { recursive: true, mode: 0o700 });
fs.writeFileSync('/etc/hostq/admin.json', JSON.stringify({
  username,
  passwordHash: bcrypt.hashSync(password, 12),
  role: 'admin',
  otpEnabled: false,
  createdAt: new Date().toISOString()
}, null, 2), { mode: 0o600 });
NODE
  log "Generated hostQ admin credentials"
else
  ADMIN_PASS=""
  warn "Existing /etc/hostq/admin.json found; admin credentials were not regenerated"
fi

export NODE_OPTIONS="--max-old-space-size=384"
npm run build
npm prune --omit=dev
log "Panel built"

pm2 delete hostq >/dev/null 2>&1 || true
pm2 start npm --name hostq --max-memory-restart 384M -- start
pm2 save
log "Panel started with PM2"

header "Configuring Nginx reverse proxy"
cat > /etc/nginx/sites-available/hostq <<'EOF'
server {
    listen 80;
    server_name _;

    client_max_body_size 64M;

    location / {
        proxy_pass http://127.0.0.1:3000;
        proxy_http_version 1.1;
        proxy_read_timeout 60s;
        proxy_buffering off;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
EOF

ln -sf /etc/nginx/sites-available/hostq /etc/nginx/sites-enabled/
rm -f /etc/nginx/sites-enabled/default
nginx -t
systemctl reload nginx
log "Nginx reverse proxy configured"

header "Firewall"
ufw allow 22/tcp comment SSH >/dev/null 2>&1 || true
ufw allow 21/tcp comment FTP >/dev/null 2>&1 || true
ufw allow 80/tcp comment HTTP >/dev/null 2>&1 || true
ufw allow 443/tcp comment HTTPS >/dev/null 2>&1 || true
ufw --force enable >/dev/null 2>&1 || true
log "Firewall configured"

header "Setup complete"
SERVER_IP=$(curl -fsS ifconfig.me 2>/dev/null || hostname -I | awk '{print $1}')
echo ""
echo -e "${GREEN}hostQ is running at: http://${SERVER_IP}${NC}"
if [[ -n "$ADMIN_PASS" ]]; then
  echo ""
  echo -e "${YELLOW}Initial hostQ admin login:${NC}"
  echo "  Username: $ADMIN_USER"
  echo "  Password: $ADMIN_PASS"
  echo ""
  echo "Save this password now. It is shown only once."
  echo "After login, change the password and enable 2FA from Admin > Security."
else
  echo "Use the existing admin account in /etc/hostq/admin.json."
fi
echo ""
echo "Useful commands:"
echo "  pm2 status"
echo "  pm2 logs hostq"
echo "  pm2 restart hostq"
echo "  mysql_secure_installation"
