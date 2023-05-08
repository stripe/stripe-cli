package requests

import (
	"crypto/ecdsa"
	"crypto/ed25519"
	"net/http"
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	ecdsaPrivateKey   = "MIGHAgEAMBMGByqGSM49AgEGCCqGSM49AwEHBG0wawIBAQQg8HGPWJFG+iJFIlBXJqbiwnjApXu8Gh2Plj8PD+9u0WOhRANCAASDqUsr3WKCRrogr/hIJxXrYZpPEanUo9c2WHgM8NYj3T9XoGlBi/Q9E9XaAgV5T4RxyKjtr++b0ETzvoLJ13O+"
	ed25519PrivateKey = "MC4CAQAwBQYDK2VwBCIEID7ElTmBLNFhFziW+YlR1SRv6/xvGq6RTbSFXmGvTBtz"
)

func TestEd25519P256Sha256GetHeaders(t *testing.T) {
	req, _ := http.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set(stripeContextHeaderName, "test-context")
	req.Header.Set(authorizationHeaderName, "STRIPE-SIG-PREFIX 123")
	getCurrentUnixTime = func() string {
		return "1683296385"
	}
	err := SignRequest(req, ed25519PrivateKey)
	require.NoError(t, err)
	require.Equal(t, "sha-256=:47DEQpj8HBSa+/TImW+5JCeuQeRkm5NMpJWZG3hSuFU=:", req.Header.Get(contentDigestHeaderName))
	require.Equal(t, `sig1=("stripe-context" "stripe-account" "authorization");created=1683296385`, req.Header.Get(signatureInputHeaderName))
	require.Regexp(t, regexp.MustCompile("sig1=:.*:"), req.Header.Get(signatureHeaderName))
}

func TestEd25519P256Sha256PostHeaders(t *testing.T) {
	req, _ := http.NewRequest(http.MethodPost, "/test", nil)
	req.Header.Set(stripeContextHeaderName, "test-context")
	req.Header.Set(authorizationHeaderName, "STRIPE-SIG-PREFIX 123")
	getCurrentUnixTime = func() string {
		return "1683296385"
	}
	err := SignRequest(req, ed25519PrivateKey)
	require.NoError(t, err)
	require.Equal(t, "sha-256=:47DEQpj8HBSa+/TImW+5JCeuQeRkm5NMpJWZG3hSuFU=:", req.Header.Get(contentDigestHeaderName))
	require.Equal(t, `sig1=("stripe-context" "stripe-account" "authorization" "content-type" "content-digest");created=1683296385`, req.Header.Get(signatureInputHeaderName))
	require.Regexp(t, regexp.MustCompile("sig1=:.*:"), req.Header.Get(signatureHeaderName))
}

func TestEcdsaP256Sha256GetHeaders(t *testing.T) {
	req, _ := http.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set(stripeContextHeaderName, "test-context")
	req.Header.Set(authorizationHeaderName, "STRIPE-SIG-PREFIX 123")
	getCurrentUnixTime = func() string {
		return "1683296385"
	}
	err := SignRequest(req, ecdsaPrivateKey)
	require.NoError(t, err)
	require.Equal(t, "sha-256=:47DEQpj8HBSa+/TImW+5JCeuQeRkm5NMpJWZG3hSuFU=:", req.Header.Get(contentDigestHeaderName))
	require.Equal(t, `sig1=("stripe-context" "stripe-account" "authorization");created=1683296385`, req.Header.Get(signatureInputHeaderName))
	require.Regexp(t, regexp.MustCompile("sig1=:.*:"), req.Header.Get(signatureHeaderName))
}

func TestEcdsaP256Sha256PostHeaders(t *testing.T) {
	req, _ := http.NewRequest(http.MethodPost, "/test", nil)
	req.Header.Set(stripeContextHeaderName, "test-context")
	req.Header.Set(authorizationHeaderName, "STRIPE-SIG-PREFIX 123")
	getCurrentUnixTime = func() string {
		return "1683296385"
	}
	err := SignRequest(req, ecdsaPrivateKey)
	require.NoError(t, err)
	require.Equal(t, "sha-256=:47DEQpj8HBSa+/TImW+5JCeuQeRkm5NMpJWZG3hSuFU=:", req.Header.Get(contentDigestHeaderName))
	require.Equal(t, `sig1=("stripe-context" "stripe-account" "authorization" "content-type" "content-digest");created=1683296385`, req.Header.Get(signatureInputHeaderName))
	require.Regexp(t, regexp.MustCompile("sig1=:.*:"), req.Header.Get(signatureHeaderName))
}

func TestBuildSignatureForGetRequest(t *testing.T) {
	req, _ := http.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set(stripeContextHeaderName, "test-context")
	req.Header.Set(authorizationHeaderName, "STRIPE-SIG-PREFIX 123")
	getCurrentUnixTime = func() string {
		return "1683296385"
	}

	headers := getCoveredHeaders(req)
	result := buildSignature(req, headers, buildSignatureInput(headers))
	require.Equal(t, "\"stripe-context\": test-context\n\"stripe-account\": \n\"authorization\": STRIPE-SIG-PREFIX 123\n\"@signature-params\": (\"stripe-context\" \"stripe-account\" \"authorization\");created=1683296385", result)
}

func TestBuildSignatureForPostRequest(t *testing.T) {
	req, _ := http.NewRequest(http.MethodPost, "/test", nil)
	req.Header.Set(stripeContextHeaderName, "test-context")
	req.Header.Set(authorizationHeaderName, "STRIPE-SIG-PREFIX 123")
	getCurrentUnixTime = func() string {
		return "1683296385"
	}

	headers := getCoveredHeaders(req)
	result := buildSignature(req, headers, buildSignatureInput(headers))
	require.Equal(t, "\"stripe-context\": test-context\n\"stripe-account\": \n\"authorization\": STRIPE-SIG-PREFIX 123\n\"content-type\": \n\"content-digest\": \n\"@signature-params\": (\"stripe-context\" \"stripe-account\" \"authorization\" \"content-type\" \"content-digest\");created=1683296385", result)
}

func TestGenerateSigningKeyPair_WithUnknownAlg(t *testing.T) {
	_, err := privateKeyFromBase64("foo")
	assert.ErrorContains(t, err, "could not parse private key; ")
}

func TestPrivateKeyFromBase64_ed25519(t *testing.T) {
	privKeyStruct, err := privateKeyFromBase64(ed25519PrivateKey)
	require.NoError(t, err)
	assert.IsType(t, ed25519.PrivateKey{}, privKeyStruct)
}

func TestPrivateKeyFromBase64_ecdsa(t *testing.T) {
	privKeyStruct, err := privateKeyFromBase64(ecdsaPrivateKey)
	require.NoError(t, err)
	assert.IsType(t, &ecdsa.PrivateKey{}, privKeyStruct)
}

func TestPrivateKeyFromBase64_UnsupportedAlg(t *testing.T) {
	// RSA
	_, err := privateKeyFromBase64("MFQCAQAwDQYJKoZIhvcNAQEBBQAEQDA+AgEAAgkAyfGahhs+Rb8CAwEAAQIIT4L9X9UUP8ECBQDlIjN3AgUA4Z9B+QIFALSbDacCBHGZAtkCBH0Z7mA=")
	assert.ErrorContains(t, err, "could not parse private key; algorithm must be ECDSA or Ed25519")
}
