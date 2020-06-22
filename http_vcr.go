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
	recordMode bool
}

func NewHttpVcr(filepath string, recordMode bool) (vcr HttpVcr, err error) {
	vcr = HttpVcr{}
	recorder, err := NewRecorder(filepath)

	vcr.recorder = recorder

	vcr.recordMode = recordMode
	return vcr, err
}

func (httpVcr *HttpVcr) handler(w http.ResponseWriter, r *http.Request) {

	// --- pass to VCR, get response back
	var resp *http.Response
	var err error
	if httpVcr.recordMode {
		resp, err = getResponseFromRemote(r)
		check(err)
	} else {
		resp, err = getNextRecordedCassetteResponse(r)
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
	httpVcr.recorder.Close() // TODO: figure out how to get recoder.Close to run
	os.Exit(0)
}

func (httpVcr *HttpVcr) StartServer(address string) {
	http.HandleFunc("/", httpVcr.handler)
	defer httpVcr.recorder.Close()
	log.Fatal(http.ListenAndServe(address, nil))
}
func main() {
	filepath := "test1.txt"
	addressString := ":8080"
	httpVcr, err := NewHttpVcr(filepath, true)
	check(err)

	fmt.Println()
	fmt.Printf("Writing to %v and listening on %v", filepath, addressString)
	fmt.Println()

	httpVcr.StartServer(addressString)
}

func getResponseFromRemote(request *http.Request) (resp *http.Response, err error) {
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
func getNextRecordedCassetteResponse(request *http.Request) (resp *http.Response, err error) {
	// the passed in request arg may not be necessary
	return nil, nil
}
