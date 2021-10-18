package stripeauth

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"

	log "github.com/sirupsen/logrus"

	"github.com/stripe/stripe-cli/pkg/stripe"
)

const stripeCLISessionPath = "/v1/stripecli/sessions"

//
// Public types
//

// Config contains the optional configuration parameters of a Client.
type Config struct {
	Log *log.Logger

	HTTPClient *http.Client

	APIBaseURL string
}

// Client is the client used to initiate new CLI sessions with Stripe.
type Client struct {
	apiKey string

	// Optional configuration parameters
	cfg *Config
}

// DeviceURLMap is a mapping of the urls that the device is listening
// for forwarded events on.
type DeviceURLMap struct {
	ForwardURL        string
	ForwardConnectURL string
}

// Authorize sends a request to Stripe to initiate a new CLI session.
func (c *Client) Authorize(ctx context.Context, deviceName string, websocketFeature string, filters *string, devURLMap *DeviceURLMap) (*StripeCLISession, error) {
	c.cfg.Log.WithFields(log.Fields{
		"prefix": "stripeauth.client.Authorize",
	}).Debug("Authenticating with Stripe...")

	parsedBaseURL, err := url.Parse(c.cfg.APIBaseURL)
	if err != nil {
		return nil, err
	}

	form := url.Values{}
	form.Add("device_name", deviceName)
	form.Add("websocket_feature", websocketFeature)

	if filters != nil {
		form.Add("filters", *filters)
	}

	if devURLMap != nil && len(devURLMap.ForwardURL) > 0 {
		form.Add("forward_to_url", devURLMap.ForwardURL)
	}

	if devURLMap != nil && len(devURLMap.ForwardConnectURL) > 0 {
		form.Add("forward_connect_to_url", devURLMap.ForwardConnectURL)
	}

	client := &stripe.Client{
		BaseURL: parsedBaseURL,
		APIKey:  c.apiKey,
	}

	resp, err := client.PerformRequest(ctx, http.MethodPost, stripeCLISessionPath, form.Encode(), nil)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		err := fmt.Errorf("Authorization failed, status=%d, body=%s", resp.StatusCode, body)
		return nil, err
	}

	var session *StripeCLISession
	if err := json.Unmarshal(body, &session); err != nil {
		return nil, err
	}

	c.cfg.Log.WithFields(log.Fields{
		"prefix":                         "stripeauth.Client.Authorize",
		"websocket_url":                  session.WebSocketURL,
		"websocket_id":                   session.WebSocketID,
		"websocket_authorized_feature":   session.WebSocketAuthorizedFeature,
		"reconnect_delay":                session.ReconnectDelay,
		"display_connect_filter_warning": session.DisplayConnectFilterWarning,
		"default_version":                session.DefaultVersion,
		"latest_version":                 session.LatestVersion,
	}).Debug("Got successful response from Stripe")

	return session, nil
}

// NewClient returns a new Client.
func NewClient(key string, cfg *Config) *Client {
	if cfg == nil {
		cfg = &Config{}
	}

	if cfg.Log == nil {
		cfg.Log = &log.Logger{Out: ioutil.Discard}
	}

	if cfg.APIBaseURL == "" {
		cfg.APIBaseURL = stripe.DefaultAPIBaseURL
	}

	return &Client{
		apiKey: key,
		cfg:    cfg,
	}
}
