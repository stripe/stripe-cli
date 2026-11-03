package main

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"os"

	"github.com/stripe/stripe-cli/pkg/errorcategory"
)

func sign(privatePEM, publicPEM string, checksums []byte) (string, error) {
	privateBlock, _ := pem.Decode([]byte(privatePEM))
	if privateBlock == nil {
		return "", errorcategory.New(errorcategory.UserInput, "invalid release signing key")
	}
	parsed, err := x509.ParsePKCS8PrivateKey(privateBlock.Bytes)
	if err != nil {
		return "", errorcategory.New(errorcategory.UserInput, "invalid release signing key")
	}
	key, ok := parsed.(*rsa.PrivateKey)
	if !ok || key.N.BitLen() < 3072 {
		return "", errorcategory.New(errorcategory.UserInput, "release signing requires an RSA key of at least 3072 bits")
	}
	publicBlock, _ := pem.Decode([]byte(publicPEM))
	if publicBlock == nil {
		return "", errorcategory.New(errorcategory.UserInput, "release public key is not configured")
	}
	public, err := x509.ParsePKIXPublicKey(publicBlock.Bytes)
	if err != nil || !key.PublicKey.Equal(public) {
		return "", errorcategory.New(errorcategory.UserInput, "release signing key does not match the pinned public key")
	}
	digest := sha256.Sum256(checksums)
	signature, err := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, digest[:])
	if err != nil {
		return "", fmt.Errorf("sign release checksums: %w", err)
	}
	return base64.StdEncoding.EncodeToString(signature), nil
}

func generateKey(keyPath string) (string, error) {
	key, err := rsa.GenerateKey(rand.Reader, 3072)
	if err != nil {
		return "", err
	}
	privateDER, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		return "", err
	}
	publicDER, err := x509.MarshalPKIXPublicKey(&key.PublicKey)
	if err != nil {
		return "", err
	}
	file, err := os.OpenFile(keyPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return "", err
	}
	if err := pem.Encode(file, &pem.Block{Type: "PRIVATE KEY", Bytes: privateDER}); err != nil {
		file.Close()
		return "", err
	}
	if err := file.Close(); err != nil {
		return "", err
	}
	return string(pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: publicDER})), nil
}

func run() error {
	if len(os.Args) == 3 && os.Args[1] == "generate-key" {
		public, err := generateKey(os.Args[2])
		if err != nil {
			return err
		}
		fmt.Print(public)
		return nil
	}
	if len(os.Args) != 3 {
		return errorcategory.New(errorcategory.UserInput, "usage: sign-release-checksums CHECKSUMS SIGNATURE | generate-key PRIVATE_KEY_FILE")
	}
	checksums, err := os.ReadFile(os.Args[1])
	if err != nil {
		return err
	}
	publicKey, err := os.ReadFile("pkg/autoupdate/release-key.pub")
	if err != nil {
		return err
	}
	signature, err := sign(os.Getenv("STRIPE_CLI_RELEASE_SIGNING_KEY"), string(publicKey), checksums)
	if err != nil {
		return err
	}
	return os.WriteFile(os.Args[2], []byte(signature+"\n"), 0644)
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
