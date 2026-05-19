#!/bin/bash
# hostQ - Go-first VPS setup script
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
echo "This script installs a lightweight Go hosting control panel:"
echo "  - hostQ Go panel as a systemd service"
echo "  - Nginx reverse proxy on ports 80 and 8090"
echo "  - MariaDB"
echo "  - PHP 8.2, 8.3, 8.4, 8.5 FPM where available"
echo "  - Certbot, WP-CLI, Pure-FTPd, phpMyAdmin"
echo "  - PHP OPcache enabled by default; Redis optional from Services"
echo ""
echo "Node.js, npm, Next.js runtime, and PM2 are not installed or required."
echo ""
read -r -p "Continue? [y/N] " confirm
[[ "$confirm" =~ ^[Yy]$ ]] || exit 0

export DEBIAN_FRONTEND=noninteractive

header "Updating system"
apt-get update -qq
apt-get upgrade -y -qq
apt-get install -y -qq ca-certificates curl gnupg lsb-release software-properties-common unzip rsync git build-essential openssl
log "System updated"

header "Installing Go"
if ! command -v go >/dev/null 2>&1; then
  apt-get install -y -qq golang-go
fi
log "$(go version) installed"

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

header "Configuring PHP OPcache"
for VER in 8.2 8.3 8.4 8.5; do
  CONF="/etc/php/${VER}/mods-available/opcache.ini"
  if [[ -f "$CONF" ]]; then
    cat > "$CONF" <<'EOF'
zend_extension=opcache.so
opcache.enable=1
opcache.enable_cli=0
opcache.memory_consumption=96
opcache.interned_strings_buffer=12
opcache.max_accelerated_files=20000
opcache.validate_timestamps=1
opcache.revalidate_freq=2
opcache.save_comments=1
EOF
    phpenmod -v "$VER" opcache >/dev/null 2>&1 || true
    systemctl restart php${VER}-fpm >/dev/null 2>&1 || true
  fi
done
log "PHP OPcache configured"

header "Configuring optional Nginx FastCGI cache"
mkdir -p /var/cache/nginx/hostq-fastcgi
chown -R www-data:www-data /var/cache/nginx/hostq-fastcgi >/dev/null 2>&1 || true
cat > /etc/nginx/conf.d/hostq-fastcgi-cache.conf <<'EOF'
fastcgi_cache_path /var/cache/nginx/hostq-fastcgi levels=1:2 keys_zone=HOSTQ_FASTCGI:64m inactive=60m max_size=512m use_temp_path=off;
fastcgi_cache_key "$scheme$request_method$host$request_uri";
EOF
nginx -t >/dev/null 2>&1 && systemctl reload nginx || true
log "Nginx FastCGI cache zone ready"

header "Installing hosting tools"
apt-get install -y -qq certbot python3-certbot-nginx python3-certbot-apache pure-ftpd pure-ftpd-common
systemctl enable pure-ftpd >/dev/null 2>&1 || true
systemctl start pure-ftpd >/dev/null 2>&1 || true

curl -fsSL -o /usr/local/bin/wp https://raw.githubusercontent.com/wp-cli/builds/gh-pages/phar/wp-cli.phar
chmod +x /usr/local/bin/wp

apt-get install -y -qq phpmyadmin >/dev/null 2>&1 || warn "phpMyAdmin package install failed; install manually if needed"
log "Certbot, WP-CLI, Pure-FTPd, and phpMyAdmin installed"

header "Setting up hostQ Go panel"
PANEL_DIR="/opt/hostq"
PANEL_PUBLIC_PORT="${PANEL_PUBLIC_PORT:-8090}"
GO_ADDR="${HOSTQ_GO_ADDR:-127.0.0.1:8091}"
mkdir -p "$PANEL_DIR" /etc/hostq
chmod 700 /etc/hostq

if [[ -f "./go.mod" ]]; then
  rsync -a --delete --exclude node_modules --exclude .next ./ "$PANEL_DIR/"
  log "Copied hostQ files to $PANEL_DIR"
else
  warn "Run this script from the hostQ directory"
fi

if [[ ! -f "$PANEL_DIR/.env.local" && -f "$PANEL_DIR/.env.example" ]]; then
  cp "$PANEL_DIR/.env.example" "$PANEL_DIR/.env.local"
  warn "Created .env.local"
fi
touch "$PANEL_DIR/.env.local"
grep -q '^HOSTQ_ALLOW_INSECURE_HTTP=' "$PANEL_DIR/.env.local" \
  && sed -i 's/^HOSTQ_ALLOW_INSECURE_HTTP=.*/HOSTQ_ALLOW_INSECURE_HTTP=true/' "$PANEL_DIR/.env.local" \
  || echo "HOSTQ_ALLOW_INSECURE_HTTP=true" >> "$PANEL_DIR/.env.local"
