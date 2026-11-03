# Release checksum signing

Release builds publish a base64-encoded RSA PKCS#1 v1.5 signature over the
SHA-256 digest of each platform's GoReleaser checksums file. The signature is
uploaded beside that file with a `.sig` suffix. Consumers can verify it with
the public key in `pkg/autoupdate/release-key.pub`.

## Key provisioning

The reviewed public key must match the private key stored in the protected
`release` environment. Release signing fails if that key is missing or mismatched.
Use the following procedure when provisioning a new signing identity.

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
The unpublished snapshot workflow skips signing and requires no release key.

## Automatic update verification

The CLI verifies signed checksums before staging an automatic update and again
before applying it. The downloaded archive must match the signed checksum for
its exact version and platform. Unsigned updates and downgrades are rejected.

## Installer verification

The shell installer for macOS and Linux and the PowerShell installer for Windows
embed the reviewed public key. They require a valid checksum signature before
downloading and extracting a release archive. The shell installer requires
OpenSSL; PowerShell uses the built-in .NET RSA API.

Unsigned releases cannot be installed, including explicitly requested versions.
