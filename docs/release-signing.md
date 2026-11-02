# Release checksum signing

Release builds publish a base64-encoded RSA PKCS#1 v1.5 signature over the
SHA-256 digest of each platform's GoReleaser checksums file. The signature is
uploaded beside that file with a `.sig` suffix. Consumers can verify it with
the public key in `pkg/autoupdate/release-key.pub`.

## Provision the key before merging

The public key is currently `UNCONFIGURED`. Release signing fails until a
matching private key and public key are provisioned.

1. On an approved secure workstation, generate a 3072-bit RSA private key outside
   the repository:

   ```sh
   go run ./scripts/sign-release-checksums generate-key /secure/location/stripe-cli-release.pem
   ```

   The command creates a file with mode 0600, refuses to overwrite an existing
   file, and prints only the public PEM key. Copy that public key into
   `pkg/autoupdate/release-key.pub` and review it with the release change.
2. Put the private PEM file's contents into the environment secret
   `STRIPE_CLI_RELEASE_SIGNING_KEY` in the existing GitHub `release` environment.
   Required reviewers and prevention of self-review must remain enabled.
   Keep the private key in approved secret storage. Repository and organization
   secrets do not provide the required environment approval boundary.
3. Publish an approved release and confirm all three platform checksum files
   have `.sig` assets. GoReleaser signs local build outputs before publication.
   The signer checks that its private key matches the reviewed public key.

The private key uses PEM PKCS#8; the public key uses PEM SubjectPublicKeyInfo.
The private key is read from the environment and never passed as an argument.
Each platform build enters the protected `release` environment to access it.

## Consumer rollout

Publishing signatures does not change existing clients. Verification must also
be enabled in the CLI and installation scripts. Validate both successful
upgrades and rejection of invalid signatures on supported operating systems.

Do not sign downloaded historical release assets. Produce signed artifacts
through an approved build. Once consumers pin a public key, rotation requires a
transition that preserves those clients' ability to verify subsequent releases.
