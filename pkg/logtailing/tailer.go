package logtailing

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/stripe/stripe-cli/pkg/stripeauth"
	"github.com/stripe/stripe-cli/pkg/websocket"
)

const requestLogsWebSocketFeature = "request_logs"

// LogFilters contains all of the potential user-provided filters for log tailing
type LogFilters struct {
	FilterAccount        []string `json:"filter_account,omitempty"`
	FilterIPAddress      []string `json:"filter_ip_address,omitempty"`
	FilterHTTPMethod     []string `json:"filter_http_method,omitempty"`
	FilterRequestPath    []string `json:"filter_request_path,omitempty"`
	FilterRequestStatus  []string `json:"filter_request_status,omitempty"`
	FilterSource         []string `json:"filter_source,omitempty"`
	FilterStatusCode     []string `json:"filter_status_code,omitempty"`
	FilterStatusCodeType []string `json:"filter_status_code_type,omitempty"`
}

// Config provides the configuration of a log tailer
type Config struct {
	APIBaseURL string

	// DeviceName is the name of the device sent to Stripe to help identify the device
	DeviceName string

	// Filters for API request logs
	Filters *LogFilters

	// Key is the API key used to authenticate with Stripe
	Key string

	// Info, error, etc. logger. Unrelated to API request logs.
	Log *log.Logger

	// Force use of unencrypted ws:// protocol instead of wss://
	NoWSS bool

	// OutCh is the channel to send logs and statuses to for processing in other packages
	OutCh chan websocket.IElement
}

// Tailer is the main interface for running the log tailing session
type Tailer struct {
	cfg *Config

	stripeAuthClient *stripeauth.Client
	webSocketClient  *websocket.Client

	interruptCh chan os.Signal
}

// EventPayload is the mapping for fields in event payloads from request log tailing
type EventPayload struct {
	CreatedAt int           `json:"created_at"`
	Livemode  bool          `json:"livemode"`
	Method    string        `json:"method"`
	RequestID string        `json:"request_id"`
	Status    int           `json:"status"`
	URL       string        `json:"url"`
	Error     RedactedError `json:"error"`
}

// RedactedError is the mapping for fields in error from an EventPayload
type RedactedError struct {
	Type        string `json:"type"`
	Charge      string `json:"charge"`
	Code        string `json:"code"`
	DeclineCode string `json:"decline_code"`
	Message     string `json:"message"`
	Param       string `json:"param"`
}

// New creates a new Tailer
func New(cfg *Config) *Tailer {
	if cfg.Log == nil {
		cfg.Log = &log.Logger{Out: ioutil.Discard}
	}

	return &Tailer{
		cfg: cfg,
		stripeAuthClient: stripeauth.NewClient(cfg.Key, &stripeauth.Config{
			Log:        cfg.Log,
			APIBaseURL: cfg.APIBaseURL,
		}),
		interruptCh: make(chan os.Signal, 1),
	}
}

const maxConnectAttempts = 3

// Run sets the websocket connection
func (t *Tailer) Run(ctx context.Context) error {
	defer close(t.cfg.OutCh)

	warned := false
	nAttempts := 0

	t.cfg.OutCh <- websocket.StateElement{
		State: websocket.Loading,
	}

	for nAttempts < maxConnectAttempts {
		session, err := t.createSession(ctx)

		if err != nil {
			t.cfg.OutCh <- websocket.ErrorElement{
				Error: fmt.Errorf("Error while authenticating with Stripe: %v", err),
			}
			return err
		}

		if session.DisplayConnectFilterWarning && !warned {
			t.cfg.OutCh <- websocket.WarningElement{
				Warning: "you specified the 'account' filter for Connect accounts but are not a Connect user, so the filter will not be applied.",
			}
			// Only display this warning once
			warned = true
		}

		t.webSocketClient = websocket.NewClient(
			session.WebSocketURL,
			session.WebSocketID,
			session.WebSocketAuthorizedFeature,
			&websocket.Config{
				EventHandler:      websocket.EventHandlerFunc(t.processRequestLogEvent),
				Log:               t.cfg.Log,
				NoWSS:             t.cfg.NoWSS,
				ReconnectInterval: time.Duration(session.ReconnectDelay) * time.Second,
			},
		)

		go func() {
			<-t.webSocketClient.Connected()
			nAttempts = 0
			t.cfg.OutCh <- websocket.StateElement{
				State: websocket.Ready,
			}
		}()

		go t.webSocketClient.Run(ctx)
		nAttempts++

		select {
		case <-ctx.Done():
			t.cfg.OutCh <- &websocket.StateElement{
				State: websocket.Done,
			}
			return nil
		case <-t.webSocketClient.NotifyExpired:
			if nAttempts < maxConnectAttempts {
				t.cfg.OutCh <- &websocket.StateElement{
					State: websocket.Reconnecting,
				}
			} else {
				err := fmt.Errorf("Session expired. Terminating after %d failed attempts to reauthorize", nAttempts)
				t.cfg.OutCh <- websocket.ErrorElement{
					Error: err,
				}
				return err
			}
		}
	}

	if t.webSocketClient != nil {
		t.webSocketClient.Stop()
	}

	log.WithFields(log.Fields{
		"prefix": "logtailing.Tailer.Run",
	}).Debug("Bye!")

	return nil
}

func (t *Tailer) createSession(ctx context.Context) (*stripeauth.StripeCLISession, error) {
	var session *stripeauth.StripeCLISession

	var err error

	exitCh := make(chan struct{})

	filters, err := jsonifyFilters(t.cfg.Filters)
	if err != nil {
		return nil, fmt.Errorf("Error while converting log filters to JSON encoding: %v", err)
	}

	go func() {
		// Try to authorize at least 5 times before failing. Sometimes we have random
		// transient errors that we just need to retry for.
		for i := 0; i <= 5; i++ {
			session, err = t.stripeAuthClient.Authorize(ctx, t.cfg.DeviceName, requestLogsWebSocketFeature, &filters, nil)

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

func (t *Tailer) processRequestLogEvent(msg websocket.IncomingMessage) {
	if msg.RequestLogEvent == nil {
		t.cfg.Log.Debug("WebSocket specified for request logs received non-request-logs event")
		return
	}

	requestLogEvent := msg.RequestLogEvent

	t.cfg.Log.WithFields(log.Fields{
		"prefix":     "logtailing.Tailer.processRequestLogEvent",
		"webhook_id": requestLogEvent.RequestLogID,
	}).Debugf("Processing request log event")

	var payload EventPayload
	if err := json.Unmarshal([]byte(requestLogEvent.EventPayload), &payload); err != nil {
		t.cfg.Log.Debug("Received malformed payload: ", err)
	}

	// at this point the message is valid so we can acknowledge it
	ackMessage := websocket.NewEventAck(requestLogEvent.RequestLogID, "")
	t.webSocketClient.SendMessage(ackMessage)

	// Don't show stripecli/sessions logs since they're generated by the CLI
	if payload.URL == "/v1/stripecli/sessions" {
		t.cfg.Log.Debug("Filtering out /v1/stripecli/sessions from logs")
		return
	}

	t.cfg.OutCh <- websocket.DataElement{
		Data:      payload,
		Marshaled: requestLogEvent.EventPayload,
	}
}

func jsonifyFilters(logFilters *LogFilters) (string, error) {
	bytes, err := json.Marshal(logFilters)
	if err != nil {
		return "", err
	}

	jsonStr := string(bytes)

	return jsonStr, nil
}
