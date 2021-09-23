package main

import (
	"context"
	"fmt"
	"sync"

	"github.com/stripe/stripe-cli/pkg/cmd"
	"github.com/stripe/stripe-cli/pkg/stripe"
)

func main() {
	ctx := context.Background()
	waitGroup := &sync.WaitGroup{}
	httpClient := stripe.NewTelemetryHTTPClient()
	telemetryClient := &stripe.AnalyticsTelemetryClient{WG: waitGroup, HttpClient: httpClient}
	contextWithTelemetry := context.WithValue(ctx, stripe.TelemetryClientKey{}, telemetryClient)
	cmd.Execute(contextWithTelemetry)
	fmt.Print("Waiting\n")
	// Wait for all telemetry calls to finish before existing the process
	// Can we add a timeout for this wait group?
	waitGroup.Wait()
	fmt.Print("Done Waiting\n")

}
