package websocket

import (
	"testing"

	assert "github.com/stretchr/testify/require"
)

func TestNewWebhookResponse(t *testing.T) {
	msg := NewWebhookResponse("wh_123", 200, "foo", map[string]string{"Response-Header": "bar"})
	assert.NotNil(t, msg.WebhookResponse)
	assert.Equal(t, "webhook_response", msg.Type)
	assert.Equal(t, "wh_123", msg.WebhookID)
	assert.Equal(t, 200, msg.Status)
	assert.Equal(t, "foo", msg.Body)
	assert.Equal(t, "bar", msg.HTTPHeaders["Response-Header"])
}
