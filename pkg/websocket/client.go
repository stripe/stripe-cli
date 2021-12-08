package websocket

import (
	"context"
	"encoding/json"
	"errors"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	ws "github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"

	"github.com/stripe/stripe-cli/pkg/useragent"
)

//
// Public types
//

// Config contains the optional configuration parameters of a Client.
type Config struct {
	ConnectAttemptWait time.Duration

	Dialer *ws.Dialer

	Log *log.Logger

	// Force use of unencrypted ws:// protocol instead of wss://
	NoWSS bool

	PingPeriod time.Duration

	PongWait time.Duration

	// Interval at which the websocket client should reset the connection
	ReconnectInterval time.Duration

	// Duration to wait before closing connection
	CloseDelayPeriod time.Duration

	WriteWait time.Duration

	EventHandler EventHandler
}

// EventHandler handles an event.
type EventHandler interface {
	ProcessEvent(IncomingMessage)
}

// EventHandlerFunc is an adapter to allow the use of ordinary
// functions as event handlers. If f is a function with the
// appropriate signature, EventHandlerFunc(f) is a
// EventHandler that calls f.
type EventHandlerFunc func(IncomingMessage)

// ProcessEvent calls f(msg).
func (f EventHandlerFunc) ProcessEvent(msg IncomingMessage) {
	f(msg)
}

// Client is the client used to receive webhook requests from Stripe
// and send back webhook responses from the local endpoint to Stripe.
type Client struct {
	// URL the client connects to
	URL string

	// ID sent by the client in the `Websocket-Id` header when connecting
	WebSocketID string

	// Feature that the websocket is specified for
	WebSocketAuthorizedFeature string

	// Optional configuration parameters
	cfg *Config

	conn        *ws.Conn
	done        chan struct{}
	isConnected bool

	NotifyExpired chan struct{}
	notifyClose   chan error
	send          chan *OutgoingMessage
	stopReadPump  chan struct{}
	stopWritePump chan struct{}
	wg            *sync.WaitGroup
}

// Connected returns a channel that's closed when the client has finished
// establishing the websocket connection.
func (c *Client) Connected() <-chan struct{} {
	d := make(chan struct{})

	go func() {
		for !c.isConnected {
			time.Sleep(100 * time.Millisecond)
		}
		close(d)
	}()

	return d
}

// Run starts listening for incoming webhook requests from Stripe.
func (c *Client) Run(ctx context.Context) {
	for {
		c.isConnected = false
		c.cfg.Log.WithFields(log.Fields{
			"prefix": "websocket.client.Run",
		}).Debug("Attempting to connect to Stripe")

		var err error
		err = c.connect(ctx)
		for err != nil {
			c.cfg.Log.WithFields(log.Fields{
				"prefix": "websocket.client.Run",
			}).Debug("Failed to connect to Stripe. Retrying...")

			if err == ErrUnknownID {
				c.cfg.Log.WithFields(log.Fields{
					"prefix": "websocket.client.Run",
				}).Debug("Websocket session is expired.")
				select {
				case <-ctx.Done():
					c.Stop()
					return
				case <-time.After(c.cfg.ConnectAttemptWait):
					c.NotifyExpired <- struct{}{}
					return
				}
			}
			select {
			case <-ctx.Done():
				c.Stop()
			case <-time.After(c.cfg.ConnectAttemptWait):
			}
			err = c.connect(ctx)
		}

		select {
		case <-ctx.Done():
			close(c.send)
			c.Close(ws.CloseNormalClosure, "Connection Done")
			return
		case <-c.done:
			close(c.send)
			close(c.NotifyExpired)
			c.Close(ws.CloseNormalClosure, "Connection Done")
			return
		case <-c.notifyClose:
			c.cfg.Log.WithFields(log.Fields{
				"prefix": "websocket.client.Run",
			}).Debug("Disconnected from Stripe")
			c.Close(ws.CloseGoingAway, "Server closed the connection")
			c.wg.Wait()
		case <-time.After(c.cfg.ReconnectInterval):
			c.cfg.Log.WithFields(log.Fields{
				"prefix": "websocket.Client.Run",
			}).Debug("Resetting the connection")
			c.Close(ws.CloseNormalClosure, "Resetting the connection")
			c.wg.Wait()
		}
	}
}

