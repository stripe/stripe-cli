// Implement a full `playback` server that can record and replay HTTP interactions

package playback

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
)

// These constants define the different playback modes
// Auto mode: If a cassette exists, replays from the cassette or else records a new cassette.
const (
	Auto   string = "auto"
	Record string = "record"
	Replay string = "replay"
)

// A Server implements the full functionality of `stripe playback` as a HTTP server.
// Acting as a proxy server for the Stripe API, it can both record and replay interactions using cassette files.
type Server struct {
	recorder Recorder
	replayer Replayer

	remoteURL         string
	cassetteDirectory string // absolute path to the root directory for all cassette filepaths

	log *log.Logger

	switchModeChan chan string

	// state machine state
	mode                  string // the user specified state (auto, record, replay)
	isRecordingInAutoMode bool   // internal state used when in auto mode to keep track of the state for the current cassette (either recording or replaying)
	cassetteLoaded        bool
}

// errorJsonOuter and errorJSONInner are used to return `playback` error messages as JSON in HTTP response bodies to
// mimic errors returned by the Stripe API. The intention is to make it easier for clients (which expect Stripe API errors)
// using `stripe playback` to parse and display a useful error message when `stripe playback` emits errors.
type errorJSONOuter struct {
	Error errorJSONInner `json:"error"`
}

type errorJSONInner struct {
	Code    string `json:"code"`
	DocURL  string `json:"doc_url"`
	Message string `json:"message"`
	Param   string `json:"param"`
	Type    string `json:"type"`
}

// NewServer instantiates a Server struct, representing the configuration and current state of a playback proxy server
// The cassetteDirectory param must be an absolute path
// initialCasssetteFilepath can be a relative path (interpreted relative to cassetteDirectory) or an absolute path
func NewServer(remoteURL string, webhookURL string, absCassetteDirectory string, mode string, initialCassetteFilepath string) (server *Server, err error) {
	server = &Server{}

	// initialize server.Recorder and server.httpReplayer first, since calls to methods like
	// server.loadCassette reference them.
	server.recorder = newRecorder(remoteURL, webhookURL, YAMLSerializer{})

	serializer := YAMLSerializer{}
	// TODO(DX-5701): We may want to allow different types of replay matching in the future (instead of simply sequential playback)
	sequentialComparator := func(req1 interface{}, req2 interface{}) (accept bool, shortCircuitNow bool) {
		return true, true
	}
	server.replayer = newReplayer(webhookURL, serializer, sequentialComparator)
	server.remoteURL = remoteURL
	server.switchModeChan = make(chan string)

	err = server.switchMode(mode)
	if err != nil {
		return server, err
	}

	// cassette directory needs to be set before loading the initial cassette
	err = server.setCassetteDir(absCassetteDirectory)
	if err != nil {
		return server, err
	}

	err = server.loadCassette(initialCassetteFilepath)
	if err != nil {
		return server, err
	}

	server.log = log.New()

	return server, nil
}

