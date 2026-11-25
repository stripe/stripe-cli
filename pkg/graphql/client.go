// Package graphql provides a genqlient client for Stripe's GraphQL gateway
// (graphql.stripe.com) that authenticates with the CLI's credentials.
//
// The query functions in generated.go are produced from operations.graphql
// and schema.graphql; run `go generate ./pkg/graphql` after changing either.
package graphql

//go:generate go tool genqlient genqlient.yaml

import (
	"context"
	"fmt"
	"net/http"
	"time"

	genqlient "github.com/Khan/genqlient/graphql"

	"github.com/stripe/stripe-cli/pkg/config"
	"github.com/stripe/stripe-cli/pkg/errorcategory"
	"github.com/stripe/stripe-cli/pkg/login"
	"github.com/stripe/stripe-cli/pkg/requests"
	"github.com/stripe/stripe-cli/pkg/stripe"
)

const (
	// DefaultHost is the production GraphQL gateway host.
	DefaultHost = "graphql.stripe.com"
	// QAHost is the QA GraphQL gateway host.
	QAHost = "qa-graphql.stripe.com"

	defaultPath    = "/graphql"
	defaultTimeout = 30 * time.Second
)

// getWorkspaceContext and getContextStatuses are variables so tests can stub
// the workspace lookups.
var (
	getWorkspaceContext = requests.GetWorkspaceContext
	getContextStatuses  = requests.GetContextStatuses
)

// Options configures NewClient.
type Options struct {
	// Profile supplies the CLI credentials. Required.
	Profile *config.Profile
	// Livemode selects live or test mode credentials.
	Livemode bool
	// APIBaseURL is the Stripe API base URL used to look up the workspace for
	// OAuth credentials. Defaults to stripe.DefaultAPIBaseURL.
	APIBaseURL string
	// Host is the GraphQL gateway host. Defaults to DefaultHost.
	Host string
	// StripeVersion is sent as the Stripe-Version header. Defaults to the
	// CLI's preview API version.
	StripeVersion string
	// Transport is the underlying RoundTripper. Defaults to http.DefaultTransport.
	Transport http.RoundTripper
}

// HostForAccessBaseURL returns the gateway host that matches the OAuth access
// base URL the CLI logged in with, so QA credentials use the QA gateway.
func HostForAccessBaseURL(accessBaseURL string) string {
	if accessBaseURL == login.QAAccessBaseURL {
		return QAHost
	}
	return DefaultHost
}

// NewClient returns a genqlient client for the GraphQL gateway that sends the
// profile's credentials with every request.
func NewClient(ctx context.Context, opts Options) (genqlient.Client, error) {
	if opts.Profile == nil {
		return nil, errorcategory.New(errorcategory.Internal, "graphql: a profile is required")
	}
	if opts.APIBaseURL == "" {
		opts.APIBaseURL = stripe.DefaultAPIBaseURL
	}
	if opts.Host == "" {
		opts.Host = DefaultHost
	}
	if opts.StripeVersion == "" {
		opts.StripeVersion = requests.StripePreviewVersionHeaderValue
	}
	if opts.Transport == nil {
		opts.Transport = http.DefaultTransport
	}

	creds, err := opts.Profile.ResolveCredentials(opts.Livemode)
	if err != nil {
		return nil, err
	}
	creds, err = withWorkspaceContext(ctx, opts, creds)
	if err != nil {
		return nil, err
	}

	httpClient := &http.Client{
		Timeout: defaultTimeout,
		Transport: &authTransport{
			creds:         creds,
			stripeVersion: opts.StripeVersion,
			base:          opts.Transport,
		},
	}

	return genqlient.NewClient("https://"+opts.Host+defaultPath, httpClient), nil
}

// withWorkspaceContext replaces the account ID (acct_...) in OAuth credentials'
// Stripe-Context with the ID of the workspace it belongs to. The CLI's OAuth
// grant is on the workspace, which is what the gateway authorizes against.
// Plain API key credentials carry no context and are returned unchanged.
func withWorkspaceContext(ctx context.Context, opts Options, creds stripe.Credentials) (stripe.Credentials, error) {
	if creds.OAKContext == "" {
		return creds, nil
	}

	var workspaceID string
	var err error
	if creds.Livemode() {
		workspaceID, err = liveWorkspaceID(ctx, opts, creds)
	} else {
		workspaceID, err = sandboxWorkspaceID(ctx, opts, creds)
	}
	if err != nil {
		return stripe.Credentials{}, fmt.Errorf("could not resolve the workspace for %s: %w", creds.OAKContext, err)
	}
	if workspaceID == "" {
		return stripe.Credentials{}, errorcategory.Errorf(errorcategory.API, "no workspace found for %s", creds.OAKContext)
	}

	creds.OAKContext = workspaceID
	return creds, nil
}

// liveWorkspaceID looks up a live account's workspace. The workspace_context
// endpoint only supports live mode.
func liveWorkspaceID(ctx context.Context, opts Options, creds stripe.Credentials) (string, error) {
	wc, err := getWorkspaceContext(ctx, opts.APIBaseURL, opts.Profile, creds, true)
	if err != nil {
		return "", err
	}
	return wc.WorkspaceID, nil
}

// sandboxWorkspaceID looks up a sandbox's own test mode workspace
// (wksp_test_...) through context_statuses, which only returns it when the
// sandbox is within the CLI's permission grant.
func sandboxWorkspaceID(ctx context.Context, opts Options, creds stripe.Credentials) (string, error) {
	statuses, err := getContextStatuses(ctx, opts.APIBaseURL, opts.Profile, creds, []requests.ContextStatusQuery{
		{AccountID: creds.OAKContext, Livemode: false},
	})
	if err != nil {
		return "", err
	}
	for _, s := range statuses {
		if s.AccountID == creds.OAKContext && !s.Livemode && s.Valid {
			return s.WorkspaceID, nil
		}
	}
	return "", nil
}

// authTransport adds the CLI's auth and Stripe-Version headers to each request.
type authTransport struct {
	creds         stripe.Credentials
	stripeVersion string
	base          http.RoundTripper
}

func (t *authTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req = req.Clone(req.Context())
	t.creds.SetRequestHeaders(req)
	req.Header.Set("Stripe-Version", t.stripeVersion)

	return t.base.RoundTrip(req)
}
