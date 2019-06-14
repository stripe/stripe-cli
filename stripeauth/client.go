package stripeauth

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/stripe/stripe-cli/useragent"
)

//
// Public types
//

// Config contains the optional configuration parameters of a Client.
type Config struct {
	Log *log.Logger

	HTTPClient *http.Client

	UnixSocket string

	URL string
}

// Client is the client used to initiate new CLI sessions with Stripe.
type Client struct {
	apiKey string

	// Optional configuration parameters
	cfg *Config
}

// Authorize sends a request to Stripe to initiate a new CLI session.
func (c *Client) Authorize(deviceName string) (*StripeCLISession, error) {
	c.cfg.Log.WithFields(log.Fields{
		"prefix": "stripeauth.client.Authorize",
	}).Debug("Authenticating with Stripe...")

	form := url.Values{}
	form.Add("device_name", deviceName)

	req, err := http.NewRequest("POST", c.cfg.URL, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}

	// Disable compression by requiring "identity"
	req.Header.Set("Accept-Encoding", "identity")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", useragent.GetEncodedUserAgent())
	req.Header.Set("X-Stripe-Client-User-Agent", useragent.GetEncodedStripeUserAgent())

	resp, err := c.cfg.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}

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
		"prefix":          "stripeauth.Client.Authorize",
		"websocket_url":   session.WebSocketURL,
		"websocket_id":    session.WebSocketID,
		"reconnect_delay": session.ReconnectDelay,
	}).Debug("Got successful response from Stripe")

	return session, nil
}

//
// Public functions
//

// NewClient returns a new Client.
func NewClient(key string, cfg *Config) *Client {
	if cfg == nil {
		cfg = &Config{}
	}
	if cfg.Log == nil {
		cfg.Log = &log.Logger{Out: ioutil.Discard}
	}
	if cfg.HTTPClient == nil {
		cfg.HTTPClient = newHTTPClient(cfg.UnixSocket)
	}
	if cfg.URL == "" {
		cfg.URL = defaultAuthorizeURL
	}

	return &Client{
		apiKey: key,
		cfg:    cfg,
	}
}

//
// Private constants
//

const (
	defaultAuthorizeURL = "https://api.stripe.com/v1/stripecli/sessions"
)

func newHTTPClient(unixSocket string) *http.Client {
	var httpTransport *http.Transport
	if unixSocket != "" {
		dialFunc := func(network, addr string) (net.Conn, error) {
			return net.Dial("unix", unixSocket)
		}
		httpTransport = &http.Transport{
			Dial:                  dialFunc,
			DialTLS:               dialFunc,
			ExpectContinueTimeout: 10 * time.Second,
			ResponseHeaderTimeout: 30 * time.Second,
			TLSHandshakeTimeout:   10 * time.Second,
		}
	} else {
		httpTransport = &http.Transport{
			Dial: (&net.Dialer{
				Timeout:   30 * time.Second,
				KeepAlive: 30 * time.Second,
			}).Dial,
			Proxy:               http.ProxyFromEnvironment,
			TLSHandshakeTimeout: 10 * time.Second,
		}
	}
	return &http.Client{
		Transport: httpTransport,
	}
}
