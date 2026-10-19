package main

import (
	"context"
	"net/http"
	"net/url"
	"os"
	"time"

	goversion "github.com/hashicorp/go-version"

	"github.com/stripe/stripe-cli/pkg/autoupdate"
	"github.com/stripe/stripe-cli/pkg/cmd"
	"github.com/stripe/stripe-cli/pkg/reporting"
	"github.com/stripe/stripe-cli/pkg/stripe"
	"github.com/stripe/stripe-cli/pkg/version"
)

const sentryDSN = "https://0e1c83fa780a5946e14bfc0f6d0a7ddd@errors.stripe.com/11762"

func main() {
	// Apply a pending update before anything else: this re-execs the new binary,
	// so the command the user typed runs on the version they are being moved to.
	autoupdate.ApplyIfPending()

	// Check for the next update after the command has run, so the network call
	// never delays it. Deferred rather than called at the end of main because
	// every branch below returns early, and the check has to happen on all of
	// them. This still does not cover cmd.Execute's os.Exit on command failure.
	defer autoupdate.CheckForUpdate()

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

	// Set up the telemetry client and add it to the context before installing
	// the panic handler below, so a recovered panic can also report telemetry.
	httpClient := &http.Client{
		Timeout: time.Second * 3,
	}
	telemetryClient := &stripe.AnalyticsTelemetryClient{HTTPClient: httpClient}
	if raw := os.Getenv("STRIPE_TELEMETRY_URL"); raw != "" {
		if parsed, err := url.Parse(raw); err == nil {
			telemetryClient.BaseURL = parsed
		}
	}
	ctx = stripe.WithTelemetryClient(ctx, telemetryClient)

	// Attach event metadata here, before cmd.Execute, and let it populate the
	// same pointer throughout the command run: that way a panic recovered
	// below still sees the command path Execute set, instead of empty
	// metadata freshly built from this pre-command ctx.
	telemetryMetadata := stripe.NewEventMetadata()
	ctx = stripe.WithEventMetadata(ctx, telemetryMetadata)

	defer func() {
		if r := recover(); r != nil {
			reporting.RecoverAndReport(ctx, r)
			panic(r)
		}
	}()

	cmd.Execute(ctx)

	// Wait for all telemetry calls to finish before exiting the process.
	telemetryClient.Wait()
}
