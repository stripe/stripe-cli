package websocket

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"sync"
	"time"

	ws "github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"

	"github.com/stripe/stripe-cli/pkg/stripeauth"
)

// Connection represents something such as a websocket connection, that can
// transmit messages, receive messages and dies.
type Connection interface {
	// Run accepts "onMessage" and "onDisconnect" handlers, so the caller can be
	// notified of incoming messages and disconnect events. It returns a "send"
	// function, which the caller should use to transmit.
	Run(ctx context.Context, onMessage func([]byte), onDisconnect func()) func(io.Reader) error
}

type connection struct {
	conn       *ws.Conn
	log        *log.Logger
	pongWait   time.Duration
	writeWait  time.Duration
	pingPeriod time.Duration
}

type ConnectWebsocketConfig struct {
	NoWSS     bool
	Logger    *log.Logger
	PongWait  time.Duration
	WriteWait time.Duration
}

func ConnectWebsocket(ctx context.Context, session *stripeauth.StripeCLISession, cfg ConnectWebsocketConfig) (Connection, *http.Response, error) {
	url, header := wsHeader(session.WebSocketURL, session.WebSocketID, session.WebSocketAuthorizedFeature, cfg.NoWSS)

	cfg.Logger.WithFields(log.Fields{
		"prefix": "websocket.connection.ConnectWebsocket",
		"url":    url,
	}).Debug("Dialing websocket")

	dialer := newWebSocketDialer(os.Getenv("STRIPE_CLI_UNIX_SOCKET"))
	conn, resp, err := dialer.DialContext(ctx, url, header)
	if err != nil {
		return nil, resp, err
	}
	return &connection{
		conn:       conn,
		log:        cfg.Logger,
		pongWait:   cfg.PongWait,
		writeWait:  cfg.WriteWait,
		pingPeriod: time.Second,
	}, resp, nil
}

func newWebSocketDialer(unixSocket string) *ws.Dialer {
	var dialer *ws.Dialer

	if unixSocket != "" {
		dialFunc := func(network, addr string) (net.Conn, error) {
			return net.Dial("unix", unixSocket)
		}
		dialer = &ws.Dialer{
			HandshakeTimeout: 10 * time.Second,
			NetDial:          dialFunc,
			Subprotocols:     subprotocols[:],
		}
	} else {
		dialer = &ws.Dialer{
			HandshakeTimeout: 10 * time.Second,
			Proxy:            http.ProxyFromEnvironment,
			Subprotocols:     subprotocols[:],
		}
	}

	return dialer
}

// Run starts the "readPump" for dispatching incoming messages from the
// websocket connection to the onMessage handler, and returns the "send"
// function which can be used to send messages over this websocket connection.
func (c *connection) Run(ctx context.Context, onMessage func([]byte), onDisconnect func()) func(io.Reader) error {
	once := sync.Once{}
	onFirstDisconnect := func() { once.Do(onDisconnect) }
	sendMsg := c.messageSender(onFirstDisconnect)
	sendData := func(r io.Reader) error {
		data, err := ioutil.ReadAll(r)
		if err != nil {
			return err
		}
		err = sendMsg(msg{
			isPing: false,
			data:   data,
		})
		if err != nil {
			return err
		}
		return nil
	}
	sendPing := func() error {
		return sendMsg(msg{
			isPing: true,
		})
	}
	go c.readPump(onMessage, onFirstDisconnect)
	go c.pingPong(sendPing, onFirstDisconnect)
	return sendData

}

// TODO: explain function of msg
type msg struct {
	isPing bool
	data   []byte
}

