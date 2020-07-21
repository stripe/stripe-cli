package playback

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

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

func writeErrorToHTTPResponse(w http.ResponseWriter, err error, statusCode int) {
	w.WriteHeader(statusCode)
	fmt.Fprintf(w, "%v\n", err)
	fmt.Printf("\n<-- %d error: %v\n", statusCode, err)
}

// TODO: deprecate this in subsequent commit
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

// These constants define the modes the playback server can be in
const (
	// in auto mode, the server records a new cassette if the given file initially doesn't exist
	// and replays a cassette of the file does exist.
	Auto   string = "auto"
	Record string = "record"
	Replay string = "replay"
)

// A Server implements the full functionality of `stripe playback` as a HTTP server.
// Acting as a proxy server for the Stripe API, it can both record and replay interactions using cassette files.
type Server struct {
	httpRecorder recordServer
	httpReplayer replayServer

	remoteURL         string
	cassetteDirectory string // absolute path to the root directory for all cassette filepaths

	// state machine state
	mode                  string // the user specified state (auto, record, replay)
	isRecordingInAutoMode bool   // internal state used when in auto mode to keep track of the state for the current cassette (either recording or replaying)
	cassetteLoaded        bool
}

// NewServer instantiates a Server struct, representing the configuration and current state of a playback proxy server
// The cassetteDirectory param must be an absolute path
// initialCasssetteFilepath can be a relative path (inretpreted relative to cassetteDirectory) or an absolute path
func NewServer(remoteURL string, webhookURL string, absCassetteDirectory string, mode string, initialCassetteFilepath string) (server *Server, err error) {
	server = &Server{}

	err = server.switchMode(mode)
	if err != nil {
		return server, err
	}

	err = server.loadCassette(initialCassetteFilepath)
	if err != nil {
		return server, err
	}

	err = server.setCassetteDir(absCassetteDirectory)
	if err != nil {
		return server, err
	}

	server.remoteURL = remoteURL
	server.httpRecorder = newRecordServer(remoteURL, webhookURL)
	server.httpReplayer = newReplayServer(webhookURL)

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
			fmt.Println("/playback/mode: unloaded the cassette. Please load a new cassette before recording/replaying any new interactions.")
		}
		if err != nil {
			fmt.Println("Error in /playback/mode handler: ", err)
			handleErrorInHandler(w, err, 400)
		} else {
			fmt.Println("/playback/mode: Set mode to ", strings.ToUpper(modeString))
		}
	})

	customMux.HandleFunc("/playback/cassette/setroot", func(w http.ResponseWriter, r *http.Request) {
		const queryKey = "dir"
		directoryVals, ok := r.URL.Query()[queryKey]

		if !ok {
			handleErrorInHandler(w, fmt.Errorf("\"%v\" query param must be present", queryKey), 400)
			return
		}

		if len(directoryVals) > 1 {
			fmt.Printf("Multiple \"value\" param values given, ignoring all except first: %v\n", directoryVals[0])
		}

		// directoryVal can be either a relative or absolute path - filepath.Abs() no-ops if the given path is already absolute, and converts to absolute (assuming CWD) if relative
		absoluteCassetteDir, err := filepath.Abs(directoryVals[0])

		if err != nil {
			fmt.Println("Error with given directory in /playback/cassette/setroot handler: ", err)
			handleErrorInHandler(w, err, 400)
			return
		}

		err = rr.setCassetteDir(absoluteCassetteDir)

		if err != nil {
			fmt.Println("Error in /playback/cassette/setroot handler: ", err)
			handleErrorInHandler(w, err, 400)
		} else {
			// TODO: why is this not printing out the absolute path?
			fmt.Printf("Cassette directory set to \"%v\"\n", rr.cassetteDirectory)
			w.WriteHeader(200)
		}
	})

	customMux.HandleFunc("/playback/cassette/load", func(w http.ResponseWriter, r *http.Request) {
		filepathVals, ok := r.URL.Query()["filepath"]

		if !ok {
			err := fmt.Errorf("\"filepath\" query param must be present")
			writeErrorToHTTPResponse(w, err, 400)
			return
		}

		if len(filepathVals) > 1 {
			fmt.Printf("Multiple \"filepath\" param values given, ignoring all except first: %v\n", filepathVals[0])
		}

		relativeFilepath := filepathVals[0]

		if !strings.HasSuffix(strings.ToLower(relativeFilepath), ".yaml") {
			err := fmt.Errorf("%v is not a .yaml file", relativeFilepath)
			writeErrorToHTTPResponse(w, err, 400)
			return
		}

		if filepath.IsAbs(relativeFilepath) {
			err := fmt.Errorf("%v must be a relative filepath. a absolute filepath was provided", relativeFilepath)
			writeErrorToHTTPResponse(w, err, 400)
			return
		}

		err := rr.loadCassette(relativeFilepath)

		if err != nil {
			writeErrorToHTTPResponse(w, err, 500)
			return
		}

		var isRecording bool
		switch rr.mode {
		case Record:
			isRecording = true
		case Replay:
			isRecording = false
		case Auto:
			isRecording = rr.isRecordingInAutoMode
		default:
			writeErrorToHTTPResponse(w, errors.New("in unexpected mode state in handler. Please restart the playback server"), 500)
			return
		}

		var statusMsg string

		if isRecording {
			statusMsg = fmt.Sprintf("Recording to %v", relativeFilepath)
		} else {
			statusMsg = fmt.Sprintf("Replaying from %v", relativeFilepath)
		}
		fmt.Println(statusMsg)
		w.WriteHeader(200)
	})

	customMux.HandleFunc("/playback/cassette/eject", func(w http.ResponseWriter, r *http.Request) {
		err := rr.ejectCassette()

		if err != nil {
			writeErrorToHTTPResponse(w, err, 500)
		} else {
			w.WriteHeader(200)
			fmt.Println("/playback/cassette/eject: Ejected cassette")
		}
		fmt.Println("")
		fmt.Println("=======")
		fmt.Println("")
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

func (rr *Server) handler(w http.ResponseWriter, r *http.Request) {
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
func (rr *Server) webhookHandler(w http.ResponseWriter, r *http.Request) {
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

// loadCassette() takes a relative path (relative to the rr.cassetteDirectory) to a .yaml file
// The .yaml file need not exist (unless rr.mode = Replay). If intermediate directories in the relative path do not exist, loadCassette *will* create them.
func (rr *Server) loadCassette(relativeFilepath string) error {
	absoluteFilepath := filepath.Join(rr.cassetteDirectory, relativeFilepath)

	var shouldCreateNewFile bool

	switch rr.mode {
	case Record:
		shouldCreateNewFile = true
	case Replay:
		shouldCreateNewFile = false
	case Auto:
		_, err := os.Stat(absoluteFilepath)
		if os.IsNotExist(err) {
			shouldCreateNewFile = true
			rr.isRecordingInAutoMode = true
		} else {
			shouldCreateNewFile = false
			rr.isRecordingInAutoMode = false
		}
	}

	if shouldCreateNewFile {
		directoryPath := filepath.Dir(absoluteFilepath)
		err := os.MkdirAll(directoryPath, 0644)
		if err != nil {
			return fmt.Errorf("Error recursively creating nested directories for cassette file: %w", err)
		}

		fileHandle, err := os.Create(absoluteFilepath)
		if err != nil {
			return fmt.Errorf("Error creating cassette file: %w", err)
		}

		err = rr.httpRecorder.insertCassette(fileHandle)
		if err != nil {
			return fmt.Errorf("Error inserting cassette file: %w", err)
		}
	} else {
		fileHandle, err := os.Open(absoluteFilepath)
		if err != nil {
			return fmt.Errorf("Error opening cassette file: %w", err)
		}

		err = rr.httpReplayer.readCassette(fileHandle)
		if err != nil {
			return fmt.Errorf("Error parsing cassette file: %v", err)
		}
	}

	rr.cassetteLoaded = true
	return nil
}

func (rr *Server) ejectCassette() error {
	if !rr.cassetteLoaded {
		return fmt.Errorf("tried to eject when no cassette is loaded")
	}

	switch rr.mode {
	case Record:
		err := rr.httpRecorder.recorder.saveAndClose()
		if err != nil {
			rr.cassetteLoaded = false // if an error occurs, best to reset to an blank state so a new cassette can be loaded
			return fmt.Errorf("unexpected error when writing cassette. It may have failed to write properly: %w", err)
		}
	case Replay:
		// nothing
	case Auto:
		if rr.isRecordingInAutoMode {
			err := rr.httpRecorder.recorder.saveAndClose()
			if err != nil {
				rr.cassetteLoaded = false // if an error occurs, best to reset to an blank state so a new cassette can be loaded
				return fmt.Errorf("unexpected error when writing cassette. It may have failed to write properly: %w", err)
			}
		}
	}
	rr.cassetteLoaded = false
	return nil
}

func copyHTTPHeader(dest, src http.Header) {
	for k, v := range src {
		for _, subvalues := range v {
			dest.Add(k, subvalues)
		}
	}
}
