package playback

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"
)

func handleErrorInHandler(w http.ResponseWriter, err error) {
	if err == nil {
		return
	}

	w.WriteHeader(500)
	// TODO: should we crash the program or keep going?
	fmt.Println("\n<-- 500 error: ", err)
}

// HTTP *record* server that proxies requests to a remote host, and records all interactions.
// The core recording logic is handled by playback.Recorder.
type HttpRecorder struct {
	recorder   *Recorder
	remoteURL  string
	webhookURL string
}

func NewHttpRecorder(remoteURL string, webhookURL string) (httpRecorder HttpRecorder) {
	httpRecorder = HttpRecorder{}
	httpRecorder.remoteURL = remoteURL
	httpRecorder.webhookURL = webhookURL

	return httpRecorder
}

func (httpRecorder *HttpRecorder) LoadCassette(writer io.Writer) error {
	recorder, err := NewRecorder(writer)
	if err != nil {
		return err
	}
	httpRecorder.recorder = recorder

	return nil
}

func (httpRecorder *HttpRecorder) webhookHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("\n[WEBHOOK] --> STRIPE %v to %v --> POST to %v", r.Method, r.RequestURI, httpRecorder.webhookURL)

	// --- Pass request to local webhook endpoint
	resp, err := forwardRequest(r, httpRecorder.webhookURL)

	// TODO: this response is going back to Stripe, what is the correct error handling logic here?
	if err != nil {
		handleErrorInHandler(w, fmt.Errorf("Error forwarding webhook to client: %w", err))
		return
	}

	// --- Write response back to client

	// Copy header
	w.WriteHeader(resp.StatusCode)
	copyHTTPHeader(w.Header(), resp.Header)

	// Copy body
	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		handleErrorInHandler(w, fmt.Errorf("Error processing client webhook response: %w", err))
		return
	}
	io.Copy(w, bytes.NewBuffer(bodyBytes))
	resp.Body = ioutil.NopCloser(bytes.NewBuffer(bodyBytes)) // TODO: better understand and document why this is necessary

	// TODO: when reading from the cassette, we need to differentiate between webhook and non-webhook interactions (because they're replayed very differently (both in destination, and in trigger))
	err = httpRecorder.recorder.Write(IncomingInteraction, NewSerializableHttpRequest(r), NewSerializableHttpResponse(resp))
	if err != nil {
		handleErrorInHandler(w, fmt.Errorf("Error writing webhook interaction to cassette: %w", err))
		return
	}
}

func (httpRecorder *HttpRecorder) handler(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("\n--> %v to %v", r.Method, r.RequestURI)

	// --- Pass request to remote
	var resp *http.Response
	var err error

	resp, err = forwardRequest(r, httpRecorder.remoteURL+r.RequestURI)

	if err != nil {
		handleErrorInHandler(w, err)
		return
	}
	fmt.Printf("\n<-- %v from %v\n", resp.Status, strings.ToUpper(httpRecorder.remoteURL))

	defer resp.Body.Close() // we need to close the body

	// --- Write response back to client

	// Copy header
	w.WriteHeader(resp.StatusCode)
	copyHTTPHeader(w.Header(), resp.Header)

	// Copy body
	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		handleErrorInHandler(w, err)
		return
	}
	io.Copy(w, bytes.NewBuffer(bodyBytes))
	resp.Body = ioutil.NopCloser(bytes.NewBuffer(bodyBytes)) // TODO: better understand and document why this is necessary

	err = httpRecorder.recorder.Write(OutgoingInteraction, NewSerializableHttpRequest(r), NewSerializableHttpResponse(resp))
	if err != nil {
		handleErrorInHandler(w, err)
		return
	}
}

