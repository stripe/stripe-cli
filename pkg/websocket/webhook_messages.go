package websocket

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

// WebhookEventAck represents outgoing Ack message for a webhook event
// received by Stripe.
type WebhookEventAck struct {
	Type                  string `json:"type"` // always "webhook_event_ack"
	WebhookConversationID string `json:"webhook_conversation_id"`
	WebhookID             string `json:"webhook_id"` // ID of the webhook event
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

// NewWebhookEventAck returns a new WebhookEventAck message.
func NewWebhookEventAck(webhookID, webhookConversationID string) *OutgoingMessage {
	return &OutgoingMessage{
		WebhookEventAck: &WebhookEventAck{
			WebhookID:             webhookID,
			WebhookConversationID: webhookConversationID,
			Type:                  "webhook_event_ack",
		},
	}
}
