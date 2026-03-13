package proxy

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/stripe/stripe-cli/pkg/websocket"
)

func TestNewWebhookEventProcessor_HTTP2Configured(t *testing.T) {
	sendMessage := func(msg *websocket.OutgoingMessage) {}
	routes := []EndpointRoute{
		{
			URL: "https://localhost:4242",
		},
	}
	cfg := &WebhookEventProcessorConfig{
		SkipVerify: true,
		Timeout:    30,
	}

	p := NewWebhookEventProcessor(sendMessage, routes, cfg)

	require.Equal(t, 1, len(p.endpointClients))
	client := p.endpointClients[0]
	
	// Assert that the transport is an *http.Transport
	transport, ok := client.cfg.HTTPClient.Transport.(*http.Transport)
	require.True(t, ok)
	
	// http2.ConfigureTransport adds "h2" to TLSNextProto
	require.NotNil(t, transport.TLSNextProto)
	_, hasH2 := transport.TLSNextProto["h2"]
	require.True(t, hasH2, "Transport should have h2 configured in TLSNextProto")
}
