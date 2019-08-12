package websocket

// RequestLogEvent represents incoming request log event messages sent by Stripe.
type RequestLogEvent struct {
	EventPayload string `json:"event_payload"`

	// RequestLogID is the `resp_` id for the response event which is used as the request log event throughout the system.
	// This is different from the `EventPayload.RequestID` which is the `req_` id for the user's actual request, which they
	// can use to find their request in the dashboard.
	RequestLogID string `json:"request_log_id"`
	Type         string `json:"type"`
}
