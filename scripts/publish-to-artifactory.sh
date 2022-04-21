#!/bin/bash

# Adapted from https://github.com/leopardslab/dunner/blob/master/release/publish_deb_to_bintray.sh

set -e

FILENAME=$1
VERSION=$2
ARCH=$3
FORMAT=$4

REPO="stripe-cli-$FORMAT"
PACKAGE="stripe"
DISTRIBUTIONS="stable"
COMPONENTS="main"
DEBIAN_META=""
UPLOAD_URL=""

if [ -z "$ARTIFACTORY_SECRET" ]; then
  echo "ARTIFACTORY_SECRET is not set"
  exit 1
fi

artifactoryUpload () {
    if [ "$ARCH" == "386" ]; then
        ARCH="i386"
    fi

    if [ "$FORMAT" == "debian" ]; then
        DEBIAN_META="deb.distribution=$DISTRIBUTIONS;deb.component=$COMPONENTS;deb.architecture=$ARCH"
        UPLOAD_URL="https://stripe.jfrog.io/artifactory/$REPO-local/pool/$PACKAGE/$VERSION/$PACKAGE$DEBIAN_META"
    elif [ "$FORMAT" == "rpm" ]; then
        UPLOAD_URL="https://stripe.jfrog.io/artifactory/$REPO-local/$PACKAGE/$VERSION/$ARCH/$PACKAGE"
    else
        echo "unrecognised package format"
        exit 1
    fi

    echo "Uploading $UPLOAD_URL"

    RESPONSE_CODE=$(curl -X PUT -T "$FILENAME" -H "Authorization: Bearer $ARTIFACTORY_SECRET" "$UPLOAD_URL" -I -s -w "%{http_code}" -o /dev/null);
    if [[ "$(echo "$RESPONSE_CODE" | head -c2)" != "20" ]]; then
        echo "Unable to upload, HTTP response code: $RESPONSE_CODE"
        exit 1
    fi
    echo "HTTP response code: $RESPONSE_CODE"
}

snooze () {
  printf "\nSleeping for 30 seconds. Have a coffee..."
  sleep 30s;
  printf "Done sleeping\n"
}

printMeta () {
  echo "Publishing: $PACKAGE"
  echo "Version to be uploaded: $VERSION"
}

printMeta
artifactoryUpload
snooze
