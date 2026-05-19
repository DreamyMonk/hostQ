#!/bin/bash
# hostQ Go panel installer/reinstaller.
# The Go panel is now the primary runtime; Node.js/PM2 are not required.

set -euo pipefail

PANEL_DIR="${PANEL_DIR:-/opt/hostq}"
GO_ADDR="${HOSTQ_GO_ADDR:-127.0.0.1:8091}"
PANEL_PUBLIC_PORT="${PANEL_PUBLIC_PORT:-8090}"

if [[ $EUID -ne 0 ]]; then
  echo "Run as root: sudo bash scripts/install-go-panel.sh" >&2
  exit 1
fi

cd "$PANEL_DIR"

if ! command -v go >/dev/null 2>&1; then
  apt-get update -qq
  apt-get install -y -qq golang-go
fi

go version
go mod download
go build -trimpath -ldflags="-s -w" -o /usr/local/bin/hostq-panel ./cmd/hostq-panel

mkdir -p /etc/hostq
chmod 700 /etc/hostq
if [[ ! -f /etc/hostq/admin.json ]]; then
  ADMIN_PASS="$(openssl rand -base64 24 | tr -d '/+=' | cut -c1-20)"
  HOSTQ_ADMIN_USER="${HOSTQ_ADMIN_USER:-admin}" HOSTQ_ADMIN_PASS="$ADMIN_PASS" /usr/local/bin/hostq-panel init-admin
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

cat > /etc/nginx/sites-available/hostq <<EOF
server {
    listen 80;
    listen ${PANEL_PUBLIC_PORT};
    server_name _;

    client_max_body_size 64M;

    location / {
        proxy_pass http://${GO_ADDR};
        proxy_http_version 1.1;
        proxy_read_timeout 60s;
        proxy_buffering off;
        proxy_set_header Upgrade \$http_upgrade;
        proxy_set_header Connection "upgrade";
        proxy_set_header Host \$host;
        proxy_set_header X-Real-IP \$remote_addr;
        proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto \$scheme;
    }
}
EOF

ln -sf /etc/nginx/sites-available/hostq /etc/nginx/sites-enabled/hostq
rm -f /etc/nginx/sites-enabled/hostq-go /etc/nginx/sites-enabled/default
nginx -t
systemctl reload nginx
ufw allow "${PANEL_PUBLIC_PORT}/tcp" comment hostQ >/dev/null 2>&1 || true

if command -v pm2 >/dev/null 2>&1; then
  pm2 delete hostq >/dev/null 2>&1 || true
fi

echo "hostQ Go panel installed: http://SERVER_IP:${PANEL_PUBLIC_PORT}"
