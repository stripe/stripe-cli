package proxy

import (
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
	"github.com/stripe/stripe-cli/pkg/endpoint"
	"github.com/stripe/stripe-cli/pkg/stripeauth"
	"github.com/stripe/stripe-cli/pkg/websocket"
)

//
// Public types
//

// Config provides the cfguration of a Proxy
type Config struct {
	// DeviceName is the name of the device sent to Stripe to help identify the device
	DeviceName string

	// Key is the API key used to authenticate with Stripe
	Key string

	// EndpointsMap is a mapping of local webhook endpoint urls to the events they consume
	EndpointsMap map[string][]string

	APIBaseURL string

	// WebSocketURL is the websocket URL used to receive incoming events
	WebSocketURL string

	// Indicates whether to print full JSON objects to stdout
	PrintJSON bool

	// Indicates whether to filter events formatted with the default or latest API version
	UseLatestAPIVersion bool

	Log *log.Logger

	// Force use of unencrypted ws:// protocol instead of wss://
	NoWSS bool
}

// A Proxy opens a websocket connection with Stripe, listens for incoming
// webhook events, forwards them to the local endpoint and sends the response
// back to Stripe.
type Proxy struct {
	cfg *Config

	endpointClients  []*endpoint.Client
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

	session, err := p.authorize()
	if err != nil {
		// TODO: better error handling / retries
		p.cfg.Log.Fatalf("Error while authenticating with Stripe: %v", err)
	}

	p.webSocketClient = websocket.NewClient(
		session.WebSocketURL,
		session.WebSocketID,
		&websocket.Config{
			Log:                 p.cfg.Log,
			NoWSS:               p.cfg.NoWSS,
			ReconnectInterval:   time.Duration(session.ReconnectDelay) * time.Second,
			WebhookEventHandler: websocket.WebhookEventHandlerFunc(p.processWebhookEvent),
		},
	)
	go p.webSocketClient.Run()

	color := ansi.Color(p.cfg.Log.Out)
	ansi.StopSpinner(s, fmt.Sprintf("Ready! Your webhook signing secret is %s (^C to quit)", color.Bold(session.Secret)), p.cfg.Log.Out)

	for {
		select {
		case <-p.interruptCh:
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
	}
}

func (p *Proxy) authorize() (*stripeauth.StripeCLISession, error) {
	if len(p.cfg.WebSocketURL) > 0 {
		p.cfg.Log.Info("Skipping authentication step because --ws-url was passed")
		session := &stripeauth.StripeCLISession{
			WebSocketID:  "",
			WebSocketURL: p.cfg.WebSocketURL,
		}
		return session, nil
	}

	return p.stripeAuthClient.Authorize(p.cfg.DeviceName)
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

func (p *Proxy) processWebhookEvent(msg *websocket.WebhookEvent) {
	p.cfg.Log.WithFields(log.Fields{
		"prefix":     "proxy.Proxy.processWebhookEvent",
		"webhook_id": msg.WebhookID,
	}).Debugf("Processing webhook event")

	if p.filterWebhookEvent(msg) {
		return
	}

	var evt stripeEvent
	err := json.Unmarshal([]byte(msg.EventPayload), &evt)
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
		fmt.Println(msg.EventPayload)
	}

	for _, endpoint := range p.endpointClients {
		if endpoint.SupportsEventType(evt.Type) {
			go endpoint.Post(msg.WebhookID, msg.EventPayload, msg.HTTPHeaders)
		}
	}
	// TODO: handle errors returned by endpointClients
	// TODO: if no forwarding, prepare a dummy response directly in the CLI
	// to pass back to Stripe
}

func (p *Proxy) processEndpointResponse(webhookID string, resp *http.Response) {
	p.cfg.Log.Infof("Got response from local endpoint, status=%d", resp.StatusCode)

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Errorf("Failed to read response from endpoint, error = %v\n", err)
		return
	}

	headers := make(map[string]string)
	for k, v := range resp.Header {
		headers[k] = v[0]
	}

	if p.webSocketClient != nil {
		msg := websocket.NewWebhookResponse(webhookID, resp.StatusCode, string(body), headers)
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

	for url, events := range cfg.EndpointsMap {
		// append to endpointClients
		p.endpointClients = append(p.endpointClients, endpoint.NewClient(
			url,
			events,
			&endpoint.Config{
				Log:             p.cfg.Log,
				ResponseHandler: endpoint.ResponseHandlerFunc(p.processEndpointResponse),
			},
		))
	}

	return p
}
