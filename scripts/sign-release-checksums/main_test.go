package main

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateAndSign(t *testing.T) {
	path := filepath.Join(t.TempDir(), "release-key.pem")
	public, err := generateKey(path)
	require.NoError(t, err)
	private, err := os.ReadFile(path)
	require.NoError(t, err)
	body := []byte("checksum  stripe_1.55.7_linux_x86_64.tar.gz\n")
	signature, err := sign(string(private), public, body)
	require.NoError(t, err)
	decoded, err := base64.StdEncoding.DecodeString(signature)
	require.NoError(t, err)
	block, _ := pem.Decode([]byte(public))
	require.NotNil(t, block)
	parsed, err := x509.ParsePKIXPublicKey(block.Bytes)
	require.NoError(t, err)
	key := parsed.(*rsa.PublicKey)
	digest := sha256.Sum256(body)
	require.NoError(t, rsa.VerifyPKCS1v15(key, crypto.SHA256, digest[:], decoded))
	if openssl, err := exec.LookPath("openssl"); err == nil {
		dir := t.TempDir()
		pubPath, sigPath, sumsPath := filepath.Join(dir, "release.pub"), filepath.Join(dir, "release.sig"), filepath.Join(dir, "checksums")
		require.NoError(t, os.WriteFile(pubPath, []byte(public), 0600))
		require.NoError(t, os.WriteFile(sigPath, decoded, 0600))
		require.NoError(t, os.WriteFile(sumsPath, body, 0600))
		output, err := exec.Command(openssl, "dgst", "-sha256", "-verify", pubPath, "-signature", sigPath, sumsPath).CombinedOutput()
		require.NoError(t, err, "%s", output)
	}

	digest = sha256.Sum256(append(body, 'x'))
	assert.Error(t, rsa.VerifyPKCS1v15(key, crypto.SHA256, digest[:], decoded))
	for _, tc := range []struct{ private, public string }{
		{"", public}, {"invalid", public}, {string(private), "UNCONFIGURED"},
	} {
		_, err := sign(tc.private, tc.public, body)
		require.Error(t, err)
	}
	other, err := rsa.GenerateKey(rand.Reader, 3072)
	require.NoError(t, err)
	otherDER, err := x509.MarshalPKIXPublicKey(&other.PublicKey)
	require.NoError(t, err)
	_, err = sign(string(private), string(pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: otherDER})), body)
	require.Error(t, err)
	if runtime.GOOS != "windows" {
		info, err := os.Stat(path)
		require.NoError(t, err)
		assert.Equal(t, os.FileMode(0600), info.Mode().Perm())
	}
	_, err = generateKey(path)
	require.Error(t, err)
	unchanged, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, private, unchanged)
}
