#!/usr/bin/env bash
set -euo pipefail

REPO="lostcoords/wariwari-cli"
INSTALL_DIR="${WARIWARI_BIN_DIR:-$HOME/.local/bin}"

OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)

case "$ARCH" in
  x86_64|amd64) ARCH="amd64" ;;
  aarch64|arm64) ARCH="arm64" ;;
  *) echo "Unsupported architecture: $ARCH"; exit 1 ;;
esac

case "$OS" in
  linux|darwin) ;;
  mingw*|msys*|cygwin*) OS="windows" ;;
  *) echo "Unsupported OS: $OS"; exit 1 ;;
esac

VERSION=$(curl -sI "https://github.com/$REPO/releases/latest" | grep -i '^location:' | sed 's/.*tag\///' | tr -d '\r\n' || true)
if [ -z "$VERSION" ]; then
  echo "Failed to determine latest version"
  exit 1
fi

ASSET="wariwari_${VERSION}_${OS}_${ARCH}.tar.gz"
if [ "$OS" = "windows" ]; then
  ASSET="wariwari_${VERSION}_${OS}_${ARCH}.zip"
fi

TMPDIR=$(mktemp -d)
trap 'rm -rf "$TMPDIR"' EXIT

curl -fsSL "https://github.com/$REPO/releases/download/${VERSION}/${ASSET}" -o "$TMPDIR/$ASSET"
curl -fsSL "https://github.com/$REPO/releases/download/${VERSION}/checksums.txt" -o "$TMPDIR/checksums.txt"

cd "$TMPDIR"
EXPECTED=$(grep "  $ASSET$" checksums.txt | awk '{print $1}')
if [ -z "$EXPECTED" ]; then
  echo "ERROR: checksum not found for $ASSET"
  exit 1
fi
ACTUAL=$(shasum -a 256 "$ASSET" | awk '{print $1}')
if [ "$EXPECTED" != "$ACTUAL" ]; then
  echo "ERROR: checksum mismatch"
  exit 1
fi

if [ "$OS" = "windows" ]; then
  unzip -q "$ASSET"
else
  tar -xzf "$ASSET"
fi

mkdir -p "$INSTALL_DIR"
BINARY="wariwari"
if [ "$OS" = "windows" ]; then
  BINARY="wariwari.exe"
fi
cp "$TMPDIR/$BINARY" "$INSTALL_DIR/$BINARY"
chmod +x "$INSTALL_DIR/$BINARY"

echo "wariwari ${VERSION} installed to $INSTALL_DIR/$BINARY"
"$INSTALL_DIR/$BINARY" setup || echo "Run '$INSTALL_DIR/$BINARY setup' to get started."
