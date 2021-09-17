package stripe

import (
	"context"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPerformRequest_ParamsEncoding_Delete(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/delete", r.URL.Path)
		require.Equal(t, "key_a=value_a&key_b=value_b", r.URL.RawQuery)

		body, err := ioutil.ReadAll(r.Body)
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

		body, err := ioutil.ReadAll(r.Body)
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

		body, err := ioutil.ReadAll(r.Body)
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
		BaseURL: baseURL,
		APIKey:  "sk_test_1234",
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

	resp, err := client.PerformRequest(context.Background(), http.MethodGet, "/get", "", func(r *http.Request) {
		r.Header.Add("Stripe-Version", "2019-07-10")
	})
	require.NoError(t, err)

	defer resp.Body.Close()
}
