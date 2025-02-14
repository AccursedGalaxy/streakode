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
LATEST_VERSION=$(curl -s https://api.github.com/repos/AccursedGalaxy/streakode/releases/latest | grep '"tag_name":' | cut -d'"' -f4)

if [ -z "${LATEST_VERSION}" ]; then
    echo "Error: Could not determine latest version"
    exit 1
fi

# Format the download URL according to GoReleaser's naming convention
DOWNLOAD_URL="https://github.com/AccursedGalaxy/streakode/releases/download/${LATEST_VERSION}/streakode_$(tr '[:lower:]' '[:upper:]' <<< ${OS:0:1})${OS:1}_${ARCH}.tar.gz"

echo "Download URL: ${DOWNLOAD_URL}"

# Installation Directory
INSTALL_DIR="/usr/local/bin"
if [ ! -w "${INSTALL_DIR}" ]; then
    INSTALL_DIR="${HOME}/.local/bin"
    mkdir -p "${INSTALL_DIR}"
fi

# Create temporary directory for extraction
TMP_DIR=$(mktemp -d)
trap 'rm -rf "${TMP_DIR}"' EXIT

# Download and extract
echo "Downloading streakode ${LATEST_VERSION}..."
if curl -sL "${DOWNLOAD_URL}" | tar xz -C "${TMP_DIR}"; then
    # Move binary to installation directory
    mv "${TMP_DIR}/streakode" "${INSTALL_DIR}/"
    chmod +x "${INSTALL_DIR}/streakode"

    # Create symlink
    if [ ! -e "${INSTALL_DIR}/sk" ]; then
        ln -sf "${INSTALL_DIR}/streakode" "${INSTALL_DIR}/sk"
    fi

    echo "✨ Installation complete! You can now use 'streakode' or 'sk' to run the CLI"
else
    echo "❌ Installation failed! Could not download or extract the binary"
    echo "Download URL was: ${DOWNLOAD_URL}"
    exit 1
fi
