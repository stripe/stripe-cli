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
	"os/exec"
	"strconv"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"

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
	Number    int64  `json:"number"`
	Email     string `json:"email"`
	Name      string `json:"name,omitempty"`
}

// ProvisionResponse wraps the server response loosely so field additions
// or type changes don't break the CLI.
type ProvisionResponse struct {
	raw map[string]interface{}
}

func (r *ProvisionResponse) UnmarshalJSON(data []byte) error {
	return json.Unmarshal(data, &r.raw)
}

func (r *ProvisionResponse) getString(keys ...string) string {
	for _, k := range keys {
		if v, ok := r.raw[k].(string); ok && v != "" {
			return v
		}
	}
	return ""
}

func (r *ProvisionResponse) GetSecretKey() string {
	return r.getString("restricted_key", "secret_key")
}

func (r *ProvisionResponse) GetPublishableKey() string {
	return r.getString("publishable_key")
}

func (r *ProvisionResponse) GetClaimURL() string {
	return r.getString("claim_url")
}

func (r *ProvisionResponse) GetAccountID() string {
	return r.getString("account_id", "merchant_token")
}

func (r *ProvisionResponse) GetExpiresAt() string {
	v, ok := r.raw["expires_at"]
	if !ok {
		return ""
	}
	switch val := v.(type) {
	case string:
		if t, err := time.Parse(time.RFC3339, val); err == nil {
			return t.Format("2006-01-02")
		}
		return val
	case float64:
		return time.Unix(int64(val), 0).UTC().Format("2006-01-02")
	default:
		return ""
	}
}

type Client struct {
	BaseURL    string
	HTTPClient *http.Client
}

func NewClient(baseURL string) *Client {
	// Normalize: add https:// if no scheme, ensure trailing slash doesn't matter
	if !strings.HasPrefix(baseURL, "http://") && !strings.HasPrefix(baseURL, "https://") {
		baseURL = "https://" + baseURL
	}
	baseURL = strings.TrimRight(baseURL, "/")

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

	log.Debugf("sandbox: POST /keys/challenge email=%s", email)

	resp, err := c.doRequest(ctx, http.MethodPost, "/keys/challenge", body)
	if err != nil {
		log.WithFields(log.Fields{"error": err}).Debug("sandbox: challenge request failed")
		return nil, err
	}
	defer resp.Body.Close()

	actionID := getActionID(resp)

	if resp.StatusCode != http.StatusOK {
		httpErr := readErrorResponse(resp)
		log.WithFields(log.Fields{
			"status":    resp.StatusCode,
			"action_id": actionID,
			"error":     httpErr,
		}).Debug("sandbox: challenge returned non-200")
		return nil, httpErr
	}

	var challenge ChallengeResponse
	if err := json.NewDecoder(resp.Body).Decode(&challenge); err != nil {
		return nil, fmt.Errorf("failed to decode challenge response: %w", err)
	}

	if challenge.Algorithm == "" || challenge.Challenge == "" {
		return nil, fmt.Errorf("invalid challenge response: missing algorithm or challenge")
	}

	log.WithFields(log.Fields{
		"action_id": actionID,
		"algorithm": challenge.Algorithm,
	}).Debug("sandbox: challenge succeeded")

	return &challenge, nil
}

func (c *Client) Provision(ctx context.Context, req ProvisionRequest) (*ProvisionResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	log.Debugf("sandbox: POST /keys/provision email=%s", req.Email)

	resp, err := c.doRequest(ctx, http.MethodPost, "/keys/provision", body)
	if err != nil {
		log.WithFields(log.Fields{"error": err}).Debug("sandbox: provision request failed")
		return nil, err
	}
	defer resp.Body.Close()

	actionID := getActionID(resp)

	if resp.StatusCode != http.StatusOK {
		httpErr := readErrorResponse(resp)
		log.WithFields(log.Fields{
			"status":    resp.StatusCode,
			"action_id": actionID,
			"error":     httpErr,
		}).Debug("sandbox: provision returned non-200")
		return nil, httpErr
	}

	var provision ProvisionResponse
	if err := json.NewDecoder(resp.Body).Decode(&provision); err != nil {
		return nil, fmt.Errorf("failed to decode provision response: %w", err)
	}

	log.WithFields(log.Fields{
		"action_id":  actionID,
		"account_id": provision.GetAccountID(),
	}).Debug("sandbox: provision succeeded")

	return &provision, nil
}

func (c *Client) doRequest(ctx context.Context, method, path string, body []byte) (*http.Response, error) {
	fullURL := c.BaseURL + path

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
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	return &HTTPError{StatusCode: resp.StatusCode, Body: string(body)}
}

func getActionID(resp *http.Response) string {
	return resp.Header.Get("Stripe-Action-Id")
}

// SolveChallenge brute-forces the proof-of-work: finds n where SHA-256(salt + n) == challenge.
func SolveChallenge(ctx context.Context, algorithm, challenge, salt string) (int64, error) {
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

	for n := int64(0); n < maxIterations; n++ {
		if n&0xFFF == 0 {
			if err := ctx.Err(); err != nil {
				return 0, err
			}
		}

		h.Reset()
		h.Write([]byte(salt))
		h.Write(strconv.AppendInt(numBuf[:0], n, 10))

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
