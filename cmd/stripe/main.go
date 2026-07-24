package main

import (
	"context"
	"net/http"
	"net/url"
	"os"
	"time"

	goversion "github.com/hashicorp/go-version"

	"github.com/stripe/stripe-cli/pkg/cmd"
	"github.com/stripe/stripe-cli/pkg/reporting"
	"github.com/stripe/stripe-cli/pkg/stripe"
	"github.com/stripe/stripe-cli/pkg/version"
)

const sentryDSN = "https://0e1c83fa780a5946e14bfc0f6d0a7ddd@errors.stripe.com/11762"

func main() {
	ctx := context.Background()

	if stripe.TelemetryOptedOut(os.Getenv("STRIPE_CLI_TELEMETRY_OPTOUT")) || stripe.TelemetryOptedOut(os.Getenv("DO_NOT_TRACK")) {
		// Proceed without telemetry or error reporting if the user opted out.
		cmd.Execute(ctx)
		return
	}

	if _, err := goversion.NewSemver(version.Version); err != nil {
		cmd.Execute(ctx)
		return
	}

	reporting.Init(sentryDSN, version.Version) //nolint:errcheck
	defer reporting.Flush()
	defer func() {
		if r := recover(); r != nil {
			reporting.RecoverAndReport(r)
			panic(r)
		}
	}()

	// Set up the telemetry client and add it to the context.
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

	// Wait for all telemetry calls to finish before exiting the process.
	telemetryClient.Wait()
}
