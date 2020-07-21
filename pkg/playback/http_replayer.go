package playback

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
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

func (httpReplayer *replayServer) handler(w http.ResponseWriter, r *http.Request) {
	httpReplayer.replayLock.Wait()       // wait to make sure no webhooks are in the middle of being fired
	httpReplayer.replayLock.Add(1)       // acquire the lock so we can handle this request
	defer httpReplayer.replayLock.Done() // release the lock when handler func is done

	fmt.Printf("\n--> %v to %v", r.Method, r.RequestURI)

	// --- Read matching response from cassette
	var resp *http.Response
	var err error
	resp, err = httpReplayer.getNextRecordedCassetteResponse(r)
	if err != nil {
		writeErrorToHTTPResponse(w, err, 500)
		return
	}
	fmt.Printf("\n<-- %v from %v\n", resp.Status, "CASSETTE")
	defer resp.Body.Close() // we need to close the body

	// --- Write response back to client
	var bodyBytes []byte
	bodyBytes, err = ioutil.ReadAll(resp.Body)
	defer resp.Body.Close()
	if err != nil {
		writeErrorToHTTPResponse(w, err, 500)
		return
	}

	// The header *must* be written first, since writing the body with implicitly and irreversibly set
	// the status code to 200 if not already set.
	w.WriteHeader(resp.StatusCode)
	copyHTTPHeader(w.Header(), resp.Header)
	io.Copy(w, bytes.NewBuffer(bodyBytes))
	resp.Body = ioutil.NopCloser(bytes.NewBuffer(bodyBytes))
	defer resp.Body.Close()

	// --- Handle webhooks
	// Check the cassette to see if there are pending webhooks we should fire

	// Note on calling resp.Body().Close() and avoiding a resource leak: all response bodies will be closed in the goroutine below
	webhookRequests, webhookResponses, err := httpReplayer.readAnyPendingWebhookRecordingsFromCassette() // nolint

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

			webhookPayload, err := ioutil.ReadAll(webhookReq.Body)
			defer webhookReq.Body.Close()
			if err != nil {
				fmt.Printf("ERROR when forwarding webhook requests: %v", err)
				continue
			}
			err = json.Unmarshal(webhookPayload, &evt)
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
func (httpReplayer *replayServer) getNextRecordedCassetteResponse(request *http.Request) (resp *http.Response, err error) {
	// the passed in request arg may not be necessary
	responseWrapper, err := httpReplayer.replayer.write(request)
	if err != nil {
		return &http.Response{}, err
	}

	response := (*responseWrapper).(*http.Response)

	return response, err
}

// Reads any contiguous set of webhook recordings from the start of the cassette
// The caller must call response.Body().Close() on each of the returned http.Response's, otherwise there will be a resource leak
func (httpReplayer *replayServer) readAnyPendingWebhookRecordingsFromCassette() (webhookRequests []*http.Request, webhookResponses []*http.Response, err error) {
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
	webhookRequests = make([]*http.Request, 0)
	webhookResponses = make([]*http.Response, 0)
	for _, rawWebhookBytes := range webhookBytes {
		var reqReader io.Reader = bytes.NewReader(rawWebhookBytes.Request)
		webhookHTTPRequest, err := httpRequestfromBytes(&reqReader)
		if err != nil {
			return nil, nil, fmt.Errorf("Error when deserializing cassette to replay webhooks: %w", err)
		}

		webhookRequests = append(webhookRequests, webhookHTTPRequest.(*http.Request))

		var respReader io.Reader = bytes.NewReader(rawWebhookBytes.Response)
		webhookHTTPResponse, err := httpResponsefromBytes(&respReader)
		if err != nil {
			return nil, nil, fmt.Errorf("Error when deserializing cassette to replay webhooks: %w", err)
		}

		// Caller of this function is expected to call .Body().Close() on each http.Response
		webhookResponses = append(webhookResponses, webhookHTTPResponse.(*http.Response)) // nolint
	}

	return webhookRequests, webhookResponses, nil
}

func (httpReplayer *replayServer) initializeServer(address string) *http.Server {
	customMux := http.NewServeMux()
	customMux.HandleFunc("/", httpReplayer.handler)

	server := &http.Server{Addr: address, Handler: customMux}

	return server
}
