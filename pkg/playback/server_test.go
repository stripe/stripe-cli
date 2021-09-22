package playback

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

const defaultLocalAddress = "localhost:8080"
const defaultLocalWebhookAddress = "localhost:8888"

func assertHTTPResponsesAreEqual(t *testing.T, resp1 *http.Response, resp2 *http.Response) error {
	// Read the response bodies
	// resp1 body
	bodyBytes1, err := ioutil.ReadAll(resp1.Body)

	if err != nil {
		return err
	}

	bodyString1 := string(bodyBytes1)
	// reset the response body to the original unread state
	err = resp1.Body.Close()
	if err != nil {
		return err
	}
	resp1.Body = ioutil.NopCloser(bytes.NewBuffer(bodyBytes1))

	// resp2 body
	bodyBytes2, err := ioutil.ReadAll(resp2.Body)
	if err != nil {
		return err
	}
	bodyString2 := string(bodyBytes2)
	// reset the response body to the original unread state
	err = resp2.Body.Close()
	if err != nil {
		return err
	}
	resp2.Body = ioutil.NopCloser(bytes.NewBuffer(bodyBytes2))

	assert.Equal(t, bodyString1, bodyString2, "Response bodies differ.")
	assert.Equal(t, resp1.Status, resp2.Status, "Response statuses differ.")

	return nil
}

func startMockServer(responses []httpResponse) *httptest.Server {
	responseCount := 0

	// Set up a local mock of a remote server will serve the provided http.Responses from the fixture files
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if responseCount >= len(responses) {
			w.WriteHeader(500)
			fmt.Fprintln(w, "Mock server ran out of http.Response fixtures to serve. Is the test case written properly?")
			return
		}

		wrappedResponse := responses[responseCount]
		responseCount++

		// --- Write response back to client
		// Copy header
		copyHTTPHeader(w.Header(), wrappedResponse.Headers)

		// WriteHeader must be called after setting header map
		w.WriteHeader(wrappedResponse.StatusCode)

		// Copy body
		io.Copy(w, bytes.NewBuffer(wrappedResponse.Body))
	}))
}

// A simple test of the record and replay servers separately (without going through a full playback server)
func TestSimpleRecordReplayServerSeparately(t *testing.T) {
	var remoteURL string

	// Initialize the mock responses we want to serve
	mockResponse1 := httpResponse{}
	mockResponse1.StatusCode = 200
	mockResponse1.Headers = make(http.Header)
	mockResponse1.Headers.Add("testHeader", "testHeaderValue")
	mockResponse1.Body = []byte("body 1")

	mockResponse2 := httpResponse{}
	mockResponse2.StatusCode = 402
	mockResponse2.Headers = make(http.Header)
	mockResponse2.Body = []byte("body 2")
	fixtureResponses := []httpResponse{mockResponse1, mockResponse2}

	ts := startMockServer(fixtureResponses)
	defer ts.Close()
	remoteURL = ts.URL

	// Spin up an instance of the HTTP playback server in record mode
	addressString := defaultLocalAddress
	webhookURL := defaultLocalWebhookAddress // not used in this test

	httpWrapper, err := NewServer(remoteURL, webhookURL, os.TempDir(), Record, "cassette.yaml")
	check(t, err)

	server := httpWrapper.InitializeServer(addressString)
	serverReady := make(chan struct{})
	go func() {
		addr := server.Addr
		ln, err := net.Listen("tcp", addr)
		if err != nil {
			log.Fatalf("server startup failed - Listen(): %v", err)
		}

		close(serverReady)

		if err := server.Serve(ln); err != http.ErrServerClosed {
			// unexpected error
			log.Fatalf("server startup failed - Serve(): %v", err)
		}
	}()

	<-serverReady

	// Send it 2 requests
	res1, err := http.Get("http://localhost:8080/")
	assert.NoError(t, err)
	assert.Equal(t, mockResponse1.StatusCode, res1.StatusCode)

	res2, err := http.Get("http://localhost:8080/")
	assert.NoError(t, err)
	assert.Equal(t, mockResponse2.StatusCode, res2.StatusCode)

	// Shutdown record server
	resShutdown, err := http.Get("http://localhost:8080/playback/cassette/eject")
	server.Shutdown(context.Background())
	assert.NoError(t, err)
	defer resShutdown.Body.Close()

	// --- Set up a replay server
	httpWrapper, err = NewServer(remoteURL, webhookURL, os.TempDir(), Replay, "cassette.yaml")
	check(t, err)
	server = httpWrapper.InitializeServer(addressString)

	go func() {
		if err := server.ListenAndServe(); err != http.ErrServerClosed {
			// unexpected error
			log.Fatalf("ListenAndServe(): %v", err)
		}
	}()

	// make sure server is ready
	time.Sleep(10 * time.Millisecond)

	// Send it the same 2 requests:
	replay1, err := http.Get("http://localhost:8080/")
	assert.NoError(t, err)
	replay2, err := http.Get("http://localhost:8080/")
	assert.NoError(t, err)

	// Assert that the original responses and replayed responses are the same
	check(t, assertHTTPResponsesAreEqual(t, res1, replay1))
	check(t, assertHTTPResponsesAreEqual(t, res2, replay2))

	// Also sanity check that the mock server is responding with the expected responses
	assert.Equal(t, "testHeaderValue", res1.Header.Get("testHeader"))
	bodyBytes1, err := ioutil.ReadAll(res1.Body)
	assert.NoError(t, err)
	assert.Equal(t, mockResponse1.Body, bodyBytes1)

	bodyBytes2, err := ioutil.ReadAll(res2.Body)
	assert.NoError(t, err)
	assert.Equal(t, mockResponse2.Body, bodyBytes2)

	// Shutdown replay server
	server.Shutdown(context.Background())
}

