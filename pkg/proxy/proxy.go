package proxy

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"reflect"
	"strconv"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/stripe/stripe-cli/pkg/ansi"
	"github.com/stripe/stripe-cli/pkg/config"
	"github.com/stripe/stripe-cli/pkg/requests"
	"github.com/stripe/stripe-cli/pkg/stripe"
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

	// Headers to forward to endpoints
	ForwardHeaders []string

	// Connect indicates whether the endpoint should receive normal (when false) or Connect (when true) events.
	Connect bool

	// EventTypes is the list of event types that should be sent to the endpoint.
	EventTypes []string

	// Status is whether or not the endpoint is enabled.
	Status string
}

// EndpointResponse describes the response to a Stripe event from an endpoint
type EndpointResponse struct {
	Event *StripeEvent
	Resp  *http.Response
}

// FailedToReadResponseError describes a failure to read the response from an endpoint
type FailedToReadResponseError struct {
	Err error
}

func (f FailedToReadResponseError) Error() string {
	return f.Err.Error()
}

// Config provides the configuration of a Proxy
type Config struct {
	// DeviceName is the name of the device sent to Stripe to help identify the device
	DeviceName string
	// Key is the API key used to authenticate with Stripe
	Key string
	// URL to which requests are sent
	APIBaseURL string

	// URL to which events are forwarded to
	ForwardURL string
	// Headers to inject when forwarding events
	ForwardHeaders []string
	// URL to which Connect events are forwarded to
	ForwardConnectURL string
	// Headers to inject when forwarding Connect events
	ForwardConnectHeaders []string
	// UseConfiguredWebhooks loads webhooks config from user's account
	UseConfiguredWebhooks bool

	// EndpointsRoutes is a mapping of local webhook endpoint urls to the events they consume
	EndpointRoutes []EndpointRoute
	// List of events to listen and proxy
	Events []string

	// WebSocketFeature is the feature specified for the websocket connection
	WebSocketFeature string
	// Indicates whether to print full JSON objects to stdout
	PrintJSON bool

	// Specifies the format to print to stdout.
	Format string

	// Indicates whether to filter events formatted with the default or latest API version
	UseLatestAPIVersion bool
	// Indicates whether to skip certificate verification when forwarding webhooks to HTTPS endpoints
	SkipVerify bool
	// The logger used to log messages to stdin/err
	Log *log.Logger
	// Force use of unencrypted ws:// protocol instead of wss://
	NoWSS bool

	// OutCh is the channel to send logs and statuses to for processing in other packages
	OutCh chan websocket.IElement
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

const maxConnectAttempts = 3

// Run sets the websocket connection and starts the Goroutines to forward
// incoming events to the local endpoint.
func (p *Proxy) Run(ctx context.Context) error {
	defer close(p.cfg.OutCh)

	p.cfg.OutCh <- websocket.StateElement{
		State: websocket.Loading,
	}

	nAttempts := 0

	for nAttempts < maxConnectAttempts {
		session, err := p.createSession(ctx)

		if err != nil {
			p.cfg.OutCh <- websocket.ErrorElement{
				Error: fmt.Errorf("Error while authenticating with Stripe: %v", err),
			}
			return err
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

		go func() {
			<-p.webSocketClient.Connected()
			nAttempts = 0

			displayedAPIVersion := ""
			if p.cfg.UseLatestAPIVersion && session.LatestVersion != "" {
				displayedAPIVersion = "You are using Stripe API Version [" + session.LatestVersion + "]. "
			} else if !p.cfg.UseLatestAPIVersion && session.DefaultVersion != "" {
				displayedAPIVersion = "You are using Stripe API Version [" + session.DefaultVersion + "]. "
			}

			p.cfg.OutCh <- websocket.StateElement{
				State: websocket.Ready,
				Data:  []string{displayedAPIVersion, session.Secret},
			}
		}()

		go p.webSocketClient.Run(ctx)
		nAttempts++

		select {
		case <-ctx.Done():
			p.cfg.OutCh <- &websocket.StateElement{
				State: websocket.Done,
			}
			return nil
		case <-p.webSocketClient.NotifyExpired:
			if nAttempts < maxConnectAttempts {
				p.cfg.OutCh <- &websocket.StateElement{
					State: websocket.Reconnecting,
				}
			} else {
				err := fmt.Errorf("Session expired. Terminating after %d failed attempts to reauthorize", nAttempts)
				p.cfg.OutCh <- websocket.ErrorElement{
					Error: err,
				}
				return err
			}
		}
	}

	if p.webSocketClient != nil {
		p.webSocketClient.Stop()
	}

	log.WithFields(log.Fields{
		"prefix": "proxy.Proxy.Run",
	}).Debug("Bye!")

	return nil
}

// GetSessionSecret creates a session and returns the webhook signing secret.
func GetSessionSecret(ctx context.Context, deviceName, key, baseURL string) (string, error) {
	p, err := Init(ctx, &Config{
		DeviceName:       deviceName,
		Key:              key,
		APIBaseURL:       baseURL,
		EndpointRoutes:   make([]EndpointRoute, 0),
		WebSocketFeature: "webhooks",
	})
	if err != nil {
		log.WithFields(log.Fields{
			"prefix": "proxy.Proxy.GetSessionSecret",
		}).Debug(err)
		return "", err
	}

	session, err := p.createSession(ctx)
	if err != nil {
		log.WithFields(log.Fields{
			"prefix": "proxy.Proxy.GetSessionSecret",
		}).Debug(fmt.Sprintf("Error while authenticating with Stripe: %v", err))
		return "", err
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
			devURLMap := stripeauth.DeviceURLMap{
				ForwardURL:        p.cfg.ForwardURL,
				ForwardConnectURL: p.cfg.ForwardConnectURL,
			}

			session, err = p.stripeAuthClient.Authorize(ctx, p.cfg.DeviceName, p.cfg.WebSocketFeature, nil, &devURLMap)

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
			"api_version": getAPIVersionString(msg.Endpoint.APIVersion),
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

// This function outputs the event payload in the format specified.
// Currently only supports JSON.
func (p *Proxy) formatOutput(format string, eventPayload string) string {
	var event map[string]interface{}
	err := json.Unmarshal([]byte(eventPayload), &event)
	if err != nil {
		p.cfg.Log.Debug("Received malformed event from Stripe, ignoring")
		return fmt.Sprint(err)
	}
	switch strings.ToUpper(format) {
	// The distinction between this and PrintJSON is that this output is stripped of all pretty format.
	case outputFormatJSON:
		outputJSON, _ := json.Marshal(event)
		return fmt.Sprintln(ansi.ColorizeJSON(string(outputJSON), false, os.Stdout))
	default:
		return fmt.Sprintf("Unrecognized output format %s\n" + format)
	}
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

	var evt StripeEvent

	err := json.Unmarshal([]byte(webhookEvent.EventPayload), &evt)
	if err != nil {
		p.cfg.Log.Debug("Received malformed event from Stripe, ignoring")
		return
	}

	req, err := ExtractRequestData(evt.RequestData)

	if err != nil {
		p.cfg.Log.Debug("Received malformed event from Stripe, ignoring")
		return
	}

	evt.Request = req

	p.cfg.Log.WithFields(log.Fields{
		"prefix":                  "proxy.Proxy.processWebhookEvent",
		"webhook_id":              webhookEvent.WebhookID,
		"webhook_conversation_id": webhookEvent.WebhookConversationID,
		"event_id":                evt.ID,
		"event_type":              evt.Type,
		"api_version":             getAPIVersionString(msg.Endpoint.APIVersion),
	}).Trace("Webhook event trace")

	// at this point the message is valid so we can acknowledge it
	ackMessage := websocket.NewEventAck(webhookEvent.WebhookID, webhookEvent.WebhookConversationID)
	p.webSocketClient.SendMessage(ackMessage)

	if p.filterWebhookEvent(webhookEvent) {
		return
	}

	evtCtx := eventContext{
		webhookID:             webhookEvent.WebhookID,
		webhookConversationID: webhookEvent.WebhookConversationID,
		event:                 &evt,
	}

	if p.events["*"] || p.events[evt.Type] {
		p.cfg.OutCh <- websocket.DataElement{
			Data:      evt,
			Marshaled: p.formatOutput(outputFormatJSON, webhookEvent.EventPayload),
		}

		for _, endpoint := range p.endpointClients {
			if endpoint.SupportsEventType(evt.IsConnect(), evt.Type) {
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

func (p *Proxy) processEndpointResponse(evtCtx eventContext, forwardURL string, resp *http.Response) {
	buf, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		p.cfg.OutCh <- websocket.ErrorElement{
			Error: FailedToReadResponseError{Err: err},
		}
		return
	}

	body := truncate(string(buf), maxBodySize, true)

	p.cfg.OutCh <- websocket.DataElement{
		Data: EndpointResponse{
			Event: evtCtx.event,
			Resp:  resp,
		},
	}

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
		msg := websocket.NewWebhookResponse(
			evtCtx.webhookID,
			evtCtx.webhookConversationID,
			forwardURL,
			resp.StatusCode,
			body,
			headers,
		)
		p.webSocketClient.SendMessage(msg)
	}
}

//
// Public functions
//

// Init initializes a new Proxy
func Init(ctx context.Context, cfg *Config) (*Proxy, error) {
	if cfg.Log == nil {
		cfg.Log = &log.Logger{Out: ioutil.Discard}
	}

	// validate forward-urls args
	if cfg.UseConfiguredWebhooks && len(cfg.ForwardURL) > 0 {
		if strings.HasPrefix(cfg.ForwardURL, "/") {
			return nil, errors.New("forward_to cannot be a relative path when loading webhook endpoints from the API")
		}
		if strings.HasPrefix(cfg.ForwardConnectURL, "/") {
			return nil, errors.New("forward_connect_to cannot be a relative path when loading webhook endpoints from the API")
		}
	} else if cfg.UseConfiguredWebhooks && len(cfg.ForwardURL) == 0 {
		return nil, errors.New("load_from_webhooks_api requires a location to forward to with forward_to")
	}

	// if no events are passed, listen for all events
	if len(cfg.Events) == 0 {
		cfg.Events = []string{"*"}
	} else {
		for _, event := range cfg.Events {
			if _, found := validEvents[event]; !found {
				cfg.Log.Infof("Warning: You're attempting to listen for \"%s\", which isn't a valid event\n", event)
			}
		}
	}

	// build from --forward-to urls if --forward-connect-to was not provided
	if len(cfg.ForwardConnectURL) == 0 {
		cfg.ForwardConnectURL = cfg.ForwardURL
	}
	if len(cfg.ForwardConnectHeaders) == 0 {
		cfg.ForwardConnectHeaders = cfg.ForwardHeaders
	}

	// build endpoint routes
	var endpointRoutes []EndpointRoute
	if cfg.UseConfiguredWebhooks {
		// build from user's API config
		endpoints := getEndpointsFromAPI(ctx, cfg.Key, cfg.APIBaseURL)
		if len(endpoints.Data) == 0 {
			return nil, errors.New("You have not defined any webhook endpoints on your account. Go to the Stripe Dashboard to add some: https://dashboard.stripe.com/test/webhooks")
		}
		var err error
		endpointRoutes, err = buildEndpointRoutes(endpoints, parseURL(cfg.ForwardURL), parseURL(cfg.ForwardConnectURL), cfg.ForwardHeaders, cfg.ForwardConnectHeaders)
		if err != nil {
			return nil, err
		}
	} else if len(cfg.ForwardURL) > 0 {
		// non-connect endpoints
		endpointRoutes = append(endpointRoutes, EndpointRoute{
			URL:            parseURL(cfg.ForwardURL),
			ForwardHeaders: cfg.ForwardHeaders,
			Connect:        false,
			EventTypes:     cfg.Events,
		})

		// connect endpoints
		endpointRoutes = append(endpointRoutes, EndpointRoute{
			URL:            parseURL(cfg.ForwardConnectURL),
			ForwardHeaders: cfg.ForwardConnectHeaders,
			Connect:        true,
			EventTypes:     cfg.Events,
		})
	}

	p := &Proxy{
		cfg: cfg,
		stripeAuthClient: stripeauth.NewClient(cfg.Key, &stripeauth.Config{
			Log:        cfg.Log,
			APIBaseURL: cfg.APIBaseURL,
		}),
		events: convertToMap(cfg.Events),
	}

	for _, route := range endpointRoutes {
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
						TLSClientConfig: &tls.Config{InsecureSkipVerify: cfg.SkipVerify},
					},
				},
				Log:             p.cfg.Log,
				ResponseHandler: EndpointResponseHandlerFunc(p.processEndpointResponse),
				OutCh:           p.cfg.OutCh,
			},
		))
	}

	return p, nil
}

// ExtractRequestData takes an interface with request data from a Stripe event payload
// and properly parses it into a StripeRequest struct before returning it
func ExtractRequestData(data interface{}) (StripeRequest, error) {
	var req StripeRequest

	reqDataValue := reflect.ValueOf(data)
	// here we check the type of the RequestData as it could be either a string or a map
	// this depends on which API version is in use when listening to events
	// versions including and prior to 2017-05-25 present the request field as a string
	switch reqDataValue.Kind() {
	case reflect.String:
		req = StripeRequest{
			ID:             data.(string),
			IdempotencyKey: "",
		}
	case reflect.Map:
		reqDataMap := reqDataValue.Interface().(map[string]interface{})

		var id = ""
		if rawID, ok := reqDataMap["id"]; ok && rawID != nil {
			id = rawID.(string)
		}

		var idempotencyKey = ""
		if rawKey, ok := reqDataMap["idempotency_key"]; ok && rawKey != nil {
			idempotencyKey = rawKey.(string)
		}

		req = StripeRequest{
			ID:             id,
			IdempotencyKey: idempotencyKey,
		}
	default:
		return StripeRequest{}, errors.New("Received malformed event from Stripe")
	}
	return req, nil
}

//
// Private types
//

type eventContext struct {
	webhookID             string
	webhookConversationID string
	event                 *StripeEvent
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

const outputFormatJSON = "JSON"

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

// TODO: move to some helper somewhere
// parseURL parses the potentially incomplete URL provided in the configuration
// and returns a full URL
func parseURL(url string) string {
	_, err := strconv.Atoi(url)
	if err == nil {
		// If the input is just a number, assume it's a port number
		url = "localhost:" + url
	}

	if strings.HasPrefix(url, "/") {
		// If the input starts with a /, assume it's a relative path
		url = "localhost" + url
	}

	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		// Add the protocol if it's not already there
		url = "http://" + url
	}

	return url
}

func getEndpointsFromAPI(ctx context.Context, secretKey, apiBaseURL string) requests.WebhookEndpointList {
	if apiBaseURL == "" {
		apiBaseURL = stripe.DefaultAPIBaseURL
	}

	return requests.WebhookEndpointsList(ctx, apiBaseURL, stripe.APIVersion, secretKey, &config.Profile{})
}

func buildEndpointRoutes(endpoints requests.WebhookEndpointList, forwardURL, forwardConnectURL string, forwardHeaders []string, forwardConnectHeaders []string) ([]EndpointRoute, error) {
	endpointRoutes := make([]EndpointRoute, 0)

	for _, endpoint := range endpoints.Data {
		// Ensure the endpoint is enabled.
		if endpoint.Status == "disabled" {
			continue
		}

		u, err := url.Parse(endpoint.URL)
		// Silently skip over invalid paths
		if err == nil {
			// Since webhooks in the dashboard may have a more generic url, only extract
			// the path. We'll use this with `localhost` or with the `--forward-to` flag
			if endpoint.Application == "" {
				url, err := buildForwardURL(forwardURL, u)
				if err != nil {
					return nil, err
				}
				endpointRoutes = append(endpointRoutes, EndpointRoute{
					URL:            url,
					ForwardHeaders: forwardHeaders,
					Connect:        false,
					EventTypes:     endpoint.EnabledEvents,
					Status:         endpoint.Status,
				})
			} else {
				url, err := buildForwardURL(forwardConnectURL, u)
				if err != nil {
					return nil, err
				}
				endpointRoutes = append(endpointRoutes, EndpointRoute{
					URL:            url,
					ForwardHeaders: forwardConnectHeaders,
					Connect:        true,
					EventTypes:     endpoint.EnabledEvents,
				})
			}
		}
	}

	return endpointRoutes, nil
}

func buildForwardURL(forwardURL string, destination *url.URL) (string, error) {
	f, err := url.Parse(forwardURL)
	if err != nil {
		return "", fmt.Errorf("Provided forward url cannot be parsed: %s", forwardURL)
	}

	return fmt.Sprintf(
		"%s://%s%s%s",
		f.Scheme,
		f.Host,
		strings.TrimSuffix(f.Path, "/"), // avoids having a double "//"
		destination.Path,
	), nil
}

func getAPIVersionString(str *string) string {
	var APIVersion string

	if str == nil {
		APIVersion = "null"
	} else {
		APIVersion = *str
	}

	return APIVersion
}
