package cmd

// overview_account_facts.go fetches the account's default API version for
// `stripe overview` from the CLI-only account facts endpoint.

import (
	"context"
	"fmt"
	"io"

	log "github.com/sirupsen/logrus"

	"github.com/stripe/stripe-cli/pkg/ansi"
	"github.com/stripe/stripe-cli/pkg/config"
	"github.com/stripe/stripe-cli/pkg/requests"
	"github.com/stripe/stripe-cli/pkg/stripe"
)

// startOverviewDefaultAPIVersion is a variable so tests can stub the lookup.
var startOverviewDefaultAPIVersion = startDefaultAPIVersionLookup

var getAccountFacts = requests.GetAccountFacts

func startDefaultAPIVersionLookup(ctx context.Context, profile *config.Profile, apiBaseURL string, creds stripe.Credentials) <-chan string {
	ch := make(chan string, 1)
	go func() {
		ch <- defaultAPIVersion(ctx, apiBaseURL, profile, creds, creds.Livemode())
	}()
	return ch
}

func defaultAPIVersion(ctx context.Context, apiBaseURL string, profile *config.Profile, creds stripe.Credentials, livemode bool) string {
	if creds.OAKContext == "" {
		// API keys aren't accepted by the endpoint.
		return ""
	}

	facts, err := getAccountFacts(ctx, apiBaseURL, profile, creds, livemode)
	if err != nil {
		log.Debugf("Skipping default API version: %v", err)
		return ""
	}
	return facts.DefaultAPIVersion
}

func renderDefaultAPIVersion(w io.Writer, version string) {
	if version == "" {
		return
	}
	fmt.Fprintf(w, "Your account's default API version is %s.\n\n", ansi.Bold(version))
}
