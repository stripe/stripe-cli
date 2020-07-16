package playback

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
	"github.com/stripe/stripe-go"
	"github.com/stripe/stripe-go/charge"
	"github.com/stripe/stripe-go/customer"
)

const stripeAPIURL = "https://api.stripe.com"
const defaultLocalAddress = "localhost:8080"
const defaultLocalWebhookAddress = "localhost:8888"

var stripeKey string
var runningInCI bool

// setup for tests
func init() {
	// When running locally (not in CI), we may want to load a .env file so we
	// can develop tests directly against testmode. But in CI we do not want to
	// load our actual keys, so use a dummy variable.
	// We also use the presence of a .env file to determine whether we are running locally, or in CI
	err := godotenv.Load("./.env")
	if err != nil {
		stripeKey = "sk_test_123"
		runningInCI = true
	} else {
		stripeKey = os.Getenv("STRIPE_SECRET_KEY")
		runningInCI = false
	}

	fmt.Println("Stripe key = ", stripeKey)
}

func assertHTTPResponsesAreEqual(t *testing.T, resp1 *http.Response, resp2 *http.Response) error {
	// Read the response bodies
	// resp1 body
	bodyBytes1, err := ioutil.ReadAll(resp1.Body)

	if err != nil {
		return err
	}

	bodyString1 := string(bodyBytes1)
	//reset the response body to the original unread state
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
	//reset the response body to the original unread state
	err = resp2.Body.Close()
	if err != nil {
		return err
	}
	resp2.Body = ioutil.NopCloser(bytes.NewBuffer(bodyBytes2))

	assert.Equal(t, bodyString1, bodyString2, "Response bodies differ.")
	assert.Equal(t, resp1.Status, resp2.Status, "Response statuses differ.")

	return nil
}

func startMockFixturesServer(responseFixtureFiles []string) *httptest.Server {
	responseCount := 0

	// Set up a local mock of a remote server will serve the provided http.Responses from the fixture files
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if responseCount >= len(responseFixtureFiles) {
			w.WriteHeader(500)
			fmt.Fprintln(w, "Mock server ran out of http.Response fixtures to serve. Is the test case written properly?")
			return
		}

		fixtureFileName := responseFixtureFiles[responseCount]
		responseCount++

		fullPath, err := filepath.Abs(filepath.Join("test-data/", fixtureFileName))
		if err != nil {
			w.WriteHeader(500)
			fmt.Fprintf(w, "Unexpected error when joining filepath: %v\n", err)
			return
		}
		var fileReader io.Reader
		fileReader, err = os.Open(fullPath)
		if err != nil {
			w.WriteHeader(500)
			fmt.Fprintf(w, "Unexpected error when reading fixtures file: %v\n", err)
			return
		}

		respGeneric, err := httpResponsefromBytes(&fileReader)
		if err != nil {
			w.WriteHeader(500)
			fmt.Fprintf(w, "Unexpected error when deserializing fixtures file: %v\n", err)
			return
		}

		resp := respGeneric.(*http.Response)

		// --- Write response back to client
		// Copy header
		w.WriteHeader(resp.StatusCode)
		copyHTTPHeader(w.Header(), resp.Header)

		// Copy body
		bodyBytes, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			handleErrorInHandler(w, err, 500)
			return
		}
		io.Copy(w, bytes.NewBuffer(bodyBytes))
		resp.Body = ioutil.NopCloser(bytes.NewBuffer(bodyBytes))
	}))
}

// Integration test for HTTP wrapper against simple HTTP serving remote
func TestGetFromSimpleWebsite(t *testing.T) {
	var remoteURL string
	if runningInCI {
		fixtureResponses := []string{"/simple-get-test/simpleRes1.bin", "/simple-get-test/simpleRes2.bin", "/simple-get-test/simpleRes3.bin"}
		ts := startMockFixturesServer(fixtureResponses)
		defer ts.Close()
		remoteURL = ts.URL
	} else {
		remoteURL = "https://gobyexample.com"
	}

	// Spin up an instance of the HTTP playback server in record mode
	var cassetteBuffer bytes.Buffer
	addressString := defaultLocalAddress
	webhookURL := defaultLocalWebhookAddress // not used in this test

	httpRecorder := newRecordServer(remoteURL, webhookURL)
	err := httpRecorder.insertCassette(&cassetteBuffer)
	check(t, err)

	server := httpRecorder.initializeServer(addressString)
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
	resShutdown, err := http.Get("http://localhost:8080/playback/stop")
	server.Shutdown(context.TODO())
	assert.NoError(t, err)
	defer resShutdown.Body.Close()

	// --- Set up a replay server
	httpReplayer := newReplayServer(webhookURL)
	err = httpReplayer.readCassette(&cassetteBuffer)
	check(t, err)

	replayServer := httpReplayer.initializeServer(addressString)
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
	check(t, assertHTTPResponsesAreEqual(t, res1, replay1))

	replay2, err := http.Get("http://localhost:8080/")
	assert.NoError(t, err)
	check(t, assertHTTPResponsesAreEqual(t, res2, replay2))

	replay3, err := http.Get("http://localhost:8080/")
	assert.NoError(t, err)
	check(t, assertHTTPResponsesAreEqual(t, res3, replay3))

	// Shutdown replay server
	replayServer.Shutdown(context.TODO())
}

