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

if [ -z "$ARTIFACTORY_SECRET" ]; then
  echo "ARTIFACTORY_SECRET is not set"
  exit 1
fi

artifactoryUpload () {
  if [ "$ARCH" == "386" ]; then
      ARCH="i386"
  fi

  if [[ $FORMAT == "debian" ]]
  then
      if [ $ARCH == "x86_64" ]; then
        ARCH="amd64"
      fi

      echo "setting deployment to debian repo"
      UPLOAD_URL="https://stripe.jfrog.io/artifactory/$REPO-local/pool/$PACKAGE/$VERSION/$ARCH/$PACKAGE.deb;deb.distribution=$DISTRIBUTIONS;deb.component=$COMPONENTS;deb.architecture=$ARCH"
  elif [[ $FORMAT == "rpm" ]]
  then
      echo "setting deployment to rpm repo"
      UPLOAD_URL="https://stripe.jfrog.io/artifactory/$REPO-local/$PACKAGE/$VERSION/$ARCH/$PACKAGE.rpm"
  else
      echo "unrecognised package format"
      exit 1
  fi

  echo "Uploading $UPLOAD_URL"

  RESPONSE_CODE=$(curl -X PUT -T "$FILENAME" -H "Authorization: Bearer $ARTIFACTORY_SECRET" "$UPLOAD_URL" -I -s -w "%{http_code}" -o /dev/null)

  if [[ "$(echo "$RESPONSE_CODE" | head -c2)" != "20" ]]; then
      echo "Unable to upload, HTTP response code: $RESPONSE_CODE"
      exit 1
  fi

  echo "HTTP response code: $RESPONSE_CODE"
}

printMeta () {
  echo "Publishing: $PACKAGE"
  echo "Version to be uploaded: $VERSION"
}

printMeta
artifactoryUpload
