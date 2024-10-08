package stripeauth

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/stripe/stripe-cli/pkg/stripe"
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

		require.Equal(t, "my-device", r.FormValue("device_name"))
		require.Equal(t, "webhooks", r.FormValue("websocket_features[]"))
	}))
	defer ts.Close()

	baseURL, _ := url.Parse(ts.URL)
	client := NewClient(&stripe.Client{APIKey: "sk_test_123", BaseURL: baseURL}, nil)

	session, err := client.Authorize(context.Background(), CreateSessionRequest{
		DeviceName:        "my-device",
		WebSocketFeatures: []string{"webhooks"},
	})
	require.NoError(t, err)
	require.NoError(t, err)
	require.Equal(t, "some-id", session.WebSocketID)
	require.Equal(t, "wss://example.com/subscribe/acct_123", session.WebSocketURL)
	require.Equal(t, "webhook-payloads", session.WebSocketAuthorizedFeature)
	require.Equal(t, "2014-01-31", session.DefaultVersion)
	require.Equal(t, "2020-08-27", session.LatestVersion)
}

func TestUserAgent(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Regexp(t, regexp.MustCompile(`^Stripe/v1 stripe-cli/\w+$`), r.Header.Get("User-Agent"))
		w.Write([]byte(`{}`))
	}))
	defer ts.Close()

	baseURL, _ := url.Parse(ts.URL)
	client := NewClient(&stripe.Client{APIKey: "sk_test_123", BaseURL: baseURL}, nil)

	_, err := client.Authorize(context.Background(), CreateSessionRequest{
		DeviceName:        "my-device",
		WebSocketFeatures: []string{"webhooks"},
	})
	require.NoError(t, err)
}

func TestStripeClientUserAgent(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		encodedUserAgent := r.Header.Get("X-Stripe-Client-User-Agent")
		require.NotEmpty(t, encodedUserAgent)

		var userAgent map[string]string
		err := json.Unmarshal([]byte(encodedUserAgent), &userAgent)
		require.NoError(t, err)

		// Just test a few headers that we know to be stable.
		require.Equal(t, "stripe-cli", userAgent["name"])
		require.Equal(t, "stripe", userAgent["publisher"])

		w.Write([]byte(`{}`))
	}))
	defer ts.Close()

	baseURL, _ := url.Parse(ts.URL)
	client := NewClient(&stripe.Client{APIKey: "sk_test_123", BaseURL: baseURL}, nil)

	_, err := client.Authorize(context.Background(), CreateSessionRequest{
		DeviceName:        "my-device",
		WebSocketFeatures: []string{"webhooks"},
	})
	require.NoError(t, err)
}

func TestAuthorizeWithURLDeviceMap(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "my-device", r.FormValue("device_name"))
		require.Equal(t, "webhooks", r.FormValue("websocket_features[]"))
		require.Equal(t, "http://localhost:3000/events", r.FormValue("forward_to_url"))
		require.Equal(t, "http://localhost:3000/connect/events", r.FormValue("forward_connect_to_url"))
		require.Equal(t, "http://localhost:3000/thin/events", r.FormValue("forward_thin_to_url"))
		require.Equal(t, "http://localhost:3000/thin/connect/events", r.FormValue("forward_thin_connect_to_url"))

		w.Write([]byte(`{}`))
	}))
	defer ts.Close()

	baseURL, _ := url.Parse(ts.URL)
	client := NewClient(&stripe.Client{APIKey: "sk_test_123", BaseURL: baseURL}, nil)

	devURLMap := DeviceURLMap{
		ForwardURL:            "http://localhost:3000/events",
		ForwardConnectURL:     "http://localhost:3000/connect/events",
		ForwardThinURL:        "http://localhost:3000/thin/events",
		ForwardThinConnectURL: "http://localhost:3000/thin/connect/events",
	}

	_, err := client.Authorize(context.Background(), CreateSessionRequest{
		DeviceName:        "my-device",
		WebSocketFeatures: []string{"webhooks"},
		DeviceURLMap:      &devURLMap,
	})
	require.NoError(t, err)
}
