package stripeauth

// StripeCLISession is the API resource returned by Stripe when initiating
// a new CLI session.
type StripeCLISession struct {
	ReconnectDelay   int              `json:"reconnect_delay"`
	Secret           string           `json:"secret"`
	WebSocketAuthorizedFeature string `json:"websocket_authorized_feature"`
	WebSocketID      string           `json:"websocket_id"`
	WebSocketURL     string           `json:"websocket_url"`
}
