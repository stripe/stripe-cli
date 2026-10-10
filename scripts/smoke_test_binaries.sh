#!/usr/bin/env bash
#
# smoke_test_binaries.sh — validate that built plugin binaries launch as valid
# HashiCorp go-plugin servers before they are published to the artifact store.
#
# Usage:
#   smoke_test_binaries.sh <bin_dir> [<plugin_file>]
#
#   <bin_dir>      Directory containing the freshly built plugin binaries.
#   <plugin_file>  Path to the repo's ".plugin" file. Defaults to "./.plugin".
#
# Why this exists:
#   Plugin binaries are HashiCorp go-plugin servers. In production mode they
#   perform a handshake: on launch they read a magic-cookie environment variable
#   and, if it matches, print a single handshake line to stdout of the form
#
#       <core>|<app>|<network>|<address>|<protocol>
#
#   (e.g. "1|3|unix|/tmp/plugin1234|grpc") and then serve over RPC. If the magic
#   cookie is missing or wrong, the binary refuses to run and exits non-zero.
#
#   The host CLI derives the handshake env var name as "plugin_<shortname>" and
#   the value from the ".plugin" file. This script reproduces that handshake to
#   prove each built binary is a valid, launchable plugin — a fast smoke test
#   that does not require a full install flow.
#
# The ".plugin" file format is a single line: "<shortname> <magic-cookie-value>".
#
# For each binary in <bin_dir>, this script:
#   1. Skips binaries built for a different OS than the current runner (a Linux
#      runner cannot exec a macOS binary and vice versa). A build matrix is
#      expected to cover every OS on its own leg.
#   2. Ensures the binary is executable (chmod +x, and clears the macOS
#      quarantine attribute if present).
#   3. Launches it with the handshake env var set, captures stdout with a
#      timeout, and checks for the go-plugin handshake line.
#   4. Terminates the process once the handshake is seen.
#
# It prints a per-binary PASS/FAIL summary and exits non-zero if any binary
# fails, so a release workflow can gate publishing on it.

set -euo pipefail

# Seconds to wait for a binary to emit the handshake line before failing it.
HANDSHAKE_TIMEOUT_SECONDS="${HANDSHAKE_TIMEOUT_SECONDS:-10}"

usage() {
  echo "Usage: $0 <bin_dir> [<plugin_file>]" >&2
}

if [ "$#" -lt 1 ] || [ "$#" -gt 2 ]; then
  usage
  exit 2
fi

bin_dir="$1"
plugin_file="${2:-.plugin}"

if [ ! -d "$bin_dir" ]; then
  echo "error: bin_dir '$bin_dir' is not a directory" >&2
  exit 2
fi

if [ ! -f "$plugin_file" ]; then
  echo "error: plugin file '$plugin_file' not found" >&2
  exit 2
fi

# Parse "<shortname> <magic-cookie-value>" from the first non-empty, non-comment
# line of the .plugin file.
plugin_line="$(grep -v -e '^[[:space:]]*#' -e '^[[:space:]]*$' "$plugin_file" | head -n 1 || true)"
shortname="$(printf '%s\n' "$plugin_line" | awk '{print $1}')"
magic_cookie="$(printf '%s\n' "$plugin_line" | awk '{print $2}')"

if [ -z "$shortname" ] || [ -z "$magic_cookie" ]; then
  echo "error: could not parse '<shortname> <magic-cookie>' from '$plugin_file'" >&2
  exit 2
fi

# The host CLI's MagicCookieKey is "plugin_<shortname>".
handshake_key="plugin_${shortname}"

current_os="$(uname -s)"

echo "Smoke-testing plugin '${shortname}' binaries in '${bin_dir}' (runner OS: ${current_os})"

# Determine the OS a binary was built for using the `file` tool. Returns one of
# "Linux", "Darwin", or "" (unknown / not a native executable, e.g. a script).
binary_os() {
  local bin="$1"
  local desc
  desc="$(file -b "$bin" 2>/dev/null || true)"
  case "$desc" in
    *ELF*) echo "Linux" ;;
    *Mach-O*) echo "Darwin" ;;
    *) echo "" ;;
  esac
}

# Ensure the binary can be executed on this runner.
prepare_binary() {
  local bin="$1"
  if [ ! -x "$bin" ]; then
    chmod +x "$bin" 2>/dev/null || true
  fi
  if [ "$current_os" = "Darwin" ]; then
    # Clear the quarantine attribute that macOS sets on downloaded artifacts.
    xattr -d com.apple.quarantine "$bin" 2>/dev/null || true
  fi
}

# Launch a single binary and confirm it emits the go-plugin handshake line.
# Returns 0 on success, 1 on failure.
smoke_test_one() {
  local bin="$1"
  local out
  out="$(mktemp)"

  # Launch with the handshake env var set so the plugin starts its RPC server.
  env "${handshake_key}=${magic_cookie}" "$bin" >"$out" 2>/dev/null &
  local pid=$!

  local found=1
  local iters=$((HANDSHAKE_TIMEOUT_SECONDS * 5)) # poll every 0.2s
  local i=0
  while [ "$i" -lt "$iters" ]; do
    # go-plugin handshake line begins with "<core>|<app>|", both integers.
    if grep -Eq '^[0-9]+\|[0-9]+\|' "$out" 2>/dev/null; then
      found=0
      break
    fi
    # Bail out early if the process died before printing the handshake.
    if ! kill -0 "$pid" 2>/dev/null; then
      break
    fi
    sleep 0.2
    i=$((i + 1))
  done

  # Reap the plugin process; it would otherwise sit and serve indefinitely.
  kill "$pid" 2>/dev/null || true
  wait "$pid" 2>/dev/null || true

  if [ "$found" -eq 0 ]; then
    echo "PASS: $bin"
    rm -f "$out"
    return 0
  fi

  echo "FAIL: $bin (no go-plugin handshake within ${HANDSHAKE_TIMEOUT_SECONDS}s)"
  if [ -s "$out" ]; then
    echo "----- captured output -----"
    sed 's/^/  /' "$out"
    echo "---------------------------"
  fi
  rm -f "$out"
  return 1
}

total=0
tested=0
skipped=0
failed=0

# Iterate over regular files directly under bin_dir (non-recursive).
for bin in "$bin_dir"/*; do
  [ -f "$bin" ] || continue
  total=$((total + 1))

  bos="$(binary_os "$bin")"
  if [ -n "$bos" ] && [ "$bos" != "$current_os" ]; then
    echo "SKIP: $bin (built for ${bos}, runner is ${current_os})"
    skipped=$((skipped + 1))
    continue
  fi

  prepare_binary "$bin"
  tested=$((tested + 1))
  if ! smoke_test_one "$bin"; then
    failed=$((failed + 1))
  fi
done

echo ""
echo "Summary: ${total} found, ${tested} tested, ${skipped} skipped, ${failed} failed"

if [ "$total" -eq 0 ]; then
  echo "error: no binaries found in '$bin_dir'" >&2
  exit 1
fi

if [ "$failed" -gt 0 ]; then
  exit 1
fi

if [ "$tested" -eq 0 ]; then
  echo "warning: all binaries were skipped on this runner (${current_os}); another matrix leg is expected to test them"
fi

exit 0
