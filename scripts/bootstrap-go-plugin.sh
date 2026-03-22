#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
DEST_DIR="${ROOT_DIR}/third_party/go-plugin"
PATCH_FILE="${ROOT_DIR}/patches/go-plugin-v1.7.0-buffer.patch"
STATE_FILE="${DEST_DIR}/.stripe-bootstrap-state"
UPSTREAM_REPO="https://github.com/hashicorp/go-plugin"
UPSTREAM_VERSION="v1.7.0"

hash_file() {
  if command -v shasum >/dev/null 2>&1; then
    shasum -a 256 "$1" | awk '{print $1}'
    return
  fi

  sha256sum "$1" | awk '{print $1}'
}

expected_state="${UPSTREAM_VERSION}:$(hash_file "${PATCH_FILE}")"

if [[ -f "${STATE_FILE}" ]] && [[ "$(<"${STATE_FILE}")" == "${expected_state}" ]]; then
  exit 0
fi

tmp_dir="$(mktemp -d)"
trap 'rm -rf "${tmp_dir}"' EXIT

git -c advice.detachedHead=false clone --quiet --depth 1 --branch "${UPSTREAM_VERSION}" "${UPSTREAM_REPO}" "${tmp_dir}/go-plugin"
git -C "${tmp_dir}/go-plugin" apply "${PATCH_FILE}"
rm -rf "${tmp_dir}/go-plugin/.git"

mkdir -p "${ROOT_DIR}/third_party"
rm -rf "${DEST_DIR}"
mv "${tmp_dir}/go-plugin" "${DEST_DIR}"
printf '%s\n' "${expected_state}" > "${STATE_FILE}"
