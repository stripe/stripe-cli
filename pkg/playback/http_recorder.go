// Core server logic to record HTTP interactions by acting as a proxy server

package playback

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	log "github.com/sirupsen/logrus"
)

// A recordServer proxies requests to a remote host, and records all interactions.
type recordServer struct {
	recorder   *interactionRecorder
	remoteURL  string
	webhookURL string

	log *log.Logger
}

func newRecordServer(remoteURL string, webhookURL string) (httpRecorder recordServer) {
	httpRecorder = recordServer{}
	httpRecorder.remoteURL = remoteURL
	httpRecorder.webhookURL = webhookURL

	httpRecorder.log = log.New()

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

// Struct used to parse the event.Type from recorded webhook JSON bodies
type stripeEvent struct {
	Type string `json:"type"`
}

// Handler for the webhook endpoint that forwards incoming webhooks to the local application,
// while recording the webhook and local app's response to the cassette.
func (httpRecorder *recordServer) webhookHandler(w http.ResponseWriter, r *http.Request) {
	wrappedReq, err := newHTTPRequest(r)
	if err != nil {
		writeErrorToHTTPResponse(w, httpRecorder.log, fmt.Errorf("unexpected error processing incoming webhook request: %w", err), 500)
		return
	}

	var evt stripeEvent
	err = json.Unmarshal(wrappedReq.Body, &evt)
	if err != nil {
		writeErrorToHTTPResponse(w, httpRecorder.log, fmt.Errorf("unexpected error processing incoming webhook request: %w", err), 500)
		return
	}

	// --- Pass request to local webhook endpoint
	httpRecorder.log.Infof("[WEBHOOK] %v [%v] to %v --> FORWARDED to %v", r.Method, evt.Type, r.RequestURI, httpRecorder.webhookURL)

	resp, err := forwardRequest(&wrappedReq, httpRecorder.webhookURL)
	// NOTE: this response is going back to Stripe, so for internal `playback` server errors - return a 500 response with an error msg in the body
	// The details of this response will be visible on the Developer Dashboard under "Webhook CLI Responses"
	// (^ this all assuming that playback is using `stripe listen` to receive webhooks)
	if err != nil {
		writeErrorToHTTPResponse(w, httpRecorder.log, fmt.Errorf("unexpected error forwarding [%v] webhook to client: %w", evt.Type, err), 500)
		return
	}
	wrappedResp, err := newHTTPResponse(resp)
	if err != nil {
		writeErrorToHTTPResponse(w, httpRecorder.log, fmt.Errorf("unexpected error forwarding [%v] webhook to client: %w", evt.Type, err), 500)
		return
	}

	// --- Write response back to client

	// We defer writing anything to the response until their final values are known, since certain fields can
	// only be written once. (golang's implementation streams the response, and immediately writes data as it is set)
	err = httpRecorder.recorder.write(incomingInteraction, wrappedReq, wrappedResp)
	if err != nil {
		writeErrorToHTTPResponse(w, httpRecorder.log, fmt.Errorf("unexpected error writing [%v] webhook interaction to cassette: %w", evt.Type, err), 500)
		return
	}

	// Now we can write to the response:
	// The header *must* be written first, since writing the body with implicitly and irreversibly set
	// the status code to 200 if not already set.
	copyHTTPHeader(w.Header(), wrappedResp.Headers) // header map must be written before calling w.WriteHeader
	w.WriteHeader(wrappedResp.StatusCode)
	_, err = io.Copy(w, bytes.NewBuffer(wrappedResp.Body))

	// Since at this point, we can't signal an error by writing the HTTP status code, and this is a significant failure - we log.Fatal
	// so that the failure is clear to the user.
	if err != nil {
		httpRecorder.log.Fatal(err)
	}
}

// Respond to incoming Stripe API requests sent to the proxy `playback` server when in REPLAY mode.
// The incoming requests are forwarded to the real Stripe API, and the resulting response is passed along to the original client.
// The original request and Stripe API response are recorded to the cassette.
func (httpRecorder *recordServer) handler(w http.ResponseWriter, r *http.Request) {
	httpRecorder.log.Infof("--> %v to %v", r.Method, r.RequestURI)

	wrappedReq, err := newHTTPRequest(r)
	if err != nil {
		writeErrorToHTTPResponse(w, httpRecorder.log, fmt.Errorf("unexpected error processing incoming API request: %w", err), 500)
		return
	}

	// --- Pass request to remote
	var resp *http.Response

	resp, err = forwardRequest(&wrappedReq, httpRecorder.remoteURL+r.RequestURI)
	if err != nil {
		writeErrorToHTTPResponse(w, httpRecorder.log, fmt.Errorf("unexpected error processing incoming API request: %w", err), 500)
		return
	}

	wrappedResp, err := newHTTPResponse(resp)
	if err != nil {
		writeErrorToHTTPResponse(w, httpRecorder.log, fmt.Errorf("unexpected error processing incoming API request: %w", err), 500)
		return
	}

	httpRecorder.log.Infof("<-- %v from %v", resp.Status, strings.ToUpper(httpRecorder.remoteURL))

	// --- Write response back to client

	// We defer writing anything to the response until their final values are known, since certain fields can
	// only be written once. (golang's implementation streams the response, and immediately writes data as it is set)
	err = httpRecorder.recorder.write(outgoingInteraction, wrappedReq, wrappedResp)
	if err != nil {
		writeErrorToHTTPResponse(w, httpRecorder.log, fmt.Errorf("error when recording HTTP response to cassette: %w", err), 500)
		return
	}
	// Now we can write to the response:
	// The header *must* be written first, since writing the body with implicitly and irreversibly set
	// the status code to 200 if not already set.
	copyHTTPHeader(w.Header(), wrappedResp.Headers) // header map must be written before calling w.WriteHeader
	w.WriteHeader(wrappedResp.StatusCode)
	_, err = io.Copy(w, bytes.NewBuffer(wrappedResp.Body))

	// Since at this point, we can't signal an error by writing the HTTP status code, and this is a significant failure - we log.Fatal
	// so that the failure is clear to the user.
	if err != nil {
		httpRecorder.log.Fatal(err)
	}
}
