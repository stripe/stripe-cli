package requests

import (
	"bytes"
	"crypto"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type signingFunction = func([]byte) ([]byte, error)

const (
	signatureInputHeaderName = "Signature-Input"
	signatureHeaderName      = "Signature"
	contentDigestHeaderName  = "Content-Digest"
	contentTypeHeaderName    = "Content-Type"
	authorizationHeaderName  = "Authorization"
	stripeAccountHeaderName  = "Stripe-Account"
	stripeContextHeaderName  = "Stripe-Context"
)

// SignRequest takes the http request and signs the request to send to Stripe. You MUST call
// this function AFTER all headers on the request have been set.
func SignRequest(req *http.Request, privKey string) error {
	var body []byte
	if req.Body != nil {
		body, err := io.ReadAll(req.Body)
		if err != nil {
			return fmt.Errorf("unable to sign request: %s", err.Error())
		}
		req.Body = io.NopCloser(bytes.NewReader(body))
	}

	digest := sha256.New()
	_, err := digest.Write(body)
	if err != nil {
		return err
	}

	contentDigestHeaderVal := fmt.Sprintf("sha-256=:%s:", base64.StdEncoding.EncodeToString(digest.Sum(nil)))
	req.Header.Set(contentDigestHeaderName, contentDigestHeaderVal)

	signer, err := getSigningFunction(privKey)
	if err != nil {
		return err
	}

	return sign(req, getCoveredHeaders(req), signer)
}

func getCoveredHeaders(req *http.Request) []string {
	coveredHeaders := []string{stripeContextHeaderName, stripeAccountHeaderName, authorizationHeaderName}
	if req.Method != http.MethodGet {
		coveredHeaders = append(coveredHeaders, contentTypeHeaderName, contentDigestHeaderName)
	}
	return coveredHeaders
}

// getSigningFunction takes the user's encoded private key, and returns the appropriate
// signing function based on the private key
func getSigningFunction(encodedKey string) (signingFunction, error) {
	decodedKey, err := privateKeyFromBase64(encodedKey)
	if err != nil {
		return nil, err
	}

	switch privateKey := decodedKey.(type) {
	case *ecdsa.PrivateKey:
		return func(in []byte) ([]byte, error) {
			hash := crypto.SHA256.New()
			defer hash.Reset()
			hash.Write(in)
			digest := hash.Sum(nil)

			r, s, err := ecdsa.Sign(rand.Reader, privateKey, digest)
			if err != nil {
				return nil, err
			}

			sig := make([]byte, 64)
			r.FillBytes(sig[:32])
			s.FillBytes(sig[32:])

			return sig, nil
		}, nil

	case ed25519.PrivateKey:
		return func(in []byte) ([]byte, error) {
			return ed25519.Sign(privateKey, in), nil
		}, nil
	}
	return nil, errors.New("could not resolve private key algorithm; algorithm must be ECDSA or Ed25519")
}

func privateKeyFromBase64(priv string) (crypto.Signer, error) {
	privDer, err := base64.StdEncoding.DecodeString(priv)
	if err != nil {
		return nil, errors.New("could not parse private key; " + err.Error())
	}

	key, err := x509.ParsePKCS8PrivateKey(privDer)
	if err != nil {
		return nil, err
	}

	switch k := key.(type) {
	case ed25519.PrivateKey:
		return k, nil
	case *ecdsa.PrivateKey:
		return k, nil
	default:
		return nil, errors.New("could not parse private key; algorithm must be ECDSA or Ed25519")
	}
}

func sign(req *http.Request, headers []string, signer signingFunction) error {
	signatureInput := buildSignatureInput(headers)
	req.Header.Set(signatureInputHeaderName, fmt.Sprintf("sig1=%s", signatureInput))

	signature := buildSignature(req, headers, signatureInput)
	signed, err := signer([]byte(signature))
	if err != nil {
		return err
	}
	encodedSignature := base64.StdEncoding.EncodeToString(signed)

	req.Header.Set(signatureHeaderName, fmt.Sprintf("sig1=:%s:", encodedSignature))

	return nil
}

var getCurrentUnixTime = func() string {
	return strconv.FormatInt(time.Now().Unix(), 10)
}

func buildSignatureInput(headers []string) string {
	var lowercasedHeaders []string
	for _, header := range headers {
		lowercasedHeaders = append(lowercasedHeaders, fmt.Sprintf(`"%s"`, strings.ToLower(header)))
	}

	signature := fmt.Sprintf("(%s)", strings.Join(lowercasedHeaders, " "))
	created := fmt.Sprintf("created=%s", getCurrentUnixTime())

	return fmt.Sprintf("%s;%s", signature, created)
}

func buildSignature(req *http.Request, headers []string, signatureInput string) string {
	var inputs []string

	for _, header := range headers {
		inputs = append(inputs, fmt.Sprintf(`"%s": %s%s`, strings.ToLower(header), req.Header.Get(header), "\n"))
	}

	return fmt.Sprintf(`%s"@signature-params": %s`, strings.Join(inputs, ""), signatureInput)
}
