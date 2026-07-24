package docs

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync"
	"testing"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/stripe/stripe-cli/pkg/useragent"
)

// capturingHook collects logrus entries for assertions.
type capturingHook struct {
	mu      sync.Mutex
	entries []*log.Entry
}

func (h *capturingHook) Levels() []log.Level { return log.AllLevels }

func (h *capturingHook) Fire(e *log.Entry) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.entries = append(h.entries, e)
	return nil
}

// newTestLogger returns a logrus logger wired to h, plus an entry for injection.
func newTestLogger(h *capturingHook) *log.Entry {
	l := log.New()
	l.SetLevel(log.DebugLevel)
	l.AddHook(h)
	return log.NewEntry(l)
}

type mockCache struct {
	data map[string][]byte
	hits int
}

func newMockCache() *mockCache {
	return &mockCache{data: make(map[string][]byte)}
}

func (m *mockCache) Get(key string) ([]byte, time.Time, bool, error) {
	if d, ok := m.data[key]; ok {
		m.hits++
		return d, time.Now(), true, nil
	}
	return nil, time.Time{}, false, nil
}

func (m *mockCache) Set(key string, data []byte) error {
	m.data[key] = data
	return nil
}

func TestNewClient_UserAgent(t *testing.T) {
	got := NewClient("1.2.3")
	assert.Equal(t, useragent.GetEncodedUserAgent(), got.userAgent)
}

func TestNewClient_Defaults(t *testing.T) {
	got := NewClient("0.1.0")
	assert.Equal(t, defaultBaseURL, got.baseURL.String())
	assert.Nil(t, got.cache)
	assert.NotNil(t, got.http)
}

func TestWithOptions(t *testing.T) {
	tests := []struct {
		name      string
		opts      []ClientOption
		wantCheck func(t *testing.T, c *Client)
	}{
		{
			name: "override base URL",
			opts: []ClientOption{WithBaseURL("https://example.com")},
			wantCheck: func(t *testing.T, c *Client) {
				assert.Equal(t, "https://example.com", c.baseURL.String())
			},
		},
		{
			name: "set custom HTTP client",
			opts: []ClientOption{WithHTTPClient(&http.Client{Timeout: 99 * time.Second})},
			wantCheck: func(t *testing.T, c *Client) {
				assert.Equal(t, 99*time.Second, c.http.Timeout)
			},
		},
		{
			name: "multiple options applied in order",
			opts: []ClientOption{WithBaseURL("https://first.com"), WithBaseURL("https://second.com")},
			wantCheck: func(t *testing.T, c *Client) {
				assert.Equal(t, "https://second.com", c.baseURL.String())
			},
		},
		{
			name: "set API key",
			opts: []ClientOption{WithAPIKey("sk_test_abc123")},
			wantCheck: func(t *testing.T, c *Client) {
				assert.Equal(t, "sk_test_abc123", c.apiKey)
			},
		},
		{
			name: "set cache key prefix",
			opts: []ClientOption{WithCacheKeyPrefix("acct_123")},
			wantCheck: func(t *testing.T, c *Client) {
				assert.Equal(t, "acct_123", c.cacheKeyPrefix)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewClient("0.1.0").WithOptions(tt.opts...)
			tt.wantCheck(t, got)
		})
	}
}

