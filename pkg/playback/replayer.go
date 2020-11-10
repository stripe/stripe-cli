// Core server logic to replay previously-recorded HTTP interactions

package playback

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"

	log "github.com/sirupsen/logrus"
)

// An HTTPReplayer receives incoming requests and returns recorded responses from the provided cassette.
type HTTPReplayer struct {
	replayer   *interactionReplayer
	webhookURL string
	replayLock *sync.WaitGroup // used to

	log *log.Logger
}

func newHTTPReplayer(webhookURL string) (httpReplayer HTTPReplayer) {
	httpReplayer = HTTPReplayer{}
	httpReplayer.webhookURL = webhookURL
	httpReplayer.replayLock = &sync.WaitGroup{}

	httpReplayer.log = log.New()

	return httpReplayer
}

// Reads a cassette file, decodes it with serializer and loads it in the cassette.
func (httpReplayer *HTTPReplayer) readCassette(reader io.Reader) error {
	// TODO(DX-5701): We may want to allow different types of replay matching in the future (instead of simply sequential playback)
	sequentialComparator := func(req1 interface{}, req2 interface{}) (accept bool, shortCircuitNow bool) {
		return true, true
	}

	replayer, err := newInteractionReplayer(reader, YAMLSerializer{}, sequentialComparator)
	if err != nil {
		return err
	}

	httpReplayer.replayer = replayer

	return nil
}

// Respond to incoming Stripe API requests sent to the proxy `playback` server when in REPLAY mode.
// The incoming request is compared with the request/response pairs recorded in the cassette, and the matching response is returned.
// This handler also fires any webhooks that were recorded immediately after the matching response.
func (httpReplayer *HTTPReplayer) handler(w http.ResponseWriter, r *http.Request) {
	httpReplayer.replayLock.Wait()       // wait to make sure no webhooks are in the middle of being fired
	httpReplayer.replayLock.Add(1)       // acquire the lock so we can handle this request
	defer httpReplayer.replayLock.Done() // release the lock when handler func is done

	httpReplayer.log.Infof("--> %v to %v", r.Method, r.RequestURI)

	wrappedRequest, err := newHTTPRequest(r)
	if err != nil {
		writeErrorToHTTPResponse(w, httpReplayer.log, err, 500)
		return
	}

	// --- Read matching response from cassette
	var wrappedResponse *httpResponse
	wrappedResponse, err = httpReplayer.getNextRecordedCassetteResponse(&wrappedRequest)
	if err != nil {
		writeErrorToHTTPResponse(w, httpReplayer.log, err, 500)
		return
	}

	httpReplayer.log.Infof("<-- %v from %v", wrappedResponse.StatusCode, "CASSETTE")

	// --- Write response back to client
	// The header *must* be written first, since writing the body with implicitly and irreversibly set
	// the status code to 200 if not already set.
	copyHTTPHeader(w.Header(), wrappedResponse.Headers) // header map must be written before calling w.WriteHeader
	w.WriteHeader(wrappedResponse.StatusCode)
	_, err = io.Copy(w, bytes.NewBuffer(wrappedResponse.Body))
	if err != nil {
		httpReplayer.log.Fatal(err)
	}

	// --- Handle webhooks
	// Check the cassette to see if there are pending webhooks we should fire
	webhookRequests, webhookResponses, err := httpReplayer.readAnyPendingWebhookRecordingsFromCassette()

	if err != nil {
		httpReplayer.log.Errorf("Error when checking cassette for webhooks to replay: %v", err)
	}
	httpReplayer.log.Infof("Replaying %d webhooks", len(webhookRequests))

	// Send the webhooks
	go func() {
		// Note: if there are any errors in processing recorded webhooks here,
		// we log the error and keep going.
		httpReplayer.replayLock.Wait()       // only send webhooks after the previous request/response is handled
		httpReplayer.replayLock.Add(1)       // acquire lock so we can send webhooks
		defer httpReplayer.replayLock.Done() // release lock when done sending webhooks

		for i, webhookReq := range webhookRequests {
			var evt stripeEvent
			err = json.Unmarshal(webhookReq.Body, &evt)
			if err != nil {
				httpReplayer.log.Errorf("Error when forwarding webhook request [%v]: %v", evt.Type, err)
			}

			resp, err := forwardRequest(webhookReq, httpReplayer.webhookURL)
			if err != nil {
				httpReplayer.log.Errorf("Error when forwarding webhook request [%v]: %v", evt.Type, err)
				continue
			}
			defer resp.Body.Close()

			expectedResp := webhookResponses[i]

			if err != nil {
				httpReplayer.log.Errorf("Error when forwarding webhook request [%v]: %v", evt.Type, err)
				continue
			}

			httpReplayer.log.Infof("	> Forwarding webhook [%v].\n", evt.Type)
			httpReplayer.log.Infof("	> Received %v from client. Expected %v.\n\n", resp.StatusCode, expectedResp.StatusCode)
			if err != nil {
				httpReplayer.log.Errorf("Error when forwarding webhook request [%v]: %v", evt.Type, err)
				continue
			}
		}
	}()
}

// returns error if something doesn't match the cassette
func (httpReplayer *HTTPReplayer) getNextRecordedCassetteResponse(request *httpRequest) (response *httpResponse, err error) {
	// the passed in request arg may not be necessary
	uncastResponse, err := httpReplayer.replayer.write(request)
	if err != nil {
		return &httpResponse{}, err
	}

	wrappedResponse := (*uncastResponse).(httpResponse)

	return &wrappedResponse, err
}

// Reads any contiguous set of webhook recordings from the start of the cassette
func (httpReplayer *HTTPReplayer) readAnyPendingWebhookRecordingsFromCassette() (webhookRequests []*httpRequest, webhookResponses []*httpResponse, err error) {
	interactions := make([]interaction, 0)

	// --- Read the pending webhook interactions (stored as raw bytes) from the cassette
	for httpReplayer.replayer.interactionsRemaining() > 0 {
		interaction, err := httpReplayer.replayer.peekFront()
		if err != nil {
			return nil, nil, fmt.Errorf("error when checking webhooks: %w", err)
		}

		if interaction.Type == incomingInteraction {
			interactions = append(interactions, interaction)
			_, err = httpReplayer.replayer.popFront()
			if err != nil {
				return nil, nil, fmt.Errorf("unexpectedly reached end of cassette when checking for pending webhooks: %w", err)
			}
		} else {
			break
		}
	}

	// --- Deserialize the bytes into HTTP request & response pairs
	webhookRequests = make([]*httpRequest, 0)
	webhookResponses = make([]*httpResponse, 0)
	for _, interaction := range interactions {
		webhookRequests = append(webhookRequests, &interaction.Request)
		webhookResponses = append(webhookResponses, &interaction.Response)
	}

	return webhookRequests, webhookResponses, nil
}
