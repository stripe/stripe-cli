package main

import (
	"context"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/stripe/stripe-cli/pkg/autoupdate"
	"github.com/stripe/stripe-cli/pkg/cmd"
	"github.com/stripe/stripe-cli/pkg/stripe"
)

func main() {
	// Apply pending update (downloads + re-execs if an update was staged)
	autoupdate.ApplyIfPending()

	ctx := context.Background()

	if stripe.TelemetryOptedOut(os.Getenv("STRIPE_CLI_TELEMETRY_OPTOUT")) || stripe.TelemetryOptedOut(os.Getenv("DO_NOT_TRACK")) {
		// Proceed without the telemetry client if client opted out.
		cmd.Execute(ctx)
	} else {
		// Set up the telemetry client and add it to the context
		httpClient := &http.Client{
			Timeout: time.Second * 3,
		}
		telemetryClient := &stripe.AnalyticsTelemetryClient{HTTPClient: httpClient}
		if raw := os.Getenv("STRIPE_TELEMETRY_URL"); raw != "" {
			if parsed, err := url.Parse(raw); err == nil {
				telemetryClient.BaseURL = parsed
			}
		}
		contextWithTelemetry := stripe.WithTelemetryClient(ctx, telemetryClient)

		cmd.Execute(contextWithTelemetry)

		// Wait for all telemetry calls to finish before existing the process
		telemetryClient.Wait()
	}

	// Check for updates after command completes (once per day, synchronous)
	autoupdate.CheckForUpdate()
}
