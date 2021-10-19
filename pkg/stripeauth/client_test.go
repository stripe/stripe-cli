package stripeauth

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAuthorize(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		session := StripeCLISession{
			WebSocketID:                "some-id",
			WebSocketURL:               "wss://example.com/subscribe/acct_123",
			WebSocketAuthorizedFeature: "webhook-payloads",
			DefaultVersion:             "2014-01-31",
			LatestVersion:              "2020-08-27",
		}
		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(session)

		require.Equal(t, http.MethodPost, r.Method)
		require.Equal(t, "Bearer sk_test_123", r.Header.Get("Authorization"))
		require.NotEmpty(t, r.UserAgent())
		require.NotEmpty(t, r.Header.Get("X-Stripe-Client-User-Agent"))

		body, err := ioutil.ReadAll(r.Body)
		require.NoError(t, err)
		require.Equal(t, "device_name=my-device&websocket_feature=webhooks", string(body))
	}))
	defer ts.Close()

	client := NewClient("sk_test_123", &Config{
		APIBaseURL: ts.URL,
	})
	session, err := client.Authorize(context.Background(), "my-device", "webhooks", nil, nil)
	require.NoError(t, err)
	require.Equal(t, "some-id", session.WebSocketID)
	require.Equal(t, "wss://example.com/subscribe/acct_123", session.WebSocketURL)
	require.Equal(t, "webhook-payloads", session.WebSocketAuthorizedFeature)
	require.Equal(t, "2014-01-31", session.DefaultVersion)
	require.Equal(t, "2020-08-27", session.LatestVersion)
}

func TestUserAgent(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)

		require.Regexp(t, regexp.MustCompile(`^Stripe/v1 stripe-cli/\w+$`), r.Header.Get("User-Agent"))
	}))
	defer ts.Close()

	client := NewClient("sk_test_123", &Config{
		APIBaseURL: ts.URL,
	})
	client.Authorize(context.Background(), "my-device", "webhooks", nil, nil)
}

func TestStripeClientUserAgent(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)

		encodedUserAgent := r.Header.Get("X-Stripe-Client-User-Agent")
		require.NotEmpty(t, encodedUserAgent)

		var userAgent map[string]string
		err := json.Unmarshal([]byte(encodedUserAgent), &userAgent)
		require.NoError(t, err)

		// Just test a few headers that we know to be stable.
		require.Equal(t, "stripe-cli", userAgent["name"])
		require.Equal(t, "stripe", userAgent["publisher"])
	}))
	defer ts.Close()

	client := NewClient("sk_test_123", &Config{
		APIBaseURL: ts.URL,
	})
	client.Authorize(context.Background(), "my-device", "webhooks", nil, nil)
}

func TestAuthorizeWithURLDeviceMap(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)

		require.Equal(t, "my-device", r.FormValue("device_name"))
		require.Equal(t, "webhooks", r.FormValue("websocket_feature"))
		require.Equal(t, "http://localhost:3000/events", r.FormValue("forward_to_url"))
		require.Equal(t, "http://localhost:3000/connect/events", r.FormValue("forward_connect_to_url"))
	}))
	defer ts.Close()

	client := NewClient("sk_test_123", &Config{
		APIBaseURL: ts.URL,
	})

	devURLMap := DeviceURLMap{
		ForwardURL:        "http://localhost:3000/events",
		ForwardConnectURL: "http://localhost:3000/connect/events",
	}

	client.Authorize(context.Background(), "my-device", "webhooks", nil, &devURLMap)
}
