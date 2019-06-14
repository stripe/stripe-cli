package websocket

import (
	"encoding/json"
	"fmt"
)

// WebhookEndpoint contains properties about the fake "endpoint" used to
// format the webhook event.
type WebhookEndpoint struct {
	APIVersion *string `json:"api_version"`
}

// WebhookEvent represents incoming webhook event messages sent by Stripe.
type WebhookEvent struct {
	Endpoint     WebhookEndpoint   `json:"endpoint"`
	EventPayload string            `json:"event_payload"`
	HTTPHeaders  map[string]string `json:"http_headers"`
	Type         string            `json:"type"`
	WebhookID    string            `json:"webhook_id"`
}

// IncomingMessage represents any incoming message sent by Stripe.
type IncomingMessage struct {
	*WebhookEvent
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
	if incomingMessageTypeOnly.Type == "webhook_event" {
		var evt WebhookEvent
		if err := json.Unmarshal(data, &evt); err != nil {
			return err
		}
		m.WebhookEvent = &evt
	} else {
		return fmt.Errorf("Unexpected message type: %s", incomingMessageTypeOnly.Type)
	}
	return nil
}

// WebhookResponse represents outgoing webhook response messages sent to
// Stripe.
type WebhookResponse struct {
	Status      int               `json:"status"`
	HTTPHeaders map[string]string `json:"http_headers"`
	Body        string            `json:"body"`
	Type        string            `json:"type"`
	WebhookID   string            `json:"webhook_id"`
}

// MarshalJSON serializes outgoing messages sent to Stripe.
func (m OutgoingMessage) MarshalJSON() ([]byte, error) {
	if m.WebhookResponse != nil {
		return json.Marshal(m.WebhookResponse)
	}

	return json.Marshal(nil)
}

// NewWebhookResponse returns a new webhookResponse message.
func NewWebhookResponse(webhookID string, status int, body string, headers map[string]string) *OutgoingMessage {
	return &OutgoingMessage{
		WebhookResponse: &WebhookResponse{
			WebhookID:   webhookID,
			Status:      status,
			Body:        body,
			HTTPHeaders: headers,
			Type:        "webhook_response",
		},
	}
}

// OutgoingMessage represents any outgoing message sent to Stripe.
type OutgoingMessage struct {
	*WebhookResponse
}
