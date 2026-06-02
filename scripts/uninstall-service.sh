#!/usr/bin/env bash
set -euo pipefail

if [[ ${EUID:-$(id -u)} -ne 0 ]]; then
  echo "run as root" >&2
  exit 1
fi

SERVICE_NAME="vakt.service"
SERVICE_DEST="${SERVICE_DEST:-/etc/systemd/system/vakt.service}"
BINARY_DEST="${BINARY_DEST:-/usr/local/bin/vakt}"

systemctl disable --now "$SERVICE_NAME" 2>/dev/null || true
rm -f "$SERVICE_DEST"
rm -f "$BINARY_DEST"
systemctl daemon-reload

echo "vakt removed. configuration under /etc/vakt was left intact."
