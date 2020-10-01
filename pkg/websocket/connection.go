package websocket

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"sync"
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
			err := sendToConn(msg)
			if err != nil {
				// Requeue after a backoff
				go func() {
					c.backoff()
					writes <- msg
				}()
			}
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
	conn       *ws.Conn
	log        *log.Logger
	pongWait   time.Duration
	writeWait  time.Duration
	pingPeriod time.Duration
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

//func (c *connection) sendData(onDisconnect func()) func(data []byte) error {
//	return c.sendMsg(msg{
//		isPing: false,
//		data:   data,
//	})
//}

// sendMsg wraps gorilla/websocket's "NextWriter" io.WriteCloser interface.
func (c *connection) messageSender(onDisconnect func()) func(msg) error {
	sendIt := func(m msg) error {
		err := c.conn.SetWriteDeadline(time.Now().Add(c.writeWait))
		if err != nil {
			return err
		}

		if m.isPing {
			c.log.WithFields(log.Fields{
				"prefix": "connection.sendMsg",
			}).Debug("Sending ping message")
			if err = c.conn.WriteMessage(ws.PingMessage, m.data); err != nil {
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
	defer func() {
		ticker.Stop()
	}()

	go func() {
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
				if err = c.conn.WriteMessage(ws.PingMessage, nil); err != nil {
					if ws.IsUnexpectedCloseError(err, ws.CloseNormalClosure) {
						c.log.Error("write error: ", err)
					}
					onDisconnect()
					return
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
