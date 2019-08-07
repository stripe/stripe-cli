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

	endpoint := requests.WebhookEndpoint{
		URL:           "https://planetexpress.com/hooks",
		EnabledEvents: []string{"*"},
	}

	endpointList := requests.WebhookEndpointList{
		Data: []requests.WebhookEndpoint{endpoint},
	}

	output := buildEndpointRoutes(endpointList, localURL)
	assert.Equal(t, 1, len(output))
	assert.Equal(t, output[0].URL, "http:/localhost/hooks")
	assert.Equal(t, output[0].EventTypes, []string{"*"})
}
