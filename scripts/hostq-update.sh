#!/bin/bash
# hostQ SSH updater. Usage:
#   sudo hostq-update           # update to latest GitHub release
#   sudo hostq-update v0.2.2    # update to a specific release tag

set -euo pipefail

REPO="DreamyMonk/hostQ"
HELPER="${HOSTQ_HELPER:-/usr/local/sbin/hostq-helper}"
TAG="${1:-}"

if [[ $EUID -ne 0 ]]; then
  echo "Run as root: sudo hostq-update ${TAG}" >&2
  exit 1
fi

if [[ ! -x "$HELPER" ]]; then
  echo "hostQ helper not found or not executable: $HELPER" >&2
  exit 1
fi

if [[ -z "$TAG" || "$TAG" == "latest" ]]; then
  TAG="$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" \
    | node -e "let s='';process.stdin.on('data',d=>s+=d);process.stdin.on('end',()=>{const r=JSON.parse(s); if(!r.tag_name) process.exit(2); console.log(r.tag_name)})")"
fi

if [[ ! "$TAG" =~ ^v[0-9]+\.[0-9]+\.[0-9]+([.-][A-Za-z0-9]+)?$ ]]; then
  echo "Invalid release tag: $TAG" >&2
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

RESULT="$(node "$HELPER" "{\"task\":\"panel.update\",\"payload\":{\"tag\":\"$TAG\"}}")"
node -e "const r=JSON.parse(process.argv[1]); console.log(r.stdout || r.stderr || r.error || 'No output'); process.exit(r.success ? 0 : 1)" "$RESULT"
