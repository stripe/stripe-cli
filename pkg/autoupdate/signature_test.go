package autoupdate

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func authorizeTestMarker(t *testing.T, marker *UpdateMarker) *rsa.PrivateKey {
	t.Helper()
	private, err := rsa.GenerateKey(rand.Reader, 3072)
	require.NoError(t, err)
	original := releasePublicKey
	t.Cleanup(func() { releasePublicKey = original })
	public, err := x509.MarshalPKIXPublicKey(&private.PublicKey)
	require.NoError(t, err)
	releasePublicKey = string(pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: public}))
	marker.SignedChecksums = fmt.Sprintf("%s  %s\n", marker.Checksum, binaryAssetName(marker.Version))
	marker.ChecksumSignature = testSignature(t, private, marker.SignedChecksums)
	return private
}

func TestDownloadAndReplaceRejectsUntrustedRelease(t *testing.T) {
	for _, name := range []string{"legacy marker", "altered checksums", "altered signature", "wrong key", "unconfigured key", "different version", "altered staged checksum", "missing archive", "duplicate archive"} {
		t.Run(name, func(t *testing.T) {
			marker := &UpdateMarker{Version: "1.55.7", Checksum: strings.Repeat("a", 64)}
			private := authorizeTestMarker(t, marker)
			switch name {
			case "legacy marker":
				marker.SignedChecksums = ""
				marker.ChecksumSignature = ""
			case "altered checksums":
				marker.SignedChecksums = strings.ReplaceAll(marker.SignedChecksums, "a", "b")
			case "altered signature":
				marker.ChecksumSignature = base64.StdEncoding.EncodeToString(make([]byte, 384))
			case "wrong key":
				other, err := rsa.GenerateKey(rand.Reader, 3072)
				require.NoError(t, err)
				der, err := x509.MarshalPKIXPublicKey(&other.PublicKey)
				require.NoError(t, err)
				releasePublicKey = string(pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: der}))
			case "unconfigured key":
				releasePublicKey = "UNCONFIGURED"
			case "different version":
				marker.Version = "1.55.8"
			case "altered staged checksum":
				marker.Checksum = strings.Repeat("b", 64)
			case "missing archive":
				marker.SignedChecksums = strings.Repeat("a", 64) + "  other.tar.gz\n"
			case "duplicate archive":
				marker.SignedChecksums += marker.SignedChecksums
			}
			if name == "missing archive" || name == "duplicate archive" {
				marker.ChecksumSignature = testSignature(t, private, marker.SignedChecksums)
			}
			requests := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { requests++ }))
			defer server.Close()
			marker.DownloadURL = server.URL
			exe := filepath.Join(t.TempDir(), binaryName())
			require.NoError(t, os.WriteFile(exe, []byte("original"), 0755))
			_, err := downloadAndReplace(marker, exe)
			require.Error(t, err)
			assert.Zero(t, requests, "reject untrusted metadata before downloading")
			data, err := os.ReadFile(exe)
			require.NoError(t, err)
			assert.Equal(t, "original", string(data))
		})
	}
}

func testSignature(t *testing.T, key *rsa.PrivateKey, data string) string {
	t.Helper()
	digest := sha256.Sum256([]byte(data))
	signature, err := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, digest[:])
	require.NoError(t, err)
	return base64.StdEncoding.EncodeToString(signature)
}
