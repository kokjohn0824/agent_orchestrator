#!/usr/bin/env sh
# Install agent-orchestrator from GitHub Releases (macOS and Linux).
# Usage: curl -fsSL https://raw.githubusercontent.com/OWNER/agent_orchestrator/main/scripts/install.sh | sh
# Override repo: AGENT_ORCHESTRATOR_REPO=owner/repo curl -fsSL ... | sh
# Override install dir: PREFIX=/usr/local curl -fsSL ... | sh  (installs to /usr/local/bin)

set -e

REPO="${AGENT_ORCHESTRATOR_REPO:-kokjohn0824/agent_orchestrator}"
BINARY="agent-orchestrator"
API_LATEST="https://api.github.com/repos/${REPO}/releases/latest"

# Detect OS and arch (darwin/linux × amd64/arm64)
OS=$(uname -s)
ARCH=$(uname -m)

case "$OS" in
  Darwin)  OS="darwin" ;;
  Linux)   OS="linux" ;;
  *)
    echo "Unsupported OS: $OS. Supported: darwin (macOS), linux."
    exit 1
    ;;
esac

case "$ARCH" in
  x86_64|amd64) ARCH="amd64" ;;
  aarch64|arm64) ARCH="arm64" ;;
  *)
    echo "Unsupported arch: $ARCH. Supported: amd64, arm64."
    exit 1
    ;;
esac

ASSET="${BINARY}-${OS}-${ARCH}"
# Resolve latest release tag via GitHub API (does not rely on /releases/latest redirect)
TAG=$(curl -sSfL "${API_LATEST}" | grep '"tag_name":' | sed -E 's/.*"tag_name"[[:space:]]*:[[:space:]]*"([^"]+)".*/\1/')
if [ -z "$TAG" ]; then
  echo "Could not find latest release. Check that a release exists: https://github.com/${REPO}/releases"
  exit 1
fi
DOWNLOAD_URL="https://github.com/${REPO}/releases/download/${TAG}/${ASSET}"

if [ -n "$PREFIX" ]; then
  INSTALL_DIR="${PREFIX}/bin"
else
  INSTALL_DIR="${HOME}/bin"
fi
INSTALL_PATH="${INSTALL_DIR}/${BINARY}"

echo "Installing ${BINARY} (${OS}-${ARCH}) to ${INSTALL_PATH} ..."

mkdir -p "$INSTALL_DIR"
if ! curl -fSL -o "$INSTALL_PATH" "$DOWNLOAD_URL"; then
  echo "Download failed. Check that release ${TAG} and asset ${ASSET} exist: https://github.com/${REPO}/releases"
  exit 1
fi
chmod +x "$INSTALL_PATH"

echo "Installed: $INSTALL_PATH"
echo "Ensure ${INSTALL_DIR} is in your PATH."
