package websocket

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

func TestUnmarshalWebhookEvent(t *testing.T) {
	var data = `{"type": "webhook_event", "event_payload": "foo", "http_headers": {"Request-Header": "bar"}, "webhook_id": "wh_123"}`

	var msg IncomingMessage
	err := json.Unmarshal([]byte(data), &msg)
	require.Nil(t, err)

	require.NotNil(t, msg.WebhookEvent)
	require.Nil(t, msg.RequestLogEvent)

	require.Equal(t, "foo", msg.WebhookEvent.EventPayload)
	require.Equal(t, "bar", msg.WebhookEvent.HTTPHeaders["Request-Header"])
	require.Equal(t, "webhook_event", msg.WebhookEvent.Type)
	require.Equal(t, "wh_123", msg.WebhookEvent.WebhookID)
}

func TestUnmarshalRequestLogEvent(t *testing.T) {
	var data = `{"type": "request_log_event", "event_payload": "foo", "request_log_id": "resp_123"}`

	var msg IncomingMessage
	err := json.Unmarshal([]byte(data), &msg)
	require.Nil(t, err)

	require.NotNil(t, msg.RequestLogEvent)
	require.Nil(t, msg.WebhookEvent)

	require.Equal(t, "foo", msg.RequestLogEvent.EventPayload)
	require.Equal(t, "resp_123", msg.RequestLogEvent.RequestLogID)
	require.Equal(t, "request_log_event", msg.RequestLogEvent.Type)
}

func TestUnmarshalUnknownIncomingMsg(t *testing.T) {
	var data = `{"type": "unknown_type", "foo": "bar"}`

	var msg IncomingMessage
	err := json.Unmarshal([]byte(data), &msg)
	require.EqualError(t, err, "Unexpected message type: unknown_type")
}

func TestMarshalWebhookResponse(t *testing.T) {
	msg := NewWebhookResponse("wh_123", 200, "foo", map[string]string{"Response-Header": "bar"})

	buf, err := json.Marshal(msg)
	require.Nil(t, err)

	json := string(buf)
	require.Equal(t, "wh_123", gjson.Get(json, "webhook_id").String())
	require.Equal(t, 200, int(gjson.Get(json, "status").Num))
	require.Equal(t, "foo", gjson.Get(json, "body").String())
	require.Equal(t, "bar", gjson.Get(json, "http_headers.Response-Header").String())
}
