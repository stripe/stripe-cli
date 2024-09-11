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

// WebhookResponse represents outgoing webhook response messages sent to
// Stripe.
type WebhookResponse struct {
	ForwardURL            string            `json:"forward_url"`
	Status                int               `json:"status"`
	HTTPHeaders           map[string]string `json:"http_headers"`
	Body                  string            `json:"body"`
	Type                  string            `json:"type"`
	WebhookConversationID string            `json:"webhook_conversation_id,omitempty"`
	WebhookID             string            `json:"webhook_id,omitempty"`
	RequestHeaders        map[string]string `json:"request_headers"`
	RequestBody           string            `json:"request_body"`
	NotificationID        string            `json:"notification_id,omitempty"`
}

// NewWebhookResponse returns a new webhookResponse message.
func NewWebhookResponse(webhookID, webhookConversationID, forwardURL string, status int, body string, headers map[string]string, requestBody string, requestHeaders map[string]string, notificationID string) *OutgoingMessage {
	return &OutgoingMessage{
		WebhookResponse: &WebhookResponse{
			WebhookID:             webhookID,
			WebhookConversationID: webhookConversationID,
			ForwardURL:            forwardURL,
			Status:                status,
			Body:                  body,
			HTTPHeaders:           headers,
			Type:                  "webhook_response",
			RequestHeaders:        requestHeaders,
			RequestBody:           requestBody,
			NotificationID:        notificationID,
		},
	}
}

// StripeV2Event is the websocket wire representation of a v2 event.
type StripeV2Event struct {
	// Type is v2_event
	Type string `json:"type"`

	// Payload
	HTTPHeaders        map[string]string `json:"http_headers"`
	Payload            string            `json:"payload"`
	EventDestinationID string            `json:"destination_id"`
}
