#!/bin/sh
set -euo pipefail

INSTALL_DIR="${STRIPE_INSTALL_DIR:-$HOME/.stripe/bin}"
GITHUB_REPO="stripe/stripe-cli"

main() {
  detect_platform
  get_latest_version
  download_and_verify
  install_binary
  setup_path
  print_success
}

detect_platform() {
  OS=$(uname -s | tr '[:upper:]' '[:lower:]')
  ARCH=$(uname -m)

  case "$OS" in
    darwin) OS_LABEL="mac-os" ;;
    linux)  OS_LABEL="linux" ;;
    *)
      echo "Error: unsupported operating system: $OS"
      echo "Supported: macOS, Linux."
      exit 1
      ;;
  esac

  case "$ARCH" in
    x86_64|amd64) ARCH_LABEL="x86_64" ;;
    arm64|aarch64) ARCH_LABEL="arm64" ;;
    *)
      echo "Error: unsupported architecture: $ARCH"
      echo "Supported: x86_64, arm64."
      exit 1
      ;;
  esac

  case "$OS" in
    darwin) CHECKSUMS_FILE="stripe-mac-checksums.txt" ;;
    linux)  CHECKSUMS_FILE="stripe-linux-checksums.txt" ;;
  esac

  echo "Detected: $OS $ARCH_LABEL"
}

http_get() {
  url="$1"
  if command -v curl >/dev/null 2>&1; then
    curl -sSL "$url"
  elif command -v wget >/dev/null 2>&1; then
    wget -qO- "$url"
  else
    echo "Error: curl or wget is required but neither is installed."
    exit 1
  fi
}

http_download() {
  url="$1"
  output="$2"
  if command -v curl >/dev/null 2>&1; then
    curl -sSL -o "$output" "$url"
  elif command -v wget >/dev/null 2>&1; then
    wget -q -O "$output" "$url"
  else
    echo "Error: curl or wget is required but neither is installed."
    exit 1
  fi
}

get_latest_version() {
  VERSION=$(http_get "https://api.github.com/repos/${GITHUB_REPO}/releases/latest" | \
    sed -n 's/.*"tag_name": *"v\([^"]*\)".*/\1/p')

  if [ -z "$VERSION" ]; then
    echo "Error: could not determine latest version."
    echo "Check your internet connection or try again later."
    exit 1
  fi

  echo "Latest version: v$VERSION"
}

download_and_verify() {
  ARCHIVE="stripe_${VERSION}_${OS_LABEL}_${ARCH_LABEL}.tar.gz"
  BASE_URL="https://github.com/${GITHUB_REPO}/releases/download/v${VERSION}"

  TMP_DIR=$(mktemp -d)
  trap 'rm -rf "$TMP_DIR"' EXIT

  echo "Downloading stripe v${VERSION}..."
  http_download "$BASE_URL/$ARCHIVE" "$TMP_DIR/$ARCHIVE"
  http_download "$BASE_URL/$CHECKSUMS_FILE" "$TMP_DIR/checksums.txt"

  echo "Verifying checksum..."
  EXPECTED=$(sed -n "s/^\([a-f0-9]*\)  *${ARCHIVE}$/\1/p" "$TMP_DIR/checksums.txt")

  if [ -z "$EXPECTED" ]; then
    echo "Error: checksum entry not found for $ARCHIVE"
    exit 1
  fi

  if command -v sha256sum >/dev/null 2>&1; then
    ACTUAL=$(sha256sum "$TMP_DIR/$ARCHIVE" | cut -d' ' -f1)
  elif command -v shasum >/dev/null 2>&1; then
    ACTUAL=$(shasum -a 256 "$TMP_DIR/$ARCHIVE" | cut -d' ' -f1)
  else
    echo "Warning: no sha256sum or shasum found, skipping verification."
    ACTUAL="$EXPECTED"
  fi

  if [ "$ACTUAL" != "$EXPECTED" ]; then
    echo "Error: checksum verification failed."
    echo "  Expected: $EXPECTED"
    echo "  Actual:   $ACTUAL"
    echo "The downloaded file may be corrupted. Please try again."
    exit 1
  fi

  echo "Checksum verified."

  tar -xzf "$TMP_DIR/$ARCHIVE" -C "$TMP_DIR"
}

install_binary() {
  mkdir -p "$INSTALL_DIR"
  mv "$TMP_DIR/stripe" "$INSTALL_DIR/stripe"
  chmod +x "$INSTALL_DIR/stripe"

  # Check for existing brew install and warn
  if command -v brew >/dev/null 2>&1; then
    BREW_STRIPE=$(brew --prefix 2>/dev/null)/bin/stripe
    if [ -f "$BREW_STRIPE" ]; then
      echo ""
      echo "Note: stripe is also installed via Homebrew at $BREW_STRIPE"
      echo "You may want to run 'brew uninstall stripe' to avoid confusion."
    fi
  fi
}

setup_path() {
  case ":$PATH:" in
    *":$INSTALL_DIR:"*) return ;;
  esac

  SHELL_NAME=$(basename "$SHELL")
  case "$SHELL_NAME" in
    zsh)  PROFILE="$HOME/.zshrc" ;;
    bash)
      if [ -f "$HOME/.bashrc" ]; then
        PROFILE="$HOME/.bashrc"
      else
        PROFILE="$HOME/.bash_profile"
      fi
      ;;
    fish) PROFILE="$HOME/.config/fish/config.fish" ;;
    *)    PROFILE="$HOME/.profile" ;;
  esac

  EXPORT_LINE="export PATH=\"$INSTALL_DIR:\$PATH\""
  if [ "$SHELL_NAME" = "fish" ]; then
    EXPORT_LINE="set -gx PATH $INSTALL_DIR \$PATH"
  fi

  if [ -f "$PROFILE" ] && grep -qF "$INSTALL_DIR" "$PROFILE" 2>/dev/null; then
    return
  fi

  echo "" >> "$PROFILE"
  echo "$EXPORT_LINE" >> "$PROFILE"
  echo "Added $INSTALL_DIR to PATH in $PROFILE"
  NEEDS_SOURCE=true
}

print_success() {
  echo ""
  echo "stripe v${VERSION} installed to $INSTALL_DIR/stripe"
  echo ""
  if [ "$NEEDS_SOURCE" = "true" ]; then
    echo "Run 'source $PROFILE' or open a new terminal, then:"
  fi
  echo "  stripe login    — authenticate with your Stripe account"
  echo "  stripe --help   — see available commands"
}

main
