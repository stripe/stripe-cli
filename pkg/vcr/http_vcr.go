package vcr

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"log"
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

func NewHttpRecorder(writer io.Writer, remoteURL string) (httpRecorder HttpRecorder, err error) {
	httpRecorder = HttpRecorder{}

	vcrRecorder, err := NewVcrRecorder(writer)
	httpRecorder.recorder = vcrRecorder

	httpRecorder.remoteURL = remoteURL
	return httpRecorder, err
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

func NewHttpReplayer(reader io.Reader) (httpReplayer HttpReplayer, err error) {
	httpReplayer = HttpReplayer{}

	// TODO: should we expose matching configuration? how?
	sequentialComparator := func(req1 interface{}, req2 interface{}) (accept bool, shortCircuitNow bool) {
		return true, true
	}

	vcrReplayer, err := NewVcrReplayer(reader, HttpRequestSerializable{}, HttpResponseSerializable{}, sequentialComparator)
	httpReplayer.replayer = vcrReplayer

	return httpReplayer, err
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

func main() {
	filepath := "main_result.yaml"
	addressString := "localhost:8080"
	recordMode := false
	remoteURL := "https://api.stripe.com"
	// remoteURL := "https://gobyexample.com"

	var httpWrapper VcrHttpServer

	if recordMode {
		// delete file if exists
		if _, err := os.Stat(filepath); !os.IsNotExist(err) {
			err = os.Remove(filepath)
			check(err)
		}

		fileWriteHandle, err := os.Create(filepath)
		check(err)

		httpRecorder, err := NewHttpRecorder(fileWriteHandle, remoteURL)
		httpWrapper = &httpRecorder
		check(err)
	} else {
		// Make sure file exists
		_, err := os.Stat(filepath)
		check(err)

		fileReadHandle, err := os.Open(filepath)
		check(err)

		httpReplayer, err := NewHttpReplayer(fileReadHandle)
		httpWrapper = &httpReplayer
		check(err)
	}

	fmt.Println()
	fmt.Printf("===\nUsing cassette \"%v\".\nListening via HTTPS on %v\nRecordMode: %v\n===", filepath, addressString, recordMode)

	fmt.Println()

	server := httpWrapper.InitializeServer(addressString)

	log.Fatal(server.ListenAndServeTLS("cert.pem", "key.pem"))
}

func copyHTTPHeader(dest, src http.Header) {
	for k, v := range src {
		for _, subvalues := range v {
			dest.Add(k, subvalues)
		}
	}
}
