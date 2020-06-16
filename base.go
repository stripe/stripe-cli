package main

import (
	"fmt"
	"log"
	"net/http"
)

// Example of concrete implentation: VCR as HTTP proxy
var vcr = Vcr{}

func handler(w http.ResponseWriter, r *http.Request) {
	// response := vcr.AcceptRequest(r)

	// take response and write the httpResponse
	fmt.Fprintf(w, "Hello")
}

func main() {
	http.HandleFunc("/", handler)
	log.Fatal(http.ListenAndServe(":8080", nil))
}

// --- Public interface

type Vcr struct {
	recordMode bool // true = record, false = replay

}

type Request interface {
	requestType() string
	header() string
	body() string
}

type Response interface {
}

// Call to expose the VCR to the next event *in the sequence*.
func (vcr *Vcr) AcceptRequest(request Request) Response {

	// figure out the response
	var response *Response
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
		vcr.updateRecording(request)
	}

	// return the response
	return response

}

// --- Private helpers
func (vcr *Vcr) sendResponseToClient(response Response) {

}

func (vcr *Vcr) updateRecording(request Request) {

}

func (vcr *Vcr) getResponseFromRemote(request Request) (resp *Response, err error) {
	return nil, nil
}

// returns error if something doesn't match the cassette
func (vcr *Vcr) getNextRecordedCassetteResponse(request Request) (resp *Response, err error) {
	// the passed in request arg may not be necessary
	return nil, nil
}
