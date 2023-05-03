package websocket

import (
	"encoding/json"
)

// IncomingMessage represents any incoming message sent by Stripe.
type IncomingMessage struct {
	*WebhookEvent
	*RequestLogEvent

	// Unknown will be present if the incoming message type does not match the
	// list known to the CLI.
	Unknown *UnknownMessage
}

// UnknownMessage represents an incoming message with a type that's unknown
// to the CLI, and therefore cannot be deserialized into a structured type.
type UnknownMessage struct {
	// Type is the value of the type field in the message's data.
	Type string

	// Data contains the raw data of the message.
	Data []byte
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
		if err := json.Unmarshal(data, &m.WebhookEvent); err != nil {
			return err
		}
	case "request_log_event":
		if err := json.Unmarshal(data, &m.RequestLogEvent); err != nil {
			return err
		}
	default:
		m.Unknown = &UnknownMessage{
			Type: incomingMessageTypeOnly.Type,
			Data: data,
		}
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