// TODO: doesn't need to be a HttpRecorder method (can be a helper)
func forwardRequest(request *http.Request, destinationURL string) (resp *http.Response, err error) {
	client := &http.Client{
		// set Timeout explicitly, otherwise the client will wait indefinitely for a response
		Timeout: time.Second * 10,
	}

	// Create a identical copy of the request
	bodyBytes, err := ioutil.ReadAll(request.Body)
	if err != nil {
		return nil, err
	}
	request.Body = ioutil.NopCloser(bytes.NewBuffer(bodyBytes))
	req, err := http.NewRequest(request.Method, destinationURL, bytes.NewBuffer(bodyBytes))
	copyHTTPHeader(req.Header, request.Header)

	// Forward the request to the remote
	res, err := client.Do(req)

	if err != nil {
		return nil, err
	}
	return res, nil
}

func (httpRecorder *HttpRecorder) InitializeServer(address string) *http.Server {
	customMux := http.NewServeMux()
	server := &http.Server{Addr: address, Handler: customMux}

	// --- Recorder control handlers
	customMux.HandleFunc("/pb/stop", func(w http.ResponseWriter, r *http.Request) {
		fmt.Println()
		fmt.Println("Received /pb/stop. Stopping...")

		httpRecorder.recorder.Close()
	})

	// --- Default handler that proxies request and returns response from the remote
	customMux.HandleFunc("/", httpRecorder.handler)

	return server
}

// HTTP *replay* server that intercepts incoming requests, and replays recorded responses from the provided cassette.
// The core replay logic is handled by playback.Replayer.
type HttpReplayer struct {
	replayer   *Replayer
	webhookURL string
	replayLock *sync.WaitGroup // used to
}

func NewHttpReplayer(webhookURL string) (httpReplayer HttpReplayer) {
	httpReplayer = HttpReplayer{}
	httpReplayer.webhookURL = webhookURL
	httpReplayer.replayLock = &sync.WaitGroup{}
	return httpReplayer
}

func (httpReplayer *HttpReplayer) LoadCassette(reader io.Reader) error {
	// TODO: should we expose matching configuration? how?
	// TODO: how will this change with webhooks?
	sequentialComparator := func(req1 interface{}, req2 interface{}) (accept bool, shortCircuitNow bool) {
		return true, true
	}

	replayer, err := NewReplayer(reader, HttpRequestSerializable{}, HttpResponseSerializable{}, sequentialComparator)
	if err != nil {
		return err
	}

	httpReplayer.replayer = replayer

	return nil
}

