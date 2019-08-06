package cmd

import (
	"testing"

	assert "github.com/stretchr/testify/require"

	"github.com/stripe/stripe-cli/pkg/requests"
)

func TestParseUrl(t *testing.T) {
	assert.Equal(t, "http://example.com/foo", parseURL("http://example.com/foo"))
	assert.Equal(t, "https://example.com/foo", parseURL("https://example.com/foo"))

	assert.Equal(t, "http://example.com/foo", parseURL("example.com/foo"))

	assert.Equal(t, "http://localhost/foo", parseURL("/foo"))

	assert.Equal(t, "http://localhost:3000", parseURL("3000"))
}

func TestBuildEndpointRoutes(t *testing.T) {
	localURL := "http://localhost"

	endpointNormal := requests.WebhookEndpoint{
		URL:           "https://planetexpress.com/hooks",
		Application:   "",
		EnabledEvents: []string{"*"},
	}

	endpointConnect := requests.WebhookEndpoint{
		URL:           "https://planetexpress.com/connect-hooks",
		Application:   "ca_123",
		EnabledEvents: []string{"*"},
	}

	endpointList := requests.WebhookEndpointList{
		Data: []requests.WebhookEndpoint{endpointNormal, endpointConnect},
	}

	output := buildEndpointRoutes(endpointList, localURL, localURL)
	assert.Equal(t, 2, len(output))
	assert.Equal(t, "http:/localhost/hooks", output[0].URL)
	assert.Equal(t, false, output[0].Connect)
	assert.Equal(t, []string{"*"}, output[0].EventTypes)
	assert.Equal(t, "http:/localhost/connect-hooks", output[1].URL)
	assert.Equal(t, true, output[1].Connect)
	assert.Equal(t, []string{"*"}, output[1].EventTypes)
}
