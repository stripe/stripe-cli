// Core server logic to record HTTP interactions by acting as a proxy server

package playback

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// A recordServer proxies requests to a remote host, and records all interactions.
type recordServer struct {
	recorder   *interactionRecorder
	remoteURL  string
	webhookURL string
}

func newRecordServer(remoteURL string, webhookURL string) (httpRecorder recordServer) {
	httpRecorder = recordServer{}
	httpRecorder.remoteURL = remoteURL
	httpRecorder.webhookURL = webhookURL

	return httpRecorder
}

// Prepare the recorder to start recording to a new cassette.
func (httpRecorder *recordServer) insertCassette(writer io.Writer) error {
	recorder, err := newInteractionRecorder(writer, httpRequestToBytes, httpResponseToBytes)
	if err != nil {
		return err
	}
	httpRecorder.recorder = recorder

	return nil
}

// Handler for the webhook endpoint that forwards incoming webhooks to the local application,
// while recording the webhook and local app's response to the cassette.
func (httpRecorder *recordServer) webhookHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("\n[WEBHOOK] --> STRIPE %v to %v --> POST to %v", r.Method, r.RequestURI, httpRecorder.webhookURL)

	wrappedReq, err := newHTTPRequest(r)
	if err != nil {
		writeErrorToHTTPResponse(w, fmt.Errorf("Unexpected error processing incoming webhook request: %w", err), 500)
		return
	}

	// --- Pass request to local webhook endpoint
	resp, err := forwardRequest(&wrappedReq, httpRecorder.webhookURL)
	// TODO: this response is going back to Stripe, what is the correct error handling logic here?
	if err != nil {
		writeErrorToHTTPResponse(w, fmt.Errorf("Unexpected error forwarding webhook to client: %w", err), 500)
		return
	}
	wrappedResp, err := newHTTPResponse(resp)
	if err != nil {
		writeErrorToHTTPResponse(w, fmt.Errorf("Unexpected error forwarding webhook to client: %w", err), 500)
		return
	}

	// --- Write response back to client

	// We defer writing anything to the response until their final values are known, since certain fields can
	// only be written once. (golang's implementation streams the response, and immediately writes data as it is set)
	if err != nil {
		writeErrorToHTTPResponse(w, fmt.Errorf("Unexpected error forwarding webhook to client: %w", err), 500)
		return
	}
	err = httpRecorder.recorder.write(incomingInteraction, wrappedReq, wrappedResp)
	if err != nil {
		writeErrorToHTTPResponse(w, fmt.Errorf("Unexpected error writing webhook interaction to cassette: %w", err), 500)
		return
	}

	// Now we can write to the response:
	// The header *must* be written first, since writing the body with implicitly and irreversibly set
	// the status code to 200 if not already set.
	copyHTTPHeader(w.Header(), wrappedResp.Headers) // header map must be written before calling w.WriteHeader
	w.WriteHeader(wrappedResp.StatusCode)
	io.Copy(w, bytes.NewBuffer(wrappedResp.Body))
}

// Respond to incoming Stripe API requests sent to the proxy `playback` server when in REPLAY mode.
// The incoming requests are forwarded to the real Stripe API, and the resulting response is passed along to the original client.
// The original request and Stripe API response are recorded to the cassette.
func (httpRecorder *recordServer) handler(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("\n--> %v to %v", r.Method, r.RequestURI)

	wrappedReq, err := newHTTPRequest(r)
	if err != nil {
		writeErrorToHTTPResponse(w, fmt.Errorf("Unexpected error processing incoming webhook request: %w", err), 500)
		return
	}

	// --- Pass request to remote
	var resp *http.Response

	resp, err = forwardRequest(&wrappedReq, httpRecorder.remoteURL+r.RequestURI)
	if err != nil {
		writeErrorToHTTPResponse(w, err, 500)
		return
	}

	wrappedResp, err := newHTTPResponse(resp)
	if err != nil {
		writeErrorToHTTPResponse(w, err, 500)
		return
	}

	fmt.Printf("\n<-- %v from %v\n", resp.Status, strings.ToUpper(httpRecorder.remoteURL))

	// --- Write response back to client

	// We defer writing anything to the response until their final values are known, since certain fields can
	// only be written once. (golang's implementation streams the response, and immediately writes data as it is set)
	if err != nil {
		writeErrorToHTTPResponse(w, err, 500)
		return
	}
	err = httpRecorder.recorder.write(outgoingInteraction, wrappedReq, wrappedResp)
	if err != nil {
		writeErrorToHTTPResponse(w, fmt.Errorf("Error when recording HTTP response to cassette: %w", err), 500)
		return
	}
	// Now we can write to the response:
	// The header *must* be written first, since writing the body with implicitly and irreversibly set
	// the status code to 200 if not already set.
	copyHTTPHeader(w.Header(), wrappedResp.Headers) // header map must be written before calling w.WriteHeader
	w.WriteHeader(wrappedResp.StatusCode)
	io.Copy(w, bytes.NewBuffer(wrappedResp.Body))
}

func (httpRecorder *recordServer) initializeServer(address string) *http.Server {
	customMux := http.NewServeMux()
	server := &http.Server{Addr: address, Handler: customMux}

	// --- Recorder control handlers
	customMux.HandleFunc("/playback/stop", func(w http.ResponseWriter, r *http.Request) {
		fmt.Println()
		fmt.Println("Received /playback/stop. Stopping...")

		httpRecorder.recorder.saveAndClose()
	})

	// --- Default handler that proxies request and returns response from the remote
	customMux.HandleFunc("/", httpRecorder.handler)

	return server
}
