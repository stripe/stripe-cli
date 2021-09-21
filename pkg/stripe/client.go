package stripe

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/stripe/stripe-cli/pkg/useragent"
)

// DefaultAPIBaseURL is the default base URL for API requests
const DefaultAPIBaseURL = "https://api.stripe.com"

// DefaultDashboardBaseURL is the default base URL for dashboard requests
const DefaultDashboardBaseURL = "https://dashboard.stripe.com"

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

	// Cached HTTP client, lazily created the first time the Client is used to
	// send a request.
	httpClient *http.Client
}

// PerformRequest sends a request to Stripe and returns the response.
func (c *Client) PerformRequest(ctx context.Context, method, path string, params string, configure func(*http.Request)) (*http.Response, error) {
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
		configure(req)
	}

	if c.httpClient == nil {
		c.httpClient = newHTTPClient(c.Verbose, os.Getenv("STRIPE_CLI_UNIX_SOCKET"))
	}

	if ctx != nil {
		req = req.WithContext(ctx)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}

	fmt.Printf("Sending telemetry event\n")
	// Need to wait for this process to finish, but not at this level.
	livemode := strings.Contains(c.APIKey, "live")
	sendTelemetryEvent(ctx, resp, livemode)
	fmt.Printf("returning response\n")
	return resp, nil
}

func sendTelemetryEvent(ctx context.Context, response *http.Response, livemode bool) {
	// RequestID of the API Request
	requestID := response.Header.Get("Request-Id")
	// Don't throw exception if we fail to send the event
	// Also do this asynchronously.
	// resp, err := SendAPIRequestEvent(ctx, requestID, livemode)
	SendAPIRequestEvent(ctx, requestID, livemode)
	// fmt.Printf("Response: %v\n\n", resp)

	// res, _ := httputil.DumpResponse(resp, true)
	// fmt.Printf("Response Dump: %v\n", string(res))

	// req, _ := httputil.DumpRequest(resp.Request, true)

	// fmt.Printf("Request: %v\n", string(req))

	// if err != nil {
	// 	fmt.Printf("Error: %v\n", err)
	// }
	// defer resp.Body.Close()
}

func newHTTPClient(verbose bool, unixSocket string) *http.Client {
	var httpTransport *http.Transport

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

	tr := &verboseTransport{
		Transport: httpTransport,
		Verbose:   verbose,
		Out:       os.Stderr,
	}

	return &http.Client{
		Transport: tr,
	}
}
