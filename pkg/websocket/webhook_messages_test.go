package websocket

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

func TestUnmarshalWebhookEvent(t *testing.T) {
	var data = `{"type": "webhook_event", "event_payload": "foo", "http_headers": {"Request-Header": "bar"}, "webhook_id": "wh_123", "webhook_conversation_id": "wc_123"}`

	var msg IncomingMessage
	err := json.Unmarshal([]byte(data), &msg)
	require.NoError(t, err)

	require.NotNil(t, msg.WebhookEvent)
	require.Nil(t, msg.RequestLogEvent)

	require.Equal(t, "foo", msg.WebhookEvent.EventPayload)
	require.Equal(t, "bar", msg.WebhookEvent.HTTPHeaders["Request-Header"])
	require.Equal(t, "webhook_event", msg.WebhookEvent.Type)
	require.Equal(t, "wh_123", msg.WebhookEvent.WebhookID)
	require.Equal(t, "wc_123", msg.WebhookEvent.WebhookConversationID)
}

func TestMarshalWebhookResponse(t *testing.T) {
	msg := NewWebhookResponse(
		"wh_123",
		"wc_123",
		"http://localhost:5000/webhooks",
		200,
		"foo",
		map[string]string{"Response-Header": "bar"},
	)

	buf, err := json.Marshal(msg)
	require.NoError(t, err)

	json := string(buf)
	require.Equal(t, "wh_123", gjson.Get(json, "webhook_id").String())
	require.Equal(t, "wc_123", gjson.Get(json, "webhook_conversation_id").String())
	require.Equal(t, "http://localhost:5000/webhooks", gjson.Get(json, "forward_url").String())
	require.Equal(t, 200, int(gjson.Get(json, "status").Num))
	require.Equal(t, "foo", gjson.Get(json, "body").String())
	require.Equal(t, "bar", gjson.Get(json, "http_headers.Response-Header").String())
}

func TestNewWebhookResponse(t *testing.T) {
	msg := NewWebhookResponse(
		"wh_123",
		"wc_123",
		"http://localhost:5000/webhooks",
		200,
		"foo",
		map[string]string{"Response-Header": "bar"},
	)

	require.NotNil(t, msg.WebhookResponse)
	require.Equal(t, "webhook_response", msg.WebhookResponse.Type)
	require.Equal(t, "wh_123", msg.WebhookResponse.WebhookID)
	require.Equal(t, "wc_123", msg.WebhookResponse.WebhookConversationID)
	require.Equal(t, "http://localhost:5000/webhooks", msg.ForwardURL)
	require.Equal(t, 200, msg.Status)
	require.Equal(t, "foo", msg.Body)
	require.Equal(t, "bar", msg.HTTPHeaders["Response-Header"])
}
