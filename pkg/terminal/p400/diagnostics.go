package p400

import (
	"net"
	"net/http"
	"net/http/httptrace"
)

// Transport is used when we're tracing a Rabbit Service call, in order to surface DNS and connectivity related data / errors
// it helps provide a more specific / succinct error to the user in order to be more helpful when they run into trouble
type Transport struct {
	DNSIPs []net.IPAddr
	Err    error
}

// RoundTrip is a hook called from http client tracing to inspect for specific tcp dialing issues when calling Rabbit Service.
// It allows for a less verbose error than what occurs further down the call stack, and attaches it to the Transport instance
// It returns the response and any error unmodified in order to continue the completion of the http client's request work
func (t *Transport) RoundTrip(req *http.Request) (*http.Response, error) {
	res, err := http.DefaultTransport.RoundTrip(req)
	if err != nil {
		t.Err = err
	}

	return res, err
}

// DNSDone is a hook called from http client tracing to extract any DNS service IP resolutions from the tcp dialing of Rabbit Service
// it attaches this list of resolved IPs to the Transport instance so we can later check if any errors were due to DNS not resolving to a valid IP
func (t *Transport) DNSDone(info httptrace.DNSDoneInfo) {
	t.DNSIPs = info.Addrs
}
