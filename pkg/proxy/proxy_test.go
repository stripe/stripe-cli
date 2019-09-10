package proxy

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/stripe/stripe-cli/pkg/websocket"
)

func TestFilterWebhookEvent(t *testing.T) {
	proxyUseDefault := New(&Config{UseLatestAPIVersion: false}, []string{"*"})
	proxyUseLatest := New(&Config{UseLatestAPIVersion: true}, []string{"*"})

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
