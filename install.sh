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

if [[ $EUID -ne 0 ]]; then error "Run as root: sudo bash install.sh"; fi

header "hostQ VPS Setup"
echo "This script installs a lightweight hosting control panel:"
echo "  - hostQ panel as a systemd service"
echo "  - hostQ direct setup access on port 8090"
echo "  - Nginx reverse proxy on port 80"
echo "  - MariaDB"
echo "  - PHP 8.2, 8.3, 8.4, 8.5 FPM where available"
echo "  - Certbot, WP-CLI, Pure-FTPd, phpMyAdmin"
echo "  - PHP OPcache enabled by default; Redis optional from Services"
echo ""
if [[ "${HOSTQ_ASSUME_YES:-false}" != "true" ]]; then
  read -r -p "Continue? [y/N] " confirm
  [[ "$confirm" =~ ^[Yy]$ ]] || exit 0
fi

export DEBIAN_FRONTEND=noninteractive

header "Updating system"
apt-get update -qq
apt-get upgrade -y -qq
apt-get install -y -qq ca-certificates curl gnupg lsb-release software-properties-common unzip rsync git build-essential openssl cron
log "System updated"

header "Installing build toolchain"
if ! command -v go >/dev/null 2>&1; then
  apt-get install -y -qq golang-go
fi
log "Native build toolchain ready"

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

# phpMyAdmin: pre-seed debconf so the install is fully non-interactive. The
# debian package otherwise prompts for which web server to configure (nginx is
# not in its list) and for dbconfig-common credentials, which is why
# `apt-get install -y phpmyadmin` was silently failing.
header "Installing phpMyAdmin"
PMA_DB_PASS="$(openssl rand -base64 24 | tr -d '/+=' | cut -c1-24)"
apt-get install -y -qq debconf-utils >/dev/null 2>&1 || true
debconf-set-selections <<DEBCONF
phpmyadmin phpmyadmin/dbconfig-install boolean true
phpmyadmin phpmyadmin/app-password-confirm password ${PMA_DB_PASS}
phpmyadmin phpmyadmin/mysql/admin-pass password ${PMA_DB_PASS}
phpmyadmin phpmyadmin/mysql/app-pass password ${PMA_DB_PASS}
phpmyadmin phpmyadmin/password-confirm password ${PMA_DB_PASS}
phpmyadmin phpmyadmin/setup-password password ${PMA_DB_PASS}
phpmyadmin phpmyadmin/reconfigure-webserver multiselect
phpmyadmin phpmyadmin/mysql/admin-user string root
DEBCONF
if DEBIAN_FRONTEND=noninteractive apt-get install -y -qq phpmyadmin; then
  log "phpMyAdmin package installed"
else
  warn "phpMyAdmin package install failed; you can rerun later with: sudo apt-get install -y phpmyadmin"
fi

# Pick a PHP-FPM socket for phpMyAdmin. Prefer 8.3 (PMA's reference version),
# fall back to whatever is active.
PMA_FPM_SOCK=""
for VER in 8.3 8.2 8.4 8.5; do
  if [[ -S "/run/php/php${VER}-fpm.sock" ]]; then
    PMA_FPM_SOCK="/run/php/php${VER}-fpm.sock"
    break
  fi
done
if [[ -z "$PMA_FPM_SOCK" ]]; then
  warn "No active PHP-FPM socket found for phpMyAdmin; defaulting to php8.3-fpm.sock"
  PMA_FPM_SOCK="/run/php/php8.3-fpm.sock"
fi

# Nginx snippet sites can include from their server block (writeNginxSite does
# this automatically for hostQ-managed vhosts).
mkdir -p /etc/nginx/snippets
cat > /etc/nginx/snippets/hostq-pma.conf <<NGINX
# hostQ phpMyAdmin alias. Sites include this from their server block so
# https://<domain>/phpmyadmin works without a separate vhost.
location ^~ /phpmyadmin/ {
    alias /usr/share/phpmyadmin/;
    index index.php;
    try_files \$uri \$uri/ /phpmyadmin/index.php?\$args;
    location ~ ^/phpmyadmin/(.+\.php)\$ {
        alias /usr/share/phpmyadmin/\$1;
        include snippets/fastcgi-php.conf;
        fastcgi_pass unix:${PMA_FPM_SOCK};
        fastcgi_param SCRIPT_FILENAME \$request_filename;
        include fastcgi_params;
    }
    location ~* ^/phpmyadmin/(.+\.(jpg|jpeg|gif|css|png|js|ico|html|xml|txt|woff|woff2|svg|map))\$ {
        alias /usr/share/phpmyadmin/\$1;
        access_log off;
        expires 1d;
    }
}
location = /phpmyadmin { return 301 /phpmyadmin/; }
NGINX
log "phpMyAdmin Nginx snippet ready (FPM via ${PMA_FPM_SOCK})"

