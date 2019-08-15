package proxy

import (
	"crypto/tls"
	"encoding/json"
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

//
// Public types
//

// EndpointRoute describes a local endpoint's routing configuration.
type EndpointRoute struct {
	// URL is the endpoint's URL.
	URL string

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

	interruptCh chan os.Signal
}

// Run sets the websocket connection and starts the Goroutines to forward
// incoming events to the local endpoint.
func (p *Proxy) Run() error {
	s := ansi.StartSpinner("Getting ready...", p.cfg.Log.Out)

	// Intercept Ctrl+c so we can do some clean up
	signal.Notify(p.interruptCh, os.Interrupt, syscall.SIGTERM)

	session, err := p.stripeAuthClient.Authorize(p.cfg.DeviceName, p.cfg.WebSocketFeature, nil)
	if err != nil {
		// TODO: better error handling / retries
		p.cfg.Log.Fatalf("Error while authenticating with Stripe: %v", err)
	}

	p.webSocketClient = websocket.NewClient(
		session.WebSocketURL,
		session.WebSocketID,
		session.WebSocketAuthorizedFeature,
		&websocket.Config{
			Log:               p.cfg.Log,
			NoWSS:             p.cfg.NoWSS,
			ReconnectInterval: time.Duration(session.ReconnectDelay) * time.Second,
			EventHandler:      websocket.EventHandlerFunc(p.processWebhookEvent),
		},
	)
	go p.webSocketClient.Run()

	color := ansi.Color(p.cfg.Log.Out)
	ansi.StopSpinner(s, fmt.Sprintf("Ready! Your webhook signing secret is %s (^C to quit)", color.Bold(session.Secret)), p.cfg.Log.Out)

	// Block until Ctrl+C is received
	<-p.interruptCh

	log.WithFields(log.Fields{
		"prefix": "proxy.Proxy.Run",
	}).Debug("Ctrl+C received, cleaning up...")

	if p.webSocketClient != nil {
		p.webSocketClient.Stop()
	}

	log.WithFields(log.Fields{
		"prefix": "proxy.Proxy.Run",
	}).Debug("Bye!")

	return nil
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
		p.cfg.Log.Warn("WebSocket specified for Webhooks received non-webhook event")
		return
	}

	webhookEvent := msg.WebhookEvent

	p.cfg.Log.WithFields(log.Fields{
		"prefix":     "proxy.Proxy.processWebhookEvent",
		"webhook_id": webhookEvent.WebhookID,
	}).Debugf("Processing webhook event")

	if p.filterWebhookEvent(webhookEvent) {
		return
	}

	var evt stripeEvent
	err := json.Unmarshal([]byte(webhookEvent.EventPayload), &evt)
	if err != nil {
		p.cfg.Log.Warn("Received malformed event from Stripe, ignoring")
		return
	}

	maybeConnect := ""
	if evt.isConnect() {
		maybeConnect = "Connect "
	}
	p.cfg.Log.Infof(
		"Received %sevent: %s [type: %s]",
		maybeConnect,
		ansi.Linkify(evt.ID, evt.urlForEventID(), p.cfg.Log.Out),
		ansi.Linkify(evt.Type, evt.urlForEventType(), p.cfg.Log.Out),
	)

	if p.cfg.PrintJSON {
		fmt.Println(webhookEvent.EventPayload)
	}

	for _, endpoint := range p.endpointClients {
		if endpoint.SupportsEventType(evt.isConnect(), evt.Type) {
			go endpoint.Post(webhookEvent.WebhookID, webhookEvent.EventPayload, webhookEvent.HTTPHeaders)
		}
	}
	// TODO: handle errors returned by endpointClients
	// TODO: if no forwarding, prepare a dummy response directly in the CLI
	// to pass back to Stripe
}

func (p *Proxy) processEndpointResponse(webhookID string, resp *http.Response) {
	p.cfg.Log.Infof("Got response from local endpoint, status=%d", resp.StatusCode)

	buf, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Errorf("Failed to read response from endpoint, error = %v\n", err)
		return
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

	if p.webSocketClient != nil {
		msg := websocket.NewWebhookResponse(webhookID, resp.StatusCode, body, headers)
		p.webSocketClient.SendMessage(msg)
	}
}

//
// Public functions
//

// New creates a new Proxy
func New(cfg *Config) *Proxy {
	if cfg.Log == nil {
		cfg.Log = &log.Logger{Out: ioutil.Discard}
	}
	p := &Proxy{
		cfg: cfg,
		stripeAuthClient: stripeauth.NewClient(cfg.Key, &stripeauth.Config{
			Log:        cfg.Log,
			APIBaseURL: cfg.APIBaseURL,
		}),
		interruptCh: make(chan os.Signal, 1),
	}

	for _, route := range cfg.EndpointRoutes {
		// append to endpointClients
		p.endpointClients = append(p.endpointClients, NewEndpointClient(
			route.URL,
			route.Connect,
			route.EventTypes,
			&EndpointConfig{
				HTTPClient: &http.Client{
					Timeout: defaultTimeout,
					Transport: &http.Transport{
						TLSClientConfig: &tls.Config{InsecureSkipVerify: cfg.SkipVerify},
					},
				},
				Log:             p.cfg.Log,
				ResponseHandler: EndpointResponseHandlerFunc(p.processEndpointResponse),
			},
		))
	}

	return p
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
