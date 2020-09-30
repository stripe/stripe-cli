package proxy

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/stripe/stripe-cli/pkg/ansi"
	"github.com/stripe/stripe-cli/pkg/stripeauth"
	"github.com/stripe/stripe-cli/pkg/websocket"
)

const timeLayout = "2006-01-02 15:04:05"

//
// Public types
//

// EndpointRoute describes a local endpoint's routing configuration.
type EndpointRoute struct {
	// URL is the endpoint's URL.
	URL string

	// Headers to forward to endpoints
	ForwardHeaders []string

	// Connect indicates whether the endpoint should receive normal (when false) or Connect (when true) events.
	Connect bool

	// EventTypes is the list of event types that should be sent to the endpoint.
	EventTypes []string
}

// Config provides the configuration of a Proxy
type Config struct {
	// DeviceName is the name of the device sent to Stripe to help identify the device
	DeviceName string

	// Key is the API key used to authenticate with Stripe
	Key string

	// EndpointsMap is a mapping of local webhook endpoint urls to the events they consume
	EndpointRoutes []EndpointRoute

	APIBaseURL string

	// WebSocketFeature is the feature specified for the websocket connection
	WebSocketFeature string

	// Indicates whether to print full JSON objects to stdout
	PrintJSON bool

	// Indicates whether to filter events formatted with the default or latest API version
	UseLatestAPIVersion bool

	// Indicates whether to skip certificate verification when forwarding webhooks to HTTPS endpoints
	SkipVerify bool

	Log *log.Logger

	// Force use of unencrypted ws:// protocol instead of wss://
	NoWSS bool
}

// A Proxy opens a websocket connection with Stripe, listens for incoming
// webhook events, forwards them to the local endpoint and sends the response
// back to Stripe.
type Proxy struct {
	cfg *Config

	endpointClients  []*EndpointClient
	stripeAuthClient *stripeauth.Client
	webSocketClient  *websocket.Client

	// Events is the supported event types for the command
	events map[string]bool
}

func withSIGTERMCancel(ctx context.Context, onCancel func()) context.Context {
	// Create a context that will be canceled when Ctrl+C is pressed
	ctx, cancel := context.WithCancel(ctx)

	interruptCh := make(chan os.Signal, 1)
	signal.Notify(interruptCh, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-interruptCh
		onCancel()
		cancel()
	}()
	return ctx
}

const maxConnectAttempts = 3

