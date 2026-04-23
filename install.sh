#!/bin/sh
set -e

REPO="exabits-xyz/gpu-cli"
BINARY_NAME="egpu"

# Default install directory
INSTALL_DIR="/usr/local/bin"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo "==> Installing $BINARY_NAME from $REPO..."

# Detect OS
OS=$(uname -s)
case "$OS" in
    Linux)  OS="linux" ;;
    Darwin) OS="darwin" ;;
    *)      echo "${RED}Error: Unsupported operating system: $OS${NC}"; exit 1 ;;
esac

# Detect Architecture
ARCH=$(uname -m)
case "$ARCH" in
    x86_64|amd64) ARCH="amd64" ;;
    arm64|aarch64) ARCH="arm64" ;;
    *)             echo "${RED}Error: Unsupported architecture: $ARCH${NC}"; exit 1 ;;
esac

echo "==> Detected OS: $OS, Architecture: $ARCH"

# Fetch latest version if not specified
if [ -z "$VERSION" ]; then
    echo "==> Fetching latest release version..."
    VERSION=$(curl -s "https://api.github.com/repos/$REPO/releases/latest" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')
    if [ -z "$VERSION" ]; then
        echo "${RED}Error: Could not determine the latest version from GitHub.${NC}"
        exit 1
    fi
fi

echo "==> Version to install: $VERSION"

# Format the filename: egpu_linux_amd64.tar.gz
FILE_NAME="${BINARY_NAME}_${OS}_${ARCH}.tar.gz"
DOWNLOAD_URL="https://github.com/$REPO/releases/download/$VERSION/$FILE_NAME"

# Create a temporary directory
TMP_DIR=$(mktemp -d)
trap 'rm -rf "$TMP_DIR"' EXIT

echo "==> Downloading $FILE_NAME..."
curl -sL -o "$TMP_DIR/$FILE_NAME" "$DOWNLOAD_URL"

echo "==> Extracting archive..."
cd "$TMP_DIR"
tar xzf "$FILE_NAME"

if [ ! -f "$BINARY_NAME" ]; then
    echo "${RED}Error: Binary '$BINARY_NAME' not found in the downloaded archive.${NC}"
    exit 1
fi

# Fall back to ~/.local/bin if /usr/local/bin is not writable
if [ ! -w "$INSTALL_DIR" ]; then
    INSTALL_DIR="$HOME/.local/bin"
    mkdir -p "$INSTALL_DIR"

    if ! echo "$PATH" | grep -q "$INSTALL_DIR"; then
        echo "${YELLOW}Warning: $INSTALL_DIR is not in your PATH.${NC}"
        echo "Add the following line to your shell profile (~/.bashrc, ~/.zshrc, etc.):"
        echo "  export PATH=\"\$PATH:$INSTALL_DIR\""
    fi
fi

echo "==> Installing to $INSTALL_DIR..."
mv "$BINARY_NAME" "$INSTALL_DIR/"
chmod +x "$INSTALL_DIR/$BINARY_NAME"

echo "${GREEN}==> Installation complete!${NC}"
echo "Run 'egpu --help' to get started."
