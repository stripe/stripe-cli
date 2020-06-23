package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
)

type HttpVcr struct {
	recorder   *VcrRecorder
	replayer   *VcrReplayer
	recordMode bool
}

func NewHttpVcr(filepath string, recordMode bool) (vcr HttpVcr, err error) {
	vcr = HttpVcr{}

	if recordMode {
		// delete file if exists
		if _, err := os.Stat(filepath); !os.IsNotExist(err) {
			err = os.Remove(filepath)
			check(err)
		}

		recorder, e := NewRecorder(filepath)
		vcr.recorder = recorder
		err = e
	} else {
		// delete file if exists
		if _, err := os.Stat(filepath); os.IsNotExist(err) {
			return vcr, err
		}

		sequentialComparator := func(req1 interface{}, req2 interface{}) (accept bool, shortCircuitNow bool) {
			return true, true
		}

		replayer, e := NewReplayer(filepath, HttpRequestSerializable{}, HttpResponseSerializable{}, sequentialComparator)
		vcr.replayer = replayer
		err = e
	}

	vcr.recordMode = recordMode
	return vcr, err
}

func (httpVcr *HttpVcr) handler(w http.ResponseWriter, r *http.Request) {

	// --- pass to VCR, get response back
	var resp *http.Response
	var err error
	if httpVcr.recordMode {
		resp, err = httpVcr.getResponseFromRemote(r)
		check(err)
	} else {
		resp, err = httpVcr.getNextRecordedCassetteResponse(r)
		check(err)
	}
	defer resp.Body.Close() // we need to close the body

	if httpVcr.recordMode {
		err = httpVcr.recorder.Write(NewSerializableHttpRequest(r), NewSerializableHttpResponse(resp))
		check(err)
	}

	// scanner := bufio.NewScanner(resp.Body)
	// for i := 0; scanner.Scan() && i < 5; i++ {
	// 	fmt.Println(scanner.Text())
	// }

	// if err := scanner.Err(); err != nil {
	// 	panic(err)
	// }

	// take response and write the httpResponse
	w.Header().Set("Content-Type", resp.Header.Get("Content-Type"))
	w.Header().Set("Content-Length", resp.Header.Get("Content-Length"))
	io.Copy(w, resp.Body)

	fmt.Fprintf(w, "hello")

	fmt.Println("EXITING AAAA")
	// httpVcr.recorder.Close() // TODO: figure out how to get recoder.Close to run
	// os.Exit(0)
}

func (httpVcr *HttpVcr) InitializeServer(address string) *http.Server {
	customMux := http.NewServeMux()
	server := &http.Server{Addr: address, Handler: customMux}

	// --- VCR control handlers
	// TODO: only stops recording, does not shutdown the server
	customMux.HandleFunc("/vcr/stop", func(w http.ResponseWriter, r *http.Request) {
		fmt.Println()
		fmt.Println("Received /vcr/stop. Stopping...")

		httpVcr.recorder.Close()
	})

	// --- Default VCR catch-all handler
	customMux.HandleFunc("/", httpVcr.handler)

	return server
}
func main() {
	filepath := "main_result.yaml"
	addressString := "localhost:8080"
	recordMode := false

	httpVcr, err := NewHttpVcr(filepath, recordMode)
	check(err)

	fmt.Println()
	fmt.Printf("Writing to %v and listening on %v", filepath, addressString)
	fmt.Println()

	server := httpVcr.InitializeServer(addressString)

	log.Fatal(server.ListenAndServe())
}

func (httpVcr *HttpVcr) getResponseFromRemote(request *http.Request) (resp *http.Response, err error) {
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
func (httpVcr *HttpVcr) getNextRecordedCassetteResponse(request *http.Request) (resp *http.Response, err error) {
	// the passed in request arg may not be necessary

	responseWrapper, err := httpVcr.replayer.Write(NewSerializableHttpRequest(request))
	check(err)
	response := (*responseWrapper).(*http.Response)

	return response, err
}
