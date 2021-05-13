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

setVersion () {
  VERSION=$(curl -s "https://api.github.com/repos/stripe/stripe-cli/releases/latest" -u $GITHUB_TOKEN: \
| jq -r ".tag_name")
}

downloadWindowsArtifacts() {
  echo "Dowloading Windows artifacts..."
  FILES=$(curl -s "https://api.github.com/repos/stripe/stripe-cli/releases/latest" -u $GITHUB_TOKEN: \
| jq -r ".assets[].browser_download_url" \
| grep "windows" \
| grep "zip")
  echo "$FILES"
  for i in $FILES; do
    RESPONSE_CODE=$(curl -L -O -w "%{response_code}" "$i")
    echo "$RESPONSE_CODE"
    code=$(echo "$RESPONSE_CODE" | head -c2)
    if [ $code != "20" ] && [ $code != "30" ]; then
      echo "Unable to download $i HTTP response code: $RESPONSE_CODE"
      exit 1
    fi
  done;
  echo "Finished downloading"
}

virustotalUpload () {
  for i in $FILES; do
    FILENAME=${i##*/}

    echo "Uncompressing archive..."

    unzip -o $FILENAME

    echo "Uploading to VirusTotal..."

    RESPONSE=$(curl -s "https://www.virustotal.com/vtapi/v2/file/scan" -X POST -F "apikey=$VIRUSTOTAL_API_KEY" -F "file=@./stripe.exe" -w "HTTPSTATUS:%{http_code}")
    BODY=$(echo $RESPONSE | sed -e 's/HTTPSTATUS\:.*//g')
    RESPONSE_CODE=$(echo $RESPONSE | tr -d '\n' | sed -e 's/.*HTTPSTATUS://')

    if [[ "$(echo $RESPONSE_CODE | head -c2)" != "20" ]]; then
      echo "Unable to upload, HTTP response code: $RESPONSE_CODE"
      exit 1
    fi

    echo "HTTP response code: $RESPONSE_CODE"

    echo "Adding comment..."

    SHA256=$(echo $BODY | jq -r ".sha256")
    RESPONSE_CODE=$(curl "https://www.virustotal.com/vtapi/v2/comments/put" -X POST -d "apikey=$VIRUSTOTAL_API_KEY" -d "resource=$SHA256" -d "comment=Stripe CLI $VERSION, uncompressed from $i" -o /dev/null -w "%{http_code}")

    if [[ "$(echo $RESPONSE_CODE | head -c2)" != "20" ]]; then
      echo "Unable to add comment, HTTP response code: $RESPONSE_CODE"
      exit 1
    fi

    PERMALINK=$(echo $BODY | jq -r ".permalink")
    echo "Permalink: $PERMALINK"
  done;
}

printMeta () {
    echo "Uploading version: $VERSION"
}

cleanArtifacts () {
  rm -f "$(pwd)/*.zip" "$(pwd)/*.exe"
}

cleanArtifacts
downloadWindowsArtifacts
setVersion
printMeta
virustotalUpload
