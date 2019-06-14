package stripeauth

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"

	assert "github.com/stretchr/testify/require"
)

func TestAuthorize(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		session := StripeCLISession{
			WebSocketID:  "some-id",
			WebSocketURL: "wss://example.com/subscribe/acct_123",
		}
		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(session)

		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "Bearer sk_test_123", r.Header.Get("Authorization"))
		assert.NotEmpty(t, r.UserAgent())
		assert.NotEmpty(t, r.Header.Get("X-Stripe-Client-User-Agent"))

		body, err := ioutil.ReadAll(r.Body)
		assert.NoError(t, err)
		assert.Equal(t, "device_name=my-device", string(body))
	}))
	defer ts.Close()

	client := NewClient("sk_test_123", &Config{
		URL: ts.URL,
	})
	session, err := client.Authorize("my-device")
	assert.NoError(t, err)
	assert.Equal(t, "some-id", session.WebSocketID)
	assert.Equal(t, "wss://example.com/subscribe/acct_123", session.WebSocketURL)
}

func TestUserAgent(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)

		assert.Regexp(t, regexp.MustCompile(`^Stripe/v1 stripe-cli/\w+$`), r.Header.Get("User-Agent"))
	}))
	defer ts.Close()

	client := NewClient("sk_test_123", &Config{
		URL: ts.URL,
	})
	client.Authorize("my-device")
}

func TestStripeClientUserAgent(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)

		encodedUserAgent := r.Header.Get("X-Stripe-Client-User-Agent")
		assert.NotEmpty(t, encodedUserAgent)

		var userAgent map[string]string
		err := json.Unmarshal([]byte(encodedUserAgent), &userAgent)
		assert.NoError(t, err)

		// Just test a few headers that we know to be stable.
		assert.Equal(t, "stripe-cli", userAgent["name"])
		assert.Equal(t, "stripe", userAgent["publisher"])
	}))
	defer ts.Close()

	client := NewClient("sk_test_123", &Config{
		URL: ts.URL,
	})
	client.Authorize("my-device")
}
