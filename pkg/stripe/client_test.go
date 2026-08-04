package stripe

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestApplyAccountContextHeaders_APIKey(t *testing.T) {
	creds := NewAPIKeyCredentials("sk_test_123")

	t.Run("no flags", func(t *testing.T) {
		h := map[string]string{}
		creds.ApplyAccountContextHeaders(h, "", "")
		require.Empty(t, h)
	})
	t.Run("account only", func(t *testing.T) {
		h := map[string]string{}
		creds.ApplyAccountContextHeaders(h, "acct_b", "")
		require.Equal(t, map[string]string{"Stripe-Account": "acct_b"}, h)
	})
	t.Run("context only", func(t *testing.T) {
		h := map[string]string{}
		creds.ApplyAccountContextHeaders(h, "", "ctx_c")
		require.Equal(t, map[string]string{"Stripe-Context": "ctx_c"}, h)
	})
	t.Run("both", func(t *testing.T) {
		h := map[string]string{}
		creds.ApplyAccountContextHeaders(h, "acct_b", "ctx_c")
		require.Equal(t, map[string]string{"Stripe-Account": "acct_b", "Stripe-Context": "ctx_c"}, h)
	})
}

func TestApplyAccountContextHeaders_UAT(t *testing.T) {
	creds := NewOAKCredentials("oak_test_123", "acct_a", false)

	t.Run("no flags uses OAK context", func(t *testing.T) {
		h := map[string]string{}
		creds.ApplyAccountContextHeaders(h, "", "")
		require.Equal(t, map[string]string{"Stripe-Context": "acct_a"}, h)
	})
	t.Run("account different value", func(t *testing.T) {
		h := map[string]string{}
		creds.ApplyAccountContextHeaders(h, "acct_b", "")
		require.Equal(t, map[string]string{"Stripe-Account": "acct_a/acct_b"}, h)
	})
	t.Run("account same value skips prefix", func(t *testing.T) {
		h := map[string]string{}
		creds.ApplyAccountContextHeaders(h, "acct_a", "")
		require.Equal(t, map[string]string{"Stripe-Account": "acct_a"}, h)
	})
	t.Run("context different value", func(t *testing.T) {
		h := map[string]string{}
		creds.ApplyAccountContextHeaders(h, "", "acct_b")
		require.Equal(t, map[string]string{"Stripe-Context": "acct_a/acct_b"}, h)
	})
	t.Run("context same value skips prefix", func(t *testing.T) {
		h := map[string]string{}
		creds.ApplyAccountContextHeaders(h, "", "acct_a")
		require.Equal(t, map[string]string{"Stripe-Context": "acct_a"}, h)
	})
	t.Run("account set omits Stripe-Context even when context also provided", func(t *testing.T) {
		h := map[string]string{}
		creds.ApplyAccountContextHeaders(h, "acct_b", "ctx_c")
		require.Equal(t, map[string]string{"Stripe-Account": "acct_a/acct_b"}, h)
		require.NotContains(t, h, "Stripe-Context")
	})
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
