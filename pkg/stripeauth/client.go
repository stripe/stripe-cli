package stripeauth

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
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
}

// Client is the client used to initiate new CLI sessions with Stripe.
type Client struct {
	client stripe.RequestPerformer

	// Optional configuration parameters
	cfg *Config
}

// DeviceURLMap is a mapping of the urls that the device is listening
// for forwarded events on.
type DeviceURLMap struct {
	ForwardURL        string
	ForwardConnectURL string
}

// CreateSessionRequest defines the API input parameters for client.Authorize.
type CreateSessionRequest struct {
	DeviceName        string
	WebSocketFeatures []string

	Filters      *string
	DeviceURLMap *DeviceURLMap
}

// Authorize sends a request to Stripe to initiate a new CLI session.
func (c *Client) Authorize(ctx context.Context, req CreateSessionRequest) (*StripeCLISession, error) {
	c.cfg.Log.WithFields(log.Fields{
		"prefix": "stripeauth.client.Authorize",
	}).Debug("Authenticating with Stripe...")

	form := url.Values{}
	form.Add("device_name", req.DeviceName)
	for _, feature := range req.WebSocketFeatures {
		form.Add("websocket_features[]", feature)
	}

	if req.Filters != nil {
		form.Add("filters", *req.Filters)
	}

	if devURLMap := req.DeviceURLMap; devURLMap != nil {
		if len(devURLMap.ForwardURL) > 0 {
			form.Add("forward_to_url", devURLMap.ForwardURL)
		}

		if len(devURLMap.ForwardConnectURL) > 0 {
			form.Add("forward_connect_to_url", devURLMap.ForwardConnectURL)
		}
	}

	resp, err := c.client.PerformRequest(ctx, http.MethodPost, stripeCLISessionPath, form.Encode(), nil)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
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
func NewClient(client stripe.RequestPerformer, cfg *Config) *Client {
	if cfg == nil {
		cfg = &Config{}
	}

	if cfg.Log == nil {
		cfg.Log = &log.Logger{Out: io.Discard}
	}

	return &Client{
		client: client,
		cfg:    cfg,
	}
}
