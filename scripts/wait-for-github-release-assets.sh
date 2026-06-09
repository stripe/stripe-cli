#!/usr/bin/env bash

set -euo pipefail

if [ "$#" -lt 3 ]; then
  echo "Usage: $0 <repo> <tag> <asset> [<asset> ...]"
  exit 1
fi

repo="$1"
tag="$2"
shift 2

attempts="${RELEASE_ASSET_CHECK_ATTEMPTS:-20}"
sleep_seconds="${RELEASE_ASSET_CHECK_INTERVAL_SECONDS:-15}"

for asset in "$@"; do
  url="https://github.com/${repo}/releases/download/${tag}/${asset}"
  echo "Waiting for public release asset: ${url}"

  available=false
  attempt=1
  while [ "$attempt" -le "$attempts" ]; do
    status_code="$(curl -sSI -o /dev/null -w "%{http_code}" "$url" || true)"
    if [ "$status_code" = "200" ] || [ "$status_code" = "302" ]; then
      echo "Asset is reachable: ${asset}"
      available=true
      break
    fi

    echo "Attempt ${attempt}/${attempts} returned HTTP ${status_code:-unknown} for ${asset}"
    if [ "$attempt" -lt "$attempts" ]; then
      sleep "$sleep_seconds"
    fi
    attempt=$((attempt + 1))
  done

  if [ "$available" != "true" ]; then
    echo "Timed out waiting for public release asset: ${asset}"
    exit 1
  fi
done
