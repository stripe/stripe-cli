package websocket

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/stripe/stripe-cli/pkg/ansi"
	"github.com/stripe/stripe-cli/pkg/stripeauth"
)

// ConnectionManager provides an interface for interacting with a *series*
// of Connections; that establishes a new connection when the previous connection
// disconnects.
type ConnectionManagerI interface {
	Run(
		ctx context.Context,
		onMessage func([]byte),
		onTerminate func(error),
		connect func() (Connection, error),
	) func(io.Reader)
}

type ConnectionManager struct {
	cfg                    *ConnectionManagerCfg
	requeueBackoffDuration time.Duration
	Logger                 *log.Logger
	stripeAuthClient       *stripeauth.Client
}

// ConnectionManagerCfg is the config passed to create a new ConnectionManager
type ConnectionManagerCfg struct {
	requeueBackoffDuration time.Duration
	NoWSS                  bool
	Logger                 *log.Logger
	PongWait               time.Duration
	WriteWait              time.Duration
	stripeAuthClient       *stripeauth.Client
	DeviceName             string
	WebSocketFeature       string
	Key                    string
	APIBaseURL             string
	Filters                string
}

// NewConnectionManager returns a new ConnectionManager
func NewConnectionManager(cfg *ConnectionManagerCfg) *ConnectionManager {
	if cfg == nil {
		cfg = &ConnectionManagerCfg{}
	}
	cm := &ConnectionManager{
		cfg: cfg,
	}

	if cfg.requeueBackoffDuration == 0 {
		cm.requeueBackoffDuration = time.Second
	} else {
		cm.requeueBackoffDuration = cfg.requeueBackoffDuration
	}

	cm.stripeAuthClient = stripeauth.NewClient(cfg.Key, &stripeauth.Config{
		Log:        cfg.Logger,
		APIBaseURL: cfg.APIBaseURL,
	})

	cm.Logger = cfg.Logger

	return cm
}

func (c *ConnectionManager) backoff() {
	time.Sleep(c.requeueBackoffDuration)
}

// Run accepts "onMessage" and "onTerminate" handlers, so the caller can be
// notified of incoming messages and fatal errors. The caller must also pass a
// "connect" function that defines how to get a single Connection. Run returns
// a "send" function, which the caller should use to send outgoing messages out
// through the current connection.
func (c *ConnectionManager) Run(
	ctx context.Context,
	onMessage func([]byte),
	onTerminate func(error),
) func(io.Reader) {
	writes := make(chan io.Reader)
	go func() {
		for {
			conn, err := c.connect(ctx)
			if err != nil {
				onTerminate(err)
				return
			}
			onConnDisconnect := make(chan struct{})
			sendToConn := conn.Run(ctx, onMessage, func() {
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
func (c *ConnectionManager) writeLoop(
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

func (c *ConnectionManager) connect(ctx context.Context) (Connection, error) {
	session, err := c.CreateSession(ctx)
	if err != nil {
		return nil, err
	}

	sessionRefresher := c.sessionRefresher()
	for range []int{1, 2, 3} {
		conn, resp, err := ConnectWebsocket(ctx, session, ConnectWebsocketConfig{
			NoWSS:     c.cfg.NoWSS,
			Logger:    c.Logger,
			PongWait:  c.cfg.PongWait,
			WriteWait: c.cfg.WriteWait,
		})
		defer resp.Body.Close()
		if err != nil {
			session, err = sessionRefresher(ctx, err, resp)
			if err != nil {
				return nil, err
			}
		} else {
			return conn, nil
		}
	}
	return nil, err
}

func (c *ConnectionManager) CreateSession(ctx context.Context) (*stripeauth.StripeCLISession, error) {
	var session *stripeauth.StripeCLISession
	var err error
	exitCh := make(chan struct{})

	go func() {
		// Try to authorize at least 5 times before failing. Sometimes we have random
		// transient errors that we just need to retry for.
		for i := 0; i <= 5; i++ {
			session, err = c.stripeAuthClient.Authorize(ctx, c.cfg.DeviceName, c.cfg.WebSocketFeature, &c.cfg.Filters)

			if err == nil {
				exitCh <- struct{}{}
				return
			}

			select {
			case <-ctx.Done():
				exitCh <- struct{}{}
				return
			case <-time.After(1 * time.Second):
			}
		}

		exitCh <- struct{}{}
	}()
	<-exitCh

	return session, err
}

func (c *ConnectionManager) sessionRefresher() func(context.Context, error, *http.Response) (*stripeauth.StripeCLISession, error) {
	warned := false
	return func(ctx context.Context, err error, resp *http.Response) (*stripeauth.StripeCLISession, error) {
		if err != nil {
			if !isExpirationErr(err, resp) {
				return nil, nil
			}
		}
		session, err := c.CreateSession(ctx)
		if err != nil {
			return nil, err
		}

		if session.DisplayConnectFilterWarning && !warned {
			color := ansi.Color(os.Stdout)
			fmt.Printf("%s you specified the 'account' filter for Connect accounts but are not a Connect user, so the filter will not be applied.\n", color.Yellow("Warning"))
			// Only display this warning once
			warned = true
		}

		return session, nil
	}
}

func isExpirationErr(err error, resp *http.Response) bool {
	if err == nil || resp == nil {
		return false
	}
	message := readWSConnectErrorMessage(resp)
	return message == unknownIDMessage
}

var unknownIDMessage string = "Unknown WebSocket ID."

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
