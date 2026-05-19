#!/bin/bash
# hostQ Go SSH updater. Usage:
#   sudo hostq-update           # update to latest GitHub tag
#   sudo hostq-update v0.2.19    # update to a specific tag

set -euo pipefail

REPO="${HOSTQ_REPO:-DreamyMonk/hostQ}"
PANEL_DIR="${PANEL_DIR:-/opt/hostq}"
TAG="${1:-}"

if [[ $EUID -ne 0 ]]; then
  echo "Run as root: sudo hostq-update ${TAG}" >&2
  exit 1
fi

if [[ -z "$TAG" || "$TAG" == "latest" ]]; then
  TAG="$(curl -fsSL "https://api.github.com/repos/${REPO}/tags?per_page=1" | sed -n 's/.*"name"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' | head -n1)"
fi

if [[ ! "$TAG" =~ ^v[0-9]+\.[0-9]+\.[0-9]+([.-][A-Za-z0-9]+)?$ ]]; then
  echo "Invalid release tag: $TAG" >&2
  exit 1
fi

if ! command -v go >/dev/null 2>&1; then
  echo "Go is required for hostQ Go updates. Install Go first, or run setup.sh." >&2
  exit 1
fi

echo "hostQ Go updater"
echo "Repository: $REPO"
echo "Target:     $TAG"
echo ""
read -r -p "Create backup and update hostQ to ${TAG}? [y/N] " confirm
if [[ ! "$confirm" =~ ^[Yy]$ ]]; then
  echo "Update cancelled."
  exit 0
fi

BACKUP="/var/backups/hostq/panel-$(date +%s).tar.gz"
ARCHIVE="/tmp/hostq-${TAG}.tar.gz"
UNPACK="/tmp/hostq-${TAG}"

mkdir -p /var/backups/hostq "$UNPACK"
test -d "$PANEL_DIR"
tar -czf "$BACKUP" -C "$PANEL_DIR" .
curl -fsSL -o "$ARCHIVE" "https://codeload.github.com/${REPO}/tar.gz/refs/tags/${TAG}"
tar -xzf "$ARCHIVE" -C "$UNPACK" --strip-components=1
rsync -a --delete --exclude .env.local --exclude node_modules --exclude .next "$UNPACK"/ "$PANEL_DIR"/

cd "$PANEL_DIR"
go mod download
go build -trimpath -ldflags="-s -w" -o /usr/local/bin/hostq-panel ./cmd/hostq-panel
install -m 0750 -o root -g root "$PANEL_DIR/scripts/hostq-update.sh" /usr/local/bin/hostq-update
systemctl daemon-reload
systemctl restart hostq-panel
systemctl reload nginx || true
rm -rf "$UNPACK" "$ARCHIVE"

echo "Updated hostQ Go to ${TAG}."
echo "Backup: ${BACKUP}"
