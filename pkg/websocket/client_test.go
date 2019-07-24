package websocket

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	ws "github.com/gorilla/websocket"
	assert "github.com/stretchr/testify/require"
)

func TestClientWebhookEventHandler(t *testing.T) {
	wg := &sync.WaitGroup{}
	wg.Add(1)

	upgrader := ws.Upgrader{}
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.NotEmpty(t, r.UserAgent())
		assert.NotEmpty(t, r.Header.Get("X-Stripe-Client-User-Agent"))
		assert.Equal(t, "websocket-random-id", r.Header.Get("Websocket-Id"))
		c, err := upgrader.Upgrade(w, r, nil)
		assert.Nil(t, err)

		assert.Equal(t, "websocket_feature=webhook-payloads", r.URL.RawQuery)

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
		assert.Nil(t, err)

		err = c.WriteMessage(ws.TextMessage, msg)
		assert.Nil(t, err)
	}))
	defer ts.Close()

	url := "ws" + strings.TrimPrefix(ts.URL, "http")

	var rcvMsg *WebhookEvent
	client := NewClient(
		url,
		"websocket-random-id",
		"webhook-payloads",
		&Config{
			EventHandler: EventHandlerFunc(func(msg IncomingMessage) {
				rcvMsg = msg.WebhookEvent
				wg.Done()
			}),
		},
	)
	go client.Run()
	defer client.Stop()

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(500 * time.Millisecond):
		assert.FailNow(t, "Timed out waiting for response from test server")
	}

	assert.Equal(t, "TestAgent/v1", rcvMsg.HTTPHeaders["User-Agent"])
	assert.Equal(t, "t=123,v1=hunter2", rcvMsg.HTTPHeaders["Stripe-Signature"])
	assert.Equal(t, "{}", rcvMsg.EventPayload)
}
