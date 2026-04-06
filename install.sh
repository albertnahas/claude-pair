#!/bin/sh
set -e

REPO="albertnahas/claude-pair"
BINARY="claude-pair"
INSTALL_DIR="${HOME}/.local/bin"

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
mkdir -p "$INSTALL_DIR"
cp "$TMP/$BINARY" "$INSTALL_DIR/$BINARY"
chmod 755 "$INSTALL_DIR/$BINARY"
rm -rf "$TMP"
echo "  $BINARY installed to $INSTALL_DIR/$BINARY"

# Ensure ~/.local/bin is on PATH
case ":$PATH:" in
  *":$INSTALL_DIR:"*) ;;
  *) export PATH="$INSTALL_DIR:$PATH"
     NEEDS_PATH=1
     ;;
esac

# Install dependencies
install_deps() {
  if command -v brew >/dev/null 2>&1; then
    # macOS / Homebrew
    if ! command -v tmux >/dev/null 2>&1; then
      echo "  Installing tmux..."
      brew install tmux
    fi
    if ! command -v upterm >/dev/null 2>&1; then
      echo "  Installing upterm..."
      brew install --cask owenthereal/upterm/upterm
    fi
    if ! command -v ttyd >/dev/null 2>&1; then
      echo "  Installing ttyd (for --web)..."
      brew install ttyd
    fi
  elif command -v apt-get >/dev/null 2>&1; then
    # Debian / Ubuntu
    if ! command -v tmux >/dev/null 2>&1; then
      echo "  Installing tmux..."
      sudo apt-get install -y tmux
    fi
    if ! command -v upterm >/dev/null 2>&1; then
      echo "  Installing upterm..."
      UPTERM_VERSION=$(curl -fsSL "https://api.github.com/repos/owenthereal/upterm/releases/latest" | grep '"tag_name"' | sed -E 's/.*"([^"]+)".*/\1/')
      curl -fsSL "https://github.com/owenthereal/upterm/releases/download/${UPTERM_VERSION}/upterm_${OS}_${ARCH}.tar.gz" -o /tmp/upterm.tar.gz
      tar -xzf /tmp/upterm.tar.gz -C /tmp upterm
      sudo install -m 755 /tmp/upterm /usr/local/bin/upterm
      rm -f /tmp/upterm.tar.gz /tmp/upterm
    fi
    if ! command -v ttyd >/dev/null 2>&1; then
      echo "  Installing ttyd (for --web)..."
      sudo apt-get install -y ttyd 2>/dev/null || echo "  ttyd not in apt; install manually: https://github.com/tsl0922/ttyd"
    fi
  else
    echo ""
    echo "  Could not detect package manager. Install manually:"
    echo "    tmux:   https://github.com/tmux/tmux"
    echo "    upterm: https://github.com/owenthereal/upterm"
  fi
}

echo ""
echo "Checking dependencies..."
install_deps

# Summary
echo ""
if [ -n "$NEEDS_PATH" ]; then
  echo "Add to your shell profile:"
  echo "  export PATH=\"\$HOME/.local/bin:\$PATH\""
  echo ""
fi

"$INSTALL_DIR/$BINARY" doctor
