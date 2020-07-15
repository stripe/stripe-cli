package playback

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

func handleErrorInHandler(w http.ResponseWriter, err error, statusCode int) {
	if err == nil {
		return
	}

	// This line will log a relatively benign 'superfluous response.WriteHeader' line if
	// w.WriteHeader was already called on this http.ResponseWriter. Ideally, response.WriteHeader
	// should not be called before this function is called, since the HTTP response code can only be
	// set once.
	// TODO: refactor usage of this function so that we do not get this 'superfluous' log message
	w.WriteHeader(statusCode)

	fmt.Fprintf(w, "%v\n", err)
	fmt.Printf("\n<-- %d error: %v\n", statusCode, err)
}

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

func (httpRecorder *recordServer) insertCassette(writer io.Writer) error {
	recorder, err := newInteractionRecorder(writer)
	if err != nil {
		return err
	}
	httpRecorder.recorder = recorder

	return nil
}

func (httpRecorder *recordServer) webhookHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("\n[WEBHOOK] --> STRIPE %v to %v --> POST to %v", r.Method, r.RequestURI, httpRecorder.webhookURL)

	// --- Pass request to local webhook endpoint
	resp, err := forwardRequest(r, httpRecorder.webhookURL)

	// TODO: this response is going back to Stripe, what is the correct error handling logic here?
	if err != nil {
		handleErrorInHandler(w, fmt.Errorf("Unexpected error forwarding webhook to client: %w", err), 500)
		return
	}

	// --- Write response back to client
	// The header *must* be written first, since writing the body with implicitly and irreversibly set
	// the status code to 200 if not already set.
	// Copy header
	w.WriteHeader(resp.StatusCode)
	copyHTTPHeader(w.Header(), resp.Header)

	// Copy body
	bodyBytes, err := ioutil.ReadAll(resp.Body)
	defer resp.Body.Close()
	if err != nil {
		handleErrorInHandler(w, fmt.Errorf("Unexpected error processing client webhook response: %w", err), 500)
		return
	}
	io.Copy(w, bytes.NewBuffer(bodyBytes))
	resp.Body = ioutil.NopCloser(bytes.NewBuffer(bodyBytes))
	defer resp.Body.Close()

	err = httpRecorder.recorder.write(incomingInteraction, newSerializableHTTPRequest(r), newSerializableHTTPResponse(resp))
	if err != nil {
		handleErrorInHandler(w, fmt.Errorf("Unexpected error writing webhook interaction to cassette: %w", err), 500)
		return
	}
}

func (httpRecorder *recordServer) handler(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("\n--> %v to %v", r.Method, r.RequestURI)

	// --- Pass request to remote
	var resp *http.Response
	var err error

	resp, err = forwardRequest(r, httpRecorder.remoteURL+r.RequestURI)

	if err != nil {
		handleErrorInHandler(w, err, 500)
		return
	}
	fmt.Printf("\n<-- %v from %v\n", resp.Status, strings.ToUpper(httpRecorder.remoteURL))

	// --- Write response back to client
	// The header *must* be written first, since writing the body with implicitly and irreversibly set
	// the status code to 200 if not already set.
	// Copy header
	w.WriteHeader(resp.StatusCode)
	copyHTTPHeader(w.Header(), resp.Header)

	// Copy body
	bodyBytes, err := ioutil.ReadAll(resp.Body)
	defer resp.Body.Close() // we need to close the body

	if err != nil {
		handleErrorInHandler(w, err, 500)
		return
	}
	io.Copy(w, bytes.NewBuffer(bodyBytes))
	// we need to reset the reader to be able to read the body again
	resp.Body = ioutil.NopCloser(bytes.NewBuffer(bodyBytes))
	defer resp.Body.Close()

	err = httpRecorder.recorder.write(outgoingInteraction, newSerializableHTTPRequest(r), newSerializableHTTPResponse(resp))
	if err != nil {
		handleErrorInHandler(w, err, 500)
		return
	}

	// Reset the body reader in case we add code later that performs another read
	resp.Body = ioutil.NopCloser(bytes.NewBuffer(bodyBytes))
	defer resp.Body.Close()
}

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
	defer request.Body.Close()

	request.Body = ioutil.NopCloser(bytes.NewBuffer(bodyBytes))
	defer request.Body.Close()
	req, err := http.NewRequest(request.Method, destinationURL, bytes.NewBuffer(bodyBytes))
	copyHTTPHeader(req.Header, request.Header)

	// Forward the request to the remote
	res, err := client.Do(req)

	if err != nil {
		return nil, err
	}
	return res, nil
}

