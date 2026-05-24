#!/usr/bin/env bash
set -euo pipefail

# MIST Server one-line installer
# Usage: curl -fsSL https://mist.viloris.org/install-server.sh | bash

DOWNLOAD_BASE="${DOWNLOAD_BASE:-https://mist.viloris.org}"
BIN_NAME="mist-server"
INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
NC='\033[0m'

msg() { echo -e "${GREEN}[mist-server]${NC} $*"; }
warn() { echo -e "${YELLOW}[mist-server]${NC} $*"; }
err() { echo -e "${RED}[mist-server]${NC} $*"; exit 1; }

detect_platform() {
	case "$(uname -s)" in
		Linux) ;;
		*) err "Unsupported OS: $(uname -s). Server only supports Linux." ;;
	esac

	case "$(uname -m)" in
		x86_64|amd64) echo "linux-amd64" ;;
		aarch64|arm64) echo "linux-arm64" ;;
		*) err "Unsupported architecture: $(uname -m)" ;;
	esac
}

PLATFORM=$(detect_platform)
DOWNLOAD_URL="${DOWNLOAD_BASE}/${BIN_NAME}-${PLATFORM}"
INSTALL_PATH="${INSTALL_DIR}/${BIN_NAME}"

msg "Detected platform: ${PLATFORM}"
msg "Downloading ${BIN_NAME}..."

if command -v curl &>/dev/null; then
	curl -fsSL "${DOWNLOAD_URL}" -o "/tmp/${BIN_NAME}"
elif command -v wget &>/dev/null; then
	wget -q "${DOWNLOAD_URL}" -O "/tmp/${BIN_NAME}"
else
	err "Neither curl nor wget found. Please install one of them."
fi

chmod +x "/tmp/${BIN_NAME}"

if [ "$(id -u)" -eq 0 ]; then
	mv "/tmp/${BIN_NAME}" "${INSTALL_PATH}"
else
	sudo mv "/tmp/${BIN_NAME}" "${INSTALL_PATH}"
fi

msg "Installed to ${INSTALL_PATH}"

echo ""
msg "To start the server:"
echo "  ${BIN_NAME} -l 0.0.0.0:8443 -p <password> -cert-type self-signed"
echo ""
msg "For a full interactive setup with systemd:"
echo "  curl -fsSL https://mist.viloris.org/install.sh | bash"
