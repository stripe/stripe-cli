package endpoint

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"time"

	log "github.com/sirupsen/logrus"
)

//
// Public types
//

// Config contains the optional configuration parameters of a Client.
type Config struct {
	HTTPClient *http.Client

	Log *log.Logger

	ResponseHandler ResponseHandler
}

// ResponseHandler handles a response from the endpoint.
type ResponseHandler interface {
	ProcessResponse(string, *http.Response)
}

// ResponseHandlerFunc is an adapter to allow the use of ordinary
// functions as response handlers. If f is a function with the
// appropriate signature, ResponseHandler(f) is a
// ResponseHandler that calls f.
type ResponseHandlerFunc func(string, *http.Response)

// ProcessResponse calls f(webhookID, resp).
func (f ResponseHandlerFunc) ProcessResponse(webhookID string, resp *http.Response) {
	f(webhookID, resp)
}

// Client is the client used to POST webhook requests to the local endpoint.
type Client struct {
	// URL the client sends POST requests to
	URL string

	events map[string]bool

	// Optional configuration parameters
	cfg *Config
}

// SupportsEventType takes an event of a webhook and compares it to the internal
// list of supported events
func (c *Client) SupportsEventType(eventType string) bool {
	// Endpoint supports all events, always return true
	if c.events["*"] || c.events[eventType] {
		return true
	}

	return false
}

// Post sends a message to the local endpoint.
func (c *Client) Post(webhookID string, body string, headers map[string]string) error {
	c.cfg.Log.WithFields(log.Fields{
		"prefix": "endpoint.Client.Post",
	}).Debug("Forwarding event to local endpoint")

	req, err := http.NewRequest(http.MethodPost, c.URL, bytes.NewBuffer([]byte(body)))
	if err != nil {
		return err
	}
	for k, v := range headers {
		req.Header.Add(k, v)
	}

	resp, err := c.cfg.HTTPClient.Do(req)
	if err != nil {
		c.cfg.Log.Errorf("Failed to POST event to local endpoint, error = %v\n", err)
		return err
	}

	c.cfg.ResponseHandler.ProcessResponse(webhookID, resp)

	return nil
}

//
// Public functions
//

// NewClient returns a new Client.
func NewClient(url string, events []string, cfg *Config) *Client {
	if cfg == nil {
		cfg = &Config{}
	}
	if cfg.Log == nil {
		cfg.Log = &log.Logger{Out: ioutil.Discard}
	}
	if cfg.HTTPClient == nil {
		cfg.HTTPClient = &http.Client{
			Timeout: defaultTimeout,
		}
	}
	if cfg.ResponseHandler == nil {
		cfg.ResponseHandler = nullResponseHandler
	}

	return &Client{
		URL:    url,
		events: convertToMap(events),
		cfg:    cfg,
	}
}

//
// Private constants
//

const (
	defaultTimeout = 30 * time.Second
)

//
// Private variables
//

var nullResponseHandler = ResponseHandlerFunc(func(string, *http.Response) {})

//
// Private functions
//

func convertToMap(events []string) map[string]bool {
	eventsMap := make(map[string]bool)
	for _, event := range events {
		eventsMap[event] = true
	}

	return eventsMap
}
