package docs

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/stripe/stripe-cli-docs-plugin/internal/agentskills"
)

// capturingHandler collects slog records for assertions.
type capturingHandler struct {
	mu      sync.Mutex
	records []slog.Record
}

func (h *capturingHandler) Enabled(_ context.Context, _ slog.Level) bool { return true }

func (h *capturingHandler) Handle(_ context.Context, r slog.Record) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.records = append(h.records, r)
	return nil
}

func (h *capturingHandler) WithAttrs(_ []slog.Attr) slog.Handler { return h }
func (h *capturingHandler) WithGroup(_ string) slog.Handler      { return h }

func (h *capturingHandler) maxLevel() (slog.Level, bool) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if len(h.records) == 0 {
		return 0, false
	}
	max := h.records[0].Level
	for _, r := range h.records[1:] {
		if r.Level > max {
			max = r.Level
		}
	}
	return max, true
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
	want := fmt.Sprintf("stripe-cli docs-plugin/1.2.3 (%s; %s; %s)", runtime.GOOS, runtime.GOARCH, runtime.Version())
	assert.Equal(t, want, got.userAgent)
}

func TestWithAgent(t *testing.T) {
	base := fmt.Sprintf("stripe-cli docs-plugin/1.2.3 (%s; %s; %s)", runtime.GOOS, runtime.GOARCH, runtime.Version())
	tests := []struct {
		name  string
		agent string
		want  string
	}{
		{"appends agent", "claude_code", base + " AIAgent/claude_code"},
		{"empty agent is no-op", "", base},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewClient("1.2.3").WithOptions(WithAgent(tt.agent))
			assert.Equal(t, tt.want, got.userAgent)
		})
	}
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
				assert.Contains(t, r.Header.Get("User-Agent"), "stripe-cli docs-plugin/")
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

	h := &capturingHandler{}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	client := NewClient("0.1.0").WithOptions(WithBaseURL(server.URL), WithLogger(slog.New(h)))
	_, err := client.FetchPage(ctx, &url.URL{Path: "/slow"})
	require.Error(t, err)

	if level, ok := h.maxLevel(); ok {
		assert.Less(t, level, slog.LevelError, "context cancellation must not produce an error-level log")
	}

	h.mu.Lock()
	defer h.mu.Unlock()
	var found bool
	for _, r := range h.records {
		if r.Level != slog.LevelDebug {
			continue
		}
		r.Attrs(func(a slog.Attr) bool {
			if a.Key == "cause" {
				found = true
				return false
			}
			return true
		})
	}
	assert.True(t, found, "expected a debug log with a cause attribute")
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

func TestFetchSkills(t *testing.T) {
	tests := []struct {
		name      string
		handler   http.HandlerFunc
		wantErr   string
		wantCheck func(t *testing.T, got *agentskills.Index)
	}{
		{
			name: "success",
			handler: func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "/.well-known/skills/index.json", r.URL.Path)
				assert.Equal(t, "application/json", r.Header.Get("Accept"))
				assert.Contains(t, r.Header.Get("User-Agent"), "stripe-cli docs-plugin/")
				w.Header().Set("Content-Type", "application/json")
				fmt.Fprint(w, `{"skills":[{"name":"stripe-best-practices","description":"Guides Stripe integration decisions","files":["SKILL.md","references/payments.md"]}]}`)
			},
			wantCheck: func(t *testing.T, got *agentskills.Index) {
				require.Len(t, got.Skills, 1)
				assert.Equal(t, "stripe-best-practices", got.Skills[0].Name)
				assert.Equal(t, "Guides Stripe integration decisions", got.Skills[0].Description)
				assert.Equal(t, []string{"SKILL.md", "references/payments.md"}, got.Skills[0].Files)
			},
		},
		{
			name: "multiple skills",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				fmt.Fprint(w, `{"skills":[{"name":"skill-a","description":"desc a","files":["a.md"]},{"name":"skill-b","description":"desc b","files":["b.md"]}]}`)
			},
			wantCheck: func(t *testing.T, got *agentskills.Index) {
				require.Len(t, got.Skills, 2)
				assert.Equal(t, "skill-a", got.Skills[0].Name)
				assert.Equal(t, "skill-b", got.Skills[1].Name)
			},
		},
		{
			name: "empty skills list",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				fmt.Fprint(w, `{"skills":[]}`)
			},
			wantCheck: func(t *testing.T, got *agentskills.Index) {
				assert.Empty(t, got.Skills)
			},
		},
		{
			name: "non-200 response returns error",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusNotFound)
			},
			wantErr: "returned 404",
		},
		{
			name: "invalid json returns error",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				fmt.Fprint(w, `{"skills":[`)
			},
			wantErr: "skills: unmarshal response",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(tt.handler)
			defer server.Close()

			client := NewClient("0.1.0").WithOptions(WithBaseURL(server.URL))
			got, err := client.FetchSkills(context.Background())

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

// evictingMockCache always returns a miss, simulating TTL expiry.
type evictingMockCache struct{}

func (m *evictingMockCache) Get(key string) ([]byte, time.Time, bool, error) {
	return nil, time.Time{}, false, nil
}

func (m *evictingMockCache) Set(key string, data []byte) error {
	return nil
}