grep -q '^HOSTQ_GO_ADDR=' "$PANEL_DIR/.env.local" \
  && sed -i "s/^HOSTQ_GO_ADDR=.*/HOSTQ_GO_ADDR=${GO_ADDR}/" "$PANEL_DIR/.env.local" \
  || echo "HOSTQ_GO_ADDR=${GO_ADDR}" >> "$PANEL_DIR/.env.local"

install -m 0750 -o root -g root "$PANEL_DIR/scripts/hostq-update.sh" /usr/local/bin/hostq-update

cd "$PANEL_DIR"
go mod download
go build -trimpath -ldflags="-s -w" -o /usr/local/bin/hostq-panel ./cmd/hostq-panel
log "Built /usr/local/bin/hostq-panel"

ADMIN_USER="admin"
ADMIN_PASS="$(openssl rand -base64 24 | tr -d '/+=' | cut -c1-20)"
if [[ ! -f /etc/hostq/admin.json ]]; then
  INIT_OUTPUT="$(HOSTQ_ADMIN_USER="$ADMIN_USER" HOSTQ_ADMIN_PASS="$ADMIN_PASS" /usr/local/bin/hostq-panel init-admin)"
  log "Generated hostQ admin credentials"
else
  INIT_OUTPUT="Existing admin account found; not regenerated."
  ADMIN_PASS=""
  warn "Existing /etc/hostq/admin.json found; admin credentials were not regenerated"
fi

cat > /etc/systemd/system/hostq-panel.service <<EOF
[Unit]
Description=hostQ Go hosting control panel
After=network.target nginx.service mariadb.service

[Service]
Type=simple
Environment=HOSTQ_GO_ADDR=${GO_ADDR}
Environment=HOSTQ_DATA_DIR=/etc/hostq
Environment=WEB_ROOT=/var/www
EnvironmentFile=-${PANEL_DIR}/.env.local
ExecStart=/usr/local/bin/hostq-panel
Restart=always
RestartSec=3
User=root
WorkingDirectory=${PANEL_DIR}
NoNewPrivileges=false

[Install]
WantedBy=multi-user.target
EOF
systemctl daemon-reload
systemctl enable --now hostq-panel
log "hostQ Go service started"

if command -v pm2 >/dev/null 2>&1; then
  pm2 delete hostq >/dev/null 2>&1 || true
fi
systemctl disable --now hostq >/dev/null 2>&1 || true

header "Configuring Nginx reverse proxy"
cat > /etc/nginx/sites-available/hostq <<'EOF'
server {
    listen 80;
    listen __PANEL_PUBLIC_PORT__;
    server_name _;

    client_max_body_size 64M;

    location / {
        proxy_pass http://__GO_ADDR__;
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
sed -i "s/__PANEL_PUBLIC_PORT__/${PANEL_PUBLIC_PORT}/g" /etc/nginx/sites-available/hostq
sed -i "s#__GO_ADDR__#${GO_ADDR}#g" /etc/nginx/sites-available/hostq

ln -sf /etc/nginx/sites-available/hostq /etc/nginx/sites-enabled/hostq
rm -f /etc/nginx/sites-enabled/default /etc/nginx/sites-enabled/hostq-go
nginx -t
systemctl reload nginx
log "Nginx reverse proxy configured"

header "Firewall"
ufw allow 22/tcp comment SSH >/dev/null 2>&1 || true
ufw allow 21/tcp comment FTP >/dev/null 2>&1 || true
ufw allow 80/tcp comment HTTP >/dev/null 2>&1 || true
ufw allow 443/tcp comment HTTPS >/dev/null 2>&1 || true
ufw allow ${PANEL_PUBLIC_PORT}/tcp comment hostQ >/dev/null 2>&1 || true
ufw --force enable >/dev/null 2>&1 || true
log "Firewall configured"

header "Setup complete"
SERVER_IP=$(curl -fsS ifconfig.me 2>/dev/null || hostname -I | awk '{print $1}')
echo ""
echo -e "${GREEN}hostQ Go is running at: http://${SERVER_IP}${NC}"
echo -e "${GREEN}Direct setup URL: http://${SERVER_IP}:${PANEL_PUBLIC_PORT}${NC}"
if [[ -n "$ADMIN_PASS" ]]; then
  echo ""
  echo -e "${YELLOW}${INIT_OUTPUT}${NC}"
  echo ""
  echo "Save this password now. It is shown only once."
else
  echo "Use the existing admin account in /etc/hostq/admin.json."
fi
echo ""
echo "Useful commands:"
echo "  systemctl status hostq-panel --no-pager -l"
echo "  journalctl -u hostq-panel -f"
echo "  sudo hostq-update"
echo "  sudo hostq-update v0.2.19"
echo "  mysql_secure_installation"
