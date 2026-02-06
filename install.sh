#!/bin/bash
set -e

REPO="thinesjs/gha-freeze"
INSTALL_DIR="/usr/local/bin"

get_latest_release() {
    curl -s "https://api.github.com/repos/$REPO/releases/latest" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/'
}

detect_os_arch() {
    OS=$(uname -s | tr '[:upper:]' '[:lower:]')
    ARCH=$(uname -m)

    case "$OS" in
        linux) OS="linux" ;;
        darwin) OS="macOS" ;;
        *) echo "Unsupported OS: $OS"; exit 1 ;;
    esac

    case "$ARCH" in
        x86_64) ARCH="amd64" ;;
        amd64) ARCH="amd64" ;;
        arm64) ARCH="arm64" ;;
        aarch64) ARCH="arm64" ;;
        *) echo "Unsupported architecture: $ARCH"; exit 1 ;;
    esac
}

main() {
    echo "Installing gha-freeze..."

    detect_os_arch
    VERSION=$(get_latest_release)
    VERSION_NUM=${VERSION#v}

    echo "Latest version: $VERSION"
    echo "Platform: $OS $ARCH"

    FILENAME="gha-freeze_${VERSION_NUM}_${OS}_${ARCH}.tar.gz"
    URL="https://github.com/$REPO/releases/download/$VERSION/$FILENAME"

    echo "Downloading from $URL..."
    curl -L "$URL" | tar xz

    if [ -w "$INSTALL_DIR" ]; then
        mv gha-freeze "$INSTALL_DIR/"
    else
        echo "Installing to $INSTALL_DIR (requires sudo)..."
        sudo mv gha-freeze "$INSTALL_DIR/"
    fi

    echo "✓ gha-freeze installed successfully!"
    echo "Run: gha-freeze --help"
}

main