func (httpRecorder *recordServer) initializeServer(address string) *http.Server {
	customMux := http.NewServeMux()
	server := &http.Server{Addr: address, Handler: customMux}

	// --- Recorder control handlers
	customMux.HandleFunc("/pb/stop", func(w http.ResponseWriter, r *http.Request) {
		fmt.Println()
		fmt.Println("Received /pb/stop. Stopping...")

		httpRecorder.recorder.close()
	})

	// --- Default handler that proxies request and returns response from the remote
	customMux.HandleFunc("/", httpRecorder.handler)

	return server
}

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

	replayer, err := newInteractionReplayer(reader, httpRequestSerializable{}, httpResponseSerializable{}, sequentialComparator)
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
		handleErrorInHandler(w, err, 500)
		return
	}
	fmt.Printf("\n<-- %v from %v\n", resp.Status, "CASSETTE")
	defer resp.Body.Close() // we need to close the body

	// --- Write response back to client
	// The header *must* be written first, since writing the body with implicitly and irreversibly set
	// the status code to 200 if not already set.
	w.WriteHeader(resp.StatusCode)
	copyHTTPHeader(w.Header(), resp.Header)

	// Copy the response exactly as received, and pass on to client
	bodyBytes, err := ioutil.ReadAll(resp.Body)
	defer resp.Body.Close()

	if err != nil {
		handleErrorInHandler(w, err, 500)
		return
	}
	io.Copy(w, bytes.NewBuffer(bodyBytes))
	resp.Body = ioutil.NopCloser(bytes.NewBuffer(bodyBytes))
	defer resp.Body.Close()

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
	responseWrapper, err := httpReplayer.replayer.write(newSerializableHTTPRequest(request))
	if err != nil {
		return &http.Response{}, err
	}

	response := (*responseWrapper).(*http.Response)

	return response, err
}

// Reads any contiguous set of webhook recordings from the start of the cassette
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
		// TODO(bwang): having to create a new instance to get at the fromBytes() method doesn't feel great.
		// But is it unavoidable since Golang doesn't have 'static' methods? Is there some other refactoring we can do?
		requestSerializer := newSerializableHTTPRequest(&http.Request{})
		var reqReader io.Reader = bytes.NewReader(rawWebhookBytes.Request)
		webhookHTTPRequest, err := requestSerializer.fromBytes(&reqReader)
		if err != nil {
			return nil, nil, fmt.Errorf("Error when deserializing cassette to replay webhooks: %w", err)
		}

		webhookRequests = append(webhookRequests, webhookHTTPRequest.(*http.Request))

		responseSerializer := newSerializableHTTPResponse(&http.Response{})
		var respReader io.Reader = bytes.NewReader(rawWebhookBytes.Response)
		webhookHTTPResponse, err := responseSerializer.fromBytes(&respReader)
		if err != nil {
			return nil, nil, fmt.Errorf("Error when deserializing cassette to replay webhooks: %w", err)
		}

		webhookResponses = append(webhookResponses, webhookHTTPResponse.(*http.Response))
	}

	return webhookRequests, webhookResponses, nil
}

func (httpReplayer *replayServer) initializeServer(address string) *http.Server {
	customMux := http.NewServeMux()
	customMux.HandleFunc("/", httpReplayer.handler)

	server := &http.Server{Addr: address, Handler: customMux}

	return server
}

// These constants define the modes the playback server can be in
const (
	// in auto mode, the server records a new cassette if the given file initially doesn't exist
	// and replays a cassette of the file does exist.
	Auto   string = "auto"
	Record string = "record"
	Replay string = "replay"
)

// A RecordReplayServer implements the full functionality of `stripe playback` as a HTTP server.
// Acting as a proxy server for the Stripe API, it can both record and replay interactions using cassette files.
type RecordReplayServer struct {
	httpRecorder recordServer
	httpReplayer replayServer

	remoteURL         string
	cassetteDirectory string // root directory for all cassette filepaths

	// state machine state
	mode                  string // the user specified state (auto, record, replay)
	isRecordingInAutoMode bool   // internal state used when in auto mode to keep track of the state for the current cassette (either recording or replaying)
	cassetteLoaded        bool
}

