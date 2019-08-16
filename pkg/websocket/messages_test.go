package websocket

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/tidwall/gjson"
)

func TestUnmarshalWebhookEvent(t *testing.T) {
	var data = `{"type": "webhook_event", "event_payload": "foo", "http_headers": {"Request-Header": "bar"}, "webhook_id": "wh_123"}`

	var msg IncomingMessage
	err := json.Unmarshal([]byte(data), &msg)
	assert.Nil(t, err)

	assert.NotNil(t, msg.WebhookEvent)
	assert.Nil(t, msg.RequestLogEvent)

	assert.Equal(t, "foo", msg.WebhookEvent.EventPayload)
	assert.Equal(t, "bar", msg.WebhookEvent.HTTPHeaders["Request-Header"])
	assert.Equal(t, "webhook_event", msg.WebhookEvent.Type)
	assert.Equal(t, "wh_123", msg.WebhookEvent.WebhookID)
}

func TestUnmarshalRequestLogEvent(t *testing.T) {
	var data = `{"type": "request_log_event", "event_payload": "foo", "request_log_id": "resp_123"}`

	var msg IncomingMessage
	err := json.Unmarshal([]byte(data), &msg)
	assert.Nil(t, err)

	assert.NotNil(t, msg.RequestLogEvent)
	assert.Nil(t, msg.WebhookEvent)

	assert.Equal(t, "foo", msg.RequestLogEvent.EventPayload)
	assert.Equal(t, "resp_123", msg.RequestLogEvent.RequestLogID)
	assert.Equal(t, "request_log_event", msg.RequestLogEvent.Type)
}

func TestUnmarshalUnknownIncomingMsg(t *testing.T) {
	var data = `{"type": "unknown_type", "foo": "bar"}`

	var msg IncomingMessage
	err := json.Unmarshal([]byte(data), &msg)
	assert.EqualError(t, err, "Unexpected message type: unknown_type")
}

func TestMarshalWebhookResponse(t *testing.T) {
	msg := NewWebhookResponse("wh_123", 200, "foo", map[string]string{"Response-Header": "bar"})

	buf, err := json.Marshal(msg)
	assert.Nil(t, err)

	json := string(buf)
	assert.Equal(t, "wh_123", gjson.Get(json, "webhook_id").String())
	assert.Equal(t, 200, int(gjson.Get(json, "status").Num))
	assert.Equal(t, "foo", gjson.Get(json, "body").String())
	assert.Equal(t, "bar", gjson.Get(json, "http_headers.Response-Header").String())
}