# Default :80 vhost so /phpmyadmin/ works when the user hits a bare IP or any
# hostname that doesn't match a per-site server_name. Without it, nginx picks
# one of the hostQ-managed vhosts as the default — and if that vhost has SSL,
# the HTTP→HTTPS redirect bounces the request into a too-many-redirects loop.
cat > /etc/nginx/sites-available/hostq-default <<'NGINX'
# hostQ default vhost — answers any unmatched host on :80 so phpMyAdmin
# works without a real domain. Only /phpmyadmin/ is exposed; everything
# else returns a placeholder 404 so this can't accidentally serve a site.
server {
    listen 80 default_server;
    listen [::]:80 default_server;
    server_name _;
    access_log /var/log/nginx/hostq-default.access.log combined;

    include snippets/hostq-pma.conf;

    location / {
        return 404 "hostQ default vhost. Add a site or use a real domain.\n";
    }
}
NGINX
ln -sf /etc/nginx/sites-available/hostq-default /etc/nginx/sites-enabled/hostq-default
nginx -t >/dev/null 2>&1 && systemctl reload nginx || warn "Default vhost write triggered nginx -t errors; check 'sudo nginx -t' output"
log "Default Nginx vhost ready — /phpmyadmin/ works on any host"

log "Certbot, WP-CLI, Pure-FTPd, and phpMyAdmin installed"

header "Setting up hostQ panel"
PANEL_DIR="/opt/hostq"
PANEL_PUBLIC_PORT="${PANEL_PUBLIC_PORT:-8090}"
PANEL_ADDR="${HOSTQ_ADDR:-0.0.0.0:${PANEL_PUBLIC_PORT}}"
PANEL_UPSTREAM_ADDR="${HOSTQ_UPSTREAM_ADDR:-127.0.0.1:${PANEL_PUBLIC_PORT}}"
mkdir -p "$PANEL_DIR" /etc/hostq
chmod 700 /etc/hostq

if [[ -f "./go.mod" ]]; then
  SOURCE_DIR="$(pwd)"
  if [[ "$SOURCE_DIR" != "$PANEL_DIR" ]]; then
    rsync -a --delete ./ "$PANEL_DIR/"
    log "Copied hostQ files to $PANEL_DIR"
  else
    log "Using hostQ files already in $PANEL_DIR"
  fi
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
grep -q '^HOSTQ_ADDR=' "$PANEL_DIR/.env.local" \
  && sed -i "s/^HOSTQ_ADDR=.*/HOSTQ_ADDR=${PANEL_ADDR}/" "$PANEL_DIR/.env.local" \
  || echo "HOSTQ_ADDR=${PANEL_ADDR}" >> "$PANEL_DIR/.env.local"

install -m 0750 -o root -g root "$PANEL_DIR/scripts/hostq-update.sh" /usr/local/bin/hostq-update

cd "$PANEL_DIR"
go mod download
go build -trimpath -ldflags="-s -w" -o /usr/local/bin/hostq-panel .
log "Built /usr/local/bin/hostq-panel"
# `hostq` convenience alias so operators can run recovery commands like
# `hostq doctor` (detect + restore zero-byte nginx vhosts, validate, reload).
ln -sf /usr/local/bin/hostq-panel /usr/local/bin/hostq

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
Description=hostQ hosting control panel
After=network.target nginx.service mariadb.service

[Service]
Type=simple
Environment=HOSTQ_ADDR=${PANEL_ADDR}
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
log "hostQ service started"

touch /etc/cron.d/hostq-user-jobs
chmod 0644 /etc/cron.d/hostq-user-jobs
cat > /etc/cron.d/hostq-backups <<EOF
SHELL=/bin/bash
PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin
17 * * * * root /usr/local/bin/hostq-panel run-backups >/var/log/hostq-backups.log 2>&1
EOF
chmod 0644 /etc/cron.d/hostq-backups
log "Automatic backup runner installed"

systemctl disable --now hostq >/dev/null 2>&1 || true

header "Configuring Nginx reverse proxy"
cat > /etc/nginx/sites-available/hostq <<'EOF'
server {
    listen 80;
    server_name _;

    client_max_body_size 64M;

    location / {
        proxy_pass http://__PANEL_UPSTREAM_ADDR__;
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
sed -i "s#__PANEL_UPSTREAM_ADDR__#${PANEL_UPSTREAM_ADDR}#g" /etc/nginx/sites-available/hostq

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
echo -e "${GREEN}hostQ is running at: http://${SERVER_IP}${NC}"
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
echo "  sudo hostq-update v0.3.5"
echo "  hostq status        # health snapshot (nginx, php-fpm, panel, ssl, sites)"
echo "  hostq validate      # nginx -t + php-fpm config test (read-only)"
echo "  sudo hostq doctor   # restore missing/zero-byte nginx configs, validate + reload"
echo "  sudo hostq repair   # regenerate every vhost from /etc/hostq/sites/*.json"
echo "  sudo hostq rebuild  # idempotent regenerate from metadata"
echo "  sudo hostq backup   # snapshot all vhosts into the revision history"
echo "  sudo hostq restore [domain] [#N]  # roll back to a saved good config"
echo "  hostq deploy-log    # tail the configuration deployment journal"
echo "  mysql_secure_installation"
