#!/bin/bash

set -e

# Enable debug output
DEBUG=flase

debug() {
    if [ "$DEBUG" = true ]; then
        echo "DEBUG: $1"
    fi
}

# Find existing installation
EXISTING_PATH=$(which streakode 2>/dev/null || true)
debug "Existing installation found at: ${EXISTING_PATH}"

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

# Get the current user's home directory
USER_HOME=$(eval echo ~${SUDO_USER:-$USER})
debug "User home directory: ${USER_HOME}"

# Check if the current version is already installed
if command -v streakode >/dev/null 2>&1; then
    # Use the user's home directory for config
    HOME="${USER_HOME}" STREAKODE_CONFIG="${USER_HOME}/.streakodeconfig" CURRENT_VERSION=$(streakode version 2>/dev/null | cut -d' ' -f3 || echo "unknown")
    debug "Current version: $CURRENT_VERSION"
    if [ "$CURRENT_VERSION" = "$LATEST_VERSION" ]; then
        echo "Latest version $LATEST_VERSION is already installed"
        exit 0
    fi
fi

# Determine installation directory from existing installation or default
if [ -n "${EXISTING_PATH}" ]; then
    INSTALL_DIR=$(dirname "${EXISTING_PATH}")
    debug "Using existing installation directory: ${INSTALL_DIR}"
else
    INSTALL_DIR="/usr/local/bin"
    if [ ! -w "${INSTALL_DIR}" ]; then
        INSTALL_DIR="${USER_HOME}/.local/bin"
        mkdir -p "${INSTALL_DIR}"
    fi
    debug "Using new installation directory: ${INSTALL_DIR}"
fi

# Check if we have write permissions to the installation directory
if [ ! -w "${INSTALL_DIR}" ]; then
    echo "❌ No write permission to ${INSTALL_DIR}"
    echo "Please run with sudo or choose a different installation directory"
    exit 1
fi

# Create temporary directory for extraction
TMP_DIR=$(mktemp -d)
trap 'rm -rf "${TMP_DIR}"' EXIT

# Download and extract
echo "Downloading streakode ${LATEST_VERSION}..."

# Download to a temporary file first
TEMP_TAR="${TMP_DIR}/streakode.tar.gz"
if ! curl -sL "${DOWNLOAD_URL}" -o "${TEMP_TAR}"; then
    echo "❌ Failed to download the binary"
    exit 1
fi

debug "Downloaded archive size: $(ls -lh "${TEMP_TAR}" | awk '{print $5}')"

# Extract the archive
if ! tar xzf "${TEMP_TAR}" -C "${TMP_DIR}"; then
    echo "❌ Failed to extract the archive"
    exit 1
fi

debug "Extracted files in temp dir: $(ls -la ${TMP_DIR})"

# Check if the binary exists in the temp directory
if [ ! -f "${TMP_DIR}/streakode" ]; then
    echo "❌ Binary not found in the extracted archive"
    debug "Archive contents:"
    tar tvf "${TEMP_TAR}"
    exit 1
fi

# Stop the current binary if it's running
if pidof streakode >/dev/null 2>&1; then
    echo "Stopping running instance of streakode..."
    killall streakode || true
fi

# Backup existing binary
if [ -f "${INSTALL_DIR}/streakode" ]; then
    mv "${INSTALL_DIR}/streakode" "${INSTALL_DIR}/streakode.backup"
    debug "Created backup of existing binary"
fi

# Force remove existing binary and symlink
rm -f "${INSTALL_DIR}/streakode"
rm -f "${INSTALL_DIR}/sk"

# Move binary to installation directory and set permissions
if ! mv "${TMP_DIR}/streakode" "${INSTALL_DIR}/"; then
    echo "❌ Failed to install binary"
    # Restore backup if it exists
    if [ -f "${INSTALL_DIR}/streakode.backup" ]; then
        mv "${INSTALL_DIR}/streakode.backup" "${INSTALL_DIR}/streakode"
    fi
    exit 1
fi

# Set ownership and permissions
if [ -n "${SUDO_USER}" ]; then
    chown ${SUDO_USER}:$(id -g ${SUDO_USER}) "${INSTALL_DIR}/streakode"
fi
chmod 755 "${INSTALL_DIR}/streakode"

# Create symlink
ln -sf "${INSTALL_DIR}/streakode" "${INSTALL_DIR}/sk"
if [ -n "${SUDO_USER}" ]; then
    chown -h ${SUDO_USER}:$(id -g ${SUDO_USER}) "${INSTALL_DIR}/sk"
fi

# Verify installation
HOME="${USER_HOME}" STREAKODE_CONFIG="${USER_HOME}/.streakodeconfig" INSTALLED_VERSION=$("${INSTALL_DIR}/streakode" version 2>/dev/null | cut -d' ' -f3 || echo "unknown")
debug "Installed version: $INSTALLED_VERSION"
debug "Expected version: ${LATEST_VERSION}"

if [ "${INSTALLED_VERSION}" != "${LATEST_VERSION#v}" ] && [ "${INSTALLED_VERSION}" != "${LATEST_VERSION}" ]; then
    echo "❌ Installation verification failed"
    # Restore backup if it exists
    if [ -f "${INSTALL_DIR}/streakode.backup" ]; then
        mv "${INSTALL_DIR}/streakode.backup" "${INSTALL_DIR}/streakode"
    fi
    exit 1
fi

# Remove backup if everything succeeded
rm -f "${INSTALL_DIR}/streakode.backup"

echo "✨ Installation complete! Streakode ${LATEST_VERSION} has been installed successfully"
echo "You can now use 'streakode' or 'sk' to run the CLI"

# Show version to confirm
HOME="${USER_HOME}" STREAKODE_CONFIG="${USER_HOME}/.streakodeconfig" "${INSTALL_DIR}/streakode" version

# If installed in /usr/local/bin, we're done
if [ "${INSTALL_DIR}" = "/usr/local/bin" ]; then
    exit 0
fi

# Add installation directory to PATH if it's not already there
if [[ ":$PATH:" != *":${INSTALL_DIR}:"* ]]; then
    echo "Adding ${INSTALL_DIR} to PATH..."
    echo "export PATH=\"\$PATH:${INSTALL_DIR}\"" >> "${USER_HOME}/.bashrc"
    echo "You may need to restart your shell or run 'source ~/.bashrc'"
fi
