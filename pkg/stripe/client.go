// Package stripe provides the HTTP client for the Stripe API.
package stripe

import (
	"context"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/stripe/stripe-cli/pkg/useragent"
)

// APIVersion is API version used in CLI
const APIVersion = "2019-03-14"

// v1 and v2 API interop constants
const (
	V1ContentType = "application/x-www-form-urlencoded"
	V2ContentType = "application/json"
	V1Request     = "v1"
	V2Request     = "v2"
)

// Credentials holds the auth credentials for a Stripe API request. For plain
// API keys only Token is set. For OAK tokens all three fields are populated.
type Credentials struct {
	Token       string
	OAKContext  string
	OAKLivemode *bool
}

// ApplyAccountContextHeaders sets Stripe-Account and/or Stripe-Context in
// headers from the caller-supplied account and context values. For UAT
// credentials the OAK compartment ID is prepended as "context/value" unless
// the value already equals the compartment ID. When account is set under UAT,
// Stripe-Context is intentionally omitted.
func (c Credentials) ApplyAccountContextHeaders(headers http.Header, account, context string) {
	if c.OAKContext != "" {
		switch {
		case account != "":
			if c.OAKContext != account {
				headers.Set("Stripe-Account", c.OAKContext+"/"+account)
			} else {
				headers.Set("Stripe-Account", account)
			}
			headers.Del("Stripe-Context")
		case context != "":
			if c.OAKContext != context {
				headers.Set("Stripe-Context", c.OAKContext+"/"+context)
			} else {
				headers.Set("Stripe-Context", context)
			}
		default:
			headers.Set("Stripe-Context", c.OAKContext)
		}
	} else {
		switch {
		case account != "":
			headers.Set("Stripe-Account", account)
			if context != "" {
				headers.Set("Stripe-Context", context)
			}
		case context != "":
			headers.Set("Stripe-Context", context)
		}
	}
}

// NewAPIKeyCredentials returns Credentials using a standard Stripe API key.
func NewAPIKeyCredentials(key string) Credentials {
	return Credentials{Token: key}
}

// NewOAKCredentials returns Credentials for an OAK (User Access Token).
func NewOAKCredentials(token, context string, livemode bool) Credentials {
	return Credentials{Token: token, OAKContext: context, OAKLivemode: &livemode}
}

// SetRequestHeaders applies all auth-related headers to req.
func (c Credentials) SetRequestHeaders(req *http.Request) {
	if c.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.Token)
	}
	if c.OAKContext != "" {
		req.Header.Set("Stripe-Context", c.OAKContext)
	}
	if c.OAKLivemode != nil {
		req.Header.Set("Stripe-Livemode", strconv.FormatBool(*c.OAKLivemode))
	}
}

// Livemode reports the effective livemode for the credentials. For OAK tokens
// this is explicit; for plain API keys it is inferred from the key string.
func (c Credentials) Livemode() bool {
	if c.OAKLivemode != nil {
		return *c.OAKLivemode
	}
	return strings.Contains(c.Token, "live")
}

// Client is the API client used to send requests to Stripe.
type Client struct {
	// The base URL (protocol + hostname) used for all requests sent by this
	// client.
	BaseURL *url.URL

	// Credentials holds the auth credentials for the request. If Token is
	// empty, the Authorization header is omitted.
	Credentials Credentials

	// When this is enabled, request and response headers will be printed to
	// stdout.
	Verbose bool

	// List of request and response headers that should be printed when Verbose is true.
	// Defaults to the standard set of relevant for Stripe headers.
	VerbosePrintableHeaders []string

	// Cached HTTP client, lazily created the first time the Client is used to
	// send a request.
	httpClient *http.Client
}

// RequestPerformer is an interface for executing requests against the Stripe
// API, usually satisfied by providing a stripe.Client.
type RequestPerformer interface {
	PerformRequest(ctx context.Context, method, path string, params string, configure func(*http.Request) error) (*http.Response, error)
}

// PerformRequest sends a request to Stripe and returns the response.
func (c *Client) PerformRequest(ctx context.Context, method, path string, params string, configure func(*http.Request) error) (*http.Response, error) {
	url, err := url.Parse(path)
	if err != nil {
		return nil, err
	}

	url = c.BaseURL.ResolveReference(url)

	var body io.Reader
	if method == http.MethodPost {
		body = strings.NewReader(params)
	} else {
		url.RawQuery = params
	}

	req, err := http.NewRequest(method, url.String(), body)
	if err != nil {
		return nil, err
	}

	// if path starts with v1
	if IsV2Path(path) {
		req.Header.Set("Content-Type", V2ContentType)
	} else {
		req.Header.Set("Content-Type", V1ContentType)
	}
	req.Header.Set("Accept-Encoding", "identity")
	req.Header.Set("User-Agent", useragent.GetEncodedUserAgent())
	req.Header.Set("X-Stripe-Client-User-Agent", useragent.GetEncodedStripeUserAgent())

	c.Credentials.SetRequestHeaders(req)

	if configure != nil {
		if err := configure(req); err != nil {
			return nil, err
		}
	}

	if c.httpClient == nil {
		c.httpClient = newHTTPClient(c.Verbose, c.VerbosePrintableHeaders, os.Getenv("STRIPE_CLI_UNIX_SOCKET"))
	}

	if ctx != nil {
		req = req.WithContext(ctx)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}

	// RequestID of the API Request
	requestID := resp.Header.Get("Request-Id")
	go sendTelemetryEvent(ctx, requestID, c.Credentials.Livemode())
	return resp, nil
}

func sendTelemetryEvent(ctx context.Context, requestID string, livemode bool) {
	telemetryClient := GetTelemetryClient(ctx)
	if telemetryClient != nil {
		resp, err := telemetryClient.SendAPIRequestEvent(ctx, requestID, livemode)
		// Don't throw exception if we fail to send the event
		if err != nil {
			log.Debugf("Error while sending telemetry data: %v\n", err)
		}
		if resp != nil {
			resp.Body.Close()
		}
	}
}

func newHTTPClient(verbose bool, printableHeaders []string, unixSocket string) *http.Client {
	var httpTransport http.RoundTripper

	if unixSocket != "" {
		dialFunc := func(network, addr string) (net.Conn, error) {
			return net.Dial("unix", unixSocket)
		}
		dialContext := func(_ context.Context, _, _ string) (net.Conn, error) {
			return net.Dial("unix", unixSocket)
		}
		httpTransport = &http.Transport{
			DialContext:           dialContext,
			DialTLS:               dialFunc,
			ResponseHeaderTimeout: 30 * time.Second,
			ExpectContinueTimeout: 10 * time.Second,
			TLSHandshakeTimeout:   10 * time.Second,
		}
	} else {
		httpTransport = &http.Transport{
			Proxy: http.ProxyFromEnvironment,
			DialContext: (&net.Dialer{
				Timeout:   30 * time.Second,
				KeepAlive: 30 * time.Second,
			}).DialContext,
			TLSHandshakeTimeout: 10 * time.Second,
		}
	}

	if verbose {
		if printableHeaders == nil {
			printableHeaders = inspectHeaders
		}

		httpTransport = &verboseTransport{
			Transport:        httpTransport,
			Out:              os.Stderr,
			PrintableHeaders: printableHeaders,
		}
	}

	return &http.Client{
		Transport: httpTransport,
	}
}

// IsV2Path checks if the path is for V1 API
func IsV2Path(path string) bool {
	return strings.HasPrefix(path, "/"+V2Request)
}
