package websocket

// RequestLogEvent represents incoming request log event messages sent by Stripe.
type RequestLogEvent struct {
	EventPayload string `json:"event_payload"`
	RequestLogID string `json:"request_log_id"`
	Type         string `json:"type"`
}
