package websocket

// RequestLogEvent represents incoming request log event messages sent by Stripe.
type RequestLogEvent struct {
	EventPayload string            `json:"event_payload"`
	Type         string            `json:"type"`
	RequestLogID string            `json:"request_log_id"`
}