// Test the full server by switching between modes, loading and ejecting cassettes, and sending real stripe requests
func TestPlaybackSingleRunCreateCustomerAndStandaloneCharge(t *testing.T) {
	var remoteURL string

	// Initialize the mock responses we want to serve
	mockResponse1 := httpResponse{}
	mockResponse1.StatusCode = 200
	mockResponse1.Headers = make(http.Header)
	mockResponse1.Headers.Add("testHeader", "testHeaderValue")
	mockResponse1.Body = []byte("body 1")

	mockResponse2 := httpResponse{}
	mockResponse2.StatusCode = 402
	mockResponse2.Headers = make(http.Header)
	mockResponse2.Body = []byte("body 2")
	fixtureResponses := []httpResponse{mockResponse1, mockResponse2}

	ts := startMockServer(fixtureResponses)
	defer ts.Close()
	remoteURL = ts.URL

	// -- Setup Playback server
	addressString := "localhost:13111"
	cassetteFilepath := "test_record_replay_single_run.yaml"

	tempCassetteDir, err := ioutil.TempDir("", "playback-test-data-")
	defer os.RemoveAll(tempCassetteDir)

	assert.NoError(t, err)

	webhookURL := defaultLocalWebhookAddress // not used in this test
	httpWrapper, err := NewServer(remoteURL, webhookURL, tempCassetteDir, Record, cassetteFilepath)
	assert.NoError(t, err)

	server := httpWrapper.InitializeServer(addressString)
	serverReady := make(chan struct{})
	go func() {
		addr := server.Addr
		ln, err := net.Listen("tcp", addr)
		if err != nil {
			log.Fatalf("server startup failed - Listen(): %v", err)
		}

		close(serverReady)

		if err := server.Serve(ln); err != http.ErrServerClosed {
			// unexpected error
			log.Fatalf("server startup failed - Serve(): %v", err)
		}
	}()
	defer server.Shutdown(context.Background())
	<-serverReady

	fullAddressString := "http://" + addressString
	// --- Start interacting in RECORD MODE

	// Send it 2 requests
	res1, err := http.Get(fullAddressString + "/r1")
	assert.NoError(t, err)
	assert.Equal(t, mockResponse1.StatusCode, res1.StatusCode)

	res2, err := http.Get(fullAddressString + "/r2")
	assert.NoError(t, err)
	assert.Equal(t, mockResponse2.StatusCode, res2.StatusCode)

	// Tell server to save recording
	resp, err := http.Post(fullAddressString+"/playback/cassette/eject", "text/plain", nil)
	assert.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, 200, resp.StatusCode)
	// --- END RECORD MODE

	// --- Start interacting in REPLAY MODE
	resp, err = http.Post(fullAddressString+"/playback/mode/replay", "text/plain", nil)
	assert.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, 200, resp.StatusCode)

	resp, err = http.Post(fullAddressString+"/playback/cassette/load?filepath="+cassetteFilepath, "text/plain", nil)
	assert.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, 200, resp.StatusCode)

	replay1, err := http.Get(fullAddressString + "/r1")
	assert.NoError(t, err)
	replay2, err := http.Get(fullAddressString + "/r2")
	assert.NoError(t, err)

	// Assert that the original responses and replayed responses are the same
	check(t, assertHTTPResponsesAreEqual(t, res1, replay1))
	check(t, assertHTTPResponsesAreEqual(t, res2, replay2))

	// Also sanity check that the mock server is responding with the expected responses
	assert.Equal(t, "testHeaderValue", res1.Header.Get("testHeader"))
	bodyBytes1, err := ioutil.ReadAll(res1.Body)
	assert.NoError(t, err)
	assert.Equal(t, mockResponse1.Body, bodyBytes1)

	bodyBytes2, err := ioutil.ReadAll(res2.Body)
	assert.NoError(t, err)
	assert.Equal(t, mockResponse2.Body, bodyBytes2)

	// --- END REPLAY MODE
}

// TODO(DX-5699, DX-5700): add more test coverage