// Close executes a proper closure handshake then closes the connection
// list of close codes: https://datatracker.ietf.org/doc/html/rfc6455#section-7.4
func (c *Client) Close(closeCode int, text string) {
	close(c.stopReadPump)
	close(c.stopWritePump)
	if c.conn != nil {
		message := ws.FormatCloseMessage(closeCode, text)

		err := c.conn.WriteControl(ws.CloseMessage, message, time.Now().Add(c.cfg.WriteWait))
		if err != nil {
			c.cfg.Log.WithFields(log.Fields{
				"prefix": "websocket.Client.Close",
				"error":  err,
			}).Debug("Error while trying to send close frame")
		}
		time.Sleep(c.cfg.CloseDelayPeriod)
		c.conn.Close()
	}
}

// Stop stops listening for incoming webhook events.
func (c *Client) Stop() {
	close(c.done)
}

// SendMessage sends a message to Stripe through the websocket.
func (c *Client) SendMessage(msg *OutgoingMessage) {
	c.send <- msg
}

func readWSConnectErrorMessage(resp *http.Response) string {
	if resp == nil {
		return ""
	}
	if resp.Body == nil {
		return ""
	}

	se := struct {
		InnerError struct {
			Message string `json:"message"`
		} `json:"error"`
	}{}

	body, err := ioutil.ReadAll(resp.Body)

	if err != nil {
		return ""
	}

	err = json.Unmarshal(body, &se)
	if err != nil {
		return ""
	}

	return se.InnerError.Message
}

var unknownIDMessage = "Unknown WebSocket ID."

// ErrUnknownID can occur when the websocket session is expired or invalid
var ErrUnknownID = errors.New(unknownIDMessage)

// connect makes a single attempt to connect to the websocket URL. It returns
// the success of the attempt.

func (c *Client) connect(ctx context.Context) error {
	header := http.Header{}
	// Disable compression by requiring "identity"
	header.Set("Accept-Encoding", "identity")
	header.Set("User-Agent", useragent.GetEncodedUserAgent())
	header.Set("X-Stripe-Client-User-Agent", useragent.GetEncodedStripeUserAgent())
	header.Set("Websocket-Id", c.WebSocketID)

	url := c.URL
	if c.cfg.NoWSS && strings.HasPrefix(url, "wss") {
		url = "ws" + strings.TrimPrefix(c.URL, "wss")
	}

	url = url + "?websocket_feature=" + c.WebSocketAuthorizedFeature

	c.cfg.Log.WithFields(log.Fields{
		"prefix": "websocket.Client.connect",
		"url":    url,
	}).Debug("Dialing websocket")

	conn, resp, err := c.cfg.Dialer.DialContext(ctx, url, header)
	if err != nil {
		message := readWSConnectErrorMessage(resp)
		c.cfg.Log.WithFields(log.Fields{
			"prefix":  "websocket.Client.connect",
			"error":   err,
			"message": message,
		}).Debug("Websocket connection error")
		if message == unknownIDMessage {
			return ErrUnknownID
		}
		return err
	}

	defer resp.Body.Close()

	c.changeConnection(conn)
	c.isConnected = true

	c.wg = &sync.WaitGroup{}
	c.wg.Add(2)

	go c.readPump()

	go c.writePump()

	c.cfg.Log.WithFields(log.Fields{
		"prefix": "websocket.client.connect",
	}).Debug("Connected!")

	return err
}

// changeConnection takes a new connection and recreates the channels.
func (c *Client) changeConnection(conn *ws.Conn) {
	c.conn = conn
	c.notifyClose = make(chan error)
	c.stopReadPump = make(chan struct{})
	c.stopWritePump = make(chan struct{})
}

