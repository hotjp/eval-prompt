#!/usr/bin/env sh
set -e

REPO="hotjp/eval-prompt"
BINARY_NAME="ep"
INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"
TMPDIR="${TMPDIR:-/tmp}"
SUDO=""

usage() {
    echo "Usage: curl -fsSL https://raw.githubusercontent.com/${REPO}/main/install.sh | sudo sh [-s -- --help]"
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

# Check if install dir is writable, use sudo if needed
if [ ! -w "$INSTALL_DIR" ]; then
    echo "Note: $INSTALL_DIR is not writable, using sudo..."
    SUDO="sudo"
fi

# Check git dependency (required)
install_git() {
    echo "git is required but not installed. Attempting to install..."

    if [ "$(uname)" = "Darwin" ]; then
        # Try Homebrew first
        if command -v brew >/dev/null 2>&1; then
            echo "Installing git via Homebrew..."
            brew install git && return 0
        fi
        # Homebrew not available or failed, show manual instructions
        echo "Could not auto-install git. Please install manually:"
        echo "  1. Download: https://git-scm.com/download/mac"
        echo "  2. Double-click the .pkg file"
        echo "  3. Follow the installer"
        exit 1
    elif [ -f /etc/debian_version ]; then
        echo "Installing git via apt..."
        $SUDO apt-get update && $SUDO apt-get install -y git
    elif [ -f /etc/redhat-release ] || [ -f /etc/yum.conf ]; then
        echo "Installing git via yum..."
        $SUDO yum install -y git
    elif command -v dnf >/dev/null 2>&1; then
        echo "Installing git via dnf..."
        $SUDO dnf install -y git
    elif command -v pacman >/dev/null 2>&1; then
        echo "Installing git via pacman..."
        $SUDO pacman -S --noconfirm git
    else
        echo "Could not auto-install git. Please install from: https://git-scm.com"
        exit 1
    fi
}

if ! command -v git >/dev/null 2>&1; then
    install_git
fi

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
$SUDO mkdir -p "${INSTALL_DIR}"

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
$SUDO rm -f "${INSTALL_DIR}/${BINARY_NAME}"
$SUDO mv "${FILENAME}" "${INSTALL_DIR}/${BINARY_NAME}"
$SUDO chmod +x "${INSTALL_DIR}/${BINARY_NAME}"

# Cleanup
rm -f "${TMPDIR}/${FILENAME}"

echo ""
echo "Installed: ${INSTALL_DIR}/${BINARY_NAME}"
echo ""
echo "Next steps:"
echo "  1. Start server:  ep serve"
echo "  2. Open browser:  open http://127.0.0.1:8080"
echo ""
echo "Run 'ep --version' to verify."