func TestFetchPage(t *testing.T) {
	tests := []struct {
		name      string
		ref       *url.URL
		handler   http.HandlerFunc
		wantErr   string
		wantCheck func(t *testing.T, got Page)
	}{
		{
			name: "success with params",
			ref:  &url.URL{Path: "/payments", RawQuery: "api_version=2024-06-30"},
			handler: func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "text/plain", r.Header.Get("Accept"))
				assert.Contains(t, r.Header.Get("User-Agent"), "stripe-cli/")
				assert.Equal(t, "/payments", r.URL.Path)
				assert.Equal(t, "2024-06-30", r.URL.Query().Get("api_version"))
				fmt.Fprint(w, "page content")
			},
			wantCheck: func(t *testing.T, got Page) {
				assert.Equal(t, []byte("page content"), got.Content)
				assert.Contains(t, got.URL.String(), "/payments?api_version=2024-06-30")
				assert.WithinDuration(t, time.Now(), got.FetchedAt, 2*time.Second)
			},
		},
		{
			name: "success with no params",
			ref:  &url.URL{Path: "/overview"},
			handler: func(w http.ResponseWriter, r *http.Request) {
				assert.Empty(t, r.URL.RawQuery)
				fmt.Fprint(w, "overview")
			},
			wantCheck: func(t *testing.T, got Page) {
				assert.Equal(t, []byte("overview"), got.Content)
			},
		},
		{
			name: "404 returns error",
			ref:  &url.URL{Path: "/missing"},
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusNotFound)
			},
			wantErr: "returned 404",
		},
		{
			name: "500 returns error",
			ref:  &url.URL{Path: "/error"},
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
			},
			wantErr: "returned 500",
		},
		{
			name: "unsupported content type returns error",
			ref:  &url.URL{Path: "/html"},
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "text/html; charset=utf-8")
				fmt.Fprint(w, "<html></html>")
			},
			wantErr: "unsupported content type",
		},
		{
			name: "text/plain with charset is accepted",
			ref:  &url.URL{Path: "/plain"},
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "text/plain; charset=utf-8")
				fmt.Fprint(w, "plain content")
			},
			wantCheck: func(t *testing.T, got Page) {
				assert.Equal(t, []byte("plain content"), got.Content)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(tt.handler)
			defer server.Close()

			client := NewClient("0.1.0").WithOptions(WithBaseURL(server.URL))
			got, err := client.FetchPage(context.Background(), tt.ref)

			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
				return
			}

			require.NoError(t, err)
			if tt.wantCheck != nil {
				tt.wantCheck(t, got)
			}
		})
	}
}

func TestFetchPage_ContextCanceled(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(5 * time.Second)
	}))
	defer server.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	client := NewClient("0.1.0").WithOptions(WithBaseURL(server.URL))
	_, err := client.FetchPage(ctx, &url.URL{Path: "/slow"})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "context canceled")
}

func TestFetchPage_ContextCanceled_LoggedAtDebug(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(5 * time.Second)
	}))
	defer server.Close()

	h := &capturingHook{}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	client := NewClient("0.1.0").WithOptions(WithBaseURL(server.URL), WithLogger(newTestLogger(h)))
	_, err := client.FetchPage(ctx, &url.URL{Path: "/slow"})
	require.Error(t, err)

	h.mu.Lock()
	defer h.mu.Unlock()
	for _, e := range h.entries {
		assert.GreaterOrEqual(t, e.Level, log.WarnLevel, "context cancellation must not produce an error-level log")
	}
	var foundCause bool
	for _, e := range h.entries {
		if e.Level == log.DebugLevel {
			if _, ok := e.Data["cause"]; ok {
				foundCause = true
			}
		}
	}
	assert.True(t, foundCause, "expected a debug log with a cause field")
}

func TestFetchPage_CacheHit(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		fmt.Fprint(w, "fresh content")
	}))
	defer server.Close()

	cache := newMockCache()
	client := NewClient("0.1.0").WithOptions(WithBaseURL(server.URL), WithCache(cache))

	got, err := client.FetchPage(context.Background(), &url.URL{Path: "/cached"})
	require.NoError(t, err)
	assert.Equal(t, []byte("fresh content"), got.Content)
	assert.WithinDuration(t, time.Now(), got.FetchedAt, 2*time.Second)
	assert.Equal(t, 1, calls)

	got, err = client.FetchPage(context.Background(), &url.URL{Path: "/cached"})
	require.NoError(t, err)
	assert.Equal(t, []byte("fresh content"), got.Content)
	assert.WithinDuration(t, time.Now(), got.FetchedAt, 2*time.Second)
	assert.Equal(t, 1, calls)
	assert.Equal(t, 1, cache.hits)
}

func TestFetchPage_CacheMissRefetches(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		fmt.Fprintf(w, "response %d", calls)
	}))
	defer server.Close()

	cache := &evictingMockCache{}
	client := NewClient("0.1.0").WithOptions(WithBaseURL(server.URL), WithCache(cache))

	got, err := client.FetchPage(context.Background(), &url.URL{Path: "/expiring"})
	require.NoError(t, err)
	assert.Equal(t, []byte("response 1"), got.Content)
	assert.WithinDuration(t, time.Now(), got.FetchedAt, 2*time.Second)

	got, err = client.FetchPage(context.Background(), &url.URL{Path: "/expiring"})
	require.NoError(t, err)
	assert.Equal(t, []byte("response 2"), got.Content)
	assert.WithinDuration(t, time.Now(), got.FetchedAt, 2*time.Second)
	assert.Equal(t, 2, calls)
}

