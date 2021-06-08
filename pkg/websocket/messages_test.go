package websocket

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

func TestUnmarshalUnknownIncomingMsg(t *testing.T) {
	var data = `{"type": "unknown_type", "foo": "bar"}`

	var msg IncomingMessage
	err := json.Unmarshal([]byte(data), &msg)
	require.EqualError(t, err, "Unexpected message type: unknown_type")
}

func TestMarshalWebhookEventAck(t *testing.T) {
	msg := NewEventAck(
		"wh_123",
		"wc_123",
	)

	buf, err := json.Marshal(msg)
	require.NoError(t, err)

	json := string(buf)
	require.Equal(t, "wh_123", gjson.Get(json, "event_id").String())
	require.Equal(t, "wc_123", gjson.Get(json, "webhook_conversation_id").String())
	require.Equal(t, "event_ack", gjson.Get(json, "type").String())
}

func TestMarshalWebhookEventAckRequestLog(t *testing.T) {
	msg := NewEventAck(
		"wh_123",
		"",
	)

	buf, err := json.Marshal(msg)
	require.NoError(t, err)

	json := string(buf)
	require.Equal(t, "wh_123", gjson.Get(json, "event_id").String())
	require.Equal(t, "", gjson.Get(json, "webhook_conversation_id").String())
	require.Equal(t, "event_ack", gjson.Get(json, "type").String())
}

func TestNewWebhookEventAck(t *testing.T) {
	msg := NewEventAck(
		"wh_123",
		"wc_123",
	)

	require.NotNil(t, msg.EventAck)
	require.Equal(t, "event_ack", msg.EventAck.Type)
	require.Equal(t, "wh_123", msg.EventAck.EventID)
	require.Equal(t, "wc_123", msg.EventAck.WebhookConversationID)
}

func TestNewWebhookEventAckRequestLog(t *testing.T) {
	msg := NewEventAck(
		"wh_123",
		"",
	)

	require.NotNil(t, msg.EventAck)
	require.Equal(t, "event_ack", msg.EventAck.Type)
	require.Equal(t, "wh_123", msg.EventAck.EventID)
	require.Equal(t, "", msg.EventAck.WebhookConversationID)
}
