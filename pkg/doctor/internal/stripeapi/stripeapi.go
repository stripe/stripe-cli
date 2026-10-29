// Package stripeapi is the minimal read-only Stripe API client shared by
// doctor's live callers. It resolves live-mode credentials and issues GETs
// through the CLI's normal request path.
package stripeapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"

	"github.com/stripe/stripe-cli/pkg/config"
	"github.com/stripe/stripe-cli/pkg/errorcategory"
	"github.com/stripe/stripe-cli/pkg/stripe"
)

// LoadCredentials resolves LIVE MODE credentials only: a test-mode account's
// facts aren't a meaningful stand-in for the account's real configuration.
// It errors rather than silently resolving test-mode credentials if the
// active context is in test mode.
func LoadCredentials(cfg *config.Config) (stripe.Credentials, error) {
	if cfg == nil {
		return stripe.Credentials{}, errorcategory.Errorf(errorcategory.Internal, "no CLI configuration available")
	}
	return cfg.GetProfile().ResolveCredentials(true)
}

// NewClient builds the shared stripe.Client used for read-only GETs, so
// they inherit the CLI's normal request path (unix socket / proxy
// support, standard headers).
func NewClient(creds stripe.Credentials) (*stripe.Client, error) {
	base, err := url.Parse(stripe.DefaultAPIBaseURL)
	if err != nil {
		return nil, err
	}
	return &stripe.Client{BaseURL: base, Credentials: creds}, nil
}

// Get issues a read-only GET through client, decoding the JSON response
// into out. query is the raw (already-encoded) query string, e.g.
// "limit=20"; pass it separately rather than appending it to path, since
// PerformRequest overwrites any query already present on path. headers,
// when non-nil, are set in addition to the usual auth/account headers.
func Get(client *stripe.Client, stripeAccount, path, query string, headers map[string]string, out any) error {
	configure := func(req *http.Request) error {
		client.Credentials.ApplyAccountContextHeaders(req.Header, stripeAccount, "")
		for k, v := range headers {
			req.Header.Set(k, v)
		}
		return nil
	}
	resp, err := client.PerformRequest(context.Background(), http.MethodGet, path, query, configure)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode == 401 {
		return errorcategory.Errorf(errorcategory.Auth, "credentials rejected (401) — key may be expired; run `stripe login`")
	}
	if resp.StatusCode != 200 {
		return errorcategory.Errorf(errorcategory.API, "GET %s: HTTP %d", path, resp.StatusCode)
	}
	return json.NewDecoder(resp.Body).Decode(out)
}
