#!/bin/bash
# Optional hostQ Go panel installer.
# This installs the lightweight Go preview beside the current Next.js panel.

set -euo pipefail

PANEL_DIR="${PANEL_DIR:-/opt/hostq}"
GO_ADDR="${HOSTQ_GO_ADDR:-127.0.0.1:8091}"
GO_PUBLIC_PORT="${HOSTQ_GO_PUBLIC_PORT:-8092}"

if [[ $EUID -ne 0 ]]; then
  echo "Run as root: sudo bash scripts/install-go-panel.sh" >&2
  exit 1
fi

cd "$PANEL_DIR"

apt-get update -qq
apt-get install -y -qq golang-go

go mod download
go build -trimpath -ldflags="-s -w" -o /usr/local/bin/hostq-panel ./cmd/hostq-panel

cat > /etc/systemd/system/hostq-panel.service <<EOF
[Unit]
Description=hostQ Go lightweight panel
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

cat > /etc/nginx/sites-available/hostq-go <<EOF
server {
    listen ${GO_PUBLIC_PORT};
    server_name _;

    location / {
        proxy_pass http://${GO_ADDR};
        proxy_http_version 1.1;
        proxy_set_header Host \$host;
        proxy_set_header X-Real-IP \$remote_addr;
        proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto \$scheme;
    }
}
EOF

ln -sf /etc/nginx/sites-available/hostq-go /etc/nginx/sites-enabled/hostq-go
nginx -t
systemctl reload nginx
ufw allow "${GO_PUBLIC_PORT}/tcp" comment hostQ-go >/dev/null 2>&1 || true

echo "hostQ Go panel installed: http://SERVER_IP:${GO_PUBLIC_PORT}"
