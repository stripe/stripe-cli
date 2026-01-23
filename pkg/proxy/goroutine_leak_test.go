package proxy

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"runtime"
	"sync/atomic"
	"testing"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	"github.com/stripe/stripe-cli/pkg/websocket"
)

// TestGoroutineLeak demonstrates that goroutines leak when processing
// webhook events with slow endpoints.
//
// The leak occurs in webhook_event_processor.go where:
// - Line 170: go endpoint.Post(evtCtx)
// - Line 214: go endpoint.PostV2(evtCtx)
//
// These goroutines are spawned without any tracking or way to wait for them.
// If endpoints are slow (or never respond), goroutines accumulate.
func TestGoroutineLeak(t *testing.T) {
	// Track initial goroutine count
	initialGoroutines := runtime.NumGoroutine()
	t.Logf("Initial goroutine count: %d", initialGoroutines)

	// Create a slow endpoint that never responds (simulates timeout)
	// Using a counter to track how many requests we receive
	var requestCount atomic.Int32

	slowServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount.Add(1)
		// Never respond - simulate timeout/slow endpoint
		<-time.After(10 * time.Second)
	}))
	defer slowServer.Close()

	// Create a second slow endpoint
	slowServer2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount.Add(1)
		// Never respond - simulate timeout/slow endpoint
		<-time.After(10 * time.Second)
	}))
	defer slowServer2.Close()

	// Create a processor with 2 slow endpoints, a short timeout, and a LOW concurrency limit
	cfg := &WebhookEventProcessorConfig{
		Log:                   &log.Logger{Out: io.Discard},
		Events:                []string{"*"},
		OutCh:                 make(chan websocket.IElement, 100),
		Timeout:               1, // 1 second timeout for HTTP client
		MaxForwardConcurrency: 2, // Low limit to test semaphore blocking
	}

	// Create routes pointing to our slow servers
	routes := []EndpointRoute{
		{
			URL:        slowServer.URL,
			Connect:    false,
			EventTypes: []string{"*"},
		},
		{
			URL:        slowServer2.URL,
			Connect:    false,
			EventTypes: []string{"*"},
		},
	}

	sendMessage := func(msg *websocket.OutgoingMessage) {
		// Silently discard messages
	}

	processor := NewWebhookEventProcessor(sendMessage, routes, cfg)

	// Verify the HTTP client has the right timeout
	for _, client := range processor.endpointClients {
		require.Equal(t, time.Duration(cfg.Timeout)*time.Second, client.cfg.HTTPClient.Timeout)
	}

	// Create a test webhook event
	eventPayload := map[string]interface{}{
		"id":   "evt_test123",
		"type": "payment_intent.succeeded",
		"data": map[string]interface{}{
			"object": map[string]interface{}{
				"id": "pi_test123",
			},
		},
	}
	payloadBytes, err := json.Marshal(eventPayload)
	require.NoError(t, err)

	webhookEvent := &websocket.WebhookEvent{
		Endpoint:     websocket.WebhookEndpoint{},
		EventPayload: string(payloadBytes),
		HTTPHeaders:  map[string]string{"Content-Type": "application/json"},
		Type:         "payment_intent.succeeded",
	}

	// Number of events to send
	numEvents := 5

	// Send the events
	msg := websocket.IncomingMessage{WebhookEvent: webhookEvent}
	for i := 0; i < numEvents; i++ {
		processor.ProcessEvent(msg)
	}

	// Wait for events to be processed
	time.Sleep(200 * time.Millisecond)

	// Check how many requests reached our servers
	receivedRequests := requestCount.Load()
	t.Logf("Requests received by endpoints: %d (max concurrent: %d)", receivedRequests, cfg.MaxForwardConcurrency)

	// With MaxForwardConcurrency=2 and slow endpoints (10s timeout),
	// only 2 requests should be in flight at once. Since we send events quickly
	// but endpoints are slow, we expect exactly MaxForwardConcurrency requests
	// to be received (one per endpoint).
	require.Equal(t, int32(cfg.MaxForwardConcurrency), receivedRequests, "Should have MaxForwardConcurrency concurrent forwards")

	// Call Shutdown to wait for in-flight forwards and release the semaphore
	processor.Shutdown()

	// After shutdown, the semaphore is closed and all goroutines should be done
	time.Sleep(100 * time.Millisecond)

	finalGoroutines := runtime.NumGoroutine()
	t.Logf("Final goroutine count after shutdown: %d", finalGoroutines)

	// After shutdown, goroutines should be cleaned up
	// We allow some tolerance for test infrastructure goroutines
	// (httptest servers may have some goroutines for connections)
	increase := finalGoroutines - initialGoroutines
	require.Less(t, increase, 10, "Goroutines should be mostly cleaned up after shutdown")
}