// messageSender wraps gorilla/websocket's "NextWriter" io.WriteCloser interface.
func (c *connection) messageSender(onDisconnect func()) func(msg) error {
	sendIt := func(m msg) error {
		err := c.conn.SetWriteDeadline(time.Now().Add(c.writeWait))
		if err != nil {
			return err
		}

		if m.isPing {
			fmt.Println("INSIDE IS PING")
			c.log.WithFields(log.Fields{
				"prefix": "connection.sendMsg",
			}).Debug("Sending ping message")
			if err = c.conn.WriteMessage(ws.PingMessage, m.data); err != nil {
				fmt.Println("ERROR SENDING PING")
				if ws.IsUnexpectedCloseError(err, ws.CloseNormalClosure) {
					c.log.Debug("write error: ", err)
				}
				onDisconnect()
				return err
			}
			return nil
		}

		if err = c.conn.WriteMessage(ws.TextMessage, m.data); err != nil {
			if ws.IsUnexpectedCloseError(err, ws.CloseNormalClosure) {
				c.log.Debug("write error: ", err)
			}
			onDisconnect()
			return err
		}
		return nil
	}

	// Only permit one write at a time. By default if you try and write while another write is
	// ongoing, gorilla/websocket will close the previous write and begin the new one. But it
	// is better to fail the second write and let the connectionManager requeue it after a
	// backoff.
	writing := false
	return func(m msg) error {
		if writing {
			return errors.New("concurrent write")
		}
		writing = true
		err := sendIt(m)
		writing = false
		return err
	}
}

func (c *connection) pingPong(sendPing func() error, onDisconnect func()) {
	stop := make(chan struct{})
	ticker := time.NewTicker(time.Millisecond * 200)
	fmt.Printf("tricker: %+v", ticker)

	go func() {
		defer func() {
			ticker.Stop()
		}()

		c.log.Debug("ticking...")
		for {
			select {
			case <-stop:
				c.log.Debug("stopped")
				return
			case <-ticker.C:
				c.log.Debug("sending ping...")
				sendPing()
				err := c.conn.SetWriteDeadline(time.Now().Add(c.writeWait))
				if err != nil {
					c.log.Debug("SetWriteDeadline error: ", err)
				}
			}
			c.log.Debug("ticked")
		}
	}()
}

// readPump uses gorilla/websocket's "conn.ReadMessage" to listen for messages coming over
// on the websocket connection. It also detects errors, and fires the "onDisconnect" handler
// when they occur. It also ensures that "onDisconnect" is fired if too much time has elasped
// since bytes were successfully read from the websocket connection (which shouldn't occur
// frequently if the connection is healthy, because of gorilla/websocket's built in ping/pong).
// The way this works is, if the gorilla/websocket connection's "ReadDeadline" is expired, then
// the "conn.ReadMessage" will return a close error. The ReadDeadline is extended on each
// successful pong.
func (c *connection) readPump(onMessage func([]byte), onDisconnect func()) {
	err := c.conn.SetReadDeadline(time.Now().Add(c.pongWait))
	c.log.Debug(fmt.Sprintf("%s", time.Now().Format(time.RFC3339)))
	c.log.Debug(fmt.Sprintf("%s", time.Now().Add(c.pongWait).Format(time.RFC3339)))
	if err != nil {
		c.log.Debug("SetReadDeadline error: ", err)
	}

	c.conn.SetPongHandler(func(string) error {
		c.log.WithFields(log.Fields{
			"prefix": "websocket.Connection.readPump",
		}).Debug("Received pong message")

		err := c.conn.SetReadDeadline(time.Now().Add(c.pongWait))
		if err != nil {
			c.log.Debug("SetReadDeadline error: ", err)
		}

		return nil
	})

	go func() {
		for {
			_, data, err := c.conn.ReadMessage()
			if err != nil {
				switch {
				case !ws.IsCloseError(err):
					// read errors do not prevent websocket reconnects in the CLI so we should
					// only display this on debug-level logging
					c.log.WithFields(log.Fields{
						"prefix": "websocket.Connection.readPump.Close",
					}).Debug("read error: ", err)
				case ws.IsUnexpectedCloseError(err, ws.CloseNormalClosure):
					c.log.WithFields(log.Fields{
						"prefix": "websocket.Connection.readPump.Close",
					}).Error("close error: ", err)
					c.log.WithFields(log.Fields{
						"prefix": "stripecli.ADDITIONAL_INFO",
					}).Error("If you run into issues, please re-run with `--log-level debug` and share the output with the Stripe team on GitHub.")
				default:
					c.log.Error("other error: ", err)
					c.log.WithFields(log.Fields{
						"prefix": "stripecli.ADDITIONAL_INFO",
					}).Error("If you run into issues, please re-run with `--log-level debug` and share the output with the Stripe team on GitHub.")
				}
				onDisconnect()
				return
			}

			c.log.WithFields(log.Fields{
				"prefix":  "websocket.Connection.readPump",
				"message": string(data),
			}).Debug("Incoming message")

			onMessage(data)
		}
	}()
}
