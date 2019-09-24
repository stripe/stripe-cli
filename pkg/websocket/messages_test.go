package websocket

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUnmarshalUnknownIncomingMsg(t *testing.T) {
	var data = `{"type": "unknown_type", "foo": "bar"}`

	var msg IncomingMessage
	err := json.Unmarshal([]byte(data), &msg)
	require.EqualError(t, err, "Unexpected message type: unknown_type")
}
