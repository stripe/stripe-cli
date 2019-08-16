package stripe

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPerformRequest_ParamsEncoding_Delete(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/delete", r.URL.Path)
		assert.Equal(t, "key_a=value_a&key_b=value_b", r.URL.RawQuery)

		body, err := ioutil.ReadAll(r.Body)
		assert.NoError(t, err)
		assert.Equal(t, "", string(body))
	}))
	defer ts.Close()

	baseURL, _ := url.Parse(ts.URL)
	client := Client{
		BaseURL: baseURL,
	}

	params := url.Values{}
	params.Add("key_a", "value_a")
	params.Add("key_b", "value_b")

	resp, err := client.PerformRequest(http.MethodDelete, "/delete", params.Encode(), nil)
	assert.NoError(t, err)
	defer resp.Body.Close()
}

func TestPerformRequest_ParamsEncoding_Get(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/get", r.URL.Path)
		assert.Equal(t, "key_a=value_a&key_b=value_b", r.URL.RawQuery)

		body, err := ioutil.ReadAll(r.Body)
		assert.NoError(t, err)
		assert.Equal(t, "", string(body))
	}))
	defer ts.Close()

	baseURL, _ := url.Parse(ts.URL)
	client := Client{
		BaseURL: baseURL,
	}

	params := url.Values{}
	params.Add("key_a", "value_a")
	params.Add("key_b", "value_b")

	resp, err := client.PerformRequest(http.MethodGet, "/get", params.Encode(), nil)
	assert.NoError(t, err)
	defer resp.Body.Close()
}

func TestPerformRequest_ParamsEncoding_Post(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/post", r.URL.Path)
		assert.Equal(t, "", r.URL.RawQuery)

		body, err := ioutil.ReadAll(r.Body)
		assert.NoError(t, err)
		assert.Equal(t, "key_a=value_a&key_b=value_b", string(body))
	}))
	defer ts.Close()

	baseURL, _ := url.Parse(ts.URL)
	client := Client{
		BaseURL: baseURL,
	}

	params := url.Values{}
	params.Add("key_a", "value_a")
	params.Add("key_b", "value_b")

	resp, err := client.PerformRequest(http.MethodPost, "/post", params.Encode(), nil)
	assert.NoError(t, err)
	defer resp.Body.Close()
}

func TestPerformRequest_ApiKey_Provided(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "Bearer sk_test_1234", r.Header.Get("Authorization"))
	}))
	defer ts.Close()

	baseURL, _ := url.Parse(ts.URL)
	client := Client{
		BaseURL: baseURL,
		APIKey:  "sk_test_1234",
	}

	resp, err := client.PerformRequest(http.MethodGet, "/get", "", nil)
	assert.NoError(t, err)
	defer resp.Body.Close()
}

func TestPerformRequest_ApiKey_Omitted(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "", r.Header.Get("Authorization"))
	}))
	defer ts.Close()

	baseURL, _ := url.Parse(ts.URL)
	client := Client{
		BaseURL: baseURL,
	}

	resp, err := client.PerformRequest(http.MethodGet, "/get", "", nil)
	assert.NoError(t, err)
	defer resp.Body.Close()
}

func TestPerformRequest_ConfigureFunc(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "2019-07-10", r.Header.Get("Stripe-Version"))
	}))
	defer ts.Close()

	baseURL, _ := url.Parse(ts.URL)
	client := Client{
		BaseURL: baseURL,
	}

	resp, err := client.PerformRequest(http.MethodGet, "/get", "", func(r *http.Request) {
		r.Header.Add("Stripe-Version", "2019-07-10")
	})
	assert.NoError(t, err)
	defer resp.Body.Close()
}
