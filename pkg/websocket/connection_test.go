package websocket

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"runtime/debug"
	"strings"
	"testing"
	"time"

	ws "github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

func waitFor(t *testing.T, done chan struct{}, timeout time.Duration) {
	timedOut := make(chan struct{})
	go func() {
		time.Sleep(timeout)
		close(timedOut)
	}()
	select {
	case <-done:
		return
	case <-timedOut:
		t.Fatal(fmt.Printf("Timed out waiting for event. Stack:\n%s", debug.Stack()))
	}
}

func TestConnectionRun(t *testing.T) {
	upgrader := ws.Upgrader{}
	gotMessageFromClient := make(chan struct{})
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := upgrader.Upgrade(w, r, nil)
		require.NoError(t, err)

		msg := []byte("\"hi\"")

		err = c.WriteMessage(ws.TextMessage, msg)
		require.NoError(t, err)

		go func() {
			_, reader, err := c.NextReader()
			require.NoError(t, err)
			_, _ = ioutil.ReadAll(reader)
			close(gotMessageFromClient)
		}()
	}))

	defer ts.Close()
	ctx := context.Background()
	url := "ws" + strings.TrimPrefix(ts.URL, "http")

	dialer := ws.Dialer{}
	conn, _, err := dialer.DialContext(ctx, url, http.Header{})
	require.NoError(t, err)

	myConn := connection{
		conn:       conn,
		pongWait:   1 * time.Second,
		writeWait:  1 * time.Second,
		pingPeriod: 1 * time.Second,
		log:        log.New(),
	}

	var send func(io.Reader) error
	gotMessageFromServer := make(chan struct{})

	onMessage := func(msg []byte) {
		close(gotMessageFromServer)
		send(bytes.NewReader([]byte("\"fromClient\"")))
	}

	onDisconnect := func() {}

	send = myConn.Run(context.Background(), onMessage, onDisconnect)

	waitFor(t, gotMessageFromServer, time.Second*2)
	waitFor(t, gotMessageFromClient, time.Second*2)
}

func TestConnectionRunHandlesExpiredReadDeadlines(t *testing.T) {
	upgrader := ws.Upgrader{}
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := upgrader.Upgrade(w, r, nil)
		require.NoError(t, err)

		time.Sleep(time.Second * 2)
		msg := []byte("\"hi\"")

		err = c.WriteMessage(ws.TextMessage, msg)
		require.NoError(t, err)
	}))

	defer ts.Close()
	ctx := context.Background()
	url := "ws" + strings.TrimPrefix(ts.URL, "http")

	dialer := ws.Dialer{}
	conn, _, err := dialer.DialContext(ctx, url, http.Header{})
	require.NoError(t, err)

	logger := log.New()
	logger.Level = log.DebugLevel

	myConn := connection{
		conn:     conn,
		pongWait: time.Second,
		log:      logger,
	}
	onMessage := func(msg []byte) {
		t.Fatal(string(msg))
	}

	onDisconnectCalled := make(chan struct{})
	onDisconnect := func() {
		close(onDisconnectCalled)
	}

	timedOut := make(chan struct{})
	go func() {
		time.Sleep(time.Second * 3)
		close(timedOut)
	}()
	send := myConn.Run(context.Background(), onMessage, onDisconnect)
	send(bytes.NewReader([]byte("\"fromClient\"")))
	select {
	case <-onDisconnectCalled:
		return
	case <-timedOut:
		t.Fatal("Timed out waiting for onDisconnect to be called")
	}

	err = send(bytes.NewReader([]byte("should error")))
	if err == nil {
		t.Fatal("Expected send to error after the connection was disconnected")
	}
}

type dummyConnection struct {
	gotSentMessage    chan (struct{})
	sent              [][]byte
	triggerError      error
	triggeredError    chan (struct{})
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

func TestConnectionManagerRun(t *testing.T) {
	manager := NewConnectionManager(&ConnectionManagerCfg{
		requeueBackoffDuration: (time.Millisecond * 10),
	})

	firstConnected := make(chan struct{})
	secondConnected := make(chan struct{})
	firstConnection := dummyConnection{
		onConnect: func() { close(firstConnected) },
	}
	secondConnection := dummyConnection{
		onConnect: func() { close(secondConnected) },
	}
	connN := 0
	connect := func() (Connection, error) {
		if connN == 0 {
			connN++
			return &firstConnection, nil
		}
		if connN == 1 {
			connN++
			return &secondConnection, nil
		}
		return nil, fmt.Errorf("failed to reconnect")
	}

	var messages [][]byte
	gotMessage := make(chan struct{})
	onMessage := func(msg []byte) {
		messages = append(messages, msg)
		go func() {
			gotMessage <- struct{}{}
		}()
	}

	gotDisconnect := make(chan struct{})
	var disconnectErr error
	onDisconnect := func(err error) {
		disconnectErr = err
		close(gotDisconnect)
	}
	fmt.Printf("running")

	// Run the connection manager.

	send := manager.Run(
		context.Background(),
		onMessage,
		onDisconnect,
		connect,
	)

	{
		// Wait for the first connection to connect. Trigger a message
		// from the first connection. Assert that it gets passed to
		// our `onMessage` handler.
		waitFor(t, firstConnected, time.Second*1)
		msg := []byte("foo")
		firstConnection.triggerMessage(msg)
		waitFor(t, gotMessage, time.Second*1)
		require.Equal(t, 1, len(messages))
		require.Equal(t, msg, messages[0])
	}
	{
		// Now send a message through the connection handler, and assert that
		// it is channeled through the first connection.
		msg := []byte("bar")
		send(bytes.NewReader(msg))
		waitFor(t, firstConnection.gotSentMessage, time.Second)
		require.Equal(t, 1, len(firstConnection.sent))
		require.Equal(t, msg, firstConnection.sent[0])
	}
	{
		// Now cause the connection to error, send two messages through the connection handler, cause the first connection to disconnect, wait for the second connection to be established, and assert that the messages are channeled through the second connection.
		msg1 := []byte("baz")
		msg2 := []byte("qux")
		firstConnection.triggerError = errors.New("fail")
		send(bytes.NewReader(msg1))
		send(bytes.NewReader(msg2))
		firstConnection.triggerDisconnect()
		waitFor(t, secondConnected, time.Second)
		waitFor(t, secondConnection.gotSentMessage, time.Second)
		waitFor(t, secondConnection.gotSentMessage, time.Second)
		require.Equal(t, 2, len(secondConnection.sent))
		// The messages can be sent in either order, depending
		// on the timing of the reconnect and the requeue timeout
		require.Contains(t, secondConnection.sent, msg1)
		require.Contains(t, secondConnection.sent, msg2)
	}
	{
		// Check to see that sending and receiving messages
		// is correctly wired up, after the reconnection
		sentMsg := []byte("sent")
		receivedMsg := []byte("received")
		send(bytes.NewReader(sentMsg))
		secondConnection.triggerMessage(receivedMsg)
		waitFor(t, secondConnection.gotSentMessage, time.Second)
		waitFor(t, gotMessage, time.Second)
		require.Equal(t, 3, len(secondConnection.sent))
		require.Contains(t, secondConnection.sent, sentMsg)
		require.Equal(t, 2, len(messages))
		require.Equal(t, receivedMsg, messages[1])
	}
	{
		// Cause the second connection to disconnect, and assert that the
		// disconnection error is passed through to the onDisconnect handler
		secondConnection.triggerDisconnect()
		waitFor(t, gotDisconnect, time.Second)
		require.Equal(t, disconnectErr.Error(), "failed to reconnect")
	}
}