// Integration test for HTTP wrapper against Stripe
// TODO: all the stripe tests should just use the SDK
func TestStripeSimpleGet(t *testing.T) {
	var remoteURL string
	if runningInCI {
		fixtureResponses := []string{"stripe-simple-get-test/res1.bin"}
		ts := startMockFixturesServer(fixtureResponses)
		defer ts.Close()
		remoteURL = ts.URL
	} else {
		remoteURL = stripeAPIURL
	}

	// Spin up an instance of the HTTP playback server in record mode
	var cassetteBuffer bytes.Buffer
	addressString := defaultLocalAddress
	webhookURL := defaultLocalWebhookAddress // not used in this test

	httpRecorder := newRecordServer(remoteURL, webhookURL)
	err := httpRecorder.insertCassette(&cassetteBuffer)
	check(t, err)

	server := httpRecorder.initializeServer(addressString)
	go func() {
		if err := server.ListenAndServe(); err != http.ErrServerClosed {
			// unexpected error
			log.Fatalf("ListenAndServe(): %v", err)
		}
	}()

	// GET /v1/balance
	client := http.Client{}
	req, err := http.NewRequest("GET", "http://localhost:8080/v1/balance", nil)
	assert.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+stripeKey)
	res1, err := client.Do(req)

	// Should record a 200 response
	assert.NoError(t, err)
	assert.Equal(t, 200, res1.StatusCode)

	// Shutdown record server
	resShutdown, err := http.Get("http://localhost:8080/playback/stop")
	server.Shutdown(context.TODO())
	assert.NoError(t, err)
	defer resShutdown.Body.Close()

	// --- Set up a replay server
	replayer := newReplayServer(webhookURL)
	err = replayer.readCassette(&cassetteBuffer)
	check(t, err)

	replayServer := replayer.initializeServer(addressString)
	go func() {
		if err := replayServer.ListenAndServe(); err != http.ErrServerClosed {
			// unexpected error
			log.Fatalf("ListenAndServe(): %v", err)
		}
	}()

	// Send it the same GET /v1/balance request:
	// Assert replayed message matches
	replayReq, err := http.NewRequest("GET", "http://localhost:8080/v1/balance", nil)
	assert.NoError(t, err)
	replayReq.Header.Set("Authorization", "Bearer "+stripeKey)
	replay1, err := client.Do(replayReq)
	assert.NoError(t, err)
	check(t, assertHTTPResponsesAreEqual(t, res1, replay1))

	// Shutdown replay server
	replayServer.Shutdown(context.TODO())
}

// If we make a Stripe request without the Authorization header, we should get a 401 Unauthorized
func TestStripeUnauthorizedErrorIsPassedOn(t *testing.T) {
	var remoteURL string
	if runningInCI {
		fixtureResponses := []string{"/stripe-unauth-error-test/res1.bin"}
		ts := startMockFixturesServer(fixtureResponses)
		defer ts.Close()
		remoteURL = ts.URL
	} else {
		remoteURL = stripeAPIURL
	}

	// Spin up an instance of the HTTP playback server in record mode
	var cassetteBuffer bytes.Buffer
	addressString := defaultLocalAddress
	webhookURL := defaultLocalWebhookAddress // not used in this test

	httpRecorder := newRecordServer(remoteURL, webhookURL)
	err := httpRecorder.insertCassette(&cassetteBuffer)
	check(t, err)

	server := httpRecorder.initializeServer(addressString)
	go func() {
		if err := server.ListenAndServe(); err != http.ErrServerClosed {
			// unexpected error
			log.Fatalf("ListenAndServe(): %v", err)
		}
	}()

	// GET /v1/balance
	client := http.Client{}
	req, err := http.NewRequest("GET", "http://localhost:8080/v1/balance", nil)
	assert.NoError(t, err)
	res1, err := client.Do(req)

	// Should record a 401 response
	assert.NoError(t, err)
	assert.Equal(t, 401, res1.StatusCode)

	// Shutdown record server
	resShutdown, err := http.Get("http://localhost:8080/playback/stop")
	server.Shutdown(context.TODO())
	assert.NoError(t, err)
	defer resShutdown.Body.Close()

	// --- Set up a replay server
	replayer := newReplayServer(webhookURL)
	err = replayer.readCassette(&cassetteBuffer)
	check(t, err)

	replayServer := replayer.initializeServer(addressString)
	go func() {
		if err := replayServer.ListenAndServe(); err != http.ErrServerClosed {
			// unexpected error
			log.Fatalf("ListenAndServe(): %v", err)
		}
	}()

	// Send it the same GET /v1/balance request:
	// Assert replayed message matches
	replayReq, err := http.NewRequest("GET", "http://localhost:8080/v1/balance", nil)
	assert.NoError(t, err)
	replay1, err := client.Do(replayReq)
	assert.NoError(t, err)
	check(t, assertHTTPResponsesAreEqual(t, res1, replay1))

	// Shutdown replay server
	replayServer.Shutdown(context.TODO())
}

