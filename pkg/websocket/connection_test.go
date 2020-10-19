package websocket

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"reflect"
	"runtime/debug"
	"strings"
	"testing"
	"time"

	ws "github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"

	"github.com/stripe/stripe-cli/pkg/stripeauth"
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

func TestConnectWebsocket(t *testing.T) {
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

	url := "ws" + strings.TrimPrefix(ts.URL, "http")
	stripeSession := &stripeauth.StripeCLISession{
		WebSocketURL:               url,
		WebSocketID:                "1234",
		WebSocketAuthorizedFeature: "webhook",
	}
	cfg := ConnectWebsocketConfig{
		NoWSS:  false,
		Logger: log.New(),
	}
	if stripeSession == nil || cfg.NoWSS == true {
		t.Fatal("bad")
	}
	ctx := context.Background()
	conn, resp, err := ConnectWebsocket(ctx, stripeSession, cfg)
	defer resp.Body.Close()
	if err != nil {
		t.Fatal("Should be able to create a Connection", err)
	}
	require.Equal(t, "connection", reflect.TypeOf(conn).Elem().Name())
	require.Equal(t, "Response", reflect.TypeOf(resp).Elem().Name())
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

func TestConnectionRunPingPong(t *testing.T) {
	upgrader := ws.Upgrader{}
	receivedPing := make(chan struct{})
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Fatal(fmt.Printf("Error upgrading server to ws. Stack:\n%s", debug.Stack()))
			return
		}
		c.SetPingHandler(func(message string) error {
			close(receivedPing)
			return nil
		})
		for {
			c.ReadMessage()
		}
	}))
	defer ts.Close()

	ctx := context.Background()
	url := "ws" + strings.TrimPrefix(ts.URL, "http")

	dialer := ws.Dialer{}
	conn, resp, err := dialer.DialContext(ctx, url, http.Header{})
	defer resp.Body.Close()
	require.NoError(t, err)

	myConn := connection{
		conn:       conn,
		pongWait:   1 * time.Second,
		writeWait:  1 * time.Second,
		pingPeriod: 1 * time.Second,
		log:        log.New(),
	}

	onMessage := func(msg []byte) {}
	onDisconnect := func() {}
	myConn.Run(context.Background(), onMessage, onDisconnect)

	// we expect c.pingPong to be initiated by Run and send a ping in 200ms.
	waitFor(t, receivedPing, 250*time.Millisecond)
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
	conn, resp, err := dialer.DialContext(ctx, url, http.Header{})
	defer resp.Body.Close()
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
		time.Sleep(time.Second)
		close(timedOut)
	}()
	send := myConn.Run(context.Background(), onMessage, onDisconnect)
	send(bytes.NewReader([]byte("\"fromClient\"")))
	select {
	case <-timedOut:
		t.Fatal("Timed out waiting for onDisconnect to be called")
	case <-onDisconnectCalled:
		return
	}

	err = send(bytes.NewReader([]byte("should error")))
	if err == nil {
		t.Fatal("Expected send to error after the connection was disconnected")
	}
}
