package stripe

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestApplyAccountContextHeaders_APIKey(t *testing.T) {
	creds := NewAPIKeyCredentials("sk_test_123")

	t.Run("no flags", func(t *testing.T) {
		h := make(http.Header)
		creds.ApplyAccountContextHeaders(h, "", "")
		require.Empty(t, h)
	})
	t.Run("account only", func(t *testing.T) {
		h := make(http.Header)
		creds.ApplyAccountContextHeaders(h, "acct_b", "")
		require.Equal(t, "acct_b", h.Get("Stripe-Account"))
		require.Empty(t, h.Get("Stripe-Context"))
	})
	t.Run("context only", func(t *testing.T) {
		h := make(http.Header)
		creds.ApplyAccountContextHeaders(h, "", "ctx_c")
		require.Equal(t, "ctx_c", h.Get("Stripe-Context"))
		require.Empty(t, h.Get("Stripe-Account"))
	})
	t.Run("both", func(t *testing.T) {
		h := make(http.Header)
		creds.ApplyAccountContextHeaders(h, "acct_b", "ctx_c")
		require.Equal(t, "acct_b", h.Get("Stripe-Account"))
		require.Equal(t, "ctx_c", h.Get("Stripe-Context"))
	})
}

func TestApplyAccountContextHeaders_UAT(t *testing.T) {
	creds := NewOAKCredentials("oak_test_123", "acct_a", false)

	t.Run("no flags uses OAK context", func(t *testing.T) {
		h := make(http.Header)
		creds.ApplyAccountContextHeaders(h, "", "")
		require.Equal(t, "acct_a", h.Get("Stripe-Context"))
		require.Empty(t, h.Get("Stripe-Account"))
	})
	t.Run("account different value", func(t *testing.T) {
		h := make(http.Header)
		creds.ApplyAccountContextHeaders(h, "acct_b", "")
		require.Equal(t, "acct_a/acct_b", h.Get("Stripe-Account"))
		require.Empty(t, h.Get("Stripe-Context"))
	})
	t.Run("account same value skips prefix", func(t *testing.T) {
		h := make(http.Header)
		creds.ApplyAccountContextHeaders(h, "acct_a", "")
		require.Equal(t, "acct_a", h.Get("Stripe-Account"))
		require.Empty(t, h.Get("Stripe-Context"))
	})
	t.Run("context different value", func(t *testing.T) {
		h := make(http.Header)
		creds.ApplyAccountContextHeaders(h, "", "acct_b")
		require.Equal(t, "acct_a/acct_b", h.Get("Stripe-Context"))
		require.Empty(t, h.Get("Stripe-Account"))
	})
	t.Run("context same value skips prefix", func(t *testing.T) {
		h := make(http.Header)
		creds.ApplyAccountContextHeaders(h, "", "acct_a")
		require.Equal(t, "acct_a", h.Get("Stripe-Context"))
		require.Empty(t, h.Get("Stripe-Account"))
	})
	t.Run("account set omits Stripe-Context even when context also provided", func(t *testing.T) {
		h := make(http.Header)
		creds.ApplyAccountContextHeaders(h, "acct_b", "ctx_c")
		require.Equal(t, "acct_a/acct_b", h.Get("Stripe-Account"))
		require.Empty(t, h.Get("Stripe-Context"))
	})
}

type apiStatusTelemetryClient struct {
	statuses chan *CLIAnalyticsEventMetadata
}

func (c *apiStatusTelemetryClient) SendAPIRequestEvent(ctx context.Context, _ string, _ bool) (*http.Response, error) {
	c.statuses <- GetEventMetadata(ctx)
	return nil, nil
}

func (c *apiStatusTelemetryClient) SendEvent(ctx context.Context, _ string, _ string) {
	c.statuses <- GetEventMetadata(ctx)
}

