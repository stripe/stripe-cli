package websocket

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	ws "github.com/gorilla/websocket"

	"github.com/stretchr/testify/require"
)

func TestClientWebhookEventHandler(t *testing.T) {
	upgrader := ws.Upgrader{}
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.NotEmpty(t, r.UserAgent())
		require.NotEmpty(t, r.Header.Get("X-Stripe-Client-User-Agent"))
		require.Equal(t, "websocket-random-id", r.Header.Get("Websocket-Id"))
		c, err := upgrader.Upgrade(w, r, nil)
		require.NoError(t, err)

		require.Equal(t, "websocket_feature=webhook-payloads", r.URL.RawQuery)

		defer c.Close()

		evt := WebhookEvent{
			EventPayload: "{}",
			HTTPHeaders: map[string]string{
				"User-Agent":       "TestAgent/v1",
				"Stripe-Signature": "t=123,v1=hunter2",
			},
			Type: "webhook_event",
		}

		msg, err := json.Marshal(evt)
		require.NoError(t, err)

		err = c.WriteMessage(ws.TextMessage, msg)
		require.NoError(t, err)
	}))

	defer ts.Close()

	url := "ws" + strings.TrimPrefix(ts.URL, "http")

	var rcvMsg WebhookEvent

	rcvMsgChan := make(chan WebhookEvent)

	client := NewClient(
		url,
		"websocket-random-id",
		"webhook-payloads",
		&Config{
			EventHandler: EventHandlerFunc(func(msg IncomingMessage) {
				rcvMsgChan <- *msg.WebhookEvent
			}),
		},
	)

	go client.Run(context.Background())

	defer client.Stop()

	select {
	case rcvMsg = <-rcvMsgChan:
	case <-time.After(500 * time.Millisecond):
		require.FailNow(t, "Timed out waiting for response from test server")
	}

	require.Equal(t, "TestAgent/v1", rcvMsg.HTTPHeaders["User-Agent"])
	require.Equal(t, "t=123,v1=hunter2", rcvMsg.HTTPHeaders["Stripe-Signature"])
	require.Equal(t, "{}", rcvMsg.EventPayload)
}

func TestClientWebhookV2EventHandler(t *testing.T) {
	upgrader := ws.Upgrader{}
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.NotEmpty(t, r.UserAgent())
		require.NotEmpty(t, r.Header.Get("X-Stripe-Client-User-Agent"))
		require.Equal(t, "websocket-random-id", r.Header.Get("Websocket-Id"))
		c, err := upgrader.Upgrade(w, r, nil)
		require.NoError(t, err)

		require.Equal(t, "websocket_feature=webhook-payloads", r.URL.RawQuery)

		defer c.Close()

		evt := StripeV2Event{
			Payload: "{}",
			HTTPHeaders: map[string]string{
				"User-Agent":       "TestAgent/v1",
				"Stripe-Signature": "t=123,v1=hunter2",
			},
			Type: "v2_event",
		}

		msg, err := json.Marshal(evt)
		require.NoError(t, err)

		err = c.WriteMessage(ws.TextMessage, msg)
		require.NoError(t, err)
	}))

	defer ts.Close()

	url := "ws" + strings.TrimPrefix(ts.URL, "http")

	var rcvMsg StripeV2Event

	rcvMsgChan := make(chan StripeV2Event)

	client := NewClient(
		url,
		"websocket-random-id",
		"webhook-payloads",
		&Config{
			EventHandler: EventHandlerFunc(func(msg IncomingMessage) {
				rcvMsgChan <- *msg.StripeV2Event
			}),
		},
	)

	go client.Run(context.Background())

	defer client.Stop()

	select {
	case rcvMsg = <-rcvMsgChan:
	case <-time.After(500 * time.Millisecond):
		require.FailNow(t, "Timed out waiting for response from test server")
	}

	require.Equal(t, "TestAgent/v1", rcvMsg.HTTPHeaders["User-Agent"])
	require.Equal(t, "t=123,v1=hunter2", rcvMsg.HTTPHeaders["Stripe-Signature"])
	require.Equal(t, "{}", rcvMsg.Payload)
}

func TestClientRequestLogEventHandler(t *testing.T) {
	wg := &sync.WaitGroup{}
	wg.Add(1)

	upgrader := ws.Upgrader{}
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.NotEmpty(t, r.UserAgent())
		require.NotEmpty(t, r.Header.Get("X-Stripe-Client-User-Agent"))
		require.Equal(t, "websocket-random-id", r.Header.Get("Websocket-Id"))
		c, err := upgrader.Upgrade(w, r, nil)
		require.NoError(t, err)

		require.Equal(t, "websocket_feature=request-log-payloads", r.URL.RawQuery)

		defer c.Close()

		evt := RequestLogEvent{
			EventPayload: "{}",
			RequestLogID: "resp_123",
			Type:         "request_log_event",
		}

		msg, err := json.Marshal(evt)
		require.NoError(t, err)

		err = c.WriteMessage(ws.TextMessage, msg)
		require.NoError(t, err)
	}))

	defer ts.Close()

	url := "ws" + strings.TrimPrefix(ts.URL, "http")

	var rcvMsg *RequestLogEvent

	client := NewClient(
		url,
		"websocket-random-id",
		"request-log-payloads",
		&Config{
			EventHandler: EventHandlerFunc(func(msg IncomingMessage) {
				rcvMsg = msg.RequestLogEvent
				wg.Done()
			}),
		},
	)

	go client.Run(context.Background())

	defer client.Stop()

	done := make(chan struct{})

	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(500 * time.Millisecond):
		require.FailNow(t, "Timed out waiting for response from test server")
	}

	require.Equal(t, "resp_123", rcvMsg.RequestLogID)
	require.Equal(t, "request_log_event", rcvMsg.Type)
	require.Equal(t, "{}", rcvMsg.EventPayload)
}

func TestClientExpiredError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(401)
		_, err := w.Write([]byte("{\"error\": {\"message\": \"Unknown WebSocket ID.\"}}"))
		require.NoError(t, err)
	}))

	defer ts.Close()

	url := "ws" + strings.TrimPrefix(ts.URL, "http")

	client := NewClient(
		url,
		"websocket-random-id",
		"webhook-payloads",
		&Config{
			ConnectAttemptWait: 1,
		},
	)

	go client.Run(context.Background())

	select {
	case <-client.NotifyExpired:
	case <-time.After(500 * time.Millisecond):
		require.FailNow(t, "Timed out waiting for response from test server")
	}
}
