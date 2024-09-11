package proxy

import (
	"crypto/tls"
	"encoding/json"
	"io"
	"net/http"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/stripe/stripe-cli/pkg/websocket"
)

// WebhookEventProcessorConfig defines the external inputs that infuence the
// behavior of a WebhookEventProcessor.
type WebhookEventProcessorConfig struct {
	// The logger used to log messages to stdin/err
	Log *log.Logger

	// List of events to listen and proxy
	Events []string

	// List of thin events to listen and proxy
	ThinEvents []string

	// OutCh is the channel to send logs and statuses to for processing in other packages
	OutCh chan websocket.IElement

	// Indicates whether to filter events formatted with the default or latest API version
	UseLatestAPIVersion bool

	// Indicates whether to skip certificate verification when forwarding webhooks to HTTPS endpoints
	SkipVerify bool

	// Override default timeout
	Timeout int64
}

// WebhookEventProcessor encapsulates logic around processing and forwarding
// webhook events.
type WebhookEventProcessor struct {
	cfg *WebhookEventProcessorConfig

	// Events is the supported event types for the command
	events          map[string]bool
	thinEvents      map[string]bool
	endpointClients []*EndpointClient
	sendMessage     func(*websocket.OutgoingMessage)
}

// NewWebhookEventProcessor constructs a WebhookEventProcessor from the provided
// websocket delivery function, route table, and config.
func NewWebhookEventProcessor(sendMessage func(*websocket.OutgoingMessage), routes []EndpointRoute, cfg *WebhookEventProcessorConfig) *WebhookEventProcessor {
	p := &WebhookEventProcessor{
		cfg:         cfg,
		events:      convertToMap(cfg.Events),
		sendMessage: sendMessage,
		thinEvents:  convertToMap(cfg.ThinEvents),
	}

	for _, route := range routes {
		// append to endpointClients
		p.endpointClients = append(p.endpointClients, NewEndpointClient(
			route.URL,
			route.ForwardHeaders,
			route.Connect,
			route.EventTypes,
			route.IsEventDestination,
			&EndpointConfig{
				HTTPClient: &http.Client{
					CheckRedirect: func(req *http.Request, via []*http.Request) error {
						return http.ErrUseLastResponse
					},
					Timeout: time.Duration(cfg.Timeout) * time.Second,
					Transport: &http.Transport{
						TLSClientConfig: &tls.Config{InsecureSkipVerify: cfg.SkipVerify},
					},
				},
				Log:             cfg.Log,
				ResponseHandler: EndpointResponseHandlerFunc(p.processEndpointResponse),
				OutCh:           cfg.OutCh,
			},
		))
	}

	return p
}

// ProcessEvent processes webhook events, notifying listeners via the configured
// OutCh, sending acknowledgements with the configured websocket sender, and
// forwarding events to configured endpoints.
//
// ProcessEvent implements the websocket.EndpointResponseHandler interface.
func (p *WebhookEventProcessor) ProcessEvent(msg websocket.IncomingMessage) {
	switch {
	case msg.WebhookEvent != nil:
		p.processEvent(msg.WebhookEvent)
	case msg.StripeV2Event != nil:
		p.processV2Event(msg.StripeV2Event)
	default:
		p.cfg.Log.Debug("WebSocket specified for Webhooks received non-webhook event")
		return
	}
}

func (p *WebhookEventProcessor) processEvent(webhookEvent *websocket.WebhookEvent) {
	p.cfg.Log.WithFields(log.Fields{
		"prefix":                   "proxy.WebhookEventProcessor.ProcessEvent",
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
		"prefix":                  "proxy.WebhookEventProcessor.ProcessEvent",
		"webhook_id":              webhookEvent.WebhookID,
		"webhook_conversation_id": webhookEvent.WebhookConversationID,
		"event_id":                evt.ID,
		"event_type":              evt.Type,
		"api_version":             getAPIVersionString(webhookEvent.Endpoint.APIVersion),
	}).Trace("Webhook event trace")

	// at this point the message is valid so we can acknowledge it
	ackMessage := websocket.NewEventAck(evt.ID, webhookEvent.WebhookConversationID, webhookEvent.WebhookID)
	p.sendMessage(ackMessage)

	if p.filterWebhookEvent(webhookEvent) {
		return
	}

	evtCtx := eventContext{
		webhookID:             webhookEvent.WebhookID,
		webhookConversationID: webhookEvent.WebhookConversationID,
		event:                 &evt,
		requestBody:           webhookEvent.EventPayload,
		requestHeaders:        webhookEvent.HTTPHeaders,
	}

	if p.events["*"] || p.events[evt.Type] {
		p.cfg.OutCh <- websocket.DataElement{
			Data:      evt,
			Marshaled: formatOutput(outputFormatJSON, webhookEvent.EventPayload),
		}

		for _, endpoint := range p.endpointClients {
			if endpoint.SupportsEventType(evt.IsConnect(), evt.Type) && !endpoint.isEventDestination {
				// TODO: handle errors returned by endpointClients
				go endpoint.Post(evtCtx)
			}
		}
	}
}