// NewRecordReplayServer instantiates a RecordReplayServer struct, representing the configuration of a playback proxy server
func NewRecordReplayServer(remoteURL string, webhookURL string, cassetteDirectory string) (server *RecordReplayServer, err error) {
	server = &RecordReplayServer{}
	server.mode = Auto
	server.cassetteLoaded = false
	server.remoteURL = remoteURL
	server.cassetteDirectory = cassetteDirectory

	server.httpRecorder = newRecordServer(remoteURL, webhookURL)
	server.httpReplayer = newReplayServer(webhookURL)

	return server, nil
}

func (rr *RecordReplayServer) handler(w http.ResponseWriter, r *http.Request) {
	if !rr.cassetteLoaded {
		w.WriteHeader(400)
		fmt.Fprint(w, "No cassette is loaded.\n")
		return
	}

	switch rr.mode {
	case Record:
		rr.httpRecorder.handler(w, r)
	case Replay:
		rr.httpReplayer.handler(w, r)
	case Auto:
		if rr.isRecordingInAutoMode {
			rr.httpRecorder.handler(w, r)
		} else {
			rr.httpReplayer.handler(w, r)
		}
	default:
		// Should never get here
		handleErrorInHandler(w, fmt.Errorf("got an unexpected playback mode \"%s\". It must be either \"record\", \"replay\", or \"auto\"", rr.mode), 500)
		return
	}
}

// Webhooks are handled slightly differently from outgoing API requests
func (rr *RecordReplayServer) webhookHandler(w http.ResponseWriter, r *http.Request) {
	if !rr.cassetteLoaded {
		w.WriteHeader(400)
		fmt.Fprint(w, "No cassette is loaded.\n")
		return
	}

	switch rr.mode {
	case Record:
		rr.httpRecorder.webhookHandler(w, r)
	case Replay:
		fmt.Println("Error: webhook endpoint should never be called in replay mode")
	case Auto:
		if rr.isRecordingInAutoMode {
			rr.httpRecorder.webhookHandler(w, r)
		} else {
			fmt.Println("Error: webhook endpoint should never be called in replay mode")
		}
	}
}

func (rr *RecordReplayServer) loadCassetteHandler(w http.ResponseWriter, r *http.Request) {
	filepathVals, ok := r.URL.Query()["filepath"]

	if !ok {
		w.WriteHeader(400)
		fmt.Fprint(w, "\"filepath\" query param must be present.\n")
		return
	}

	if len(filepathVals) > 1 {
		fmt.Printf("Multiple \"filepath\" param values given, ignoring all except first: %v\n", filepathVals[0])
	}

	relativeFilepath := filepathVals[0]

	if !strings.HasSuffix(strings.ToLower(relativeFilepath), ".yaml") {
		w.WriteHeader(400)
		fmt.Fprint(w, "\"filepath\" must specify a .yaml file.\n")
		return
	}

	if filepath.IsAbs(relativeFilepath) {
		w.WriteHeader(400)
		fmt.Fprint(w, "\"filepath\" must be a relative filepath, you provided a absolute filepath.\n")
		return
	}

	absoluteFilepath := filepath.Join(rr.cassetteDirectory, relativeFilepath)

	var shouldCreateNewFile bool

	switch rr.mode {
	case Record:
		fmt.Println("/pb/cassette/load: Recording to: ", absoluteFilepath)
		shouldCreateNewFile = true
	case Replay:
		fmt.Println("/pb/cassette/load: Replaying from: ", absoluteFilepath)
		shouldCreateNewFile = false
	case Auto:
		_, err := os.Stat(absoluteFilepath)
		if os.IsNotExist(err) {
			fmt.Println("/pb/cassette/load: Recording to: ", absoluteFilepath)
			shouldCreateNewFile = true
			rr.isRecordingInAutoMode = true
		} else {
			fmt.Println("/pb/cassette/load: Replaying from: ", absoluteFilepath)
			shouldCreateNewFile = false
			rr.isRecordingInAutoMode = false
		}
	}

	if shouldCreateNewFile {
		fileHandle, err := os.Create(absoluteFilepath)
		if err != nil {
			handleErrorInHandler(w, err, 500)
		}

		err = rr.httpRecorder.insertCassette(fileHandle)
		if err != nil {
			handleErrorInHandler(w, err, 500)
		}
	} else {
		fileHandle, err := os.Open(absoluteFilepath)
		if err != nil {
			handleErrorInHandler(w, err, 500)
		}

		err = rr.httpReplayer.readCassette(fileHandle)
		if err != nil {
			handleErrorInHandler(w, err, 500)
		}
	}

	rr.cassetteLoaded = true
}

