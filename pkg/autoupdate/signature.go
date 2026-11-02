package autoupdate

import (
	"crypto"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	_ "embed"
	"encoding/base64"
	"encoding/hex"
	"encoding/pem"
	"fmt"
	"strings"

	"github.com/stripe/stripe-cli/pkg/errorcategory"
)

// This trust root must be reviewed independently of GitHub release assets.
//
//go:embed release-key.pub
var releasePublicKey string

func verifiedReleaseChecksum(ver, checksums, signature string) (string, error) {
	block, _ := pem.Decode([]byte(releasePublicKey))
	if block == nil {
		return "", errorcategory.New(errorcategory.Internal, "release verification key is not configured")
	}
	parsed, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return "", fmt.Errorf("parse release verification key: %w", err)
	}
	key, ok := parsed.(*rsa.PublicKey)
	if !ok || key.N.BitLen() < 3072 {
		return "", errorcategory.New(errorcategory.Internal, "invalid release verification key")
	}
	sig, err := base64.StdEncoding.DecodeString(strings.TrimSpace(signature))
	if err != nil {
		return "", fmt.Errorf("decode release signature: %w", err)
	}
	digest := sha256.Sum256([]byte(checksums))
	if err := rsa.VerifyPKCS1v15(key, crypto.SHA256, digest[:], sig); err != nil {
		return "", fmt.Errorf("verify checksum signature: %w", err)
	}

	asset := binaryAssetName(ver)
	checksum := ""
	for _, line := range strings.Split(checksums, "\n") {
		fields := strings.Fields(line)
		if len(fields) != 2 || fields[1] != asset {
			continue
		}
		digest, err := hex.DecodeString(fields[0])
		if err != nil || len(digest) != 32 || checksum != "" {
			return "", errorcategory.New(errorcategory.Network, "invalid or duplicate archive checksum")
		}
		checksum = strings.ToLower(fields[0])
	}
	if checksum == "" {
		return "", errorcategory.Errorf(errorcategory.Network, "signed checksums do not contain %s", asset)
	}
	return checksum, nil
}
