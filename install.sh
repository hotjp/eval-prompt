#!/usr/bin/env sh
set -e

REPO="hotjp/eval-prompt"
BINARY_NAME="ep"
INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"
TMPDIR="${TMPDIR:-/tmp}"

usage() {
    echo "Usage: curl -fsSL https://raw.githubusercontent.com/${REPO}/main/install.sh | sh [-s -- --help]"
    echo ""
    echo "Options:"
    echo "  -b DIR   Install binary to DIR (default: /usr/local/bin)"
    echo "  -v VER   Install specific version (default: latest)"
    echo "  --help   Show this help"
    exit 0
}

# Parse args
while [ $# -gt 0 ]; do
    case "$1" in
        -b) INSTALL_DIR="$2"; shift 2 ;;
        -v) VERSION="$2"; shift 2 ;;
        --help|-h) usage ;;
        *) shift ;;
    esac
done

# Detect OS
OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
case "$OS" in
    darwin) OS="darwin" ;;
    linux)  OS="linux" ;;
    *)      echo "Unsupported OS: $OS"; exit 1 ;;
esac

# Detect ARCH
ARCH="$(uname -m)"
case "$ARCH" in
    x86_64)  ARCH="amd64" ;;
    aarch64|arm64) ARCH="arm64" ;;
    armv7l)  ARCH="arm" ;;
    *)       echo "Unsupported architecture: $ARCH"; exit 1 ;;
esac

# Determine version
if [ -z "$VERSION" ]; then
    echo "Fetching latest version..."
    VERSION="$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" | grep '"tag_name":' | sed -E 's/.*"tag_name": "([^"]+)".*/\1/')"
fi

if [ -z "$VERSION" ]; then
    echo "Failed to determine version. Is the repository public and has releases?"
    exit 1
fi

FILENAME="${BINARY_NAME}-${OS}-${ARCH}"
DOWNLOAD_URL="https://github.com/${REPO}/releases/download/${VERSION}/${FILENAME}"
CHECKSUM_URL="https://github.com/${REPO}/releases/download/${VERSION}/${FILENAME}.sha256"

echo "eval-prompt installer"
echo "  Version:    ${VERSION}"
echo "  Platform:   ${OS}/${ARCH}"
echo "  URL:        ${DOWNLOAD_URL}"
echo "  Install to: ${INSTALL_DIR}"
echo ""

# Create install dir if needed
mkdir -p "${INSTALL_DIR}"

# Download binary
cd "${TMPDIR}"
echo "Downloading..."
curl -fsSL -o "${FILENAME}" "${DOWNLOAD_URL}"

# Download checksum if exists
if curl -fsSL -o /dev/null "${CHECKSUM_URL}"; then
    echo "Verifying checksum..."
    curl -fsSL "${CHECKSUM_URL}" | sha256sum -c --status - && echo "Checksum OK" || {
        echo "Checksum mismatch!"
        rm -f "${FILENAME}"
        exit 1
    }
fi

# Install
echo "Installing to ${INSTALL_DIR}/${BINARY_NAME}..."
rm -f "${INSTALL_DIR}/${BINARY_NAME}"
mv "${FILENAME}" "${INSTALL_DIR}/${BINARY_NAME}"
chmod +x "${INSTALL_DIR}/${BINARY_NAME}"

# Cleanup
rm -f "${TMPDIR}/${FILENAME}"

echo ""
echo "Installed: ${INSTALL_DIR}/${BINARY_NAME}"
echo "Run 'ep --version' to verify."
