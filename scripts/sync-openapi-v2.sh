#!/usr/bin/env bash
set -eu -o pipefail

pushd "$HOME/stripe/zoolander"

# Pull master.
echo "Bringing master up to date."
git checkout master && git pull

# Grab SHA so we can save this to a file for some kind of "paper trail".
SHA=$(git rev-parse HEAD)

echo "⏳ Retrieving v2 openapi spec..."

./scripts/api-services/apiv2 apispecdump --version="2025-06-30.basil" --format OPENAPI_JSON --variant CLI --out-file spec3.v2.sdk.json
./scripts/api-services/apiv2 apispecdump --version="2025-06-30.basil" --format OPENAPI_JSON --variant CLI_PUBLIC_PREVIEW --out-file spec3.v2.sdk.preview.json

popd

rm -f api/openapi-spec/spec3.v2.sdk.json
rm -f api/openapi-spec/spec3.v2.sdk.preview.json

echo "$SHA" > api/ZOOLANDER_SHA

cp ~/stripe/zoolander/spec3.v2.sdk.json api/openapi-spec/
cp ~/stripe/zoolander/spec3.v2.sdk.preview.json api/openapi-spec/

echo "⏳ Generating resource commands..."

make build

echo "✅ Successfully generated resource commands and rebuilt CLI."

