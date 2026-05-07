// Package sandbox provisions anonymous Stripe sandbox environments.
package sandbox

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/stripe/stripe-cli/pkg/useragent"
)

const maxIterations = 100_000_000

var (
	ErrUnsupportedAlgorithm = errors.New("unsupported algorithm: only SHA-256 is supported")
	ErrMaxIterationsReached = errors.New("proof-of-work exceeded maximum iterations")
)

// HTTPError is returned when the sandbox server responds with a non-200 status.
type HTTPError struct {
	StatusCode int
	Body       string
}

func (e *HTTPError) Error() string {
	if e.Body != "" {
		return fmt.Sprintf("server returned %d: %s", e.StatusCode, e.Body)
	}
	return fmt.Sprintf("server returned %d", e.StatusCode)
}

type ChallengeResponse struct {
	Algorithm string `json:"algorithm"`
	Challenge string `json:"challenge"`
	Salt      string `json:"salt"`
	Signature string `json:"signature"`
}

type ProvisionRequest struct {
	Algorithm string `json:"algorithm"`
	Challenge string `json:"challenge"`
	Salt      string `json:"salt"`
	Signature string `json:"signature"`
	Number    int    `json:"number"`
	Email     string `json:"email"`
	Name      string `json:"name,omitempty"`
}

type ProvisionResponse struct {
	SecretKey      string `json:"secret_key"`
	PublishableKey string `json:"publishable_key"`
	ClaimURL       string `json:"claim_url,omitempty"`
	ExpiresAt      string `json:"expires_at,omitempty"`
	AccountID      string `json:"account_id,omitempty"`
}

type Client struct {
	BaseURL    string
	HTTPClient *http.Client
}

func NewClient(baseURL string) *Client {
	return &Client{
		BaseURL: baseURL,
		HTTPClient: &http.Client{
			Transport: &http.Transport{
				Proxy: http.ProxyFromEnvironment,
				DialContext: (&net.Dialer{
					Timeout:   30 * time.Second,
					KeepAlive: 30 * time.Second,
				}).DialContext,
				TLSHandshakeTimeout:   10 * time.Second,
				ResponseHeaderTimeout: 30 * time.Second,
			},
		},
	}
}

func (c *Client) GetChallenge(ctx context.Context, email string) (*ChallengeResponse, error) {
	body, err := json.Marshal(map[string]string{"email": email})
	if err != nil {
		return nil, err
	}

	resp, err := c.doRequest(ctx, http.MethodPost, "/keys/challenge", body)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, readErrorResponse(resp)
	}

	var challenge ChallengeResponse
	if err := json.NewDecoder(resp.Body).Decode(&challenge); err != nil {
		return nil, fmt.Errorf("failed to decode challenge response: %w", err)
	}

	if challenge.Algorithm == "" || challenge.Challenge == "" {
		return nil, fmt.Errorf("invalid challenge response: missing algorithm or challenge")
	}

	return &challenge, nil
}

func (c *Client) Provision(ctx context.Context, req ProvisionRequest) (*ProvisionResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	resp, err := c.doRequest(ctx, http.MethodPost, "/keys/provision", body)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, readErrorResponse(resp)
	}

	var provision ProvisionResponse
	if err := json.NewDecoder(resp.Body).Decode(&provision); err != nil {
		return nil, fmt.Errorf("failed to decode provision response: %w", err)
	}

	return &provision, nil
}

func (c *Client) doRequest(ctx context.Context, method, path string, body []byte) (*http.Response, error) {
	u, err := url.Parse(c.BaseURL)
	if err != nil {
		return nil, fmt.Errorf("invalid base URL: %w", err)
	}

	ref, err := url.Parse(path)
	if err != nil {
		return nil, err
	}
	fullURL := u.ResolveReference(ref).String()

	var reqBody io.Reader
	if body != nil {
		reqBody = bytes.NewReader(body)
	}

	req, err := http.NewRequestWithContext(ctx, method, fullURL, reqBody)
	if err != nil {
		return nil, err
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("User-Agent", useragent.GetEncodedUserAgent())
	req.Header.Set("X-Stripe-Client-User-Agent", useragent.GetEncodedStripeUserAgent())

	return c.HTTPClient.Do(req)
}

func readErrorResponse(resp *http.Response) error {
	body, _ := io.ReadAll(resp.Body)
	return &HTTPError{StatusCode: resp.StatusCode, Body: string(body)}
}

// SolveChallenge brute-forces the proof-of-work: finds n where SHA-256(salt + n) == challenge.
func SolveChallenge(ctx context.Context, algorithm, challenge, salt string) (int, error) {
	normalized := strings.ToLower(strings.ReplaceAll(algorithm, "-", ""))
	if normalized != "sha256" {
		return 0, fmt.Errorf("%w: got %q", ErrUnsupportedAlgorithm, algorithm)
	}

	target, err := hex.DecodeString(challenge)
	if err != nil {
		return 0, fmt.Errorf("invalid challenge hex: %w", err)
	}

	h := sha256.New()
	sumBuf := make([]byte, 0, sha256.Size)
	numBuf := make([]byte, 0, 20)

	for n := 0; n < maxIterations; n++ {
		if n&0xFFF == 0 {
			if err := ctx.Err(); err != nil {
				return 0, err
			}
		}

		h.Reset()
		h.Write([]byte(salt))
		h.Write(strconv.AppendInt(numBuf[:0], int64(n), 10))

		if bytes.Equal(h.Sum(sumBuf[:0]), target) {
			return n, nil
		}
	}

	return 0, ErrMaxIterationsReached
}

// GitConfigFunc resolves git config values. Replaceable in tests.
var GitConfigFunc = defaultGitConfig

func defaultGitConfig(key string) string {
	output, err := exec.Command("git", "config", key).Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(output))
}
