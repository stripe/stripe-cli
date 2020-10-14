package websocket

import (
	"context"
	"io"
	"time"
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

// ConnectionManagerCfg is the config passed to create a new ConnectionManager
type ConnectionManagerCfg struct {
	requeueBackoffDuration time.Duration
}

// NewConnectionManager returns a new ConnectionManager
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
// transmit messages, receive mes
