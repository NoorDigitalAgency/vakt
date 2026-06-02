#!/usr/bin/env bash
set -euo pipefail

if [[ ${EUID:-$(id -u)} -ne 0 ]]; then
  echo "run as root" >&2
  exit 1
fi

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
BINARY_DEST="${BINARY_DEST:-/usr/local/bin/vakt}"
CONFIG_DEST="${CONFIG_DEST:-/etc/vakt/config.yaml}"
SERVICE_DEST="${SERVICE_DEST:-/etc/systemd/system/vakt.service}"

"$ROOT_DIR/scripts/build.sh"
install -Dm755 "$ROOT_DIR/dist/vakt" "$BINARY_DEST"
if [[ ! -f "$CONFIG_DEST" ]]; then
  install -Dm640 "$ROOT_DIR/configs/config.example.yaml" "$CONFIG_DEST"
fi
install -Dm644 "$ROOT_DIR/packaging/vakt.service" "$SERVICE_DEST"

systemctl daemon-reload
systemctl enable --now vakt.service

echo "vakt installed. review $CONFIG_DEST before relying on alerts."
