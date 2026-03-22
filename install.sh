#!/bin/bash
set -euo pipefail

REPO="saadjs/genie-cli"
BINARY="genie"
INSTALL_DIR="/usr/local/bin"

echo "Installing genie..."

# Detect OS
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
case "$OS" in
    linux|darwin) ;;
    *) echo "Error: Unsupported OS: $OS"; exit 1 ;;
esac

# Detect architecture
ARCH=$(uname -m)
case "$ARCH" in
    x86_64)        ARCH="amd64" ;;
    aarch64|arm64) ARCH="arm64" ;;
    *) echo "Error: Unsupported architecture: $ARCH"; exit 1 ;;
esac

# Get latest release tag
LATEST=$(curl -sL "https://api.github.com/repos/$REPO/releases/latest" | grep '"tag_name"' | head -1 | cut -d'"' -f4)
if [ -z "$LATEST" ]; then
    echo "Error: Could not determine latest release"
    exit 1
fi

# Download and install
URL="https://github.com/$REPO/releases/download/$LATEST/genie_${OS}_${ARCH}.tar.gz"
echo "Downloading genie $LATEST for ${OS}/${ARCH}..."

TMP=$(mktemp -d)
trap 'rm -rf "$TMP"' EXIT

curl -sL "$URL" | tar xz -C "$TMP"

if [ -w "$INSTALL_DIR" ]; then
    install -m 755 "$TMP/$BINARY" "$INSTALL_DIR/$BINARY"
else
    echo "Need sudo to install to $INSTALL_DIR"
    sudo install -m 755 "$TMP/$BINARY" "$INSTALL_DIR/$BINARY"
fi

echo "genie $LATEST installed to $INSTALL_DIR/$BINARY"
echo ""
echo "Try it out: genie \"show me all files in this folder\""
