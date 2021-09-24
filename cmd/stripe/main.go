package main

import (
	"context"
	"net/http"
	"sync"
	"time"

	"github.com/stripe/stripe-cli/pkg/cmd"
	"github.com/stripe/stripe-cli/pkg/stripe"
)

func main() {
	ctx := context.Background()

	// Set up the telemetry client and add it to the context
	waitGroup := &sync.WaitGroup{}
	httpClient := &http.Client{
		Timeout: time.Second * 3,
	}
	telemetryClient := &stripe.AnalyticsTelemetryClient{WG: waitGroup, HTTPClient: httpClient}
	contextWithTelemetry := context.WithValue(ctx, stripe.TelemetryClientKey{}, telemetryClient)

	cmd.Execute(contextWithTelemetry)

	// Wait for all telemetry calls to finish before existing the process
	waitGroup.Wait()
}
