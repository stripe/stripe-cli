package main

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
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
	fmt.Printf("\n<-- %v from %v\n", resp.Status, "REMOTE")

	defer resp.Body.Close() // we need to close the body

	// --- Write response back to client
	// TODO: this is kind of a piecemeal way to transfer data from the proxied response
	// 		 Is there a way to copy and return the entire proxied response? (and not worry about missing a field)
	w.WriteHeader(resp.StatusCode)
	w.Header().Set("Content-Type", resp.Header.Get("Content-Type"))
	w.Header().Set("Content-Length", resp.Header.Get("Content-Length"))
	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		handleErrorInHandler(w, err)
		return
	}

	io.Copy(w, bytes.NewBuffer(bodyBytes)) // TODO: there is an ordering bug between this and recorder.Write() below

	resp.Body = ioutil.NopCloser(bytes.NewBuffer(bodyBytes))

	err = httpRecorder.recorder.Write(NewSerializableHttpRequest(r), NewSerializableHttpResponse(resp))
	if err != nil {
		handleErrorInHandler(w, err)
		return
	}
}

func (httpRecorder *HttpRecorder) getResponseFromRemote(request *http.Request) (resp *http.Response, err error) {
	// TODO: placeholder proxy a request to some random website. Later - this should pass on the request
	// We need to pass on the entire request (or at least the Authorization part of the header)

	client := &http.Client{}
	req, err := http.NewRequest(request.Method, httpRecorder.remoteURL+request.RequestURI, nil)
	req.Header.Add("Authorization", request.Header.Get("Authorization"))

	res, err := client.Do(req)
	// res, err := http.Get(remoteUrl + request.URL.RequestURI())

	if err != nil {
		return nil, err
	}

	// If returning the response to code that expects to read it, we cannot call res.Body.Close() here.
	// defer res.Body.Close()

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
	// TODO: this is kind of a piecemeal way to transfer data from the proxied response
	// 		 Is there a way to copy and return the entire proxied response? (and not worry about missing a field)
	w.WriteHeader(resp.StatusCode)
	w.Header().Set("Content-Type", resp.Header.Get("Content-Type"))
	w.Header().Set("Content-Length", resp.Header.Get("Content-Length"))
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
	server := &http.Server{Addr: address, Handler: customMux}

	// --- Default VCR catch-all handler
	customMux.HandleFunc("/", httpReplayer.handler)

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
	recordMode := true
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