func (p *WebhookEventProcessor) processV2Event(v2Event *websocket.StripeV2Event) {
	var evt V2EventPayload

	err := json.Unmarshal([]byte(v2Event.Payload), &evt)
	if err != nil {
		p.cfg.Log.Debug("Received malformed event from Stripe, ignoring")
		return
	}

	p.cfg.Log.WithFields(log.Fields{
		"prefix":     "proxy.WebhookEventProcessor.ProcessV2Event",
		"event_id":   evt.ID,
		"event_type": evt.Type,
	}).Debugf("Processing webhook event")

	// ack the event
	p.sendMessage(websocket.NewEventAck(evt.ID, "", v2Event.EventDestinationID))

	// skip further event processing if the event type is not enabled
	if !p.thinEvents[evt.Type] && !p.thinEvents["*"] {
		return
	}

	// notify consumers
	p.cfg.OutCh <- websocket.DataElement{
		Data: evt,
	}

	evtCtx := eventContext{
		webhookID:             v2Event.EventDestinationID,
		webhookConversationID: "",
		v2Event:               &evt,
		requestBody:           v2Event.Payload,
		requestHeaders:        v2Event.HTTPHeaders,
	}

	for _, endpoint := range p.endpointClients {
		if endpoint.isEventDestination && endpoint.SupportsContext(evt.Context) {
			go endpoint.PostV2(evtCtx)
		}
	}
}

func (p *WebhookEventProcessor) filterWebhookEvent(msg *websocket.WebhookEvent) bool {
	if msg.Endpoint.APIVersion != nil && !p.cfg.UseLatestAPIVersion {
		p.cfg.Log.WithFields(log.Fields{
			"prefix":      "proxy.WebhookEventProcessor.filterWebhookEvent",
			"api_version": getAPIVersionString(msg.Endpoint.APIVersion),
		}).Debugf("Received event with non-default API version, ignoring")

		return true
	}

	if msg.Endpoint.APIVersion == nil && p.cfg.UseLatestAPIVersion {
		p.cfg.Log.WithFields(log.Fields{
			"prefix": "proxy.WebhookEventProcessor.filterWebhookEvent",
		}).Debugf("Received event with default API version, ignoring")

		return true
	}

	return false
}

func (p *WebhookEventProcessor) processEndpointResponse(evtCtx eventContext, forwardURL string, resp *http.Response) {
	buf, err := io.ReadAll(resp.Body)
	if err != nil {
		p.cfg.OutCh <- websocket.ErrorElement{
			Error: FailedToReadResponseError{Err: err},
		}
		return
	}

	body := truncate(string(buf), maxBodySize, true)
	var eventID string
	if evtCtx.event != nil {
		eventID = evtCtx.event.ID
		p.cfg.OutCh <- websocket.DataElement{
			Data: EndpointResponse{
				Event: evtCtx.event,
				Resp:  resp,
			},
		}
	} else if evtCtx.v2Event != nil {
		eventID = evtCtx.v2Event.ID
		p.cfg.OutCh <- websocket.DataElement{
			Data: EndpointResponse{
				V2Event: evtCtx.v2Event,
				Resp:    resp,
			},
		}
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

	msg := websocket.NewWebhookResponse(
		evtCtx.webhookID,
		evtCtx.webhookConversationID,
		forwardURL,
		resp.StatusCode,
		body,
		headers,
		evtCtx.requestBody,
		evtCtx.requestHeaders,
		eventID,
	)
	p.sendMessage(msg)
}
