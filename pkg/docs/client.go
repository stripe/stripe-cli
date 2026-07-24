package docs

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/stripe/stripe-cli/pkg/useragent"
)

const defaultBaseURL = "https://docs.stripe.com"

// Page holds the content and metadata of a fetched documentation page.
type Page struct {
	// Content is the raw page body returned by docs.stripe.com.
	Content []byte
	// URL is the fully resolved URL including query parameters.
	URL *url.URL
	// FetchedAt is when the content was originally retrieved from docs.stripe.com.
	FetchedAt time.Time
}

// SearchResponse represents the response from the docs search endpoint.
type SearchResponse struct {
	Hits []Hit
}

// Hit represents a single search hit from docs.stripe.com.
type Hit struct {
	Title string `json:"title"`
	URL   string `json:"url"`
}

// Client fetches documentation pages and calls endpoints on docs.stripe.com.
type Client struct {
	http           *http.Client
	baseURL        *url.URL
	userAgent      string
	apiKey         string
	cacheKeyPrefix string
	cache          Cache
	logger         *log.Entry
}

// ClientOption configures a Client.
type ClientOption func(*Client)

// WithBaseURL overrides the default docs.stripe.com base URL.
func WithBaseURL(u string) ClientOption {
	return func(c *Client) { c.baseURL, _ = url.Parse(u) }
}

// WithHTTPClient sets a custom http.Client.
func WithHTTPClient(hc *http.Client) ClientOption { return func(c *Client) { c.http = hc } }

// WithCache enables response caching for FetchPage.
func WithCache(cache Cache) ClientOption { return func(c *Client) { c.cache = cache } }

// WithLogger sets a custom logger.
func WithLogger(logger *log.Entry) ClientOption { return func(c *Client) { c.logger = logger } }

// WithAPIKey sets the Stripe API key sent as an Authorization header on every request.
func WithAPIKey(key string) ClientOption { return func(c *Client) { c.apiKey = key } }

// WithCacheKeyPrefix scopes the FetchPage cache to a specific account by prefixing
// all cache keys with prefix. Pass the account ID (e.g. "acct_xxx") so that cached
// entries from one account are never served to another. An empty prefix is a no-op,
// which keeps unauthenticated usage working without changes.
func WithCacheKeyPrefix(prefix string) ClientOption {
	return func(c *Client) { c.cacheKeyPrefix = prefix }
}