func TestPerformRequest_TelemetryHTTPOutcome(t *testing.T) {
	statuses := []int{200, 201, 204, 302, 400, 401, 403, 404, 429, 500}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		status, err := strconv.Atoi(r.URL.Query().Get("status"))
		if err != nil {
			t.Error(err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		w.WriteHeader(status)
	}))
	defer server.Close()
	baseURL, err := url.Parse(server.URL)
	require.NoError(t, err)
	client := Client{BaseURL: baseURL, httpClient: server.Client()}
	telemetry := &apiStatusTelemetryClient{statuses: make(chan *CLIAnalyticsEventMetadata, len(statuses)+1)}
	metadata := &CLIAnalyticsEventMetadata{CommandPath: "stripe get", InvocationID: "test-invocation"}
	original := *metadata
	ctx := WithEventMetadata(WithTelemetryClient(context.Background(), telemetry), metadata)

	var wg sync.WaitGroup
	for _, status := range statuses {
		wg.Go(func() {
			resp, err := client.PerformRequest(ctx, http.MethodGet, "/v1/test", "status="+strconv.Itoa(status), nil)
			if err != nil {
				t.Error(err)
				return
			}
			defer resp.Body.Close()
			if resp.StatusCode != status {
				t.Errorf("response status = %d, want %d", resp.StatusCode, status)
			}
		})
	}
	wg.Wait()

	var gotOutcomes []string
	for range statuses {
		select {
		case eventMetadata := <-telemetry.statuses:
			require.NotSame(t, metadata, eventMetadata)
			require.Equal(t, original.CommandPath, eventMetadata.CommandPath)
			require.Equal(t, original.InvocationID, eventMetadata.InvocationID)
			gotOutcomes = append(gotOutcomes, eventMetadata.HTTPOutcome)
		case <-time.After(time.Second):
			t.Fatal("timed out waiting for API request telemetry")
		}
	}
	require.ElementsMatch(t, []string{"2xx", "2xx", "2xx", "other_non_200", "other_4xx", "other_4xx", "other_4xx", "not_found", "other_4xx", "5xx"}, gotOutcomes)
	require.Equal(t, original, *metadata)
	telemetry.SendEvent(ctx, "Command Invoked", "")
	require.Empty(t, (<-telemetry.statuses).HTTPOutcome)
}

func TestAPIRequestTelemetryImmediatelyBeforeWait(t *testing.T) {
	var requests atomic.Int64
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
	}))
	defer server.Close()
	baseURL, err := url.Parse(server.URL)
	require.NoError(t, err)
	telemetry := &AnalyticsTelemetryClient{BaseURL: baseURL, HTTPClient: server.Client()}
	ctx := WithEventMetadata(WithTelemetryClient(context.Background(), telemetry), &CLIAnalyticsEventMetadata{})

	for i := range 100 {
		sendTelemetryEvent(ctx, "", false)
		telemetry.Wait()
		require.EqualValues(t, i+1, requests.Load())
	}
}

func TestPerformRequest_NoResponseHasNoOutcomeTelemetry(t *testing.T) {
	baseURL, err := url.Parse("http://127.0.0.1")
	require.NoError(t, err)
	client := Client{BaseURL: baseURL}
	telemetry := &apiStatusTelemetryClient{statuses: make(chan *CLIAnalyticsEventMetadata, 1)}
	ctx := WithEventMetadata(WithTelemetryClient(context.Background(), telemetry), &CLIAnalyticsEventMetadata{})
	ctx, cancel := context.WithCancel(ctx)
	cancel()

	resp, err := client.PerformRequest(ctx, http.MethodGet, "/v1/test", "", nil)
	if resp != nil {
		defer resp.Body.Close()
	}

	require.ErrorIs(t, err, context.Canceled)
	require.Nil(t, resp)
	require.Empty(t, telemetry.statuses)
}

