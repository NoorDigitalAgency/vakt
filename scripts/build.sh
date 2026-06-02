#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
DIST_DIR="$ROOT_DIR/dist"

mkdir -p "$DIST_DIR"
cd "$ROOT_DIR"

go test ./...
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o "$DIST_DIR/vakt" ./cmd/vakt
