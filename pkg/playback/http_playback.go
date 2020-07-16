package playback

import (
	"bytes"
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
	cassetteDirectory string // root directory for all cassette filepaths

	// state machine state
	mode                  string // the user specified state (auto, record, replay)
	isRecordingInAutoMode bool   // internal state used when in auto mode to keep track of the state for the current cassette (either recording or replaying)
	cassetteLoaded        bool
}

// NewServer instantiates a Server struct, representing the configuration of a playback proxy server
func NewServer(remoteURL string, webhookURL string, cassetteDirectory string) (server *Server, err error) {
	server = &Server{}
	server.mode = Auto
	server.cassetteLoaded = false
	server.remoteURL = remoteURL
	server.cassetteDirectory = cassetteDirectory

	server.httpRecorder = newRecordServer(remoteURL, webhookURL)
	server.httpReplayer = newReplayServer(webhookURL)

	return server, nil
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

func (rr *Server) loadCassetteHandler(w http.ResponseWriter, r *http.Request) {
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
		fmt.Println("/playback/cassette/load: Recording to: ", absoluteFilepath)
		shouldCreateNewFile = true
	case Replay:
		fmt.Println("/playback/cassette/load: Replaying from: ", absoluteFilepath)
		shouldCreateNewFile = false
	case Auto:
		_, err := os.Stat(absoluteFilepath)
		if os.IsNotExist(err) {
			fmt.Println("/playback/cassette/load: Recording to: ", absoluteFilepath)
			shouldCreateNewFile = true
			rr.isRecordingInAutoMode = true
		} else {
			fmt.Println("/playback/cassette/load: Replaying from: ", absoluteFilepath)
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
func (rr *Server) InitializeServer(address string) *http.Server {
	customMux := http.NewServeMux()
	server := &http.Server{Addr: address, Handler: customMux}

	// --- Webhook endpoint
	customMux.HandleFunc("/playback/webhooks", rr.webhookHandler)

	// --- Server control handlers
	customMux.HandleFunc("/playback/mode/", func(w http.ResponseWriter, r *http.Request) {
		// get mode
		modeString := strings.TrimPrefix(r.URL.Path, "/playback/mode/")

		switch strings.ToLower(modeString) {
		case Record:
			fmt.Println("/playback/mode/: mode set to RECORD")
			rr.mode = Record
			w.WriteHeader(200)
		case Replay:
			fmt.Println("/playback/mode/: mode set to REPLAY")
			rr.mode = Replay
			w.WriteHeader(200)
		case Auto:
			fmt.Println("/playback/mode/: mode set to AUTO")
			rr.mode = Auto
			w.WriteHeader(200)
		default:
			w.WriteHeader(400)
			fmt.Fprintf(w, "\"%s\" is not a valid playback mode. It must be either \"record\", \"replay\", or \"auto\".\n", modeString)
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

	customMux.HandleFunc("/playback/cassette/load", rr.loadCassetteHandler)

	customMux.HandleFunc("/playback/cassette/eject", func(w http.ResponseWriter, r *http.Request) {
		if !rr.cassetteLoaded {
			fmt.Println("Tried to eject when no cassette is loaded.")
			w.WriteHeader(400)
			return
		}

		switch rr.mode {
		case Record:
			err := rr.httpRecorder.recorder.saveAndClose()
			if err != nil {
				handleErrorInHandler(w, fmt.Errorf("Unexpected error when writing cassette. It may have failed to write properly: %w", err), 500)
			}
		case Replay:
			// nothing
		case Auto:
			if rr.isRecordingInAutoMode {
				err := rr.httpRecorder.recorder.saveAndClose()
				if err != nil {
					handleErrorInHandler(w, fmt.Errorf("Unexpected error when writing cassette. It may have failed to write properly: %w", err), 500)
				}
			}
		}

		rr.cassetteLoaded = false

		fmt.Println("/playback/cassette/eject: Ejected cassette")
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
