package websocket

import (
	"net/http"
)

// WebhookEndpoint contains properties about the fake "endpoint" used to
// format the webhook event.
type WebhookEndpoint struct {
	APIVersion *string `json:"api_version"`
}

// WebhookEvent represents incoming webhook event messages sent by Stripe.
type WebhookEvent struct {
	Endpoint              WebhookEndpoint   `json:"endpoint"`
	EventPayload          string            `json:"event_payload"`
	HTTPHeaders           map[string]string `json:"http_headers"`
	Type                  string            `json:"type"`
	WebhookConversationID string            `json:"webhook_conversation_id"`
	WebhookID             string            `json:"webhook_id"`
}

// WebhookResponse represents outgoing webhook response messages sent to
// Stripe.
type WebhookResponse struct {
	ForwardURL            string            `json:"forward_url"`
	Status                int               `json:"status"`
	HTTPHeaders           map[string]string `json:"http_headers"`
	Body                  string            `json:"body"`
	Type                  string            `json:"type"`
	WebhookConversationID string            `json:"webhook_conversation_id"`
	WebhookID             string            `json:"webhook_id"`
}

// NewWebhookResponse returns a new webhookResponse message.
func NewWebhookResponse(webhookID, webhookConversationID, forwardURL string, status int, body string, headers map[string]string) *OutgoingMessage {
	return &OutgoingMessage{
		WebhookResponse: &WebhookResponse{
			WebhookID:             webhookID,
			WebhookConversationID: webhookConversationID,
			ForwardURL:            forwardURL,
			Status:                status,
			Body:                  body,
			HTTPHeaders:           headers,
			Type:                  "webhook_response",
		},
	}
}

// StripeV2Event is the websocket wire representation of a v2 event.
type StripeV2Event struct {
	// Type is v2_event
	Type string `json:"type"`

	// Payload
	Payload V2EventPayload `json:"event_payload"`
}

// V2EventWebhookResponse is a tuple of a v2 event and the response returned by
// the server after forwarding the event.
type V2EventWebhookResponse struct {
	Event *V2EventPayload
	Resp  *http.Response
}
