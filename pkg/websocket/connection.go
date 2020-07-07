package websocket

import (
	"context"
	"io"
	"time"

	ws "github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"
)

// ConnectionManager provides an interface for interacting with a *series*
// of Connections; that establishes a new connection when the previous connection
// disconnects.
type ConnectionManager interface {
	// Run accepts "onMessage" and "onFinish" handlers, so the caller can be
	// notified of incoming messages and fatal errors. The caller must also pass a
	// "connect" function that defines how to get a single Connection. Run returns
	// a "send" function, which the caller should use to send outgoing messages out
	// through the current connection.

	Run(
		ctx context.Context,
		onMessage func([]byte),
		onTerminate func(error),
		connect func() (Connection, error),
	) func(io.Reader)
}

type ConnectionManagerCfg struct {
	requeueBackoffDuration time.Duration
}

func NewConnectionManager(cfg *ConnectionManagerCfg) ConnectionManager {
	if cfg == nil {
		cfg = &ConnectionManagerCfg{}
	}
	ret := &connectionManager{}
	if cfg.requeueBackoffDuration == 0 {
		ret.requeueBackoffDuration = time.Second
	} else {
		ret.requeueBackoffDuration = cfg.requeueBackoffDuration
	}
	return ret
}

type connectionManager struct {
	requeueBackoffDuration time.Duration
}

func (c *connectionManager) backoff() {
	time.Sleep(c.requeueBackoffDuration)
}

func (c *connectionManager) Run(
	ctx context.Context,
	onMessage func([]byte),
	onTerminate func(error),
	connect func() (Connection, error),
) func(io.Reader) {
	writes := make(chan io.Reader)
	go func() {
		for {
			conn, err := connect()
			if err != nil {
				onTerminate(err)
				return
			}
			onConnDisconnect := make(chan struct{})
			sendToConn := conn.Run(ctx, onMessage, func() {
				close(onConnDisconnect)
			})
			c.writeLoop(writes, sendToConn, onConnDisconnect)
		}
	}()
	send := func(msg io.Reader) {
		writes <- msg
	}
	return send
}

// writeLoop blocks, sending messages from the `writes` channel via the
// `sendToConn` function, until onConnDisconnect is triggered.
func (c *connectionManager) writeLoop(
	writes chan io.Reader,
	sendToConn func(io.Reader) error,
	onConnDisconnect chan struct{},
) {
	for {
		select {
		case msg := <-writes:
			go func() {
				err := sendToConn(msg)
				if err != nil {
					// Requeue after a backoff
					c.backoff()
					writes <- msg
				}
			}()
		case <-onConnDisconnect:
			return
		}
	}

}

// Connection represents something such as a websocket connection, that can
// transmit messages, receive messages, and die.
type Connection interface {
	// Run accepts "onMessage" and "onDisconnect" handlers, so the caller can be
	// notified of incoming messages and disconnect events. It returns a "send"
	// function, which the caller should use to transmit.
	Run(ctx context.Context, onMessage func([]byte), onDisconnect func()) func(io.Reader) error
}

type connection struct {
	conn     *ws.Conn
	log      *log.Logger
	pongWait time.Duration
}

// Run starts the "readPump" for dispatching incoming messages from the
// websocket connection to the onMessage handler, and returns the "send"
// function which can be used to send messages over this websocket connection.
func (c *connection) Run(ctx context.Context, onMessage func([]byte), onDisconnect func()) func(io.Reader) error {
	go c.readPump(c.conn, onMessage, onDisconnect)
	return c.messageWriter(c.conn)
}

// messageWriter wraps gorilla/websocket's "NextWriter" io.WriteCloser interface.
func (c *connection) messageWriter(conn *ws.Conn) func(r io.Reader) error {
	return func(r io.Reader) error {
		w, err := conn.NextWriter(ws.BinaryMessage)
		if err != nil {
			return err
		}
		conn.NextWriter(ws.BinaryMessage)
		if _, err := io.Copy(w, r); err != nil {
			return err
		}

		if err := w.Close(); err != nil {
			return err
		}
		return nil
	}
}

// readPump uses gorilla/websocket's "conn.ReadMessage" to listen for messages coming over
// on the websocket connection. It also detects errors, and fires the "onDisconnect" handler
// when they occur. It also ensures that "onDisconnect" is fired if too much time has elasped
// since bytes were successfully read from the websocket connection (which shouldn't occur
// frequently if the connection is healthy, because of gorilla/websocket's built in ping/pong).
// The way this works is, if the gorilla/websocket connection's "ReadDeadline" is expired, then
// the "conn.ReadMessage" will return a close error. The ReadDeadline is extended on each
// successful pong.
func (c *connection) readPump(conn *ws.Conn, onMessage func([]byte), onDisconnect func()) {
	err := conn.SetReadDeadline(time.Now().Add(c.pongWait))
	if err != nil {
		c.log.Debug("SetReadDeadline error: ", err)
	}

	conn.SetPongHandler(func(string) error {
		c.log.WithFields(log.Fields{
			"prefix": "websocket.Client.readPump",
		}).Debug("Received pong message")

		err := conn.SetReadDeadline(time.Now().Add(c.pongWait))
		if err != nil {
			c.log.Debug("SetReadDeadline error: ", err)
		}

		return nil
	})

	go func() {
		for {
			_, data, err := conn.ReadMessage()
			if err != nil {
				switch {
				case !ws.IsCloseError(err):
					// read errors do not prevent websocket reconnects in the CLI so we should
					// only display this on debug-level logging
					c.log.WithFields(log.Fields{
						"prefix": "websocket.Client.Close",
					}).Debug("read error: ", err)
				case ws.IsUnexpectedCloseError(err, ws.CloseNormalClosure):
					c.log.WithFields(log.Fields{
						"prefix": "websocket.Client.Close",
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
				"prefix":  "websocket.Client.readPump",
				"message": string(data),
			}).Debug("Incoming message")

			onMessage(data)
		}
	}()
}
