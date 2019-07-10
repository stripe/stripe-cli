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

	_, err := client.PerformRequest("DELETE", "/delete", params, nil)
	assert.NoError(t, err)
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

	_, err := client.PerformRequest("GET", "/get", params, nil)
	assert.NoError(t, err)
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

	_, err := client.PerformRequest("POST", "/post", params, nil)
	assert.NoError(t, err)
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

	_, err := client.PerformRequest("GET", "/get", nil, nil)
	assert.NoError(t, err)
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

	_, err := client.PerformRequest("GET", "/get", nil, nil)
	assert.NoError(t, err)
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

	_, err := client.PerformRequest("GET", "/get", nil, func(r *http.Request) {
		r.Header.Add("Stripe-Version", "2019-07-10")
	})
	assert.NoError(t, err)
}