func TestFetchPage_DifferentParamsCacheSeparately(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		fmt.Fprintf(w, "response for %s", r.URL.RawQuery)
	}))
	defer server.Close()

	cache := newMockCache()
	client := NewClient("0.1.0").WithOptions(WithBaseURL(server.URL), WithCache(cache))

	_, err := client.FetchPage(context.Background(), &url.URL{Path: "/api", RawQuery: "v=1"})
	require.NoError(t, err)

	_, err = client.FetchPage(context.Background(), &url.URL{Path: "/api", RawQuery: "v=2"})
	require.NoError(t, err)

	assert.Equal(t, 2, calls)
}

func TestFetchPage_ParamOrderNormalized(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		fmt.Fprint(w, "content")
	}))
	defer server.Close()

	cache := newMockCache()
	client := NewClient("0.1.0").WithOptions(WithBaseURL(server.URL), WithCache(cache))

	_, err := client.FetchPage(context.Background(), &url.URL{Path: "/api", RawQuery: "lang=go&api_version=2024-06-30"})
	require.NoError(t, err)

	_, err = client.FetchPage(context.Background(), &url.URL{Path: "/api", RawQuery: "api_version=2024-06-30&lang=go"})
	require.NoError(t, err)

	assert.Equal(t, 1, calls, "second call should hit cache despite different param order")
	assert.Equal(t, 1, cache.hits)
}

func TestSearch(t *testing.T) {
	tests := []struct {
		name      string
		query     string
		handler   http.HandlerFunc
		wantErr   string
		wantCheck func(t *testing.T, got *SearchResponse)
	}{
		{
			name:  "success",
			query: "payments search",
			handler: func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "/_endpoint/search", r.URL.Path)
				assert.Equal(t, "payments search", r.URL.Query().Get("query"))
				assert.Equal(t, "application/json", r.Header.Get("Accept"))
				w.Header().Set("Content-Type", "application/json")
				fmt.Fprint(w, `{"hits":[{"title":"Accept a payment","url":"https://docs.stripe.com/payments/accept-a-payment"}]}`)
			},
			wantCheck: func(t *testing.T, got *SearchResponse) {
				require.Len(t, got.Hits, 1)
				assert.Equal(t, "Accept a payment", got.Hits[0].Title)
				assert.Equal(t, "https://docs.stripe.com/payments/accept-a-payment", got.Hits[0].URL)
			},
		},
		{
			name:  "empty query returns empty response without calling server",
			query: "",
			handler: func(w http.ResponseWriter, r *http.Request) {
				t.Error("server should not be called for empty query")
			},
			wantCheck: func(t *testing.T, got *SearchResponse) {
				assert.Empty(t, got.Hits)
			},
		},
		{
			name:  "empty search response",
			query: "no matches",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				fmt.Fprint(w, `{"hits":[]}`)
			},
			wantCheck: func(t *testing.T, got *SearchResponse) {
				assert.Empty(t, got.Hits)
			},
		},
		{
			name:  "non-200 response returns error",
			query: "fails",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusBadGateway)
			},
			wantErr: "returned 502",
		},
		{
			name:  "invalid json returns error",
			query: "bad json",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				fmt.Fprint(w, `{"hits":[`)
			},
			wantErr: "search: unmarshal response",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(tt.handler)
			defer server.Close()

			client := NewClient("0.1.0").WithOptions(WithBaseURL(server.URL))
			got, err := client.Search(context.Background(), tt.query)

			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
				return
			}

			require.NoError(t, err)
			if tt.wantCheck != nil {
				tt.wantCheck(t, got)
			}
		})
	}
}

func strPtr(s string) *string { return &s }

