package websocket

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"time"

	ws "github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"

	"github.com/stripe/stripe-cli/pkg/stripeauth"
	"github.com/stripe/stripe-cli/pkg/useragent"
)

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
	GetReconnectInterval func(session *stripeauth.StripeCLISession) time.Duration

	WriteWait time.Duration
}

// Client is the client used to receive webhook requests from Stripe
// and send back webhook responses from the local endpoint to Stripe.
type Client struct {
	// Optional configuration parameters
	cfg *Config

	conn *ws.Conn
	done chan struct{}

	notifyClose   chan error
	send          chan *OutgoingMessage
	stopWritePump chan struct{}
}

type sessionRefresher = func(context.Context, error, *http.Response) (*stripeauth.StripeCLISession, error)

func cancellableWait(ctx context.Context, duration time.Duration, onCancel func()) {
	select {
	case <-ctx.Done():
		onCancel()
		return
	case <-time.After(duration):
		return
	}
}

func (c *Client) connectRefreshLoop(ctx context.Context, initialSession *stripeauth.StripeCLISession, refreshSession sessionRefresher) (*ws.Conn, error) {
	c.cfg.Log.WithFields(log.Fields{
		"prefix": "websocket.client.Run",
	}).Debug("Attempting to connect to Stripe")

	conn, errResp, err := c.connect(ctx, initialSession)
	for err != nil {
		c.cfg.Log.WithFields(log.Fields{
			"prefix": "websocket.client.Run",
		}).Debug("Failed to connect to Stripe. Retrying...")

		session, err := refreshSession(ctx, err, errResp)
		if err != nil {
			return nil, err
		}
		cancellableWait(ctx, c.cfg.ConnectAttemptWait, func() { c.Stop() })
		conn, errResp, err = c.connect(ctx, session)
	}
	return conn, err
}

type IncomingMessageEvent struct {
	Msg IncomingMessage
	Err error
}

// Run starts listening for incoming webhook requests from Stripe.
func (c *Client) Run(ctx context.Context, refreshSession sessionRefresher) chan IncomingMessageEvent {
	messages := make(chan IncomingMessageEvent, 100)
	sendError := func(err error) chan IncomingMessageEvent {
		messages <- IncomingMessageEvent{Err: err}
		close(messages)
		return messages
	}

	session, err := refreshSession(ctx, nil, nil)
	if err != nil {
		return sendError(err)
	}
	go func() {
		for {
			conn, err := c.connectRefreshLoop(ctx, session, refreshSession)
			if err != nil {
				sendError(err)
			}
			c.changeConnection(conn)

			go func() {
				connMessages := c.readPump(conn)
				for msg := range connMessages {
					messages <- IncomingMessageEvent{Msg: msg}
				}
			}()

			go c.writePump()

			c.cfg.Log.WithFields(log.Fields{
				"prefix": "websocket.client.connect",
			}).Debug("Connected!")

			select {
			case <-ctx.Done():
				close(c.send)
				close(c.stopWritePump)

			case <-c.done:
				close(c.send)
				close(c.stopWritePump)

			case <-c.notifyClose:
				c.cfg.Log.WithFields(log.Fields{
					"prefix": "websocket.client.Run",
				}).Debug("Disconnected from Stripe")
				close(c.stopWritePump)
			case <-time.After(c.cfg.GetReconnectInterval(session)):
				c.cfg.Log.WithFields(log.Fields{
					"prefix": "websocket.Client.Run",
				}).Debug("Resetting the connection")
				close(c.stopWritePump)

				if c.conn != nil {
					c.conn.Close() // #nosec G104
				}
			}
		}
	}()
	return messages
}

// Stop stops listening for incoming webhook events.
func (c *Client) Stop() {
	close(c.done)
}

// SendMessage sends a message to Stripe through the websocket.
func (c *Client) SendMessage(msg *OutgoingMessage) {
	c.send <- msg
}

func wsHeader(baseURL string, webSocketID string, webSocketAuthorizedFeature string, noWSS bool) (string, http.Header) {
	header := http.Header{}
	// Disable compression by requiring "identity"
	header.Set("Accept-Encoding", "identity")
	header.Set("User-Agent", useragent.GetEncodedUserAgent())
	header.Set("X-Stripe-Client-User-Agent", useragent.GetEncodedStripeUserAgent())
	header.Set("Websocket-Id", webSocketID)

	url := baseURL
	if noWSS && strings.HasPrefix(baseURL, "wss") {
		url = "ws" + strings.TrimPrefix(baseURL, "wss")
	}

	url = url + "?websocket_feature=" + webSocketAuthorizedFeature
	return url, header
}