// readPump pumps messages from the websocket connection and pushes them into
// RequestHandler's ProcessWebhookRequest.
//
// The application runs readPump in a per-connection goroutine. The application
// ensures that there is at most one reader on a connection by executing all
// reads from this goroutine.
func (c *Client) readPump() {
	defer c.wg.Done()

	err := c.conn.SetReadDeadline(time.Now().Add(c.cfg.PongWait))
	if err != nil {
		c.cfg.Log.Debug("SetReadDeadline error: ", err)
	}

	c.conn.SetPongHandler(func(string) error {
		c.cfg.Log.WithFields(log.Fields{
			"prefix": "websocket.Client.readPump",
		}).Debug("Received pong message")

		err := c.conn.SetReadDeadline(time.Now().Add(c.cfg.PongWait))
		if err != nil {
			c.cfg.Log.Debug("SetReadDeadline error: ", err)
		}

		return nil
	})

	for {
		_, data, err := c.conn.ReadMessage()
		if err != nil {
			select {
			case <-c.stopReadPump:
				c.cfg.Log.WithFields(log.Fields{
					"prefix": "websocket.Client.readPump",
				}).Debug("stopReadPump")
			default:
				switch {
				case !ws.IsCloseError(err):
					// read errors do not prevent websocket reconnects in the CLI so we should
					// only display this on debug-level logging
					c.cfg.Log.WithFields(log.Fields{
						"prefix": "websocket.Client.Close",
					}).Debug("read error: ", err)
				case ws.IsUnexpectedCloseError(err, ws.CloseNormalClosure):
					c.cfg.Log.WithFields(log.Fields{
						"prefix": "websocket.Client.Close",
					}).Error("close error: ", err)
					c.cfg.Log.WithFields(log.Fields{
						"prefix": "stripecli.ADDITIONAL_INFO",
					}).Error("If you run into issues, please re-run with `--log-level debug` and share the output with the Stripe team on GitHub.")
				default:
					c.cfg.Log.Error("other error: ", err)
					c.cfg.Log.WithFields(log.Fields{
						"prefix": "stripecli.ADDITIONAL_INFO",
					}).Error("If you run into issues, please re-run with `--log-level debug` and share the output with the Stripe team on GitHub.")
				}
				c.notifyClose <- err
			}

			return
		}

		c.cfg.Log.WithFields(log.Fields{
			"prefix":  "websocket.Client.readPump",
			"message": string(data),
		}).Debug("Incoming message")

		var msg IncomingMessage
		if err = json.Unmarshal(data, &msg); err != nil {
			c.cfg.Log.Debug("Received malformed message: ", err)

			continue
		}

		go c.cfg.EventHandler.ProcessEvent(msg)
	}
}

