package websocket

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUnmarshalRequestLogEvent(t *testing.T) {
	var data = `{"type": "request_log_event", "event_payload": "foo", "request_log_id": "resp_123"}`

	var msg IncomingMessage
	err := json.Unmarshal([]byte(data), &msg)
	require.NoError(t, err)

	require.NotNil(t, msg.RequestLogEvent)
	require.Nil(t, msg.WebhookEvent)

	require.Equal(t, "foo", msg.RequestLogEvent.EventPayload)
	require.Equal(t, "resp_123", msg.RequestLogEvent.RequestLogID)
	require.Equal(t, "request_log_event", msg.RequestLogEvent.Type)
}
