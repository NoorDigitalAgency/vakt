#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
DIST_DIR="$ROOT_DIR/dist"
TARGET_OS="${TARGET_OS:-linux}"
TARGET_ARCH="${TARGET_ARCH:-amd64}"
VERSION="${VERSION:-dev}"
PACKAGE_NAME="vakt_${TARGET_OS}_${TARGET_ARCH}"
PACKAGE_DIR="$DIST_DIR/$PACKAGE_NAME"
ARCHIVE_PATH="$DIST_DIR/${PACKAGE_NAME}.tar.gz"
CHECKSUM_PATH="$DIST_DIR/${PACKAGE_NAME}.tar.gz.sha256"
LDFLAGS="-X github.com/NoorDigitalAgency/vakt/internal/buildinfo.Version=${VERSION}"

rm -rf "$PACKAGE_DIR" "$ARCHIVE_PATH" "$CHECKSUM_PATH"
mkdir -p "$PACKAGE_DIR"

cd "$ROOT_DIR"
go test ./...
CGO_ENABLED=0 GOOS="$TARGET_OS" GOARCH="$TARGET_ARCH" go build -ldflags "$LDFLAGS" -o "$PACKAGE_DIR/vakt" ./cmd/vakt
install -m644 "$ROOT_DIR/configs/config.example.yaml" "$PACKAGE_DIR/config.yaml"
install -m644 "$ROOT_DIR/packaging/vakt.service" "$PACKAGE_DIR/vakt.service"
printf '%s\n' "$VERSION" > "$PACKAGE_DIR/VERSION"

tar -C "$DIST_DIR" -czf "$ARCHIVE_PATH" "$PACKAGE_NAME"
(
  cd "$DIST_DIR"
  sha256sum "$(basename "$ARCHIVE_PATH")" > "$CHECKSUM_PATH"
)
