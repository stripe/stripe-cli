package main

import (
	"bytes"
	"context"
	"io/ioutil"
	"log"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func assertHttpResponsesAreEqual(t *testing.T, resp1 *http.Response, resp2 *http.Response) error {
	// Read the response bodies
	// resp1 body
	bodyBytes1, err := ioutil.ReadAll(resp1.Body)

	if err != nil {
		return err
	}

	bodyString1 := string(bodyBytes1)
	//reset the response body to the original unread state
	resp1.Body = ioutil.NopCloser(bytes.NewBuffer(bodyBytes1))

	// resp2 body
	bodyBytes2, err := ioutil.ReadAll(resp2.Body)
	if err != nil {
		return err
	}
	bodyString2 := string(bodyBytes2)
	//reset the response body to the original unread state
	resp2.Body = ioutil.NopCloser(bytes.NewBuffer(bodyBytes2))

	assert.Equal(t, bodyString1, bodyString2, "Response bodies differ.")
	assert.Equal(t, resp1.Status, resp2.Status, "Response statuses differ.")

	return nil
}

// Integration test for HTTP wrapper
// TODO: not working
func TestHttpWrapperSimpleIntegration(t *testing.T) {
	// Spin up an instance of the HTTP vcr server in record mode
	filepath := "test_data/simple_integration.yaml"
	addressString := "localhost:8080"

	httpVcr, err := NewHttpVcr(filepath, true)
	check(err)

	server := httpVcr.InitializeServer(addressString)
	go func() {
		if err := server.ListenAndServe(); err != http.ErrServerClosed {
			// unexpected error
			log.Fatalf("ListenAndServe(): %v", err)
		}
	}()

	// fmt.Println("Sleeping for 1s TODO: necessary?")
	// time.Sleep(time.Second)

	// Send it 3 requests
	res1, err := http.Get("http://localhost:8080/a")
	assert.NoError(t, err)

	res2, err := http.Get("http://localhost:8080/b")
	assert.NoError(t, err)

	res3, err := http.Get("http://localhost:8080/c")
	assert.NoError(t, err)

	// Shutdown record server
	_, err = http.Get("http://localhost:8080/vcr/stop")
	server.Shutdown(context.TODO())
	assert.NoError(t, err)

	// --- Set up a replay server
	replayVcr, err := NewHttpVcr(filepath, false)
	check(err)

	replayServer := replayVcr.InitializeServer(addressString)
	go func() {
		if err := replayServer.ListenAndServe(); err != http.ErrServerClosed {
			// unexpected error
			log.Fatalf("ListenAndServe(): %v", err)
		}
	}()

	// TODO: do this better
	// fmt.Println("Sleeping for 1s TODO: necessary?")
	// time.Sleep(time.Second)

	// Send it the same 3 requests:
	// Assert on the replay messages
	replay1, err := http.Get("http://localhost:8080/a")
	assert.NoError(t, err)
	check(assertHttpResponsesAreEqual(t, res1, replay1))

	replay2, err := http.Get("http://localhost:8080/b")
	assert.NoError(t, err)
	check(assertHttpResponsesAreEqual(t, res2, replay2))

	replay3, err := http.Get("http://localhost:8080/c")
	assert.NoError(t, err)
	check(assertHttpResponsesAreEqual(t, res3, replay3))

}
