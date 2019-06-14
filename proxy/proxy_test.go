package proxy

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/stripe/stripe-cli/websocket"
)

func TestFilterWebhookEvent(t *testing.T) {
	proxyUseDefault := New(&Config{UseLatestAPIVersion: false})
	proxyUseLatest := New(&Config{UseLatestAPIVersion: true})

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

	assert.False(t, proxyUseDefault.filterWebhookEvent(evtDefault))
	assert.True(t, proxyUseDefault.filterWebhookEvent(evtLatest))

	assert.True(t, proxyUseLatest.filterWebhookEvent(evtDefault))
	assert.False(t, proxyUseLatest.filterWebhookEvent(evtLatest))
}
