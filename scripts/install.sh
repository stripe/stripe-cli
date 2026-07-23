#!/bin/sh
set -eu

INSTALL_DIR="${STRIPE_INSTALL_DIR:-$HOME/.stripe/bin}"
GITHUB_REPO="stripe/stripe-cli"
NEEDS_SOURCE=false
TELEMETRY_URL="${STRIPE_TELEMETRY_URL:-https://r.stripe.com/0}"
INSTALL_SUCCESS=false

main() {
  detect_platform
  get_latest_version
  download_and_verify
  install_binary
  setup_path
  INSTALL_SUCCESS=true
  send_telemetry "Install Succeeded" "version=$VERSION"
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
    if [ -n "${GITHUB_TOKEN:-}" ]; then
      curl -sSL -H "Authorization: Bearer $GITHUB_TOKEN" "$url"
    else
      curl -sSL "$url"
    fi
  elif command -v wget >/dev/null 2>&1; then
    if [ -n "${GITHUB_TOKEN:-}" ]; then
      wget -qO- --header="Authorization: Bearer $GITHUB_TOKEN" "$url"
    else
      wget -qO- "$url"
    fi
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
  LATEST_VERSION=$(http_get "https://api.github.com/repos/${GITHUB_REPO}/releases/latest" | \
    sed -n 's/.*"tag_name": *"v\([^"]*\)".*/\1/p')

  if [ -z "$LATEST_VERSION" ]; then
    echo "Error: could not determine latest version."
    echo "Check your internet connection or try again later."
    exit 1
  fi

  if [ -n "${VERSION:-}" ]; then
    VERSION=$(echo "$VERSION" | sed 's/^v//')
    echo "Installing requested version: v$VERSION (latest is v$LATEST_VERSION)"
  else
    VERSION="$LATEST_VERSION"
    echo "Latest version: v$VERSION"
  fi
}

download_and_verify() {
  ARCHIVE="stripe_${VERSION}_${OS_LABEL}_${ARCH_LABEL}.tar.gz"
  BASE_URL="https://github.com/${GITHUB_REPO}/releases/download/v${VERSION}"

  TMP_DIR=$(mktemp -d)
  trap 'rm -rf "$TMP_DIR"; if [ "$INSTALL_SUCCESS" = "false" ]; then send_telemetry "Install Failed" "version=${VERSION:-unknown}"; fi' EXIT

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

version_lt() {
  # Returns 0 (true) if $1 < $2 using sort -V for version comparison
  [ "$1" != "$2" ] && [ "$(printf '%s\n%s' "$1" "$2" | sort -V | head -n1)" = "$1" ]
}

send_telemetry() {
  event_name="$1"
  event_value="$2"

  case "${STRIPE_CLI_TELEMETRY_OPTOUT:-}${DO_NOT_TRACK:-}" in
    *1*|*true*|*TRUE*) return ;;
  esac

  telemetry_data="client_id=stripe-cli&event_name=${event_name}&event_value=${event_value}&os=${OS:-unknown}&arch=${ARCH_LABEL:-unknown}&cli_version=${VERSION:-unknown}&install_method=curl"

  if command -v curl >/dev/null 2>&1; then
    curl -sS --max-time 3 -X POST -H "origin: stripe-cli" -H "Content-Type: application/x-www-form-urlencoded" -d "$telemetry_data" "$TELEMETRY_URL" >/dev/null 2>&1 || true
  elif command -v wget >/dev/null 2>&1; then
    wget -q --timeout=3 -O /dev/null --post-data="$telemetry_data" --header="origin: stripe-cli" --header="Content-Type: application/x-www-form-urlencoded" "$TELEMETRY_URL" 2>/dev/null || true
  fi
}

print_success() {
  echo ""
  echo "stripe v${VERSION} installed to $INSTALL_DIR/stripe"
  echo ""
  if [ "$VERSION" != "$LATEST_VERSION" ] && version_lt "$VERSION" "$LATEST_VERSION"; then
    echo "Note: You installed v${VERSION}, but the latest is v${LATEST_VERSION}."
    echo "Auto-update will upgrade you to the latest on next run."
    echo "To stay on this version, set STRIPE_NO_AUTO_UPDATE=1 or add to ~/.config/stripe/config.toml:"
    echo "  [settings]"
    echo "  auto_update = false"
    echo ""
  fi
  if [ "$NEEDS_SOURCE" = "true" ]; then
    echo "Run 'source $PROFILE' or open a new terminal, then:"
  fi
  echo "  stripe login    — authenticate with your Stripe account"
  echo "  stripe --help   — see available commands"
}

main
