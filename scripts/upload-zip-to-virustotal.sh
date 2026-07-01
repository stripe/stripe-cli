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
VIRUSTOTAL_SCAN_URL="https://www.virustotal.com/vtapi/v2/file/scan"
VIRUSTOTAL_LARGE_FILE_UPLOAD_URL="https://www.virustotal.com/vtapi/v2/file/scan/upload_url"

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

  upload_url_response=$(curl -s "$VIRUSTOTAL_LARGE_FILE_UPLOAD_URL?apikey=$VIRUSTOTAL_API_KEY" -w "HTTPSTATUS:%{http_code}")
  upload_url_body=$(getResponseBody "$upload_url_response")
  upload_url_response_code=$(getResponseCode "$upload_url_response")

  if [[ "$(echo "$upload_url_response_code" | head -c2)" != "20" ]]; then
    echo "Unable to get large-file upload URL, HTTP response code: $upload_url_response_code" >&2
    exit 1
  fi

  upload_url=$(echo "$upload_url_body" | jq -r ".upload_url")
  if [ -z "$upload_url" ] || [ "$upload_url" = "null" ]; then
    echo "VirusTotal did not return a large-file upload URL" >&2
    exit 1
  fi

  printf '%s' "$upload_url"
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
    local upload_url
    local response
    local body
    local response_code
    FILENAME=${i##*/}

    echo "Uncompressing archive..."

    unzip -o "$FILENAME"

    echo "Uploading to VirusTotal..."

    executable_size_bytes=$(wc -c < ./stripe.exe | tr -d '[:space:]')
    upload_url="$VIRUSTOTAL_SCAN_URL"

    if [ "$executable_size_bytes" -gt "$VIRUSTOTAL_DIRECT_UPLOAD_LIMIT_BYTES" ]; then
      echo "Executable exceeds VirusTotal's 32 MB direct-upload limit ($executable_size_bytes bytes); requesting a large-file upload URL..."
      upload_url=$(getLargeFileUploadURL)
      response=$(curl -s "$upload_url" -X POST -F "file=@./stripe.exe" -w "HTTPSTATUS:%{http_code}")
    else
      response=$(curl -s "$upload_url" -X POST -F "apikey=$VIRUSTOTAL_API_KEY" -F "file=@./stripe.exe" -w "HTTPSTATUS:%{http_code}")

      if [ "$(getResponseCode "$response")" = "413" ]; then
        echo "VirusTotal rejected the direct upload with 413; retrying with a large-file upload URL..."
        upload_url=$(getLargeFileUploadURL)
        response=$(curl -s "$upload_url" -X POST -F "file=@./stripe.exe" -w "HTTPSTATUS:%{http_code}")
      fi
    fi

    body=$(getResponseBody "$response")
    response_code=$(getResponseCode "$response")

    if [[ "$(echo "$response_code" | head -c2)" != "20" ]]; then
      echo "Unable to upload, HTTP response code: $response_code"
      exit 1
    fi

    echo "HTTP response code: $response_code"

    echo "Adding comment..."

    SHA256=$(echo "$body" | jq -r ".sha256")
    if [ -z "$SHA256" ] || [ "$SHA256" = "null" ]; then
      echo "VirusTotal upload did not return a sha256 hash"
      exit 1
    fi

    RESPONSE_CODE=$(curl "https://www.virustotal.com/vtapi/v2/comments/put" -X POST -d "apikey=$VIRUSTOTAL_API_KEY" -d "resource=$SHA256" -d "comment=Stripe CLI $VERSION, uncompressed from $i" -o /dev/null -w "%{http_code}")

    if [[ "$(echo "$RESPONSE_CODE" | head -c2)" != "20" ]]; then
      echo "Unable to add comment, HTTP response code: $RESPONSE_CODE"
      exit 1
    fi

    PERMALINK=$(echo "$body" | jq -r ".permalink")
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
