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
	} else if m.EventAck != nil {
		return json.Marshal(m.EventAck)
	}

	return json.Marshal(nil)
}

// EventAck represents outgoing Ack messages
// for events received by Stripe.
type EventAck struct {
	Type                  string `json:"type"` // always "event_ack"
	WebhookConversationID string `json:"webhook_conversation_id"`
	EventID               string `json:"event_id"` // ID of the event
}

// NewEventAck returns a new EventAck message.
func NewEventAck(eventID, webhookConversationID string) *OutgoingMessage {
	return &OutgoingMessage{
		EventAck: &EventAck{
			EventID:               eventID,
			WebhookConversationID: webhookConversationID,
			Type:                  "event_ack",
		},
	}
}

// OutgoingMessage represents any outgoing message sent to Stripe.
type OutgoingMessage struct {
	*WebhookResponse
	*EventAck
}
