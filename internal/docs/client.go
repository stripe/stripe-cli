package docs

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"runtime"
	"time"
)

// Page holds the content and metadata of a fetched documentation page.
type Page struct {
	// Content is the raw page body returned by docs.stripe.com.
	Content []byte
	// URL is the fully resolved URL including query parameters.
	URL string
	// FetchedAt is when the content was originally retrieved from docs.stripe.com.
	FetchedAt time.Time
}

const defaultBaseURL = "https://docs.stripe.com"

// Client fetches documentation pages and calls endpoints on docs.stripe.com.
type Client struct {
	http      *http.Client
	baseURL   string
	userAgent string
	cache     Cache
	logger    *slog.Logger
}

// ClientOption configures a Client.
type ClientOption func(*Client)

// WithBaseURL overrides the default docs.stripe.com base URL.
func WithBaseURL(u string) ClientOption { return func(c *Client) { c.baseURL = u } }

// WithHTTPClient sets a custom http.Client.
func WithHTTPClient(hc *http.Client) ClientOption { return func(c *Client) { c.http = hc } }

// WithCache enables response caching for FetchPage.
func WithCache(cache Cache) ClientOption { return func(c *Client) { c.cache = cache } }

// WithLogger sets a custom logger.
func WithLogger(logger *slog.Logger) ClientOption { return func(c *Client) { c.logger = logger } }

// NewClient creates a Client configured with the given plugin version.
func NewClient(version string) *Client {
	client := http.Client{Timeout: 10 * time.Second}

	return &Client{
		http:      &client,
		baseURL:   defaultBaseURL,
		userAgent: fmt.Sprintf("stripe-cli docs-plugin/%s (%s; %s; %s)", version, runtime.GOOS, runtime.GOARCH, runtime.Version()),
		logger:    slog.Default(),
	}
}

// WithOptions applies the given options and returns the Client for chaining.
func (c *Client) WithOptions(opts ...ClientOption) *Client {
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// FetchPage retrieves a documentation page as plain text, using cache when available.
//
//	page, err := c.FetchPage(ctx, "/payments/accept-a-payment", map[string]string{
//	    "api_version": "2024-06-30",
//	})
func (c *Client) FetchPage(ctx context.Context, route string, params map[string]string) (Page, error) {
	resolvedURL := c.buildURL(route, params)

	if c.cache != nil {
		if data, cachedAt, ok, err := c.cache.Get(resolvedURL); err != nil {
			c.logger.Error("cache read failed", "url", resolvedURL, "err", err)
			return Page{}, fmt.Errorf("docs: cache read: %w", err)
		} else if ok {
			c.logger.Debug("cache hit", "url", resolvedURL, "age", time.Since(cachedAt).Round(time.Second))
			return Page{
				Content:   data,
				URL:       resolvedURL,
				FetchedAt: cachedAt,
			}, nil
		}
		c.logger.Debug("cache miss", "url", resolvedURL)
	}

	body, err := c.do(ctx, resolvedURL)
	if err != nil {
		return Page{}, err
	}

	if c.cache != nil {
		if err := c.cache.Set(resolvedURL, body); err != nil {
			c.logger.Error("cache write failed", "url", resolvedURL, "err", err)
			return Page{}, fmt.Errorf("docs: cache write: %w", err)
		}
	}

	return Page{
		Content:   body,
		URL:       resolvedURL,
		FetchedAt: time.Now(),
	}, nil
}

func (c *Client) buildURL(route string, params map[string]string) string {
	u, _ := url.Parse(c.baseURL + route)
	q := u.Query()
	for k, v := range params {
		q.Set(k, v)
	}
	u.RawQuery = q.Encode()
	return u.String()
}

func (c *Client) do(ctx context.Context, rawURL string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, fmt.Errorf("docs: build request: %w", err)
	}
	req.Header.Set("User-Agent", c.userAgent)
	req.Header.Set("Accept", "text/plain")

	start := time.Now()
	resp, err := c.http.Do(req)
	if err != nil {
		c.logger.Error("request failed", "url", rawURL, "err", err)
		return nil, fmt.Errorf("docs: request failed: %w", err)
	}
	defer resp.Body.Close()

	c.logger.Debug("request complete", "url", rawURL, "status", resp.StatusCode, "duration", time.Since(start).Round(time.Millisecond))

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("docs: %s returned %d", rawURL, resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("docs: read body: %w", err)
	}

	return body, nil
}
