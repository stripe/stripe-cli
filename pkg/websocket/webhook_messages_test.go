package websocket

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewWebhookResponse(t *testing.T) {
	msg := NewWebhookResponse("wh_123", 200, "foo", map[string]string{"Response-Header": "bar"})
	require.NotNil(t, msg.WebhookResponse)
	require.Equal(t, "webhook_response", msg.Type)
	require.Equal(t, "wh_123", msg.WebhookID)
	require.Equal(t, 200, msg.Status)
	require.Equal(t, "foo", msg.Body)
	require.Equal(t, "bar", msg.HTTPHeaders["Response-Header"])
}
