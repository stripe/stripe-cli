// Package stripeapi is the minimal read-only Stripe API client shared by
// doctor's live callers: judge (fetching the migration rule itself) and
// remediation implementations (fetching whatever account evidence their
// Kind needs — recent API versions, payment-method configuration, etc.).
// It knows nothing about doctor's engine; it only resolves live-mode
// credentials and issues GETs through the CLI's normal request path.
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

// LoadCredentials resolves LIVE MODE credentials only, through the CLI's
// own profile machinery: the user access token (UAT) from `stripe login`
// when present, falling back to an API key (which already honors
// STRIPE_API_KEY). doctor's account-evidence lookups (recent API
// versions, payment-method configuration, the rule fetch itself) are
// read-only but still describe the account's real, live configuration —
// a test-mode account's facts aren't a meaningful stand-in for them. If
// the active context is actually in test mode, this errors rather than
// silently resolving test-mode credentials instead.
func LoadCredentials(cfg *config.Config) (stripe.Credentials, error) {
	if cfg == nil {
		return stripe.Credentials{}, errorcategory.Errorf(errorcategory.Internal, "no CLI configuration available")
	}
	return cfg.GetProfile().ResolveCredentials(true)
}

// NewClient builds the shared stripe.Client used for read-only GETs, so
// they inherit the CLI's normal request path — unix socket / proxy
// support, standard headers — instead of a bare http.Client.
func NewClient(creds stripe.Credentials) (*stripe.Client, error) {
	base, err := url.Parse(stripe.DefaultAPIBaseURL)
	if err != nil {
		return nil, err
	}
	return &stripe.Client{BaseURL: base, Credentials: creds}, nil
}

// Get issues a read-only GET through client, decoding the JSON response
// into out. query is the raw (already-encoded) query string, e.g.
// "limit=20"; it is applied via PerformRequest rather than appended to
// path since PerformRequest overwrites any query already present on path.
// headers, when non-nil, are set on the request in addition to the usual
// auth/account headers — e.g. a Stripe-Version pin for an unreleased API.
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
