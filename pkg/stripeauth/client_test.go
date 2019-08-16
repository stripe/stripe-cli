package stripeauth

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAuthorize(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		session := StripeCLISession{
			WebSocketID:                "some-id",
			WebSocketURL:               "wss://example.com/subscribe/acct_123",
			WebSocketAuthorizedFeature: "webhook-payloads",
		}
		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(session)

		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "Bearer sk_test_123", r.Header.Get("Authorization"))
		assert.NotEmpty(t, r.UserAgent())
		assert.NotEmpty(t, r.Header.Get("X-Stripe-Client-User-Agent"))

		body, err := ioutil.ReadAll(r.Body)
		assert.NoError(t, err)
		assert.Equal(t, "device_name=my-device&websocket_feature=webhooks", string(body))
	}))
	defer ts.Close()

	client := NewClient("sk_test_123", &Config{
		APIBaseURL: ts.URL,
	})
	session, err := client.Authorize("my-device", "webhooks", nil)
	assert.NoError(t, err)
	assert.Equal(t, "some-id", session.WebSocketID)
	assert.Equal(t, "wss://example.com/subscribe/acct_123", session.WebSocketURL)
	assert.Equal(t, "webhook-payloads", session.WebSocketAuthorizedFeature)
}

func TestUserAgent(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)

		assert.Regexp(t, regexp.MustCompile(`^Stripe/v1 stripe-cli/\w+$`), r.Header.Get("User-Agent"))
	}))
	defer ts.Close()

	client := NewClient("sk_test_123", &Config{
		APIBaseURL: ts.URL,
	})
	client.Authorize("my-device", "webhooks", nil)
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
		APIBaseURL: ts.URL,
	})
	client.Authorize("my-device", "webhooks", nil)
}
