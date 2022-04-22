package requests

import (
	"net/http"
	"net/http/httptrace"

	log "github.com/sirupsen/logrus"
)

// TracedTransport is an http.RoundTripper that keeps track of the in-flight
// request and implements hooks to report HTTP tracing events
// this is a different RoundTripper implementation to stripe.verboseTransport
// and is not designed for Stripe API requests
type TracedTransport struct {
	current *http.Request
}

// RoundTrip wraps http.DefaultTransport.RoundTrip to keep track
// of the current request
func (t *TracedTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	t.current = req
	return http.DefaultTransport.RoundTrip(req)
}

// GotConn will trace log each connection for the current request
func (t *TracedTransport) GotConn(connInfo httptrace.GotConnInfo) {
	log.WithFields(log.Fields{
		"prefix":   "requests.TracedTransport",
		"connInfo": connInfo,
	}).Tracef("Connection trace for %v: %v", t.current.URL, connInfo)
}

// DNSDone will trace log each DNS lookup for the current request
func (t *TracedTransport) DNSDone(dnsInfo httptrace.DNSDoneInfo) {
	log.WithFields(log.Fields{
		"prefix":  "requests.TracedTransport",
		"dnsInfo": dnsInfo,
	}).Tracef("DNS trace for %v", t.current.URL)
}