// Test the full server by switching between modes, loading and ejecting cassettes, and sending real stripe requests
func TestRecordReplaySingleRunCreateCustomerAndStandaloneCharge(t *testing.T) {
	var remoteURL string
	if runningInCI {
		fixtureResponses := []string{"/create-customer-and-charge-test/res1.bin", "/create-customer-and-charge-test/res2.bin", "/create-customer-and-charge-test/res3.bin"}
		ts := startMockFixturesServer(fixtureResponses)
		defer ts.Close()
		remoteURL = ts.URL
	} else {
		remoteURL = stripeAPIURL
	}

	// -- Setup Playback server
	addressString := "localhost:13111"
	cassetteFilepath := "test-data/test_record_replay_single_run.yaml"

	// for now, write cassettes to this directory. Ideally we have a test output folder
	cassetteDirectory, err := filepath.Abs("")
	assert.NoError(t, err)

	webhookURL := defaultLocalWebhookAddress // not used in this test
	httpWrapper, err := NewRecordReplayServer(remoteURL, webhookURL, cassetteDirectory)
	assert.NoError(t, err)

	server := httpWrapper.InitializeServer(addressString)
	go func() {
		server.ListenAndServe()
	}()

	fullAddressString := "http://" + addressString

	resp, err := http.Get(fullAddressString + "/playback/mode/record")
	assert.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, 200, resp.StatusCode)

	resp, err = http.Get(fullAddressString + "/playback/cassette/load?filepath=" + cassetteFilepath)
	assert.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, 200, resp.StatusCode)

	// --- We'll use the stripe-go SDK in this test. Configure it to point to the server
	stripe.Key = stripeKey

	mockBackendConf := stripe.BackendConfig{
		URL: "http://" + addressString,
	}
	mockBackend := stripe.GetBackendWithConfig("api", &mockBackendConf)
	stripe.SetBackend(stripe.APIBackend, mockBackend)

	// --- Start interacting in RECORD MODE

	// Create a customer
	description := "Stripe Developer"
	email := "gostripe@stripe.com"
	params := &stripe.CustomerParams{
		Description: stripe.String(description),
		Email:       stripe.String(email),
	}

	c, err := customer.New(params)
	assert.NoError(t, err)
	assert.Equal(t, description, c.Description)
	assert.Equal(t, email, c.Email)

	// List customers
	listParams := &stripe.CustomerListParams{}
	listParams.Filters.AddFilter("limit", "", "1")
	i := customer.List(listParams)

	// Check the first customer returned (should be the most recent one we just created)
	assert.True(t, i.Next())
	newC := i.Customer()
	assert.Equal(t, c, newC)

	// Create a charge
	chargeParams := &stripe.ChargeParams{
		Amount:      stripe.Int64(2000),
		Currency:    stripe.String(string(stripe.CurrencyUSD)),
		Description: stripe.String("My First Test Charge (created for API docs)"),
		Source:      &stripe.SourceParams{Token: stripe.String("tok_mastercard")},
	}
	myCharge, err := charge.New(chargeParams)
	assert.NoError(t, err)

	assert.Equal(t, int64(2000), myCharge.Amount)
	assert.Equal(t, stripe.CurrencyUSD, myCharge.Currency)
	assert.Equal(t, "My First Test Charge (created for API docs)", myCharge.Description)

	// Tell server to save recording
	resp, err = http.Get(fullAddressString + "/playback/cassette/eject")
	assert.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, 200, resp.StatusCode)

	// --- END RECORD MODE

	// --- Start interacting in REPLAY MODE
	resp, err = http.Get(fullAddressString + "/playback/mode/replay")
	assert.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, 200, resp.StatusCode)

	resp, err = http.Get(fullAddressString + "/playback/cassette/load?filepath=" + cassetteFilepath)
	assert.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, 200, resp.StatusCode)

	// Make the same interactions, assert the same things
	// Create a customer
	replayC, err := customer.New(params)
	assert.NoError(t, err)
	assert.Equal(t, c, replayC)

	// List customers
	i = customer.List(listParams)

	// Check the first customer returned (should be the most recent one we just created)
	assert.True(t, i.Next())
	replayNewC := i.Customer()
	assert.Equal(t, newC, replayNewC)

	// Create a charge
	replayMyCharge, err := charge.New(chargeParams)
	assert.NoError(t, err)

	assert.Equal(t, myCharge, replayMyCharge)

	// --- END REPLAY MODE
}

// TODO: Test auto mode on the full server
// TODO: create a more complicated Stripe API integration test
// TODO: add a Stripe API test that depends on data sent in the request body (eg: stripe customers create)
//   	 this would be a regression test for a bug where request bodies weren't being forwarded by the playback proxy server