type sessionRefresher = func(context.Context, error) (*stripeauth.StripeCLISession, error)
type EventSource interface {
	Run(context.Context, sessionRefresher) (chan websocket.IncomingMessage, error)
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

var unknownIDMessage string = "Unknown WebSocket ID."

// ErrUnknownID can occur when the websocket session is expired or invalid
var ErrUnknownID error = errors.New(unknownIDMessage)

func isExpirationErr(err error, resp *http.Response) bool {
	if err == nil || resp == nil {
		return false
	}
	message := readWSConnectErrorMessage(resp)
	return message == unknownIDMessage
}

func (p *Proxy) refreshSession(ctx context.Context, err error, resp *http.Response) (*stripeauth.StripeCLISession, error) {
	if err != nil {
		if !isExpirationErr(err, resp) {
			return nil, nil
		}
	}
	session, err := p.createSession(ctx)
	if err != nil {
		return nil, err
	}

	return session, nil
}

// Run sets the websocket connection and starts the Goroutines to forward
// incoming events to the local endpoint.
func (p *Proxy) Run(ctx context.Context) error {
	connectionManager := websocket.NewConnectionManager(&websocket.ConnectionManagerCfg{})
	session, err := p.createSession(ctx)
	dialer := websocket.NewWebSocketDialer(os.Getenv("STRIPE_CLI_UNIX_SOCKET"))
	if err != nil {
		return err
	}
	connect := func() (websocket.Connection, error) {
		for _ = range []int{1, 2, 3} {
			//TODO: fix this PongWait
			conn, resp, err := websocket.ConnectWebsocket(ctx, session, websocket.ConnectWebsocketConfig{
				Dialer:    dialer,
				NoWSS:     p.cfg.NoWSS,
				Logger:    p.cfg.Log,
				PongWait:  10 * time.Second,
				WriteWait: 5 * time.Second,
			})
			if err != nil {
				session, err = p.refreshSession(ctx, err, resp)
				if err != nil {
					return nil, err
				}
			} else {
				return conn, nil
			}
		}
		return nil, err
	}

	onMessage := func(b []byte) {
		var msg websocket.IncomingMessage
		if err := json.Unmarshal(b, &msg); err != nil {
			p.cfg.Log.Debug("Received malformed message: ", err)

			return
		}
		p.processWebhookEvent(msg)
	}
	onTerminate := func(err error) {
		p.cfg.Log.Fatal("Terminating...", err)
	}
	send := connectionManager.Run(ctx, onMessage, onTerminate, connect)
	responses := p.listenEndpoints()

	for {
		response, ok := <-responses
		if !ok {
			return nil
		}
		msg, err := p.processEndpointResponse(response.eventContext, response.forwardURL, response.resp)
		if err == nil {
			b, err := json.Marshal(msg)
			if err != nil {
				send(bytes.NewReader(b))
			}
		}
	}
}

// GetSessionSecret creates a session and returns the webhook signing secret.
func (p *Proxy) GetSessionSecret(ctx context.Context) (string, error) {
	session, err := p.createSession(ctx)
	if err != nil {
		p.cfg.Log.Fatalf("Error while authenticating with Stripe: %v", err)
	}
	return session.Secret, nil
}

func (p *Proxy) createSession(ctx context.Context) (*stripeauth.StripeCLISession, error) {
	var session *stripeauth.StripeCLISession

	var err error

	exitCh := make(chan struct{})

	go func() {
		// Try to authorize at least 5 times before failing. Sometimes we have random
		// transient errors that we just need to retry for.
		for i := 0; i <= 5; i++ {
			session, err = p.stripeAuthClient.Authorize(ctx, p.cfg.DeviceName, p.cfg.WebSocketFeature, nil)

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

func (p *Proxy) filterWebhookEvent(msg *websocket.WebhookEvent) bool {
	if msg.Endpoint.APIVersion != nil && !p.cfg.UseLatestAPIVersion {
		p.cfg.Log.WithFields(log.Fields{
			"prefix":      "proxy.Proxy.filterWebhookEvent",
			"api_version": msg.Endpoint.APIVersion,
		}).Debugf("Received event with non-default API version, ignoring")

		return true
	}

	if msg.Endpoint.APIVersion == nil && p.cfg.UseLatestAPIVersion {
		p.cfg.Log.WithFields(log.Fields{
			"prefix": "proxy.Proxy.filterWebhookEvent",
		}).Debugf("Received event with default API version, ignoring")

		return true
	}

	return false
}

func (p *Proxy) processWebhookEvent(msg websocket.IncomingMessage) {
	if msg.WebhookEvent == nil {
		p.cfg.Log.Debug("WebSocket specified for Webhooks received non-webhook event")
		return
	}

	webhookEvent := msg.WebhookEvent

	p.cfg.Log.WithFields(log.Fields{
		"prefix":                   "proxy.Proxy.processWebhookEvent",
		"webhook_id":               webhookEvent.WebhookID,
		"webhook_converesation_id": webhookEvent.WebhookConversationID,
	}).Debugf("Processing webhook event")

	if p.filterWebhookEvent(webhookEvent) {
		return
	}

	var evt stripeEvent

	err := json.Unmarshal([]byte(webhookEvent.EventPayload), &evt)
	if err != nil {
		p.cfg.Log.Debug("Received malformed event from Stripe, ignoring")
		return
	}

	evtCtx := eventContext{
		webhookID:             webhookEvent.WebhookID,
		webhookConversationID: webhookEvent.WebhookConversationID,
		event:                 &evt,
	}

	if p.events["*"] || p.events[evt.Type] {
		if p.cfg.PrintJSON {
			fmt.Println(webhookEvent.EventPayload)
		} else {
			maybeConnect := ""
			if evt.isConnect() {
				maybeConnect = "connect "
			}

			localTime := time.Now().Format(timeLayout)

			color := ansi.Color(os.Stdout)
			outputStr := fmt.Sprintf("%s   --> %s%s [%s]",
				color.Faint(localTime),
				maybeConnect,
				ansi.Linkify(ansi.Bold(evt.Type), evt.urlForEventType(), p.cfg.Log.Out),
				ansi.Linkify(evt.ID, evt.urlForEventID(), p.cfg.Log.Out),
			)
			fmt.Println(outputStr)
		}

		for _, endpoint := range p.endpointClients {
			if endpoint.SupportsEventType(evt.isConnect(), evt.Type) {
				// TODO: handle errors returned by endpointClients
				go endpoint.Post(
					evtCtx,
					webhookEvent.EventPayload,
					webhookEvent.HTTPHeaders,
				)
			}
		}
	}
}

func (p *Proxy) processEndpointResponse(evtCtx eventContext, forwardURL string, resp *http.Response) (*websocket.OutgoingMessage, error) {
	localTime := time.Now().Format(timeLayout)

	color := ansi.Color(os.Stdout)
	outputStr := fmt.Sprintf("%s  <--  [%d] %s %s [%s]",
		color.Faint(localTime),
		ansi.ColorizeStatus(resp.StatusCode),
		resp.Request.Method,
		resp.Request.URL,
		ansi.Linkify(evtCtx.event.ID, evtCtx.event.urlForEventID(), p.cfg.Log.Out),
	)
	fmt.Println(outputStr)

	buf, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		errStr := fmt.Sprintf("%s            [%s] Failed to read response from endpoint, error = %v\n",
			color.Faint(localTime),
			color.Red("ERROR"),
			err,
		)
		log.Errorf(errStr)

		return nil, err
	}

	body := truncate(string(buf), maxBodySize, true)

	idx := 0
	headers := make(map[string]string)

	for k, v := range resp.Header {
		headers[truncate(k, maxHeaderKeySize, false)] = truncate(v[0], maxHeaderValueSize, true)
		idx++

		if idx > maxNumHeaders {
			break
		}
	}

	return websocket.NewWebhookResponse(
		evtCtx.webhookID,
		evtCtx.webhookConversationID,
		forwardURL,
		resp.StatusCode,
		body,
		headers,
	), nil
}

//
// Public functions
//

// New creates a new Proxy
func New(cfg *Config, events []string) *Proxy {
	if cfg.Log == nil {
		cfg.Log = &log.Logger{Out: ioutil.Discard}
	}

	p := &Proxy{
		cfg: cfg,
		stripeAuthClient: stripeauth.NewClient(cfg.Key, &stripeauth.Config{
			Log:        cfg.Log,
			APIBaseURL: cfg.APIBaseURL,
		}),
	}

	if len(events) > 0 {
		p.events = convertToMap(events)
	}

	return p
}

type EndpointResponse = struct {
	eventContext eventContext
	forwardURL   string
	resp         *http.Response
}

func (p *Proxy) listenEndpoints() chan EndpointResponse {
	responses := make(chan EndpointResponse)
	for _, route := range p.cfg.EndpointRoutes {
		// append to endpointClients
		p.endpointClients = append(p.endpointClients, NewEndpointClient(
			route.URL,
			route.ForwardHeaders,
			route.Connect,
			route.EventTypes,
			&EndpointConfig{
				HTTPClient: &http.Client{
					CheckRedirect: func(req *http.Request, via []*http.Request) error {
						return http.ErrUseLastResponse
					},
					Timeout: defaultTimeout,
					Transport: &http.Transport{
						TLSClientConfig: &tls.Config{InsecureSkipVerify: p.cfg.SkipVerify},
					},
				},
				Log: p.cfg.Log,
				ResponseHandler: EndpointResponseHandlerFunc(func(eventContext eventContext, forwardURL string, resp *http.Response) {
					responses <- EndpointResponse{
						eventContext: eventContext,
						forwardURL:   forwardURL,
						resp:         resp,
					}
				}),
			},
		))
	}
	return responses
}

//
// Private types
//

type eventContext struct {
	webhookID             string
	webhookConversationID string
	event                 *stripeEvent
}

//
// Private constants
//

const (
	maxBodySize        = 5000
	maxNumHeaders      = 20
	maxHeaderKeySize   = 50
	maxHeaderValueSize = 200
)

//
// Private functions
//

// truncate will truncate str to be less than or equal to maxByteLength bytes.
// It will respect UTF8 and truncate the string at a code point boundary.
// If ellipsis is true, we'll append "..." to the truncated string if the string
// was in fact truncated, and if there's enough room. Note that the
// full string returned will always be <= maxByteLength bytes long, even with ellipsis.
func truncate(str string, maxByteLength int, ellipsis bool) string {
	if len(str) <= maxByteLength {
		return str
	}

	bytes := []byte(str)

	if ellipsis && maxByteLength > 3 {
		maxByteLength -= 3
	} else {
		ellipsis = false
	}

	for maxByteLength > 0 && maxByteLength < len(bytes) && isUTF8ContinuationByte(bytes[maxByteLength]) {
		maxByteLength--
	}

	result := string(bytes[0:maxByteLength])
	if ellipsis {
		result += "..."
	}

	return result
}

func isUTF8ContinuationByte(b byte) bool {
	return (b & 0xC0) == 0x80
}
