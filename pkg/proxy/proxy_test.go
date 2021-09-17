package proxy

import (
	"context"
	"net/url"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/stripe/stripe-cli/pkg/requests"
	"github.com/stripe/stripe-cli/pkg/websocket"
)

func TestFilterWebhookEvent(t *testing.T) {
	proxyUseDefault, _ := Init(context.Background(), &Config{UseLatestAPIVersion: false})
	proxyUseLatest, _ := Init(context.Background(), &Config{UseLatestAPIVersion: true})

	evtDefault := &websocket.WebhookEvent{
		Endpoint: websocket.WebhookEndpoint{
			APIVersion: nil,
		},
	}

	apiVersion := "2019-05-04"
	evtLatest := &websocket.WebhookEvent{
		Endpoint: websocket.WebhookEndpoint{
			APIVersion: &apiVersion,
		},
	}

	require.False(t, proxyUseDefault.filterWebhookEvent(evtDefault))
	require.True(t, proxyUseDefault.filterWebhookEvent(evtLatest))

	require.True(t, proxyUseLatest.filterWebhookEvent(evtDefault))
	require.False(t, proxyUseLatest.filterWebhookEvent(evtLatest))
}

func TestTruncate(t *testing.T) {
	require.Equal(t, "Hello, World", truncate("Hello, World", 12, false))
	require.Equal(t, "Hello, Worl", truncate("Hello, World", 11, false))
	require.Equal(t, "Hello, W...", truncate("Hello, World", 11, true))

	require.Equal(t, "Hello, 世界", truncate("Hello, 世界", 13, false))
	require.Equal(t, "Hello, 世", truncate("Hello, 世界", 12, false))
	require.Equal(t, "Hello, ...", truncate("Hello, 世界", 12, true))
}

func TestBuildEndpointRoutes(t *testing.T) {
	localURL := "http://localhost"

	endpointNormal := requests.WebhookEndpoint{
		URL:           "https://planetexpress.com/hooks",
		Application:   "",
		EnabledEvents: []string{"*"},
		Status:        "enabled",
	}

	endpointConnect := requests.WebhookEndpoint{
		URL:           "https://planetexpress.com/connect-hooks",
		Application:   "ca_123",
		EnabledEvents: []string{"*"},
		Status:        "enabled",
	}

	endpointDisabled := requests.WebhookEndpoint{
		URL:         "https://test-app-url/stripe/payment-webhook",
		Application: "ca_123",
		EnabledEvents: []string{"payment_intent.payment_failed",
			"payment_intent.succeeded"},
		Status: "disabled",
	}

	endpointList := requests.WebhookEndpointList{
		Data: []requests.WebhookEndpoint{endpointNormal, endpointConnect, endpointDisabled},
	}

	output, err := buildEndpointRoutes(endpointList, localURL, localURL, []string{"Host: hostname"}, []string{"Host: connecthostname"})
	require.NoError(t, err)
	require.Equal(t, 2, len(output))
	require.Equal(t, "http://localhost/hooks", output[0].URL)
	require.Equal(t, []string{"Host: hostname"}, output[0].ForwardHeaders)
	require.Equal(t, false, output[0].Connect)
	require.Equal(t, []string{"*"}, output[0].EventTypes)
	require.Equal(t, "http://localhost/connect-hooks", output[1].URL)
	require.Equal(t, []string{"Host: connecthostname"}, output[1].ForwardHeaders)
	require.Equal(t, true, output[1].Connect)
	require.Equal(t, []string{"*"}, output[1].EventTypes)
}

func TestBuildForwardURL(t *testing.T) {
	f, err := url.Parse("http://example.com/foo/bar.php")
	require.NoError(t, err)

	// pairs of [expected, input]
	expectedInputPairs := [][]string{
		{"http://localhost/foo/bar.php", "http://localhost"},
		{"https://localhost/foo/bar.php", "https://localhost/"},
		{"http://localhost:8000/foo/bar.php", "http://localhost:8000"},
		{"http://localhost:8000/foo/bar.php", "http://localhost:8000/"},
		{"http://localhost:8000/forward/sub/path/foo/bar.php", "http://localhost:8000/forward/sub/path/"},
		{"http://localhost:8000/forward/sub/path/foo/bar.php", "http://localhost:8000/forward/sub/path"},
	}
	for _, pair := range expectedInputPairs {
		expected := pair[0]
		input := pair[1]
		forwardURL, err := buildForwardURL(input, f)
		require.NoError(t, err)
		require.Equal(t, expected, forwardURL)
	}

	f, err = url.Parse("http://example.com/bar/")
	require.NoError(t, err)

	// pairs of [expected, input]
	expectedInputPairs = [][]string{
		{"http://localhost/bar/", "http://localhost/"},
		{"http://localhost/bar/", "http://localhost"},
		{"https://localhost/bar/", "https://localhost/"},
		{"https://localhost/bar/", "https://localhost"},
		{"http://localhost:8000/bar/", "http://localhost:8000"},
		{"http://localhost:8000/bar/", "http://localhost:8000/"},
	}
	for _, pair := range expectedInputPairs {
		expected := pair[0]
		input := pair[1]
		forwardURL, err := buildForwardURL(input, f)
		require.NoError(t, err)
		require.Equal(t, expected, forwardURL)
	}
}

func TestParseUrl(t *testing.T) {
	require.Equal(t, "http://example.com/foo", parseURL("http://example.com/foo"))
	require.Equal(t, "https://example.com/foo", parseURL("https://example.com/foo"))

	require.Equal(t, "http://example.com/foo", parseURL("example.com/foo"))

	require.Equal(t, "http://localhost/foo", parseURL("/foo"))

	require.Equal(t, "http://localhost:3000", parseURL("3000"))
}