// TestGoroutineLeak_V2Events demonstrates goroutine leak with V2 events
func TestGoroutineLeak_V2Events(t *testing.T) {
	// Track initial goroutine count
	initialGoroutines := runtime.NumGoroutine()
	t.Logf("Initial goroutine count: %d", initialGoroutines)

	var requestCount atomic.Int32

	slowServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount.Add(1)
		<-time.After(10 * time.Second)
	}))
	defer slowServer.Close()

	slowServer2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount.Add(1)
		<-time.After(10 * time.Second)
	}))
	defer slowServer2.Close()

	cfg := &WebhookEventProcessorConfig{
		Log:                   &log.Logger{Out: io.Discard},
		ThinEvents:             []string{"*"},
		OutCh:                  make(chan websocket.IElement, 100),
		Timeout:                1,
		MaxForwardConcurrency:  2, // Low limit to test semaphore blocking
	}

	// For V2 events with empty context, we need Connect: true to get SupportsContext to return true
	// OR we can use a non-empty context with Connect: false
	routes := []EndpointRoute{
		{
			URL:                slowServer.URL,
			Connect:            true, // Need connect=true for events with context
			EventTypes:         []string{"*"},
			IsEventDestination: true,
		},
		{
			URL:                slowServer2.URL,
			Connect:            true,
			EventTypes:         []string{"*"},
			IsEventDestination: true,
		},
	}

	sendMessage := func(msg *websocket.OutgoingMessage) {}
	processor := NewWebhookEventProcessor(sendMessage, routes, cfg)

	// Create a test V2 event with context (for connect endpoints)
	v2EventPayload := map[string]interface{}{
		"id":      "evt_v2_test",
		"type":    "payment",
		"context": "acct_123", // Non-empty context for connect endpoints
	}
	payloadBytes, err := json.Marshal(v2EventPayload)
	require.NoError(t, err)

	v2Event := &websocket.StripeV2Event{
		Type:               "v2_event",
		Payload:            string(payloadBytes),
		HTTPHeaders:        map[string]string{"Content-Type": "application/json"},
		EventDestinationID: "dest_test",
	}

	msg := websocket.IncomingMessage{StripeV2Event: v2Event}

	// Send multiple events
	numEvents := 5
	for i := 0; i < numEvents; i++ {
		processor.ProcessEvent(msg)
	}

	time.Sleep(200 * time.Millisecond)

	receivedRequests := requestCount.Load()
	t.Logf("Requests received by endpoints: %d (max concurrent: %d)", receivedRequests, cfg.MaxForwardConcurrency)

	// With MaxForwardConcurrency=2 and slow endpoints,
	// only 2 requests should be in flight at once.
	require.Equal(t, int32(cfg.MaxForwardConcurrency), receivedRequests, "Should have MaxForwardConcurrency concurrent forwards for V2")

	// Call Shutdown to wait for in-flight forwards
	processor.Shutdown()

	// After shutdown, wait a bit for goroutines to finish
	time.Sleep(100 * time.Millisecond)

	finalGoroutines := runtime.NumGoroutine()
	t.Logf("Final goroutine count after shutdown: %d", finalGoroutines)

	// After shutdown, goroutines should be cleaned up
	// We allow some tolerance for test infrastructure goroutines
	increase := finalGoroutines - initialGoroutines
	require.Less(t, increase, 10, "Goroutines should be mostly cleaned up after shutdown")
}

// TestGoroutineLeak_Baseline is a baseline test showing normal goroutine behavior
// when NOT using the webhook event processor
func TestGoroutineLeak_Baseline(t *testing.T) {
	initialGoroutines := runtime.NumGoroutine()

	// Create a channel to coordinate goroutine lifecycle
	done := make(chan struct{})

	// Spawn some goroutines that we CAN track and wait for
	numGoroutines := 20
	for i := 0; i < numGoroutines; i++ {
		go func() {
			time.Sleep(10 * time.Millisecond)
			done <- struct{}{}
		}()
	}

	// Wait for all goroutines to complete
	for i := 0; i < numGoroutines; i++ {
		<-done
	}

	currentGoroutines := runtime.NumGoroutine()

	t.Logf("Initial goroutines: %d", initialGoroutines)
	t.Logf("Current goroutines after cleanup: %d", currentGoroutines)

	// With proper tracking and waiting, goroutine count should return to baseline
	// (allowing for some variance due to test infrastructure)
	diff := currentGoroutines - initialGoroutines
	require.LessOrEqual(t, diff, 5, "Baseline test should not leak goroutines")
}

// TestGoroutineLeak_EndpointClientDirectly tests the EndpointClient.Post method directly
// to show the goroutine leak at a lower level
func TestGoroutineLeak_EndpointClientDirectly(t *testing.T) {
	initialGoroutines := runtime.NumGoroutine()

	var requestCount atomic.Int32

	slowServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount.Add(1)
		<-time.After(10 * time.Second)
	}))
	defer slowServer.Close()

	// Create an EndpointClient with a short timeout
	client := &EndpointClient{
		URL:     slowServer.URL,
		connect: false,
		events:  map[string]bool{"*": true},
		cfg: &EndpointConfig{
			HTTPClient: &http.Client{
				Timeout: 1 * time.Second,
				CheckRedirect: func(req *http.Request, via []*http.Request) error {
					return http.ErrUseLastResponse
				},
			},
			Log:             &log.Logger{Out: io.Discard},
			ResponseHandler: EndpointResponseHandlerFunc(func(eventContext, string, *http.Response) {}),
			OutCh:           make(chan websocket.IElement, 100),
		},
	}

	// Create test event context
	eventPayload := map[string]interface{}{
		"id":   "evt_test",
		"type": "payment_intent.succeeded",
	}
	payloadBytes, _ := json.Marshal(eventPayload)

	evtCtx := eventContext{
		requestBody: string(payloadBytes),
		requestHeaders: map[string]string{
			"Content-Type": "application/json",
		},
	}

	// Spawn multiple goroutines directly, simulating what ProcessEvent does
	numGoroutines := 10
	for i := 0; i < numGoroutines; i++ {
		go client.Post(evtCtx)
	}

	time.Sleep(200 * time.Millisecond)

	currentGoroutines := runtime.NumGoroutine()
	goroutinesIncrease := currentGoroutines - initialGoroutines

	t.Logf("Initial goroutines: %d", initialGoroutines)
	t.Logf("Current goroutines: %d", currentGoroutines)
	t.Logf("Goroutine increase: %d", goroutinesIncrease)
	t.Logf("Requests received: %d", requestCount.Load())

	// Shows that spawning goroutines without tracking causes leaks
	require.Greater(t, goroutinesIncrease, numGoroutines/2, "Direct goroutine spawning shows leak")
}