// InitializeServer sets up and returns a http.Server that acts as a playback proxy
func (rr *RecordReplayServer) InitializeServer(address string) *http.Server {
	customMux := http.NewServeMux()
	server := &http.Server{Addr: address, Handler: customMux}

	// --- Webhook endpoint
	customMux.HandleFunc("/pb/webhooks", rr.webhookHandler)

	// --- Server control handlers
	customMux.HandleFunc("/pb/mode/", func(w http.ResponseWriter, r *http.Request) {
		// get mode
		modeString := strings.TrimPrefix(r.URL.Path, "/pb/mode/")

		switch strings.ToLower(modeString) {
		case Record:
			fmt.Println("/pb/mode/: mode set to RECORD")
			rr.mode = Record
			w.WriteHeader(200)
		case Replay:
			fmt.Println("/pb/mode/: mode set to REPLAY")
			rr.mode = Replay
			w.WriteHeader(200)
		case Auto:
			fmt.Println("/pb/mode/: mode set to AUTO")
			rr.mode = Auto
			w.WriteHeader(200)
		default:
			w.WriteHeader(400)
			fmt.Fprintf(w, "\"%s\" is not a valid playback mode. It must be either \"record\", \"replay\", or \"auto\".\n", modeString)
		}
	})

	customMux.HandleFunc("/pb/cassette/setroot", func(w http.ResponseWriter, r *http.Request) {
		const queryKey = "dir"
		directoryVals, ok := r.URL.Query()[queryKey]

		if !ok {
			handleErrorInHandler(w, fmt.Errorf("\"%v\" query param must be present", queryKey), 400)
			return
		}

		if len(directoryVals) > 1 {
			fmt.Printf("Multiple \"value\" param values given, ignoring all except first: %v\n", directoryVals[0])
		}

		absoluteCassetteDir := directoryVals[0]

		handle, err := os.Stat(absoluteCassetteDir)
		if err != nil {
			if os.IsNotExist(err) {
				handleErrorInHandler(w,
					fmt.Errorf("the directory \"%v\" does not exist. Please create it, then try again", absoluteCassetteDir), 400)
				return
			}
			handleErrorInHandler(w, fmt.Errorf("Unexpected error when checking cassette directory: %w", err), 500)
			return
		}

		if !handle.Mode().IsDir() {
			handleErrorInHandler(w, fmt.Errorf("the path \"%v\" is not a directory", absoluteCassetteDir), 400)
			return
		}

		rr.cassetteDirectory = absoluteCassetteDir

		fmt.Printf("Cassette directory set to \"%v\"\n", rr.cassetteDirectory)
	})

	customMux.HandleFunc("/pb/cassette/load", rr.loadCassetteHandler)

	customMux.HandleFunc("/pb/cassette/eject", func(w http.ResponseWriter, r *http.Request) {
		if !rr.cassetteLoaded {
			fmt.Println("Tried to eject when no cassette is loaded.")
			w.WriteHeader(400)
			return
		}

		switch rr.mode {
		case Record:
			err := rr.httpRecorder.recorder.close()
			if err != nil {
				handleErrorInHandler(w, fmt.Errorf("Unexpected error when writing cassette. It may have failed to write properly: %w", err), 500)
			}
		case Replay:
			// nothing
		case Auto:
			if rr.isRecordingInAutoMode {
				err := rr.httpRecorder.recorder.close()
				if err != nil {
					handleErrorInHandler(w, fmt.Errorf("Unexpected error when writing cassette. It may have failed to write properly: %w", err), 500)
				}
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
