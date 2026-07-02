#!/bin/bash

set -e

if ! [ -x "$(command -v jq)" ]; then
  echo "jq is not installed." >&2
  exit 1
fi

if ! [ -x "$(command -v unzip)" ]; then
  echo "unzip is not installed." >&2
  exit 1
fi

if [ -z "$GITHUB_TOKEN" ]; then
  echo "GITHUB_TOKEN is not set" >&2
  exit 1
fi

if [ -z "$VIRUSTOTAL_API_KEY" ]; then
  echo "VIRUSTOTAL_API_KEY is not set" >&2
  exit 1
fi

VIRUSTOTAL_DIRECT_UPLOAD_LIMIT_BYTES=$((32 * 1024 * 1024))
VIRUSTOTAL_V2_SCAN_URL="https://www.virustotal.com/vtapi/v2/file/scan"
VIRUSTOTAL_V2_COMMENT_URL="https://www.virustotal.com/vtapi/v2/comments/put"
VIRUSTOTAL_V3_LARGE_FILE_UPLOAD_URL="https://www.virustotal.com/api/v3/files/upload_url"

getResponseBody() {
  printf '%s' "$1" | sed -e 's/HTTPSTATUS\:.*//g'
}

getResponseCode() {
  printf '%s' "$1" | tr -d '\n' | sed -e 's/.*HTTPSTATUS://'
}

getLargeFileUploadURL() {
  local upload_url
  local upload_url_body
  local upload_url_response
  local upload_url_response_code

  upload_url_response=$(curl -s -H "x-apikey: $VIRUSTOTAL_API_KEY" "$VIRUSTOTAL_V3_LARGE_FILE_UPLOAD_URL" -w "HTTPSTATUS:%{http_code}")
  upload_url_body=$(getResponseBody "$upload_url_response")
  upload_url_response_code=$(getResponseCode "$upload_url_response")

  if [ "$upload_url_response_code" = "403" ]; then
    echo "VirusTotal API key is not allowed to request large-file upload URLs; skipping this oversized file." >&2
    return 3
  fi

  if [[ "$(echo "$upload_url_response_code" | head -c2)" != "20" ]]; then
    echo "Unable to get large-file upload URL, HTTP response code: $upload_url_response_code" >&2
    return 1
  fi

  upload_url=$(echo "$upload_url_body" | jq -er ".data" 2>/dev/null || true)
  if [ -z "$upload_url" ] || [ "$upload_url" = "null" ]; then
    echo "VirusTotal did not return a large-file upload URL" >&2
    return 1
  fi

  printf '%s' "$upload_url"
}

calculateSHA256() {
  if [ -x "$(command -v sha256sum)" ]; then
    sha256sum "$1" | awk '{print $1}'
    return
  fi

  if [ -x "$(command -v shasum)" ]; then
    shasum -a 256 "$1" | awk '{print $1}'
    return
  fi

  echo "Neither sha256sum nor shasum is installed." >&2
  exit 1
}

setVersion () {
  VERSION=$(curl -s "https://api.github.com/repos/stripe/stripe-cli/releases/latest" -u "${GITHUB_TOKEN}:" \
| jq -r ".tag_name")
}

downloadWindowsArtifacts() {
  echo "Dowloading Windows artifacts..."
  FILES=$(curl -s "https://api.github.com/repos/stripe/stripe-cli/releases/latest" -u "${GITHUB_TOKEN}:" \
| jq -r ".assets[].browser_download_url" \
| grep "windows" \
| grep "zip")
  echo "$FILES"
  for i in $FILES; do
    RESPONSE_CODE=$(curl -L -O -w "%{response_code}" "$i")
    echo "$RESPONSE_CODE"
    code=$(echo "$RESPONSE_CODE" | head -c2)
    if [ "$code" != "20" ] && [ "$code" != "30" ]; then
      echo "Unable to download $i HTTP response code: $RESPONSE_CODE"
      exit 1
    fi
  done;
  echo "Finished downloading"
}

virustotalUpload () {
  for i in $FILES; do
    local executable_size_bytes
    local large_file_upload_url_status
    local upload_url
    local sha256
    local response
    local response_code
    FILENAME=${i##*/}

    echo "Uncompressing archive..."

    unzip -o "$FILENAME"

    echo "Uploading to VirusTotal..."

    executable_size_bytes=$(wc -c < ./stripe.exe | tr -d '[:space:]')
    upload_url="$VIRUSTOTAL_V2_SCAN_URL"
    sha256=$(calculateSHA256 ./stripe.exe)

    if [ "$executable_size_bytes" -gt "$VIRUSTOTAL_DIRECT_UPLOAD_LIMIT_BYTES" ]; then
      echo "Executable exceeds VirusTotal's 32 MB direct-upload limit ($executable_size_bytes bytes); requesting a v3 large-file upload URL..."
      if upload_url=$(getLargeFileUploadURL); then
        response=$(curl -s "$upload_url" -X POST -F "file=@./stripe.exe" -w "HTTPSTATUS:%{http_code}")
      else
        large_file_upload_url_status=$?
        if [ "$large_file_upload_url_status" -eq 3 ]; then
          echo "Skipping VirusTotal upload for $FILENAME because the configured API key cannot upload large files."
          rm -f ./stripe.exe
          continue
        fi
        exit 1
      fi
    else
      response=$(curl -s "$upload_url" -X POST -F "apikey=$VIRUSTOTAL_API_KEY" -F "file=@./stripe.exe" -w "HTTPSTATUS:%{http_code}")

      if [ "$(getResponseCode "$response")" = "413" ]; then
        echo "VirusTotal rejected the direct upload with 413; retrying with a large-file upload URL..."
        if upload_url=$(getLargeFileUploadURL); then
          response=$(curl -s "$upload_url" -X POST -F "file=@./stripe.exe" -w "HTTPSTATUS:%{http_code}")
        else
          large_file_upload_url_status=$?
          if [ "$large_file_upload_url_status" -eq 3 ]; then
            echo "Skipping VirusTotal upload for $FILENAME because the configured API key cannot upload large files."
            rm -f ./stripe.exe
            continue
          fi
          exit 1
        fi
      fi
    fi

    response_code=$(getResponseCode "$response")

    if [[ "$(echo "$response_code" | head -c2)" != "20" ]]; then
      echo "Unable to upload, HTTP response code: $response_code"
      exit 1
    fi

    echo "HTTP response code: $response_code"

    echo "Adding comment..."

    RESPONSE_CODE=$(curl -s "$VIRUSTOTAL_V2_COMMENT_URL" -X POST -d "apikey=$VIRUSTOTAL_API_KEY" -d "resource=$sha256" -d "comment=Stripe CLI $VERSION, uncompressed from $i" -o /dev/null -w "%{http_code}")

    if [[ "$(echo "$RESPONSE_CODE" | head -c2)" != "20" ]]; then
      echo "Unable to add comment, HTTP response code: $RESPONSE_CODE"
      exit 1
    fi

    PERMALINK="https://www.virustotal.com/gui/file/$sha256"
    echo "Permalink: $PERMALINK"
  done;
}

printMeta () {
    echo "Uploading version: $VERSION"
}

cleanArtifacts () {
  rm -f ./*.zip ./*.exe
}

cleanArtifacts
downloadWindowsArtifacts
setVersion
printMeta
virustotalUpload