// writePump pumps messages to the websocket connection that are queued with
// SendWebhookResponse.
//
// A goroutine running writePump is started for each connection. The
// application ensures that there is at most one writer to a connection by
// executing all writes from this goroutine.
func (c *Client) writePump() {
	ticker := time.NewTicker(c.cfg.PingPeriod)

	defer func() {
		ticker.Stop()
		c.wg.Done()
	}()

	for {
		select {
		case outMsg, ok := <-c.send:
			err := c.conn.SetWriteDeadline(time.Now().Add(c.cfg.WriteWait))
			if err != nil {
				c.cfg.Log.Debug("SetWriteDeadline error: ", err)
			}

			if !ok {
				c.cfg.Log.WithFields(log.Fields{
					"prefix": "websocket.Client.writePump",
				}).Debug("Sending close message")

				err = c.conn.WriteMessage(ws.CloseMessage, ws.FormatCloseMessage(ws.CloseNormalClosure, ""))
				if err != nil {
					c.cfg.Log.Debug("WriteMessage error: ", err)
				}

				return
			}

			c.cfg.Log.WithFields(log.Fields{
				"prefix": "websocket.Client.writePump",
			}).Debug("Sending text message")

			err = c.conn.WriteJSON(outMsg)
			if err != nil {
				if ws.IsUnexpectedCloseError(err, ws.CloseNormalClosure) {
					c.cfg.Log.Error("write error: ", err)
				}
				// Requeue the message to be processed when writePump restarts
				c.send <- outMsg
				c.notifyClose <- err

				return
			}
		case <-ticker.C:
			err := c.conn.SetWriteDeadline(time.Now().Add(c.cfg.WriteWait))
			if err != nil {
				c.cfg.Log.Debug("SetWriteDeadline error: ", err)
			}

			c.cfg.Log.WithFields(log.Fields{
				"prefix": "websocket.Client.writePump",
			}).Debug("Sending ping message")

			if err = c.conn.WriteMessage(ws.PingMessage, nil); err != nil {
				if ws.IsUnexpectedCloseError(err, ws.CloseNormalClosure) {
					c.cfg.Log.Error("write error: ", err)
				}

				// writing to notifyClose during a reset will cause a deadlock
				select {
				case c.notifyClose <- err:
					c.cfg.Log.WithFields(log.Fields{
						"prefix": "websocket.Client.writePump",
					}).Debug("Failed to send ping; closing connection")
				case <-c.stopWritePump:
					c.cfg.Log.WithFields(log.Fields{
						"prefix": "websocket.Client.writePump",
					}).Debug("Failed to send ping; connection is resetting")
				}
				return
			}
		case <-c.stopWritePump:
			c.cfg.Log.WithFields(log.Fields{
				"prefix": "websocket.Client.writePump",
			}).Debug("stopWritePump")

			return
		}
	}
}

//
// Public functions
//

// NewClient returns a new Client.
func NewClient(url string, webSocketID string, websocketAuthorizedFeature string, cfg *Config) *Client {
	if cfg == nil {
		cfg = &Config{}
	}

	if cfg.ConnectAttemptWait == 0 {
		cfg.ConnectAttemptWait = defaultConnectAttemptWait
	}

	if cfg.Dialer == nil {
		cfg.Dialer = newWebSocketDialer(os.Getenv("STRIPE_CLI_UNIX_SOCKET"))
	}

	if cfg.Log == nil {
		cfg.Log = &log.Logger{Out: ioutil.Discard}
	}

	if cfg.PongWait == 0 {
		cfg.PongWait = defaultPongWait
	}

	if cfg.PingPeriod == 0 {
		cfg.PingPeriod = (cfg.PongWait * 2) / 10
	}

	if cfg.ReconnectInterval == 0 {
		cfg.ReconnectInterval = defaultReconnectInterval
	}

	if cfg.CloseDelayPeriod == 0 {
		cfg.CloseDelayPeriod = defaultCloseDelayPeriod
	}

	if cfg.WriteWait == 0 {
		cfg.WriteWait = defaultWriteWait
	}

	if cfg.EventHandler == nil {
		cfg.EventHandler = nullEventHandler
	}

	return &Client{
		URL:                        url,
		WebSocketID:                webSocketID,
		WebSocketAuthorizedFeature: websocketAuthorizedFeature,
		cfg:                        cfg,
		done:                       make(chan struct{}),
		send:                       make(chan *OutgoingMessage),
		NotifyExpired:              make(chan struct{}),
	}
}

//
// Private constants
//

const (
	defaultConnectAttemptWait = 10 * time.Second

	defaultPongWait = 10 * time.Second

	defaultReconnectInterval = 60 * time.Second

	defaultCloseDelayPeriod = 1 * time.Second

	defaultWriteWait = 1 * time.Second
)

//
// Private variables
//

var subprotocols = [...]string{"stripecli-devproxy-v1"}

var nullEventHandler = EventHandlerFunc(func(IncomingMessage) {})

//
// Private functions
//

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