func TestFetchPrefs(t *testing.T) {
	tests := []struct {
		name      string
		handler   http.HandlerFunc
		wantErr   string
		wantCheck func(t *testing.T, got *PrefsResponse)
	}{
		{
			name: "success with multiple prefs",
			handler: func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "/_endpoint/prefs", r.URL.Path)
				assert.Equal(t, "application/json", r.Header.Get("Accept"))
				w.Header().Set("Content-Type", "application/json")
				fmt.Fprint(w, `{"prefs":[{"id":"lang","category":"code","description":"Programming language","values":["ruby","python"],"default":"ruby"},{"id":"theme","category":null,"description":"Color theme","values":["light","dark"],"default":null}]}`)
			},
			wantCheck: func(t *testing.T, got *PrefsResponse) {
				require.Len(t, got.Prefs, 2)
				assert.Equal(t, "lang", got.Prefs[0].ID)
				assert.Equal(t, strPtr("code"), got.Prefs[0].Category)
				assert.Equal(t, "Programming language", got.Prefs[0].Description)
				assert.Equal(t, []string{"ruby", "python"}, got.Prefs[0].Values)
				assert.Equal(t, strPtr("ruby"), got.Prefs[0].Default)
				assert.Equal(t, "theme", got.Prefs[1].ID)
				assert.Nil(t, got.Prefs[1].Category)
				assert.Nil(t, got.Prefs[1].Default)
			},
		},
		{
			name: "empty prefs list",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				fmt.Fprint(w, `{"prefs":[]}`)
			},
			wantCheck: func(t *testing.T, got *PrefsResponse) {
				assert.Empty(t, got.Prefs)
			},
		},
		{
			name: "non-200 response returns error",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
			},
			wantErr: "returned 500",
		},
		{
			name: "invalid json returns error",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				fmt.Fprint(w, `{"prefs":[`)
			},
			wantErr: "prefs: unmarshal response",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(tt.handler)
			defer server.Close()

			client := NewClient("0.1.0").WithOptions(WithBaseURL(server.URL))
			got, err := client.FetchPrefs(context.Background())

			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
				return
			}

			require.NoError(t, err)
			if tt.wantCheck != nil {
				tt.wantCheck(t, got)
			}
		})
	}
}

func TestFetchPage_CacheKeyPrefix_ScopesByAccount(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		fmt.Fprintf(w, "response %d", calls)
	}))
	defer server.Close()

	cache := newMockCache()
	ref := &url.URL{Path: "/payments"}

	acct1 := NewClient("0.1.0").WithOptions(WithBaseURL(server.URL), WithCache(cache), WithCacheKeyPrefix("acct_111"))
	acct2 := NewClient("0.1.0").WithOptions(WithBaseURL(server.URL), WithCache(cache), WithCacheKeyPrefix("acct_222"))

	got1, err := acct1.FetchPage(context.Background(), ref)
	require.NoError(t, err)
	assert.Equal(t, []byte("response 1"), got1.Content)

	// same URL, different prefix — must not hit acct1's cache entry
	got2, err := acct2.FetchPage(context.Background(), ref)
	require.NoError(t, err)
	assert.Equal(t, []byte("response 2"), got2.Content)

	assert.Equal(t, 2, calls)
	assert.Equal(t, 0, cache.hits)
}

func TestFetchPage_CacheKeyPrefix_HitsWithinSameAccount(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		fmt.Fprint(w, "content")
	}))
	defer server.Close()

	cache := newMockCache()
	ref := &url.URL{Path: "/payments"}

	client := NewClient("0.1.0").WithOptions(WithBaseURL(server.URL), WithCache(cache), WithCacheKeyPrefix("acct_111"))

	_, err := client.FetchPage(context.Background(), ref)
	require.NoError(t, err)

	_, err = client.FetchPage(context.Background(), ref)
	require.NoError(t, err)

	assert.Equal(t, 1, calls)
	assert.Equal(t, 1, cache.hits)
}

func TestFetchPage_EmptyCacheKeyPrefix_UsesRawURL(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		fmt.Fprint(w, "content")
	}))
	defer server.Close()

	cache := newMockCache()
	ref := &url.URL{Path: "/payments"}

	// WithCacheKeyPrefix("") is a no-op — behaves identically to not setting a prefix
	client := NewClient("0.1.0").WithOptions(WithBaseURL(server.URL), WithCache(cache), WithCacheKeyPrefix(""))

	_, err := client.FetchPage(context.Background(), ref)
	require.NoError(t, err)

	_, err = client.FetchPage(context.Background(), ref)
	require.NoError(t, err)

	assert.Equal(t, 1, calls)
	assert.Equal(t, 1, cache.hits)
}

// evictingMockCache always returns a miss, simulating TTL expiry.
type evictingMockCache struct{}

func (m *evictingMockCache) Get(key string) ([]byte, time.Time, bool, error) {
	return nil, time.Time{}, false, nil
}

func (m *evictingMockCache) Set(key string, data []byte) error {
	return nil
}
