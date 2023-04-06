package stripe

import (
	"context"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/stripe/stripe-cli/pkg/useragent"
)

// DefaultAPIBaseURL is the default base URL for API requests
const DefaultAPIBaseURL = "https://api.stripe.com"

// DefaultFilesAPIBaseURL is the default base URL for Files API requsts
const DefaultFilesAPIBaseURL = "https://files.stripe.com"

// DefaultDashboardBaseURL is the default base URL for dashboard requests
const DefaultDashboardBaseURL = "https://dashboard.stripe.com"

// APIVersion is API version used in CLI
const APIVersion = "2019-03-14"

// Client is the API client used to sent requests to Stripe.
type Client struct {
	// The base URL (protocol + hostname) used for all requests sent by this
	// client.
	BaseURL *url.URL

	// API key used to authenticate requests sent by this client. If left
	// empty, the `Authorization` header will be omitted.
	APIKey string

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

	req.Header.Set("Accept-Encoding", "identity")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", useragent.GetEncodedUserAgent())
	req.Header.Set("X-Stripe-Client-User-Agent", useragent.GetEncodedStripeUserAgent())

	if c.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.APIKey)
	}

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
	livemode := strings.Contains(c.APIKey, "live")
	go sendTelemetryEvent(ctx, requestID, livemode)
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
