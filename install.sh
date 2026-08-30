#!/usr/bin/env sh
# Install the latest knap binary into ~/.local/bin (or $KNAP_INSTALL_DIR).
set -e

REPO="mvanduijker/knap-cli"
INSTALL_DIR="${KNAP_INSTALL_DIR:-$HOME/.local/bin}"

case "$(uname -s)" in
    Darwin) OS="darwin" ;;
    Linux) OS="linux" ;;
    *) echo "Unsupported OS: $(uname -s)" >&2; exit 1 ;;
esac

case "$(uname -m)" in
    x86_64 | amd64) ARCH="amd64" ;;
    arm64 | aarch64) ARCH="arm64" ;;
    *) echo "Unsupported architecture: $(uname -m)" >&2; exit 1 ;;
esac

VERSION="${KNAP_VERSION:-$(curl -fsSL "https://api.github.com/repos/$REPO/releases/latest" | sed -n 's/.*"tag_name": *"\([^"]*\)".*/\1/p')}"
if [ -z "$VERSION" ]; then
    echo "Could not determine the latest release." >&2
    exit 1
fi

URL="https://github.com/$REPO/releases/download/$VERSION/knap_${OS}_${ARCH}.tar.gz"
TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"' EXIT

echo "Downloading knap $VERSION for $OS/$ARCH..."
curl -fsSL "$URL" | tar -xz -C "$TMP"

mkdir -p "$INSTALL_DIR"
mv "$TMP/knap" "$INSTALL_DIR/knap"
chmod +x "$INSTALL_DIR/knap"

echo "Installed knap to $INSTALL_DIR/knap"
case ":$PATH:" in
    *":$INSTALL_DIR:"*) ;;
    *) echo "Add $INSTALL_DIR to your PATH to use it." ;;
esac
