// Core server logic to replay previously-recorded HTTP interactions

package playback

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"sync"

	log "github.com/sirupsen/logrus"
)

// An Replayer receives incoming requests and returns recorded responses from the provided cassette.
type Replayer struct {
	webhookURL string
	replayLock *sync.WaitGroup // used to

	cursor     int
	cassette   Cassette
	comparator requestComparator
	serializer serializer

	log *log.Logger
}

func newReplayer(webhookURL string, serializer serializer, comparator requestComparator) Replayer {
	replayer := Replayer{}
	replayer.cursor = 0
	replayer.webhookURL = webhookURL
	replayer.serializer = serializer
	replayer.comparator = comparator
	replayer.replayLock = &sync.WaitGroup{}

	replayer.log = log.New()

	return replayer
}

// Reads a cassette file, decodes it with serializer and loads it in the cassette.
func (replayer *Replayer) readCassette(reader io.Reader) error {
	buffer, err := ioutil.ReadAll(reader)
	if err != nil {
		return err
	}

	cassette, err := replayer.serializer.DecodeCassette(buffer)
	if err != nil {
		return err
	}

	replayer.cassette = cassette

	return nil
}

// write parses the cassette for matching responses and returns them
// core "replay" logic
func (replayer *Replayer) write(req *httpRequest) (resp *interface{}, err error) {
	if len(replayer.cassette) == 0 {
		return nil, errors.New("nothing left in cassette to replay")
	}

	var lastAccepted interface{}
	acceptedIdx := -1

	for idx, interaction := range replayer.cassette {
		accept, shortCircuit := replayer.comparator(interaction.Request, *req)

		if accept {
			lastAccepted = interaction.Response
			acceptedIdx = idx

			if shortCircuit {
				break
			}
		}
	}
	if acceptedIdx != -1 {
		// remove interactions that were accepted from tape
		replayer.cassette = append(replayer.cassette[:acceptedIdx], replayer.cassette[acceptedIdx+1:]...)
		return &lastAccepted, nil
	}

	return nil, errors.New("no matching events")
}

func (replayer *Replayer) interactionsRemaining() int {
	return len(replayer.cassette)
}

func (replayer *Replayer) peekFront() (interaction, error) {
	if len(replayer.cassette) == 0 {
		return interaction{}, errors.New("nothing left in cassette to replay")
	}

	return replayer.cassette[0], nil
}

func (replayer *Replayer) popFront() (interaction, error) {
	if len(replayer.cassette) == 0 {
		return interaction{}, errors.New("nothing left in cassette to replay")
	}

	first := replayer.cassette[0]
	replayer.cassette = replayer.cassette[1:]
	return first, nil
}

// Respond to incoming Stripe API requests sent to the proxy `playback` server when in REPLAY mode.
// The incoming request is compared with the request/response pairs recorded in the cassette, and the matching response is returned.
// This handler also fires any webhooks that were recorded immediately after the matching response.
func (replayer *Replayer) handler(w http.ResponseWriter, r *http.Request) {
	replayer.replayLock.Wait()       // wait to make sure no webhooks are in the middle of being fired
	replayer.replayLock.Add(1)       // acquire the lock so we can handle this request
	defer replayer.replayLock.Done() // release the lock when handler func is done

	replayer.log.Infof("--> %v to %v", r.Method, r.RequestURI)

	wrappedRequest, err := newHTTPRequest(r)
	if err != nil {
		writeErrorToHTTPResponse(w, replayer.log, err, 500)
		return
	}

	// --- Read matching response from cassette
	var wrappedResponse *httpResponse
	wrappedResponse, err = replayer.getNextRecordedCassetteResponse(&wrappedRequest)
	if err != nil {
		writeErrorToHTTPResponse(w, replayer.log, err, 500)
		return
	}

	replayer.log.Infof("<-- %v from %v", wrappedResponse.StatusCode, "CASSETTE")

	// --- Write response back to client
	// The header *must* be written first, since writing the body with implicitly and irreversibly set
	// the status code to 200 if not already set.
	copyHTTPHeader(w.Header(), wrappedResponse.Headers) // header map must be written before calling w.WriteHeader
	w.WriteHeader(wrappedResponse.StatusCode)
	_, err = io.Copy(w, bytes.NewBuffer(wrappedResponse.Body))
	if err != nil {
		replayer.log.Fatal(err)
	}

	// --- Handle webhooks
	// Check the cassette to see if there are pending webhooks we should fire
	webhookRequests, webhookResponses, err := replayer.readAnyPendingWebhookRecordingsFromCassette()

	if err != nil {
		replayer.log.Errorf("Error when checking cassette for webhooks to replay: %v", err)
	}
	replayer.log.Infof("Replaying %d webhooks", len(webhookRequests))

	// Send the webhooks
	go func() {
		// Note: if there are any errors in processing recorded webhooks here,
		// we log the error and keep going.
		replayer.replayLock.Wait()       // only send webhooks after the previous request/response is handled
		replayer.replayLock.Add(1)       // acquire lock so we can send webhooks
		defer replayer.replayLock.Done() // release lock when done sending webhooks

		for i, webhookReq := range webhookRequests {
			var evt stripeEvent
			err = json.Unmarshal(webhookReq.Body, &evt)
			if err != nil {
				replayer.log.Errorf("Error when forwarding webhook request [%v]: %v", evt.Type, err)
			}

			resp, err := forwardRequest(webhookReq, replayer.webhookURL)
			if err != nil {
				replayer.log.Errorf("Error when forwarding webhook request [%v]: %v", evt.Type, err)
				continue
			}
			defer resp.Body.Close()

			expectedResp := webhookResponses[i]

			if err != nil {
				replayer.log.Errorf("Error when forwarding webhook request [%v]: %v", evt.Type, err)
				continue
			}

			replayer.log.Infof("	> Forwarding webhook [%v].\n", evt.Type)
			replayer.log.Infof("	> Received %v from client. Expected %v.\n\n", resp.StatusCode, expectedResp.StatusCode)
			if err != nil {
				replayer.log.Errorf("Error when forwarding webhook request [%v]: %v", evt.Type, err)
				continue
			}
		}
	}()
}

// returns error if something doesn't match the cassette
func (replayer *Replayer) getNextRecordedCassetteResponse(request *httpRequest) (response *httpResponse, err error) {
	// the passed in request arg may not be necessary
	uncastResponse, err := replayer.write(request)
	if err != nil {
		return &httpResponse{}, err
	}

	wrappedResponse := (*uncastResponse).(httpResponse)

	return &wrappedResponse, err
}

// Reads any contiguous set of webhook recordings from the start of the cassette
func (replayer *Replayer) readAnyPendingWebhookRecordingsFromCassette() (webhookRequests []*httpRequest, webhookResponses []*httpResponse, err error) {
	interactions := make([]interaction, 0)

	// --- Read the pending webhook interactions (stored as raw bytes) from the cassette
	for replayer.interactionsRemaining() > 0 {
		interaction, err := replayer.peekFront()
		if err != nil {
			return nil, nil, fmt.Errorf("error when checking webhooks: %w", err)
		}

		if interaction.Type == incomingInteraction {
			interactions = append(interactions, interaction)
			_, err = replayer.popFront()
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
		req := interaction.Request.(httpRequest)
		res := interaction.Response.(httpResponse)
		webhookRequests = append(webhookRequests, &req)
		webhookResponses = append(webhookResponses, &res)
	}

	return webhookRequests, webhookResponses, nil
}
