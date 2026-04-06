#!/bin/sh
set -e

REPO="albertnahas/claude-pair"
BINARY="claude-pair"
INSTALL_DIR="/usr/local/bin"

# Detect OS and arch
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)
case "$ARCH" in
  x86_64) ARCH="amd64" ;;
  aarch64|arm64) ARCH="arm64" ;;
  *) echo "Unsupported architecture: $ARCH" && exit 1 ;;
esac

# Get latest version
VERSION=$(curl -fsSL "https://api.github.com/repos/$REPO/releases/latest" | grep '"tag_name"' | sed -E 's/.*"([^"]+)".*/\1/')
if [ -z "$VERSION" ]; then
  echo "Failed to fetch latest version"
  exit 1
fi

URL="https://github.com/$REPO/releases/download/$VERSION/${BINARY}_${OS}_${ARCH}.tar.gz"

echo "Installing $BINARY $VERSION ($OS/$ARCH)..."
TMP=$(mktemp -d)
curl -fsSL "$URL" -o "$TMP/archive.tar.gz"
tar -xzf "$TMP/archive.tar.gz" -C "$TMP"
sudo mkdir -p "$INSTALL_DIR"
sudo cp "$TMP/$BINARY" "$INSTALL_DIR/$BINARY"
sudo chmod 755 "$INSTALL_DIR/$BINARY"
rm -rf "$TMP"

echo "$BINARY $VERSION installed to $INSTALL_DIR/$BINARY"
echo ""
echo "Prerequisites: tmate, tmux, claude"
echo "  brew install tmate tmux  (or apt install tmate tmux)"
echo "  Run '$BINARY doctor' to verify"
