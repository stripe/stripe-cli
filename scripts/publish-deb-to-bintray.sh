#!/bin/bash

# Adapted from https://github.com/leopardslab/dunner/blob/master/release/publish_deb_to_bintray.sh

set -e

REPO="stripe-cli-deb"
PACKAGE="stripe"
DISTRIBUTIONS="stable"
COMPONENTS="main"

if [ -z "$GITHUB_TOKEN" ]; then
  echo "GITHUB_TOKEN is not set"
  exit 1
fi

if [ -z "$BINTRAY_USER" ]; then
  echo "BINTRAY_USER is not set"
  exit 1
fi

if [ -z "$BINTRAY_API_KEY" ]; then
  echo "BINTRAY_API_KEY is not set"
  exit 1
fi

setVersion () {
  VERSION=$(curl --silent "https://api.github.com/repos/stripe/stripe-cli/releases/latest" -u $GITHUB_TOKEN: | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/');
}

setUploadDirPath () {
  UPLOADDIRPATH="pool/s/$PACKAGE"
}

downloadDebianArtifacts() {
  echo "Dowloading debian artifacts"
  FILES=$(curl -s https://api.github.com/repos/stripe/stripe-cli/releases/latest -u $GITHUB_TOKEN: \
| grep "browser_download_url.*deb" \
| cut -d : -f 3 \
| sed -e 's/^/https:/' \
| tr -d '"' );
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

bintrayUpload () {
  for i in $FILES; do
    FILENAME=${i##*/}
    ARCH=$(echo ${FILENAME##*_} | cut -d '.' -f 1)
    if [ $ARCH == "386" ]; then
      ARCH="i386"
    fi

    URL="https://api.bintray.com/content/stripe/$REPO/$PACKAGE/$VERSION/$UPLOADDIRPATH/$FILENAME;deb_distribution=$DISTRIBUTIONS;deb_component=$COMPONENTS;deb_architecture=$ARCH?publish=1&override=1"
    echo "Uploading $URL"

    RESPONSE_CODE=$(curl -T $FILENAME -u$BINTRAY_USER:$BINTRAY_API_KEY $URL -I -s -w "%{http_code}" -o /dev/null);
    if [[ "$(echo $RESPONSE_CODE | head -c2)" != "20" ]]; then
      echo "Unable to upload, HTTP response code: $RESPONSE_CODE"
      exit 1
    fi
    echo "HTTP response code: $RESPONSE_CODE"
  done;
}

bintraySetDownloads () {
  for i in $FILES; do
    FILENAME=${i##*/}
    ARCH=$(echo ${FILENAME##*_} | cut -d '.' -f 1)
    if [ $ARCH == "386" ]; then
      ARCH="i386"
    fi
    URL="https://api.bintray.com/file_metadata/stripe/$REPO/$UPLOADDIRPATH/$FILENAME"

    echo "Putting $FILENAME in $PACKAGE's download list"
    RESPONSE_CODE=$(curl -X PUT -d "{ \"list_in_downloads\": true }" -H "Content-Type: application/json" -u$BINTRAY_USER:$BINTRAY_API_KEY $URL -s -w "%{http_code}" -o /dev/null);

    if [ "$(echo $RESPONSE_CODE | head -c2)" != "20" ]; then
        echo "Unable to put in download list, HTTP response code: $RESPONSE_CODE"
        exit 1
    fi
    echo "HTTP response code: $RESPONSE_CODE"
  done
}

snooze () {
    echo "\nSleeping for 30 seconds. Have a coffee..."
    sleep 30s;
    echo "Done sleeping\n"
}

printMeta () {
    echo "Publishing: $PACKAGE"
    echo "Version to be uploaded: $VERSION"
}

cleanArtifacts () {
  rm -f "$(pwd)/*.deb"
}

cleanArtifacts
downloadDebianArtifacts
setVersion
printMeta
setUploadDirPath
bintrayUpload
snooze
bintraySetDownloads