// InitializeServer sets up and returns a http.Server that acts as a playback proxy
func (rr *Server) InitializeServer(address string) *http.Server {
	customMux := http.NewServeMux()
	server := &http.Server{Addr: address, Handler: customMux}

	// --- Webhook endpoint
	customMux.HandleFunc("/playback/webhooks", rr.webhookHandler)

	// --- Server control handlers
	customMux.HandleFunc("/playback/mode/", func(w http.ResponseWriter, r *http.Request) {
		// get mode
		modeString := strings.TrimPrefix(r.URL.Path, "/playback/mode/")

		wasCassetteLoaded := rr.cassetteLoaded
		err := rr.switchMode(modeString)
		if wasCassetteLoaded {
			rr.log.Info("/playback/mode: unloaded the cassette. Please load a new cassette before recording/replaying any new interactions.")
		}
		if err != nil {
			rr.log.Error("Error in /playback/mode handler: ", err)
			writeErrorToHTTPResponse(w, rr.log, err, 400)
			return
		}

		// There might be no process listening to this channel,
		// so we need to make sure sending to it is non-blocking.
		select {
		case rr.switchModeChan <- strings.ToLower(modeString):
		default:
		}
		rr.log.Info("/playback/mode: Set mode to ", strings.ToUpper(modeString))
	})

	customMux.HandleFunc("/playback/cassette/setroot", func(w http.ResponseWriter, r *http.Request) {
		const queryKey = "dir"
		directoryVals, ok := r.URL.Query()[queryKey]

		if !ok {
			writeErrorToHTTPResponse(w, rr.log, fmt.Errorf("\"%v\" query param must be present", queryKey), 400)
			return
		}

		if len(directoryVals) > 1 {
			rr.log.Warnf("Multiple \"value\" query param values provided, ignoring all except first: %v\n", directoryVals[0])
		}

		// directoryVal can be either a relative or absolute path - filepath.Abs() no-ops if the given path is already absolute, and converts to absolute (assuming CWD) if relative
		absoluteCassetteDir, err := filepath.Abs(directoryVals[0])

		if err != nil {
			rr.log.Error("Error with given directory in /playback/cassette/setroot handler: ", err)
			writeErrorToHTTPResponse(w, rr.log, err, 400)
			return
		}

		err = rr.setCassetteDir(absoluteCassetteDir)

		if err != nil {
			rr.log.Error("Error in /playback/cassette/setroot handler: ", err)
			writeErrorToHTTPResponse(w, rr.log, err, 400)
			return
		}
		rr.log.Infof("Cassette directory set to \"%v\"", rr.cassetteDirectory)
		w.WriteHeader(200)
	})

	customMux.HandleFunc("/playback/cassette/load", func(w http.ResponseWriter, r *http.Request) {
		filepathVals, ok := r.URL.Query()["filepath"]

		if !ok {
			err := fmt.Errorf("\"filepath\" query param must be present")
			writeErrorToHTTPResponse(w, rr.log, err, 400)
			return
		}

		if len(filepathVals) > 1 {
			rr.log.Warnf("Multiple \"filepath\" param values given, ignoring all except first: %v", filepathVals[0])
		}

		relativeFilepath := filepathVals[0]

		if !strings.HasSuffix(strings.ToLower(relativeFilepath), ".yaml") {
			err := fmt.Errorf("%v is not a .yaml file", relativeFilepath)
			writeErrorToHTTPResponse(w, rr.log, err, 400)
			return
		}

		if filepath.IsAbs(relativeFilepath) {
			err := fmt.Errorf("%v must be a relative filepath. a absolute filepath was provided", relativeFilepath)
			writeErrorToHTTPResponse(w, rr.log, err, 400)
			return
		}

		err := rr.loadCassette(relativeFilepath)

		if err != nil {
			writeErrorToHTTPResponse(w, rr.log, err, 500)
			return
		}

		var statusMsg string
		isRecording := rr.isRecording()
		if isRecording {
			statusMsg = fmt.Sprintf("Recording to %v", relativeFilepath)
		} else {
			statusMsg = fmt.Sprintf("Replaying from %v", relativeFilepath)
		}
		rr.log.Info(statusMsg)
		w.WriteHeader(200)
	})

	customMux.HandleFunc("/playback/cassette/eject", func(w http.ResponseWriter, r *http.Request) {
		err := rr.ejectCassette()

		if err != nil {
			writeErrorToHTTPResponse(w, rr.log, err, 500)
		} else {
			w.WriteHeader(200)
			rr.log.Info("/playback/cassette/eject: Ejected cassette")
		}
	})

	// Display error message for unmatched routes with a control-prefix
	customMux.HandleFunc("/playback/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
		fmt.Fprintf(w, "\"%v\" is not a valid /playback/ control endpoint. Run `stripe playback --help` for a comprehensive list.\n", r.URL)
	})

	// --- Default handler
	customMux.HandleFunc("/", rr.handler)

	return server
}

// Handles incoming Stripe API requests sent to the `playback` server.
// Requests are handled differently depending on whether we are recording or replaying.
func (rr *Server) handler(w http.ResponseWriter, r *http.Request) {
	// TODO: Should we be automatically loading a cassette when none is loaded?
	if !rr.cassetteLoaded {
		err := errors.New("no cassette is loaded")
		writeErrorToHTTPResponse(w, rr.log, err, 400)
		return
	}

	isRecording := rr.isRecording()
	if isRecording {
		rr.recorder.handler(w, r)
	} else {
		rr.replayer.handler(w, r)
	}
}

// Handles incoming webhook requests sent from Stripe. Should only be receiving requests when in record mode.
// Webhook requests are forwarded to the local application and recorded.
func (rr *Server) webhookHandler(w http.ResponseWriter, r *http.Request) {
	if !rr.cassetteLoaded {
		err := errors.New("no cassette is loaded")
		writeErrorToHTTPResponse(w, rr.log, err, 400)
		return
	}

	isRecording := rr.isRecording()
	if isRecording {
		rr.recorder.webhookHandler(w, r)
	} else {
		rr.log.Error("Error: webhook endpoint should never be called in replay mode")
	}
}

// forwardRequest forwards a request to destinationURL and returns the response.
func forwardRequest(wrappedRequest *httpRequest, destinationURL string) (resp *http.Response, err error) {
	client := &http.Client{
		// set Timeout explicitly, otherwise the client will wait indefinitely for a response
		Timeout: time.Second * 10,
	}

	// Create a identical copy of the request
	req, err := http.NewRequest(wrappedRequest.Method, destinationURL, bytes.NewBuffer(wrappedRequest.Body))
	if err != nil {
		return nil, err
	}
	copyHTTPHeader(req.Header, wrappedRequest.Headers)

	// Forward the request to the remote
	res, err := client.Do(req)

	if err != nil {
		return nil, err
	}
	return res, nil
}

