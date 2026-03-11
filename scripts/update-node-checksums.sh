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

# Store the original working directory and runtime file path
ORIGINAL_DIR="$(pwd)"
RUNTIME_FILE="${ORIGINAL_DIR}/pkg/plugins/runtime.go"

# Check if runtime.go exists
if [ ! -f "$RUNTIME_FILE" ]; then
    echo "Error: Could not find runtime.go at $RUNTIME_FILE"
    echo "Make sure you run this from the project root: make update-node-checksums VERSION=X.Y.Z"
    exit 1
fi

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

# Write the new Go code entry to a temporary file
NEW_ENTRY_FILE=$(mktemp)
trap "rm -f $NEW_ENTRY_FILE" EXIT

cat > "$NEW_ENTRY_FILE" <<EOF
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
echo "Updating ${RUNTIME_FILE}..."

# Check if the major version already exists
if grep -q "\"${MAJOR_VERSION}\":" "$RUNTIME_FILE"; then
    echo "Version ${MAJOR_VERSION} already exists in runtime.go"
    echo "Replacing existing entry..."

    # Use awk to replace the existing entry
    awk -v major="$MAJOR_VERSION" -v newfile="$NEW_ENTRY_FILE" '
    BEGIN { in_block=0; skip=0 }
    {
        if ($0 ~ "\"" major "\":") {
            in_block=1
            skip=1
            # Read and print the new entry from file
            while ((getline line < newfile) > 0) {
                print line
            }
            close(newfile)
            next
        }
        if (in_block && $0 ~ /^\t},/) {
            in_block=0
            skip=0
            next
        }
        if (!skip) {
            print $0
        }
    }
    ' "$RUNTIME_FILE" > "${RUNTIME_FILE}.tmp"

    mv "${RUNTIME_FILE}.tmp" "$RUNTIME_FILE"
else
    echo "Adding new version ${MAJOR_VERSION} to runtime.go..."

    # Insert the new entry before the closing brace of the map
    # Find the line with just "}" that closes the nodeRuntimeConfigs map
    awk -v newfile="$NEW_ENTRY_FILE" '
    /^}$/ && prev_line ~ /^\t},/ {
        # Read and print the new entry from file
        while ((getline line < newfile) > 0) {
            print line
        }
        close(newfile)
    }
    { print; prev_line=$0 }
    ' "$RUNTIME_FILE" > "${RUNTIME_FILE}.tmp"

    mv "${RUNTIME_FILE}.tmp" "$RUNTIME_FILE"
fi

echo "✓ Successfully updated ${RUNTIME_FILE}"
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
echo "Next steps:"
echo "  1. Review the changes: git diff ${RUNTIME_FILE}"
echo "  2. Test the changes: make build"
echo "  3. Commit the changes: git add ${RUNTIME_FILE} && git commit"
echo ""
