#!/bin/bash

set -e

# Determine OS and architecture
OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
ARCH="$(uname -m)"

# Convert architecture names
case ${ARCH} in
    x86_64) ARCH="x86_64" ;;
    aarch64) ARCH="arm64" ;;
    *) echo "Unsupported architecture: ${ARCH}"; exit 1 ;;
esac

# Get latest version
echo "Fetching latest version..."
LATEST_VERSION=$(curl -s https://api.github.com/repos/AccursedGalaxy/streakode/releases/latest | grep -oP '"tag_name":' | cut -d'"' -f4)

# Download and extract
DOWNLOAD_URL="https://github.com/AccursedGalaxy/streakode/releases/download/${LATEST_VERSION}/streakode-${OS}-${ARCH}.tar.gz"

# Installation Directory
INSTALL_DIR="/usr/local/bin"
if [ ! -w "${INSTALL_DIR}" ]; then
    INSTALL_DIR="${HOME}/.local/bin"
    mkdir -p "${INSTALL_DIR}"
fi

# Download and extract
echo "Downloading streakode ${LATEST_VERSION}..."
curl -sL "${DOWNLOAD_URL}" | tar xz -C "${INSTALL_DIR}"

# Create symlink
if [ ! -e "${INSTALL_DIR}/sk" ]; then
    ln -s "${INSTALL_DIR}/streakode" "${INSTALL_DIR}/sk"
fi

echo "✨ Installation complete! You can now use 'streakode' or 'sk' to run the CLI