// Writes to the HTTP response using the given HTTP status code and error in the response body. Simultaneously, prints logs error text.
// Note: since http.ResponseWriter streams the response, you will not be able to change the HTTP status code after calling this function.
func writeErrorToHTTPResponse(w http.ResponseWriter, log *log.Logger, errParam error, statusCode int) { // nolint:interfacer
	errorJSON := errorJSONOuter{}
	errorJSON.Error.Message = errParam.Error()
	errorJSON.Error.Type = "stripe_playback_error"
	jsonBytes, err := json.MarshalIndent(errorJSON, "", "    ")

	// Write error as json, falling back to plain text if it fails
	if err != nil {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(statusCode)
		fmt.Fprintf(w, "%v\n", errParam)
	} else {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(statusCode)
		w.Write(jsonBytes)
	}

	log.Errorf("<-- %d: %v", statusCode, errParam)
}

func copyHTTPHeader(dest, src http.Header) {
	for k, v := range src {
		for _, subvalues := range v {
			dest.Set(k, subvalues)
		}
	}
}

// OnSwitchMode listens to the switchModeChan and calls f when it receives a mode string
func (rr *Server) OnSwitchMode(f func(string)) {
	go func() {
		mode := <-rr.switchModeChan
		f(mode)
	}()
}

// note: calling this method always sets cassetteLoaded = false
func (rr *Server) switchMode(modeString string) error {
	rr.cassetteLoaded = false
	switch strings.ToLower(modeString) {
	case Record:
		rr.mode = Record
		return nil
	case Replay:
		rr.mode = Replay
		return nil
	case Auto:
		rr.mode = Auto
		return nil
	default:
		return fmt.Errorf("\"%s\" is not a valid playback mode. It must be either \"record\", \"replay\", or \"auto\"", modeString)
	}
}

func (rr *Server) setCassetteDir(absoluteCassetteDir string) error {
	handle, err := os.Stat(absoluteCassetteDir)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("the directory \"%v\" does not exist. Please create it, then try again", absoluteCassetteDir)
		}
		return fmt.Errorf("inexpected error when checking cassette directory: %w", err)
	}

	if !handle.Mode().IsDir() {
		return fmt.Errorf("the path \"%v\" is not a directory", absoluteCassetteDir)
	}

	rr.cassetteDirectory = absoluteCassetteDir
	return nil
}

func (rr *Server) openCassetteFileForReplaying(absoluteFilepath string) error {
	fileHandle, err := os.Open(absoluteFilepath)
	if err != nil {
		return fmt.Errorf("error opening cassette file: %w", err)
	}

	err = rr.replayer.readCassette(fileHandle)
	if err != nil {
		return fmt.Errorf("error parsing cassette file: %v", err)
	}

	return nil
}

func (rr *Server) createCassetteFileForRecording(absoluteFilepath string) error {
	directoryPath := filepath.Dir(absoluteFilepath)
	err := os.MkdirAll(directoryPath, 0755)
	if err != nil {
		return fmt.Errorf("error recursively creating nested directories for cassette file: %w", err)
	}

	fileHandle, err := os.Create(absoluteFilepath)
	if err != nil {
		return fmt.Errorf("error creating cassette file: %w", err)
	}

	rr.recorder.insertCassette(fileHandle)

	return nil
}

// loadCassette() takes a relative path (relative to the rr.cassetteDirectory) to a .yaml file
// The .yaml file need not exist (unless rr.mode = Replay). If intermediate directories in the relative path do not exist, loadCassette *will* create them.
func (rr *Server) loadCassette(relativeFilepath string) error {
	absoluteFilepath := filepath.Join(rr.cassetteDirectory, relativeFilepath)

	var fileErr error

	switch rr.mode {
	case Record:
		fileErr = rr.createCassetteFileForRecording(absoluteFilepath)
	case Replay:
		fileErr = rr.openCassetteFileForReplaying(absoluteFilepath)
	case Auto:
		_, err := os.Stat(absoluteFilepath)
		if os.IsNotExist(err) {
			rr.isRecordingInAutoMode = true
			fileErr = rr.createCassetteFileForRecording(absoluteFilepath)
		} else {
			rr.isRecordingInAutoMode = false
			fileErr = rr.openCassetteFileForReplaying(absoluteFilepath)
		}
	}

	if fileErr != nil {
		return fileErr
	}

	rr.cassetteLoaded = true
	return nil
}

// ejectCassette calls recorder.saveAndClose which persists the cassette to file and closes it.
func (rr *Server) ejectCassette() error {
	if !rr.cassetteLoaded {
		return fmt.Errorf("tried to eject when no cassette is loaded")
	}

	isRecording := rr.isRecording()
	if isRecording {
		err := rr.recorder.saveAndClose()
		if err != nil {
			rr.cassetteLoaded = false // if an error occurs, best to reset to an blank state so a new cassette can be loaded
			return fmt.Errorf("unexpected error when writing cassette. It may have failed to write properly: %w", err)
		}
	}
	// ejecting is a no-op internally when in replay mode

	rr.cassetteLoaded = false
	return nil
}

func (rr *Server) isRecording() bool {
	switch rr.mode {
	case Record:
		return true
	case Replay:
		return false
	case Auto:
		return rr.isRecordingInAutoMode
	default:
		// We should never get here, since all mutations of rr.mode should have already validated the mode value.
		rr.log.Fatalf("Unexpected mode \"%v\" in playback server - this likely indicates a bug in the implementation. Please try restarting the server.", rr.mode)
		return false
	}
}
