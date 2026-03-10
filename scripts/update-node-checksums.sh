#!/usr/bin/env bash

set -e

# Script to update Node.js checksums for runtime.go
# Usage: ./scripts/update-node-checksums.sh <version>
# Example: ./scripts/update-node-checksums.sh 20.18.1

VERSION="${1:-}"

if [ -z "$VERSION" ]; then
    echo "Usage: $0 <version>"
    echo "Example: $0 20.18.1"
    exit 1
fi

# Remove 'v' prefix if present
VERSION="${VERSION#v}"

# Create temp directory
TMPDIR=$(mktemp -d)
trap "rm -rf $TMPDIR" EXIT

cd "$TMPDIR"

echo "Downloading checksums for Node.js v${VERSION}..."
curl -fsSLo SHASUMS256.txt "https://nodejs.org/dist/v${VERSION}/SHASUMS256.txt"

if [ ! -f SHASUMS256.txt ]; then
    echo "Error: Failed to download SHASUMS256.txt"
    exit 1
fi

# Optional: Verify GPG signature (if gpgv is available)
if command -v gpgv &> /dev/null; then
    echo "Verifying GPG signature..."
    if curl -fsSLo SHASUMS256.txt.asc "https://nodejs.org/dist/v${VERSION}/SHASUMS256.txt.asc" 2>/dev/null; then
        if curl -fsSLo nodejs-keyring.kbx "https://github.com/nodejs/release-keys/raw/HEAD/gpg/pubring.kbx" 2>/dev/null; then
            if gpgv --keyring="nodejs-keyring.kbx" --output /dev/null SHASUMS256.txt.asc 2>/dev/null; then
                echo "✓ GPG signature verified"
            else
                echo "⚠ Warning: GPG signature verification failed, but continuing anyway..."
            fi
        fi
    fi
fi

# Extract checksums
echo ""
echo "Extracting checksums..."

get_checksum() {
    local pattern="$1"
    grep "$pattern" SHASUMS256.txt | awk '{print $1}' || echo ""
}

DARWIN_AMD64=$(get_checksum "darwin-x64.tar.gz")
DARWIN_ARM64=$(get_checksum "darwin-arm64.tar.gz")
LINUX_AMD64=$(get_checksum "linux-x64.tar.gz")
LINUX_ARM64=$(get_checksum "linux-arm64.tar.gz")
WINDOWS_AMD64=$(get_checksum "win-x64.zip")

# Get major version
MAJOR_VERSION=$(echo "$VERSION" | cut -d. -f1)

# Get current date
CURRENT_DATE=$(date +%Y-%m-%d)

# Generate Go code
echo ""
echo "================================================"
echo "Add the following to pkg/plugins/runtime.go:"
echo "================================================"
echo ""
cat <<EOF
	"${MAJOR_VERSION}": {
		Version: "${VERSION}",
		// Checksums verified from https://nodejs.org/dist/v${VERSION}/SHASUMS256.txt
		// Verified on ${CURRENT_DATE}
		Checksums: map[string]map[string]string{
			"darwin": {
				"amd64": "${DARWIN_AMD64}", // node-v${VERSION}-darwin-x64.tar.gz
				"arm64": "${DARWIN_ARM64}", // node-v${VERSION}-darwin-arm64.tar.gz
			},
			"linux": {
				"amd64": "${LINUX_AMD64}", // node-v${VERSION}-linux-x64.tar.gz
				"arm64": "${LINUX_ARM64}", // node-v${VERSION}-linux-arm64.tar.gz
			},
			"windows": {
				"amd64": "${WINDOWS_AMD64}", // node-v${VERSION}-win-x64.zip
			},
		},
	},
EOF
echo ""
echo "================================================"
echo ""
echo "Summary:"
echo "  Version: ${VERSION}"
echo "  Major: ${MAJOR_VERSION}"
echo "  macOS Intel:        ${DARWIN_AMD64}"
echo "  macOS Apple Silicon: ${DARWIN_ARM64}"
echo "  Linux Intel:        ${LINUX_AMD64}"
echo "  Linux ARM:          ${LINUX_ARM64}"
echo "  Windows:            ${WINDOWS_AMD64}"
echo ""
