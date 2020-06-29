package vcr

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"testing"

	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
)

var stripeKey string

// setup for tests
func init() {
	err := godotenv.Load("./.env")
	check(err)

	stripeKey = os.Getenv("STRIPE_SECRET_KEY")
	fmt.Println("Stripe key = ", stripeKey)
}

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

// Integration test for HTTP wrapper against simple HTTP serving remote
func TestGetFromSimpleWebsite(t *testing.T) {
	// Spin up an instance of the HTTP vcr server in record mode
	var cassetteBuffer bytes.Buffer
	addressString := "localhost:8080"
	remoteURL := "https://gobyexample.com"

	httpRecorder, err := NewHttpRecorder(&cassetteBuffer, remoteURL)
	check(err)

	server := httpRecorder.InitializeServer(addressString)
	go func() {
		if err := server.ListenAndServe(); err != http.ErrServerClosed {
			// unexpected error
			log.Fatalf("ListenAndServe(): %v", err)
		}
	}()

	// Send it 3 requests
	res1, err := http.Get("http://localhost:8080/")
	assert.NoError(t, err)
	assert.Equal(t, 200, res1.StatusCode)

	res2, err := http.Get("http://localhost:8080/")
	assert.NoError(t, err)
	assert.Equal(t, 200, res2.StatusCode)

	res3, err := http.Get("http://localhost:8080/")
	assert.NoError(t, err)
	assert.Equal(t, 200, res3.StatusCode)

	// Shutdown record server
	_, err = http.Get("http://localhost:8080/vcr/stop")
	server.Shutdown(context.TODO())
	assert.NoError(t, err)

	// --- Set up a replay server
	httpReplayer, err := NewHttpReplayer(&cassetteBuffer)
	check(err)

	replayServer := httpReplayer.InitializeServer(addressString)
	go func() {
		if err := replayServer.ListenAndServe(); err != http.ErrServerClosed {
			// unexpected error
			log.Fatalf("ListenAndServe(): %v", err)
		}
	}()

	// Send it the same 3 requests:
	// Assert on the replay messages
	replay1, err := http.Get("http://localhost:8080/")
	assert.NoError(t, err)
	check(assertHttpResponsesAreEqual(t, res1, replay1))

	replay2, err := http.Get("http://localhost:8080/")
	assert.NoError(t, err)
	check(assertHttpResponsesAreEqual(t, res2, replay2))

	replay3, err := http.Get("http://localhost:8080/")
	assert.NoError(t, err)
	check(assertHttpResponsesAreEqual(t, res3, replay3))

	// Shutdown replay server
	replayServer.Shutdown(context.TODO())
}

// Integration test for HTTP wrapper against Stripe
// TODO: all the stripe tests should just use the SDK
func TestStripeSimpleGet(t *testing.T) {
	// Spin up an instance of the HTTP vcr server in record mode
	var cassetteBuffer bytes.Buffer
	addressString := "localhost:8080"
	remoteURL := "https://api.stripe.com"

	httpRecorder, err := NewHttpRecorder(&cassetteBuffer, remoteURL)
	check(err)

	server := httpRecorder.InitializeServer(addressString)
	go func() {
		if err := server.ListenAndServe(); err != http.ErrServerClosed {
			// unexpected error
			log.Fatalf("ListenAndServe(): %v", err)
		}
	}()

	// GET /v1/balance
	client := http.Client{}
	req, err := http.NewRequest("GET", "http://localhost:8080/v1/balance", nil)
	req.Header.Set("Authorization", "Bearer "+stripeKey)
	res1, err := client.Do(req)

	// Should record a 200 response
	assert.NoError(t, err)
	assert.Equal(t, 200, res1.StatusCode)

	// Shutdown record server
	_, err = http.Get("http://localhost:8080/vcr/stop")
	server.Shutdown(context.TODO())
	assert.NoError(t, err)

	// --- Set up a replay server
	replayVcr, err := NewHttpReplayer(&cassetteBuffer)
	check(err)

	replayServer := replayVcr.InitializeServer(addressString)
	go func() {
		if err := replayServer.ListenAndServe(); err != http.ErrServerClosed {
			// unexpected error
			log.Fatalf("ListenAndServe(): %v", err)
		}
	}()

	// Send it the same GET /v1/balance request:
	// Assert replayed message matches
	replayReq, err := http.NewRequest("GET", "http://localhost:8080/v1/balance", nil)
	replayReq.Header.Set("Authorization", "Bearer "+stripeKey)
	replay1, err := client.Do(replayReq)
	assert.NoError(t, err)
	check(assertHttpResponsesAreEqual(t, res1, replay1))

	// Shutdown replay server
	replayServer.Shutdown(context.TODO())
}

