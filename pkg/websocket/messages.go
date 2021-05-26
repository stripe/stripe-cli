package websocket

import (
	"encoding/json"
	"fmt"
)

// IncomingMessage represents any incoming message sent by Stripe.
type IncomingMessage struct {
	*WebhookEvent
	*RequestLogEvent
}

// UnmarshalJSON deserializes incoming messages sent by Stripe into the
// appropriate structure.
func (m *IncomingMessage) UnmarshalJSON(data []byte) error {
	incomingMessageTypeOnly := struct {
		Type string `json:"type"`
	}{}
	if err := json.Unmarshal(data, &incomingMessageTypeOnly); err != nil {
		return err
	}

	switch incomingMessageTypeOnly.Type {
	case "webhook_event":
		var evt WebhookEvent
		if err := json.Unmarshal(data, &evt); err != nil {
			return err
		}

		m.WebhookEvent = &evt
	case "request_log_event":
		var evt RequestLogEvent
		if err := json.Unmarshal(data, &evt); err != nil {
			return err
		}

		m.RequestLogEvent = &evt
	default:
		return fmt.Errorf("Unexpected message type: %s", incomingMessageTypeOnly.Type)
	}

	return nil
}

// MarshalJSON serializes outgoing messages sent to Stripe.
func (m OutgoingMessage) MarshalJSON() ([]byte, error) {
	if m.WebhookResponse != nil {
		return json.Marshal(m.WebhookResponse)
	} else if m.WebhookEventAck != nil {
		return json.Marshal(m.WebhookEventAck)
	}

	return json.Marshal(nil)
}

// OutgoingMessage represents any outgoing message sent to Stripe.
type OutgoingMessage struct {
	*WebhookResponse
	*WebhookEventAck
}
