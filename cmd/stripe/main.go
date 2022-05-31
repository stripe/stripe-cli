package main

import (
	"context"
	"fmt"
	"syscall/js"

	"github.com/stripe/stripe-cli/pkg/cmd"
)

func executeCommandWrapper() js.Func {
	return js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		ctx := context.Background()
		cmd.PassInArgs(args)
		cmd.Execute(ctx)
		return nil
	})
}

func main() {
	// ctx := context.Background()
	fmt.Println("Go Web Assembly From Stripe CLI!")
	js.Global().Set("stripeCli", executeCommandWrapper())
	<-make(chan bool)

	// if stripe.TelemetryOptedOut(os.Getenv("STRIPE_CLI_TELEMETRY_OPTOUT")) || stripe.TelemetryOptedOut(os.Getenv("DO_NOT_TRACK")) {
	// 	// Proceed without the telemetry client if client opted out.
	// 	cmd.Execute(ctx)
	// } else {
	// 	// Set up the telemetry client and add it to the context
	// 	httpClient := &http.Client{
	// 		Timeout: time.Second * 3,
	// 	}
	// 	telemetryClient := &stripe.AnalyticsTelemetryClient{HTTPClient: httpClient}
	// 	contextWithTelemetry := stripe.WithTelemetryClient(ctx, telemetryClient)

	// 	cmd.Execute(contextWithTelemetry)

	// 	// Wait for all telemetry calls to finish before existing the process
	// 	telemetryClient.Wait()
	// }
}