func (httpReplayer *HttpReplayer) handler(w http.ResponseWriter, r *http.Request) {
	httpReplayer.replayLock.Wait()       // wait to make sure no webhooks are in the middle of being fired
	httpReplayer.replayLock.Add(1)       // acquire the lock so we can handle this request
	defer httpReplayer.replayLock.Done() // release the lock when handler func is done

	fmt.Printf("\n--> %v to %v", r.Method, r.RequestURI)

	// --- Read matching response from cassette
	var resp *http.Response
	var err error
	resp, err = httpReplayer.getNextRecordedCassetteResponse(r)
	if err != nil {
		handleErrorInHandler(w, err)
		return
	}
	fmt.Printf("\n<-- %v from %v\n", resp.Status, "CASSETTE")
	defer resp.Body.Close() // we need to close the body

	// --- Write response back to client
	// Copy the response exactly as received, and pass on to client
	w.WriteHeader(resp.StatusCode)
	copyHTTPHeader(w.Header(), resp.Header)
	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		handleErrorInHandler(w, err)
		return
	}
	io.Copy(w, bytes.NewBuffer(bodyBytes))
	resp.Body = ioutil.NopCloser(bytes.NewBuffer(bodyBytes))

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
		// we log the error and keep going. TODO: is this the appropriate behavior?
		httpReplayer.replayLock.Wait()       // only send webhooks after the previous request/response is handled
		httpReplayer.replayLock.Add(1)       // acquire lock so we can send webhooks
		defer httpReplayer.replayLock.Done() // release lock when done sending webhooks

		for i, webhookReq := range webhookRequests {
			resp, err := forwardRequest(webhookReq, httpReplayer.webhookURL)
			expectedResp := webhookResponses[i]
			var evt stripeEvent

			webhookPayload, err := ioutil.ReadAll(webhookReq.Body)
			if err != nil {
				fmt.Printf("ERROR when forwarding webhook requests: %v", err)
				continue
			}
			err = json.Unmarshal([]byte(webhookPayload), &evt)
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
func (httpReplayer *HttpReplayer) getNextRecordedCassetteResponse(request *http.Request) (resp *http.Response, err error) {
	// the passed in request arg may not be necessary
	responseWrapper, err := httpReplayer.replayer.Write(NewSerializableHttpRequest(request))
	if err != nil {
		return &http.Response{}, err
	}

	response := (*responseWrapper).(*http.Response)

	return response, err
}

// Reads any contiguous set of webhook recordings from the start of the cassette
func (httpReplayer *HttpReplayer) readAnyPendingWebhookRecordingsFromCassette() (webhookRequests []*http.Request, webhookResponses []*http.Response, err error) {

	webhookBytes := make([]CassettePair, 0)

	// --- Read the pending webhook interactions (stored as raw bytes) from the cassette
	for httpReplayer.replayer.InteractionsRemaining() > 0 {
		interaction, err := httpReplayer.replayer.PeekFront()
		if err != nil {
			return nil, nil, fmt.Errorf("Error when checking webhooks: %w", err)
		}

		if interaction.Type == IncomingInteraction {
			webhookBytes = append(webhookBytes, interaction)
			_, err = httpReplayer.replayer.PopFront()
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
		// TODO(bwang): having to create a new instance to get at the fromBytes() method doesn't feel great.
		// But is it unavoidable since Golang doesn't have 'static' methods? Is there some other refactoring we can do?
		requestSerializer := NewSerializableHttpRequest(&http.Request{})
		webhookHTTPRequest, err := requestSerializer.fromBytes(bytes.NewBuffer(rawWebhookBytes.Request))
		if err != nil {
			return nil, nil, fmt.Errorf("Error when deserializing cassette to replay webhooks: %w", err)
		}

		webhookRequests = append(webhookRequests, webhookHTTPRequest.(*http.Request))

		responseSerializer := NewSerializableHttpResponse(&http.Response{})
		webhookHTTPResponse, err := responseSerializer.fromBytes(bytes.NewBuffer(rawWebhookBytes.Response))
		if err != nil {
			return nil, nil, fmt.Errorf("Error when deserializing cassette to replay webhooks: %w", err)
		}

		webhookResponses = append(webhookResponses, webhookHTTPResponse.(*http.Response))
	}

	return webhookRequests, webhookResponses, nil

}

func (httpReplayer *HttpReplayer) InitializeServer(address string) *http.Server {
	customMux := http.NewServeMux()
	customMux.HandleFunc("/", httpReplayer.handler)

	server := &http.Server{Addr: address, Handler: customMux}

	return server
}

// TODO: currently has issues - do manually for now
func generateSelfSignedCertificates() error {
	gorootPath := os.Getenv("GOROOT")
	fmt.Println("GOROOT: ", gorootPath)
	certGenerationScript := gorootPath + "/src/crypto/tls/generate_cert.go"
	rsaBits := "2048"
	host := "localhost, 127.0.0.1"
	startDate := "Jan 1 00:00:00 1970"
	duration := "--duration=100000h"

	cmd := exec.Command("go", "run", certGenerationScript, "--rsa-bits", rsaBits, "--host", host, "--ca", "--start-date", startDate, duration)
	// cmd := exec.Command("go env")
	// cmd := exec.Command("ls")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("generating certs failed: %w", err)
	} else {
		return nil
	}
}

type RecordReplayServer struct {
	httpRecorder HttpRecorder
	httpReplayer HttpReplayer

	remoteURL string

	// state machine state
	recordMode     bool
	cassetteLoaded bool
}

func NewRecordReplayServer(remoteURL string, webhookURL string) (server *RecordReplayServer, err error) {
	server = &RecordReplayServer{}
	server.recordMode = true
	server.cassetteLoaded = false
	server.remoteURL = remoteURL

	server.httpRecorder = NewHttpRecorder(remoteURL, webhookURL)
	server.httpReplayer = NewHttpReplayer(webhookURL)

	return server, nil
}

func (rr *RecordReplayServer) handler(w http.ResponseWriter, r *http.Request) {
	if !rr.cassetteLoaded {
		w.WriteHeader(400)
		fmt.Fprint(w, "No cassette is loaded.")
		return
	}

	if rr.recordMode {
		rr.httpRecorder.handler(w, r)
	} else {
		rr.httpReplayer.handler(w, r)
	}
}

// Webhooks are handled slightly differently from outgoing API requests
func (rr *RecordReplayServer) webhookHandler(w http.ResponseWriter, r *http.Request) {
	if !rr.cassetteLoaded {
		w.WriteHeader(400)
		fmt.Fprint(w, "No cassette is loaded.")
		return
	}

	if rr.recordMode {
		rr.httpRecorder.webhookHandler(w, r)
	} else {
		fmt.Println("Error: webhook endpoint should never be called in replay mode")
	}
}

func (rr *RecordReplayServer) InitializeServer(address string) *http.Server {
	customMux := http.NewServeMux()
	server := &http.Server{Addr: address, Handler: customMux}

	// --- Webhook endpoint
	customMux.HandleFunc("/pb/webhooks", rr.webhookHandler)

	// --- Replay control handlers
	customMux.HandleFunc("/pb/mode/", func(w http.ResponseWriter, r *http.Request) {

		// get mode
		modeString := strings.TrimPrefix(r.URL.Path, "/pb/mode/")

		fmt.Println("/pb/mode/: Setting mode to ", modeString)

		if strings.EqualFold("record", modeString) {
			rr.recordMode = true
			w.WriteHeader(200)
		} else if strings.EqualFold("replay", modeString) {
			rr.recordMode = false
			w.WriteHeader(200)
		} else {
			w.WriteHeader(400)
			fmt.Fprintf(w, "\"%s\" is not a valid playback mode. It must be either \"record\" or \"replay\".", modeString)
		}
	})

	customMux.HandleFunc("/pb/cassette/load", func(w http.ResponseWriter, r *http.Request) {
		// TODO: does previous cassette have to be ejected explcitly?
		// if we allow implicitly, we should make sure to call eject so that cleanup happens
		// get cassette
		filepath, ok := r.URL.Query()["filepath"]

		if !ok {
			w.WriteHeader(400)
			fmt.Fprint(w, "\"filepath\" query param must be present.")
			return
		}
		fmt.Println("/pb/cassette/load: Loading cassette ", filepath)

		if rr.recordMode {
			fileHandle, err := os.Create(filepath[0])
			if err != nil {
				handleErrorInHandler(w, err)
			}

			err = rr.httpRecorder.LoadCassette(fileHandle)
			if err != nil {
				handleErrorInHandler(w, err)
			}
		} else {
			fileHandle, err := os.Open(filepath[0])
			if err != nil {
				handleErrorInHandler(w, err)
			}

			err = rr.httpReplayer.LoadCassette(fileHandle)
			if err != nil {
				handleErrorInHandler(w, err)
			}
		}

		rr.cassetteLoaded = true

	})

	customMux.HandleFunc("/pb/cassette/eject", func(w http.ResponseWriter, r *http.Request) {
		if rr.recordMode {
			err := rr.httpRecorder.recorder.Close()
			if err != nil {
				handleErrorInHandler(w, err)
			}
		}
		rr.cassetteLoaded = false

		fmt.Println("/pb/cassette/eject: Ejected cassette")
		fmt.Println("")
		fmt.Println("=======")
		fmt.Println("")
	})

	// --- Default handler
	customMux.HandleFunc("/", rr.handler)

	return server
}

func copyHTTPHeader(dest, src http.Header) {
	for k, v := range src {
		for _, subvalues := range v {
			dest.Add(k, subvalues)
		}
	}
}