type ConnectWebsocketConfig2 struct {
	Dialer    *ws.Dialer
	NoWSS     bool
	Logger    *log.Logger
	PongWait  time.Duration
	WriteWait time.Duration
}

func ConnectWebsocket2(ctx context.Context, session *stripeauth.StripeCLISession, cfg ConnectWebsocketConfig2) (Connection, *http.Response, error) {
	url, header := wsHeader(session.WebSocketURL, session.WebSocketID, session.WebSocketAuthorizedFeature, cfg.NoWSS)

	cfg.Logger.WithFields(log.Fields{
		"prefix": "websocket.connection.ConnectWebsocket",
		"url":    url,
	}).Debug("Dialing websocket")

	conn, resp, err := cfg.Dialer.DialContext(ctx, url, header)
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

func (c *Client) connect(ctx context.Context, session *stripeauth.StripeCLISession) (*ws.Conn, *http.Response, error) {
	url, header := wsHeader(session.WebSocketURL, session.WebSocketID, session.WebSocketAuthorizedFeature, c.cfg.NoWSS)

	c.cfg.Log.WithFields(log.Fields{
		"prefix": "websocket.Client.connect",
		"url":    url,
	}).Debug("Dialing websocket")
	return c.cfg.Dialer.DialContext(ctx, url, header)
}

// changeConnection takes a new connection and recreates the channels.
func (c *Client) changeConnection(conn *ws.Conn) {
	c.conn = conn
	c.notifyClose = make(chan error)
	c.stopWritePump = make(chan struct{})
}

// readPump reads from a connection and writes IncomingMessages to the channel
// it returns
func (c *Client) readPump(conn *ws.Conn) chan IncomingMessage {
	err := conn.SetReadDeadline(time.Now().Add(c.cfg.PongWait))
	if err != nil {
		c.cfg.Log.Debug("SetReadDeadline error: ", err)
	}

	conn.SetPongHandler(func(string) error {
		c.cfg.Log.WithFields(log.Fields{
			"prefix": "websocket.Client.readPump",
		}).Debug("Received pong message")

		err := conn.SetReadDeadline(time.Now().Add(c.cfg.PongWait))
		if err != nil {
			c.cfg.Log.Debug("SetReadDeadline error: ", err)
		}

		return nil
	})
	messages := make(chan IncomingMessage)
	go func() {
		defer func() {
			close(messages)
		}()
		for {
			_, data, err := conn.ReadMessage()
			if err != nil {
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

			messages <- msg
		}
	}()
	return messages
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
	}()

	for {
		select {
		case whResp, ok := <-c.send:
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

			err = c.conn.WriteJSON(whResp)
			if err != nil {
				if ws.IsUnexpectedCloseError(err, ws.CloseNormalClosure) {
					c.cfg.Log.Error("write error: ", err)
				}
				// Requeue the message to be processed when writePump restarts
				c.send <- whResp
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
				c.notifyClose <- err

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
func NewClient(cfg *Config) *Client {
	if cfg == nil {
		cfg = &Config{}
	}

	if cfg.ConnectAttemptWait == 0 {
		cfg.ConnectAttemptWait = defaultConnectAttemptWait
	}

	if cfg.Dialer == nil {
		cfg.Dialer = NewWebSocketDialer(os.Getenv("STRIPE_CLI_UNIX_SOCKET"))
	}

	if cfg.Log == nil {
		cfg.Log = &log.Logger{Out: ioutil.Discard}
	}

	if cfg.PongWait == 0 {
		cfg.PongWait = defaultPongWait
	}

	if cfg.PingPeriod == 0 {
		cfg.PingPeriod = (cfg.PongWait * 9) / 10
	}

	if cfg.GetReconnectInterval == nil {
		cfg.GetReconnectInterval = defaultGetReconnectInterval
	}

	if cfg.WriteWait == 0 {
		cfg.WriteWait = defaultWriteWait
	}

	return &Client{
		cfg:  cfg,
		done: make(chan struct{}),
		send: make(chan *OutgoingMessage),
	}
}

//
// Private constants
//

const (
	defaultConnectAttemptWait = 10 * time.Second
	defaultPongWait           = 10 * time.Second
	defaultWriteWait          = 10 * time.Second
)

//
// Private variables
//

var subprotocols = [...]string{"stripecli-devproxy-v1"}

//
// Private functions
//

func defaultGetReconnectInterval(session *stripeauth.StripeCLISession) time.Duration {
	return 60 * time.Second
}
