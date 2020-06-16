package main

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"os"
)

// Example of concrete implentation: VCR as HTTP proxy
var vcr = Vcr{recordMode: true}

func handler(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Handling request")

	// --- pass to VCR, get response back
	resp := vcr.AcceptRequest(r)
	// -- end VCR interface

	defer resp.Body.Close() // we need to close the body

	scanner := bufio.NewScanner(resp.Body)
	for i := 0; scanner.Scan() && i < 5; i++ {
		fmt.Println(scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		panic(err)
	}

	// take response and write the httpResponse
	w.Header().Set("Content-Type", resp.Header.Get("Content-Type"))
	w.Header().Set("Content-Length", resp.Header.Get("Content-Length"))
	io.Copy(w, resp.Body)

	fmt.Fprintf(w, "hello")
}

// func main() {
// 	http.HandleFunc("/", handler)
// 	log.Fatal(http.ListenAndServe(":8080", nil))
// }

// --- Public interface

type Vcr struct {
	recordMode bool // true = record, false = replay
	fileHandle os.File
}

type Request interface {
	requestType() string
	header() string
	body() string
}

type Response interface {
}

func (vcr *Vcr) StartRecording(filepath string) error {
	// open a cassette file for writing
	fileHandle, err := os.Create(filepath)

	if err != nil {
		return err
	}

	vcr.fileHandle = *fileHandle

	return nil
}

func (vcr *Vcr) StopRecording() {
	vcr.fileHandle.Close()
}

// Call to expose the VCR to the next event *in the sequence*.
// TODO: separate the functions for accept request in record vs replay mode
// VCRRecorder vs VCRReplayer
func (vcr *Vcr) AcceptRequest(request *http.Request) *http.Response {

	// figure out the response
	var response *http.Response
	var err error

	if vcr.recordMode {
		response, err = vcr.getResponseFromRemote(request)
	} else {
		response, err = vcr.getNextRecordedCassetteResponse(request)
	}

	if err != nil {
		// do something
	}

	// update the recording, if in record mode
	if vcr.recordMode {
		vcr.updateRecording(request, response)
	}

	// return the response
	return response

}

// --- Private helpers
func (vcr *Vcr) sendResponseToClient(response http.Response) {

}

// Save a request/response pair to the recording
func (vcr *Vcr) updateRecording(request *http.Request, response *http.Response) {
	// serilaize the request to the cassette file

	// TODO: for now, just recording the received URL path
	_, err := vcr.fileHandle.WriteString("-> " + request.URL.Path)
	_, err = vcr.fileHandle.WriteString("<- " + request.URL.Path)

	if err != nil {
		panic(err)
	}

}

func (vcr *Vcr) getResponseFromRemote(request *http.Request) (resp *http.Response, err error) {
	// TODO: placeholder proxy a request to some random website. Later - this should pass on the request
	res, err := http.Get("http://gobyexample.com")

	if err != nil {
		return nil, err
	}

	fmt.Printf("%s", res.Body)

	// If returning the response to code that expects to read it, we cannot call res.Body.Close() here.
	// defer res.Body.Close()

	return res, nil
}

// returns error if something doesn't match the cassette
func (vcr *Vcr) getNextRecordedCassetteResponse(request *http.Request) (resp *http.Response, err error) {
	// the passed in request arg may not be necessary
	return nil, nil
}