func TestPerformRequest_ParamsEncoding_Delete(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/delete", r.URL.Path)
		require.Equal(t, "key_a=value_a&key_b=value_b", r.URL.RawQuery)

		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		require.Equal(t, "", string(body))
	}))
	defer ts.Close()

	baseURL, _ := url.Parse(ts.URL)
	client := Client{
		BaseURL: baseURL,
	}

	params := url.Values{}
	params.Add("key_a", "value_a")
	params.Add("key_b", "value_b")

	resp, err := client.PerformRequest(context.Background(), http.MethodDelete, "/delete", params.Encode(), nil)
	require.NoError(t, err)

	defer resp.Body.Close()
}

func TestPerformRequest_ParamsEncoding_Get(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/get", r.URL.Path)
		require.Equal(t, "key_a=value_a&key_b=value_b", r.URL.RawQuery)

		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		require.Equal(t, "", string(body))
	}))
	defer ts.Close()

	baseURL, _ := url.Parse(ts.URL)
	client := Client{
		BaseURL: baseURL,
	}

	params := url.Values{}
	params.Add("key_a", "value_a")
	params.Add("key_b", "value_b")

	resp, err := client.PerformRequest(context.Background(), http.MethodGet, "/get", params.Encode(), nil)
	require.NoError(t, err)

	defer resp.Body.Close()
}

func TestPerformRequest_ParamsEncoding_Post(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/post", r.URL.Path)
		require.Equal(t, "", r.URL.RawQuery)

		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		require.Equal(t, "key_a=value_a&key_b=value_b", string(body))
	}))
	defer ts.Close()

	baseURL, _ := url.Parse(ts.URL)
	client := Client{
		BaseURL: baseURL,
	}

	params := url.Values{}
	params.Add("key_a", "value_a")
	params.Add("key_b", "value_b")

	resp, err := client.PerformRequest(context.Background(), http.MethodPost, "/post", params.Encode(), nil)
	require.NoError(t, err)

	defer resp.Body.Close()
}

func TestPerformRequest_ApiKey_Provided(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "Bearer sk_test_1234", r.Header.Get("Authorization"))
	}))
	defer ts.Close()

	baseURL, _ := url.Parse(ts.URL)
	client := Client{
		BaseURL:     baseURL,
		Credentials: Credentials{Token: "sk_test_1234"},
	}

	resp, err := client.PerformRequest(context.Background(), http.MethodGet, "/get", "", nil)
	require.NoError(t, err)

	defer resp.Body.Close()
}

func TestPerformRequest_ApiKey_Omitted(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "", r.Header.Get("Authorization"))
	}))
	defer ts.Close()

	baseURL, _ := url.Parse(ts.URL)
	client := Client{
		BaseURL: baseURL,
	}

	resp, err := client.PerformRequest(context.Background(), http.MethodGet, "/get", "", nil)
	require.NoError(t, err)

	defer resp.Body.Close()
}

func TestPerformRequest_ConfigureFunc(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "2019-07-10", r.Header.Get("Stripe-Version"))
	}))
	defer ts.Close()

	baseURL, _ := url.Parse(ts.URL)
	client := Client{
		BaseURL: baseURL,
	}

	resp, err := client.PerformRequest(context.Background(), http.MethodGet, "/get", "", func(r *http.Request) error {
		r.Header.Add("Stripe-Version", "2019-07-10")
		return nil
	})
	require.NoError(t, err)

	defer resp.Body.Close()
}

func TestPerformRequest_ConfigureFuncReturnsError(t *testing.T) {
	serverCalled := false
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		serverCalled = true
	}))
	defer ts.Close()

	baseURL, _ := url.Parse(ts.URL)
	client := Client{
		BaseURL: baseURL,
	}

	resp, err := client.PerformRequest(context.Background(), http.MethodGet, "/get", "", func(r *http.Request) error {
		return errors.New("foo")
	})
	require.Equal(t, errors.New("foo"), err)
	require.False(t, serverCalled)
	if resp != nil {
		resp.Body.Close()
	}
}
