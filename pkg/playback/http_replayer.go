// Core server logic to replay previously-recorded HTTP interactions

package playback

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
)

// A replayServer is a HTTP server that intercepts incoming requests, and replays recorded responses from the provided cassette.
type replayServer struct {
	replayer   *interactionReplayer
	webhookURL string
	replayLock *sync.WaitGroup // used to
}

func newReplayServer(webhookURL string) (httpReplayer replayServer) {
	httpReplayer = replayServer{}
	httpReplayer.webhookURL = webhookURL
	httpReplayer.replayLock = &sync.WaitGroup{}
	return httpReplayer
}

// Read a previously recorded cassette into the replayer and ready it for replaying.
func (httpReplayer *replayServer) readCassette(reader io.Reader) error {
	// TODO: We may want to allow different types of replay matching in the future (instead of simply sequential playback)
	sequentialComparator := func(req1 interface{}, req2 interface{}) (accept bool, shortCircuitNow bool) {
		return true, true
	}

	replayer, err := newInteractionReplayer(reader, httpRequestfromBytes, httpResponsefromBytes, sequentialComparator)
	if err != nil {
		return err
	}

	httpReplayer.replayer = replayer

	return nil
}

// Respond to incoming Stripe API requests sent to the proxy `playback` server when in REPLAY mode.
// The incoming request is compared with the request/response pairs recorded in the cassette, and the matching response is returned.
// This handler also fires any webhooks that were recorded immediately after the matching response.
func (httpReplayer *replayServer) handler(w http.ResponseWriter, r *http.Request) {
	httpReplayer.replayLock.Wait()       // wait to make sure no webhooks are in the middle of being fired
	httpReplayer.replayLock.Add(1)       // acquire the lock so we can handle this request
	defer httpReplayer.replayLock.Done() // release the lock when handler func is done

	fmt.Printf("\n--> %v to %v", r.Method, r.RequestURI)

	wrappedRequest, err := newHTTPRequest(r)
	if err != nil {
		writeErrorToHTTPResponse(w, err, 500)
		return
	}

	// --- Read matching response from cassette
	var wrappedResponse *httpResponse
	wrappedResponse, err = httpReplayer.getNextRecordedCassetteResponse(&wrappedRequest)
	if err != nil {
		writeErrorToHTTPResponse(w, err, 500)
		return
	}

	fmt.Printf("\n<-- %v from %v\n", wrappedResponse.StatusCode, "CASSETTE")

	// --- Write response back to client
	// The header *must* be written first, since writing the body with implicitly and irreversibly set
	// the status code to 200 if not already set.
	copyHTTPHeader(w.Header(), wrappedResponse.Headers) // header map must be written before calling w.WriteHeader
	w.WriteHeader(wrappedResponse.StatusCode)
	io.Copy(w, bytes.NewBuffer(wrappedResponse.Body))

	// --- Handle webhooks
	// Check the cassette to see if there are pending webhooks we should fire
	webhookRequests, webhookResponses, err := httpReplayer.readAnyPendingWebhookRecordingsFromCassette()

	if err != nil {
		fmt.Printf("Error when checking cassette for webhooks to replay: %v\n", err)
	}
	fmt.Printf("Replaying %d webhooks.\n", len(webhookRequests))

	// Send the webhooks

	// Struct used to parse the event.Type from recorded webhook JSON bodies
	type stripeEvent struct {
		Type string `json:"type"`
	}

	go func() {
		// Note: if there are any errors in processing recorded webhooks here,
		// we log the error and keep going.
		// TODO: is the above the appropriate behavior?
		httpReplayer.replayLock.Wait()       // only send webhooks after the previous request/response is handled
		httpReplayer.replayLock.Add(1)       // acquire lock so we can send webhooks
		defer httpReplayer.replayLock.Done() // release lock when done sending webhooks

		for i, webhookReq := range webhookRequests {
			resp, err := forwardRequest(webhookReq, httpReplayer.webhookURL)
			if err != nil {
				fmt.Printf("ERROR when forwarding webhook requests: %v", err)
				continue
			}
			defer resp.Body.Close()

			expectedResp := webhookResponses[i]
			var evt stripeEvent

			if err != nil {
				fmt.Printf("ERROR when forwarding webhook requests: %v", err)
				continue
			}
			err = json.Unmarshal(webhookReq.Body, &evt)
			if err != nil {
				fmt.Printf("ERROR when forwarding webhook requests: %v", err)
				continue
			}

			fmt.Printf("	> Forwarding webhook [%v].\n", evt.Type)
			fmt.Printf("	> Received %v from client. Expected %v.\n\n", resp.StatusCode, expectedResp.StatusCode)
			if err != nil {
				fmt.Printf("ERROR when forwarding webhook requests: %v", err)
				continue
			}
		}
	}()
}

// returns error if something doesn't match the cassette
func (httpReplayer *replayServer) getNextRecordedCassetteResponse(request *httpRequest) (response *httpResponse, err error) {
	// the passed in request arg may not be necessary
	uncastResponse, err := httpReplayer.replayer.write(request)
	if err != nil {
		return &httpResponse{}, err
	}

	wrappedResponse := (*uncastResponse).(httpResponse)

	return &wrappedResponse, err
}

// Reads any contiguous set of webhook recordings from the start of the cassette
func (httpReplayer *replayServer) readAnyPendingWebhookRecordingsFromCassette() (webhookRequests []*httpRequest, webhookResponses []*httpResponse, err error) {
	webhookBytes := make([]cassettePair, 0)

	// --- Read the pending webhook interactions (stored as raw bytes) from the cassette
	for httpReplayer.replayer.interactionsRemaining() > 0 {
		interaction, err := httpReplayer.replayer.peekFront()
		if err != nil {
			return nil, nil, fmt.Errorf("Error when checking webhooks: %w", err)
		}

		if interaction.Type == incomingInteraction {
			webhookBytes = append(webhookBytes, interaction)
			_, err = httpReplayer.replayer.popFront()
			if err != nil {
				return nil, nil, fmt.Errorf("Unexpectedly reached end of cassette when checking for pending webhooks: %w", err)
			}
		} else {
			break
		}
	}

	// --- Deserialize the bytes into HTTP request & response pairs
	webhookRequests = make([]*httpRequest, 0)
	webhookResponses = make([]*httpResponse, 0)
	for _, rawWebhookBytes := range webhookBytes {
		var reqReader io.Reader = bytes.NewReader(rawWebhookBytes.Request)
		rawWhReq, err := httpRequestfromBytes(&reqReader)
		if err != nil {
			return nil, nil, fmt.Errorf("Error when deserializing cassette to replay webhooks: %w", err)
		}
		whReq := rawWhReq.(httpRequest)

		webhookRequests = append(webhookRequests, &whReq)

		var respReader io.Reader = bytes.NewReader(rawWebhookBytes.Response)
		rawWhResp, err := httpResponsefromBytes(&respReader)
		if err != nil {
			return nil, nil, fmt.Errorf("Error when deserializing cassette to replay webhooks: %w", err)
		}
		whResp := rawWhResp.(httpResponse)

		webhookResponses = append(webhookResponses, &whResp)
	}

	return webhookRequests, webhookResponses, nil
}

func (httpReplayer *replayServer) initializeServer(address string) *http.Server {
	customMux := http.NewServeMux()
	customMux.HandleFunc("/", httpReplayer.handler)

	server := &http.Server{Addr: address, Handler: customMux}

	return server
}
