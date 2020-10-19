package websocket

import (
	"context"
	"io"
	"io/ioutil"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type dummyConnection struct {
	gotSentMessage    chan (struct{})
	sent              [][]byte
	triggerError      error
	triggerMessage    func([]byte)
	triggerDisconnect func()
	onConnect         func()
}

func (dc *dummyConnection) Run(ctx context.Context, onMessage func([]byte), onDisconnect func()) func(io.Reader) error {
	dc.sent = [][]byte{}
	dc.triggerDisconnect = onDisconnect
	dc.triggerMessage = onMessage
	dc.gotSentMessage = make(chan struct{})
	send := func(r io.Reader) error {
		if dc.triggerError != nil {
			return dc.triggerError
		}
		msg, err := ioutil.ReadAll(r)
		if err != nil {
			return err
		}
		go func() {
			dc.sent = append(dc.sent, msg)
			dc.gotSentMessage <- struct{}{}
		}()
		return nil
	}
	if dc.onConnect != nil {
		dc.onConnect()
	}
	return send
}

func TestNewConnectionManager(t *testing.T) {
	cfg := &ConnectionManagerCfg{}
	cm := NewConnectionManager(cfg)
	require.Equal(t, time.Second, cm.requeueBackoffDuration, "NewConnectionManager should default requeueBackoffDuration to 1 second.")
	require.NotNil(t, cm.stripeAuthClient, "NewConnectionManager should initialize a new stripeAuthClient.")

	cfg.requeueBackoffDuration = time.Minute
	cm = NewConnectionManager(cfg)
	require.Equal(t, time.Minute, cm.requeueBackoffDuration, "NewConnectionManager should save cfg.requeueBackoffDuration.")
	require.Equal(t, cm.cfg, cfg, "NewConnectionManager should save the cfg in the ConnectionManager object.")
}

// func TestRun(t *testing.T) {
// 	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
// 		session := stripeauth.StripeCLISession{
// 			WebSocketID:                "some-id",
// 			WebSocketURL:               "wss://example.com/subscribe/acct_123",
// 			WebSocketAuthorizedFeature: "webhook-payloads",
// 		}
// 		w.WriteHeader(http.StatusOK)
// 		w.Header().Set("Content-Type", "application/json")
// 		json.NewEncoder(w).Encode(session)

// 		require.Equal(t, http.MethodPost, r.Method)
// 		require.Equal(t, "Bearer sk_test_123", r.Header.Get("Authorization"))
// 		require.NotEmpty(t, r.UserAgent())
// 		require.NotEmpty(t, r.Header.Get("X-Stripe-Client-User-Agent"))

// 		body, err := ioutil.ReadAll(r.Body)
// 		require.NoError(t, err)
// 		require.Equal(t, "device_name=my-device&websocket_feature=webhooks", string(body))
// 	}))
// 	defer ts.Close()

// 	logger := log.New()
// 	cm := NewConnectionManager(&ConnectionManagerCfg{
// 		Logger:           logger,
// 		APIBaseURL:       ts.URL,
// 		DeviceName:       "my-device",
// 		WebSocketFeature: "webhooks",
// 		Key:              "sk_test_123",
// 	})

// 	messageReceived := make(chan struct{})
// 	onMessage := func(msg []byte) {
// 		close(messageReceived)
// 	}
// 	terminated := make(chan struct{})
// 	onTerminate := func(err error) {
// 		close(terminated)
// 	}
// 	ctx := context.Background()

// 	cm.Run(ctx, onMessage, onTerminate)
// 	select {
// 	case <-messageReceived:
// 		t.Fatal("message received")
// 	case <-terminated:
// 		t.Fatal("terminated")
// 	}
// }

// func TestConnectionManagerRun(t *testing.T) {
// 	manager := NewConnectionManager(&ConnectionManagerCfg{
// 		requeueBackoffDuration: (time.Millisecond * 10),
// 	})

// 	firstConnected := make(chan struct{})
// 	secondConnected := make(chan struct{})
// 	firstConnection := dummyConnection{
// 		onConnect: func() { close(firstConnected) },
// 	}
// 	secondConnection := dummyConnection{
// 		onConnect: func() { close(secondConnected) },
// 	}

// 	var messages [][]byte
// 	gotMessage := make(chan struct{})
// 	onMessage := func(msg []byte) {
// 		messages = append(messages, msg)
// 		go func() {
// 			gotMessage <- struct{}{}
// 		}()
// 	}

// 	gotDisconnect := make(chan struct{})
// 	var disconnectErr error
// 	onDisconnect := func(err error) {
// 		disconnectErr = err
// 		close(gotDisconnect)
// 	}
// 	fmt.Println("running")

// 	// Run the connection manager.

// 	send := manager.Run(
// 		context.Background(),
// 		onMessage,
// 		onDisconnect,
// 	)

// 	{
// 		// Wait for the first connection to connect. Trigger a message
// 		// from the first connection. Assert that it gets passed to
// 		// our `onMessage` handler.
// 		waitFor(t, firstConnected, time.Second*1)
// 		msg := []byte("foo")
// 		firstConnection.triggerMessage(msg)
// 		waitFor(t, gotMessage, time.Second*1)
// 		require.Equal(t, 1, len(messages))
// 		require.Equal(t, msg, messages[0])
// 	}
// 	{
// 		// Now send a message through the connection handler, and assert that
// 		// it is channeled through the first connection.
// 		msg := []byte("bar")
// 		send(bytes.NewReader(msg))
// 		waitFor(t, firstConnection.gotSentMessage, time.Second)
// 		require.Equal(t, 1, len(firstConnection.sent))
// 		require.Equal(t, msg, firstConnection.sent[0])
// 	}
// 	{
// 		// Now cause the connection to error, send two messages through the connection handler, cause the first connection to disconnect, wait for the second connection to be established, and assert that the messages are channeled through the second connection.
// 		msg1 := []byte("baz")
// 		msg2 := []byte("qux")
// 		firstConnection.triggerError = errors.New("fail")
// 		send(bytes.NewReader(msg1))
// 		send(bytes.NewReader(msg2))
// 		firstConnection.triggerDisconnect()
// 		waitFor(t, secondConnected, time.Second)
// 		waitFor(t, secondConnection.gotSentMessage, time.Second)
// 		waitFor(t, secondConnection.gotSentMessage, time.Second)
// 		require.Equal(t, 2, len(secondConnection.sent))
// 		// The messages can be sent in either order, depending
// 		// on the timing of the reconnect and the requeue timeout
// 		require.Contains(t, secondConnection.sent, msg1)
// 		require.Contains(t, secondConnection.sent, msg2)
// 	}
// 	{
// 		// Check to see that sending and receiving messages
// 		// is correctly wired up, after the reconnection
// 		sentMsg := []byte("sent")
// 		receivedMsg := []byte("received")
// 		send(bytes.NewReader(sentMsg))
// 		secondConnection.triggerMessage(receivedMsg)
// 		waitFor(t, secondConnection.gotSentMessage, time.Second)
// 		waitFor(t, gotMessage, time.Second)
// 		require.Equal(t, 3, len(secondConnection.sent))
// 		require.Contains(t, secondConnection.sent, sentMsg)
// 		require.Equal(t, 2, len(messages))
// 		require.Equal(t, receivedMsg, messages[1])
// 	}
// 	{
// 		// Cause the second connection to disconnect, and assert that the
// 		// disconnection error is passed through to the onDisconnect handler
// 		secondConnection.triggerDisconnect()
// 		waitFor(t, gotDisconnect, time.Second)
// 		require.Equal(t, disconnectErr.Error(), "failed to reconnect")
// 	}
// }