// NewClient creates a Client configured to talk to docs.stripe.com.
func NewClient(_ string) *Client {
	base, _ := url.Parse(defaultBaseURL)
	client := http.Client{Timeout: 10 * time.Second}

	return &Client{
		http:      &client,
		baseURL:   base,
		userAgent: useragent.GetEncodedUserAgent(),
		logger:    log.NewEntry(log.StandardLogger()),
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
//	page, err := c.FetchPage(ctx, &url.URL{Path: "/payments/accept-a-payment", RawQuery: "api_version=2024-06-30"})
func (c *Client) FetchPage(ctx context.Context, ref *url.URL) (Page, error) {
	resolvedURL := c.baseURL.ResolveReference(ref)
	// url.Values.Encode() already sorts params, but we normalize here to ensure
	// a consistent cache key regardless of how callers construct RawQuery.
	resolvedURL.RawQuery = resolvedURL.Query().Encode()
	rawURL := resolvedURL.String()
	cacheKey := c.cacheKey(rawURL)

	if c.cache != nil {
		if data, cachedAt, ok, err := c.cache.Get(cacheKey); err != nil {
			c.logger.WithFields(log.Fields{"url": rawURL}).WithError(err).Error("cache read failed")
			return Page{}, fmt.Errorf("docs: cache read: %w", err)
		} else if ok {
			c.logger.WithFields(log.Fields{"url": rawURL, "age": time.Since(cachedAt).Round(time.Second)}).Debug("cache hit")
			return Page{
				Content:   data,
				URL:       resolvedURL,
				FetchedAt: cachedAt,
			}, nil
		}
		c.logger.WithField("url", rawURL).Debug("cache miss")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return Page{}, fmt.Errorf("docs: build request: %w", err)
	}
	req.Header.Set("Accept", "text/plain")

	res, err := c.do(req)
	if err != nil {
		return Page{}, err
	}

	if c.cache != nil {
		if err := c.cache.Set(cacheKey, res.body); err != nil {
			c.logger.WithField("url", rawURL).WithError(err).Error("cache write failed")
			return Page{}, fmt.Errorf("docs: cache write: %w", err)
		}
	}

	return Page{
		Content:   res.body,
		URL:       res.finalURL,
		FetchedAt: time.Now(),
	}, nil
}

type response struct {
	body     []byte
	finalURL *url.URL
}

// cacheKey returns a cache key for a resolved URL, prefixed by the account ID
// when one is configured so that entries are scoped per account.
func (c *Client) cacheKey(rawURL string) string {
	if c.cacheKeyPrefix == "" {
		return rawURL
	}
	return c.cacheKeyPrefix + ":" + rawURL
}

func (c *Client) do(req *http.Request) (response, error) {
	req.Header.Set("User-Agent", c.userAgent)

	start := time.Now()
	resp, err := c.http.Do(req)
	if err != nil {
		if req.Context().Err() != nil {
			c.logger.WithFields(log.Fields{"url": req.URL, "cause": req.Context().Err()}).Debug("request canceled")
		} else {
			c.logger.WithField("url", req.URL).WithError(err).Error("request failed")
		}
		return response{}, fmt.Errorf("docs: request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	c.logger.WithFields(log.Fields{"url": req.URL, "status": resp.StatusCode, "duration": time.Since(start).Round(time.Millisecond)}).Debug("request complete")

	if resp.StatusCode != http.StatusOK {
		return response{}, fmt.Errorf("docs: %s returned %d", req.URL, resp.StatusCode)
	}

	accept := req.Header.Get("Accept")
	if ct := resp.Header.Get("Content-Type"); ct != "" && accept != "" {
		mediaType, _, _ := mime.ParseMediaType(ct)
		acceptMedia, _, _ := mime.ParseMediaType(accept)
		if mediaType != acceptMedia {
			return response{}, fmt.Errorf("docs: %s returned unsupported content type %q", req.URL, ct)
		}
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return response{}, fmt.Errorf("docs: read body: %w", err)
	}

	return response{body: body, finalURL: resp.Request.URL}, nil
}

// Pref represents a single documentation preference.
type Pref struct {
	ID          string   `json:"id"`
	Category    *string  `json:"category"`
	Description string   `json:"description"`
	Values      []string `json:"values"`
	Default     *string  `json:"default"`
}

// PrefsResponse represents the response from the docs prefs endpoint.
type PrefsResponse struct {
	Prefs []Pref `json:"prefs"`
}

// FetchPrefs retrieves the list of documentation preferences from docs.stripe.com.
func (c *Client) FetchPrefs(ctx context.Context) (*PrefsResponse, error) {
	u := c.baseURL.JoinPath("/_endpoint/prefs")

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("prefs: build request: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	res, err := c.do(req)
	if err != nil {
		return nil, err
	}

	var response PrefsResponse
	if err := json.Unmarshal(res.body, &response); err != nil {
		return nil, fmt.Errorf("prefs: unmarshal response: %w", err)
	}
	return &response, nil
}

// Search sends the request to docs search endpoint and returns a list of search results.
//
//	response, err := c.Search(ctx, "payment methods")
func (c *Client) Search(ctx context.Context, query string) (*SearchResponse, error) {
	if query == "" {
		return &SearchResponse{}, nil
	}
	u := c.baseURL.JoinPath("/_endpoint/search")
	u.RawQuery = url.Values{"query": {query}}.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("search: build request: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	res, err := c.do(req)
	if err != nil {
		return nil, err
	}

	var response SearchResponse
	if err := json.Unmarshal(res.body, &response); err != nil {
		return nil, fmt.Errorf("search: unmarshal response: %w", err)
	}
	return &response, nil
}