// If we make a Stripe request without the Authorization header, we should get a 401 Unauthorized
func TestStripeUnauthorizedErrorIsPassedOn(t *testing.T) {
	// Spin up an instance of the HTTP vcr server in record mode
	var cassetteBuffer bytes.Buffer
	addressString := "localhost:8080"
	remoteURL := "https://api.stripe.com"

	httpRecorder, err := NewHttpRecorder(&cassetteBuffer, remoteURL)
	check(err)

	server := httpRecorder.InitializeServer(addressString)
	go func() {
		if err := server.ListenAndServe(); err != http.ErrServerClosed {
			// unexpected error
			log.Fatalf("ListenAndServe(): %v", err)
		}
	}()

	// GET /v1/balance
	client := http.Client{}
	req, err := http.NewRequest("GET", "http://localhost:8080/v1/balance", nil)
	res1, err := client.Do(req)

	// Should record a 401 response
	assert.NoError(t, err)
	assert.Equal(t, 401, res1.StatusCode)

	// Shutdown record server
	_, err = http.Get("http://localhost:8080/vcr/stop")
	server.Shutdown(context.TODO())
	assert.NoError(t, err)

	// --- Set up a replay server
	replayVcr, err := NewHttpReplayer(&cassetteBuffer)
	check(err)

	replayServer := replayVcr.InitializeServer(addressString)
	go func() {
		if err := replayServer.ListenAndServe(); err != http.ErrServerClosed {
			// unexpected error
			log.Fatalf("ListenAndServe(): %v", err)
		}
	}()

	// Send it the same GET /v1/balance request:
	// Assert replayed message matches
	replayReq, err := http.NewRequest("GET", "http://localhost:8080/v1/balance", nil)
	replay1, err := client.Do(replayReq)
	assert.NoError(t, err)
	check(assertHttpResponsesAreEqual(t, res1, replay1))

	// Shutdown replay server
	replayServer.Shutdown(context.TODO())
}

func TestStripeSimpleGetWithHttps(t *testing.T) {
	// Spin up an instance of the HTTP vcr server in record mode
	var cassetteBuffer bytes.Buffer
	addressString := "localhost:8080"
	remoteURL := "https://api.stripe.com"

	httpRecorder, err := NewHttpRecorder(&cassetteBuffer, remoteURL)
	check(err)

	recordServer := httpRecorder.InitializeServer(addressString)
	go func() {
		if err := recordServer.ListenAndServeTLS("cert.pem", "key.pem"); err != http.ErrServerClosed {
			// unexpected error
			log.Fatalf("ListenAndServeTLS() 1: %v", err)
		}
	}()
	// Initialize the http.Client used to send requests
	// our test server uses a self-signed certificate -- configure the client to accept that
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr}

	// GET /v1/balance
	req, err := http.NewRequest("GET", "https://localhost:8080/v1/balance", nil)
	req.Header.Set("Authorization", "Bearer "+stripeKey)
	res1, err := client.Do(req)

	// Should record a 200 response
	assert.NoError(t, err)
	assert.Equal(t, 200, res1.StatusCode)

	// Shutdown record server
	shutdownReq, err := http.NewRequest("GET", "https://localhost:8080/vcr/stop", nil)
	assert.NoError(t, err)
	_, err = client.Do(shutdownReq)
	assert.NoError(t, err)
	recordServer.Shutdown(context.TODO())

	// --- Set up a replay server
	replayVcr, err := NewHttpReplayer(&cassetteBuffer)
	check(err)

	replayServer := replayVcr.InitializeServer(addressString)
	go func() {
		if err := replayServer.ListenAndServeTLS("cert.pem", "key.pem"); err != http.ErrServerClosed {
			// unexpected error
			log.Fatalf("ListenAndServeTLS() 2: %v", err)
		}
	}()

	// Send it the same GET /v1/balance request:
	// Assert replayed message matches
	replayReq, err := http.NewRequest("GET", "https://localhost:8080/v1/balance", nil)
	assert.NoError(t, err)

	replayReq.Header.Set("Authorization", "Bearer "+stripeKey)

	replay1, err := client.Do(replayReq)
	assert.NoError(t, err)
	check(assertHttpResponsesAreEqual(t, res1, replay1))

	// Shutdown replay server
	replayServer.Shutdown(context.TODO())
}
