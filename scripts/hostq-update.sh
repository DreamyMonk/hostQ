#!/bin/bash
# hostQ SSH updater. Usage:
#   sudo hostq-update           # update to latest GitHub tag
#   sudo hostq-update v0.3.5     # update to a specific tag

set -euo pipefail

REPO="${HOSTQ_REPO:-DreamyMonk/hostQ}"
PANEL_DIR="${PANEL_DIR:-/opt/hostq}"
TAG="${1:-}"

if [[ $EUID -ne 0 ]]; then
  echo "Run as root: sudo hostq-update ${TAG}" >&2
  exit 1
fi

if [[ -z "$TAG" || "$TAG" == "latest" ]]; then
  # GitHub's /tags endpoint returns tags in lexicographic order, which means
  # v0.4.0 sorts above v0.14.11 ("4" > "1"). Use /releases/latest instead,
  # which honours the actual release publish order. Fall back to fetching
  # the full tag list and sorting semver-aware if no GitHub Release exists.
  TAG="$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" 2>/dev/null | sed -n 's/.*"tag_name"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' | head -n1)"
  if [[ -z "$TAG" ]]; then
    TAG="$(curl -fsSL "https://api.github.com/repos/${REPO}/tags?per_page=100" | sed -n 's/.*"name"[[:space:]]*:[[:space:]]*"\(v[0-9][^"]*\)".*/\1/p' | sort -V | tail -n1)"
  fi
  if [[ -z "$TAG" ]]; then
    echo "Could not determine latest release tag from GitHub." >&2
    exit 1
  fi
  echo "Resolved latest tag: $TAG"
fi

if [[ ! "$TAG" =~ ^v[0-9]+\.[0-9]+\.[0-9]+([.-][A-Za-z0-9]+)?$ ]]; then
  echo "Invalid release tag: $TAG" >&2
  exit 1
fi

if ! command -v go >/dev/null 2>&1; then
  echo "The native build toolchain is required for hostQ updates. Run install.sh first." >&2
  exit 1
fi

echo "hostQ updater"
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
rsync -a --delete --exclude .env.local "$UNPACK"/ "$PANEL_DIR"/

cd "$PANEL_DIR"
go mod download
go build -trimpath -ldflags="-s -w" -o /usr/local/bin/hostq-panel .
install -m 0750 -o root -g root "$PANEL_DIR/scripts/hostq-update.sh" /usr/local/bin/hostq-update
systemctl daemon-reload
systemctl restart hostq-panel
systemctl reload nginx || true
rm -rf "$UNPACK" "$ARCHIVE"

echo "Updated hostQ to ${TAG}."
echo "Backup: ${BACKUP}"
