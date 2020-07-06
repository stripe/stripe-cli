package vcr

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"strings"
)

func handleErrorInHandler(w http.ResponseWriter, err error) {
	if err == nil {
		return
	}

	w.WriteHeader(500)
	// TODO: should we crash the program or keep going?
	fmt.Println("\n<-- 500 error: ", err)
}

// Interface for a http server that implements VCR functionality (record or replay)
type VcrHttpServer interface {
	InitializeServer(address string) *http.Server
}

// HTTP VCR *record* server that proxies requests to a remote host, and records all interactions.
// The core VCR logic is handled by VcrRecorder.
type HttpRecorder struct {
	recorder  *VcrRecorder
	remoteURL string
}

func NewHttpRecorder(remoteURL string) (httpRecorder HttpRecorder) {
	httpRecorder = HttpRecorder{}
	httpRecorder.remoteURL = remoteURL

	return httpRecorder
}

func (httpRecorder *HttpRecorder) LoadCassette(writer io.Writer) error {
	vcrRecorder, err := NewVcrRecorder(writer)
	if err != nil {
		return err
	}
	httpRecorder.recorder = vcrRecorder

	return nil
}

func (httpRecorder *HttpRecorder) handler(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("\n--> %v to %v", r.Method, r.RequestURI)

	// --- Pass request to remote
	var resp *http.Response
	var err error

	resp, err = httpRecorder.getResponseFromRemote(r)

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

	err = httpRecorder.recorder.Write(NewSerializableHttpRequest(r), NewSerializableHttpResponse(resp))
	if err != nil {
		handleErrorInHandler(w, err)
		return
	}
}

func (httpRecorder *HttpRecorder) getResponseFromRemote(request *http.Request) (resp *http.Response, err error) {
	client := &http.Client{}

	// Create a identical copy of the request
	bodyBytes, err := ioutil.ReadAll(request.Body)
	if err != nil {
		return nil, err
	}
	request.Body = ioutil.NopCloser(bytes.NewBuffer(bodyBytes))
	req, err := http.NewRequest(request.Method, httpRecorder.remoteURL+request.RequestURI, bytes.NewBuffer(bodyBytes))
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

	// --- VCR control handlers
	// TODO: only stops recording, does not shutdown the server
	customMux.HandleFunc("/vcr/stop", func(w http.ResponseWriter, r *http.Request) {
		fmt.Println()
		fmt.Println("Received /vcr/stop. Stopping...")

		httpRecorder.recorder.Close()
	})

	// --- Default VCR catch-all handler
	customMux.HandleFunc("/", httpRecorder.handler)

	return server
}

// HTTP VCR *replay* server that intercepts incoming requests, and replays recorded responses from the provided cassette.
// The core VCR logic is handled by VcrReplayer.
type HttpReplayer struct {
	replayer *VcrReplayer
}

func NewHttpReplayer() (httpReplayer HttpReplayer) {
	httpReplayer = HttpReplayer{}

	return httpReplayer
}

func (httpReplayer *HttpReplayer) LoadCassette(reader io.Reader) error {
	// TODO: should we expose matching configuration? how?
	sequentialComparator := func(req1 interface{}, req2 interface{}) (accept bool, shortCircuitNow bool) {
		return true, true
	}

	vcrReplayer, err := NewVcrReplayer(reader, HttpRequestSerializable{}, HttpResponseSerializable{}, sequentialComparator)
	if err != nil {
		return err
	}

	httpReplayer.replayer = vcrReplayer

	return nil
}

func (httpReplayer *HttpReplayer) handler(w http.ResponseWriter, r *http.Request) {
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
	io.Copy(w, bytes.NewBuffer(bodyBytes)) // TODO: there is an ordering bug between this and recorder.Write() below
	resp.Body = ioutil.NopCloser(bytes.NewBuffer(bodyBytes))
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

func NewRecordReplayServer(remoteURL string) (server *RecordReplayServer, err error) {
	server = &RecordReplayServer{}
	server.recordMode = true
	server.cassetteLoaded = false
	server.remoteURL = remoteURL

	server.httpRecorder = NewHttpRecorder(remoteURL)
	server.httpReplayer = NewHttpReplayer()

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

func (rr *RecordReplayServer) InitializeServer(address string) *http.Server {
	customMux := http.NewServeMux()
	server := &http.Server{Addr: address, Handler: customMux}

	// --- VCR control handlers
	// TODO: only stops recording, does not shutdown the server
	// TODO: move this logic into eject
	// customMux.HandleFunc("/vcr/stop", func(w http.ResponseWriter, r *http.Request) {
	// 	fmt.Println()
	// 	fmt.Println("Received /vcr/stop. Stopping...")

	// 	// update state
	// 	rr.cassetteLoaded = false
	// 	if rr.recordMode {
	// 		// TODO: should we expose a Close() fn on the HttpRecorder?
	// 		rr.httpRecorder.recorder.Close()
	// 	}
	// })

	// TODO: only stops recording, does not shutdown the server
	customMux.HandleFunc("/vcr/mode/", func(w http.ResponseWriter, r *http.Request) {

		// get mode
		modeString := strings.TrimPrefix(r.URL.Path, "/vcr/mode/")

		fmt.Println("/vcr/mode/: Setting mode to ", modeString)

		if strings.EqualFold("record", modeString) {
			rr.recordMode = true
			w.WriteHeader(200)
		} else if strings.EqualFold("replay", modeString) {
			rr.recordMode = false
			w.WriteHeader(200)
		} else {
			w.WriteHeader(400)
			fmt.Fprintf(w, "\"%s\" is not a valid VCR mode. Must be \"record\" or \"replay\".", modeString)
		}
	})

	customMux.HandleFunc("/vcr/cassette/load", func(w http.ResponseWriter, r *http.Request) {
		// TODO: does previous cassette have to be ejected explcitly?
		// if we allow implicitly, we should make sure to call eject so that cleanup happens
		// get cassette
		filepath, ok := r.URL.Query()["filepath"]

		if !ok {
			w.WriteHeader(400)
			fmt.Fprint(w, "\"filepath\" query param must be present.")
			return
		}
		fmt.Println("/vcr/cassette/load: Loading cassette ", filepath)

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

	customMux.HandleFunc("/vcr/cassette/eject", func(w http.ResponseWriter, r *http.Request) {
		if rr.recordMode {
			err := rr.httpRecorder.recorder.Close()
			if err != nil {
				handleErrorInHandler(w, err)
			}
		}
		rr.cassetteLoaded = false

		fmt.Println("/vcr/cassette/eject: Ejected cassette")
		fmt.Println("")
		fmt.Println("=======")
		fmt.Println("")
	})

	// --- Default VCR catch-all handler
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
