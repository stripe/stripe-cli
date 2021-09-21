package main

import (
	"context"

	"github.com/stripe/stripe-cli/pkg/cmd"
	"github.com/stripe/stripe-cli/pkg/stripe"
)

func main() {
	ctx := context.Background()
	contextWithTelemetry := context.WithValue(ctx, stripe.TelemetryClientKey{}, &stripe.AnalyticsTelemetry{})
	cmd.Execute(contextWithTelemetry)
}
