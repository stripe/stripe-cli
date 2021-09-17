package p400

import (
	"context"
	"fmt"
	"math/rand"
	"os"

	"github.com/stripe/stripe-cli/pkg/ansi"
)

// SetUpTestPayment asks the user for their payment amount / currency, then updates the reader display, then creates a new Payment Intent
func SetUpTestPayment(ctx context.Context, tsCtx TerminalSessionContext) (TerminalSessionContext, error) {
	var err error
	tsCtx.Currency, err = ReaderChargeCurrencyPrompt()

	if err != nil {
		return tsCtx, err
	}

	tsCtx.Amount, err = ReaderChargeAmountPrompt()

	if err != nil {
		return tsCtx, err
	}

	spinner := ansi.StartNewSpinner("Setting reader display...", os.Stdout)
	tsCtx.TransactionID = int(rand.Float64() * 100000)
	tsCtx.MethodID = int(rand.Float64() * 100000)
	parentTraceID := SetParentTraceID(tsCtx.TransactionID, tsCtx.MethodID, "setReaderDisplay")

	err = SetReaderDisplay(tsCtx, parentTraceID)

	if err != nil {
		return tsCtx, err
	}

	ansi.StopSpinner(spinner, ansi.Faint("Reader display updated"), os.Stdout)
	spinner = ansi.StartNewSpinner("Creating a new Payment Intent...", os.Stdout)

	tsCtx.PaymentIntentID, err = CreatePaymentIntent(ctx, tsCtx)

	if err != nil {
		return tsCtx, err
	}

	ansi.StopSpinner(spinner, ansi.Faint(fmt.Sprintf("Payment Intent created %s", tsCtx.PaymentIntentID)), os.Stdout)

	return tsCtx, nil
}

// CompleteTestPayment sets the reader into collect payment mode, waits for the payment, confirms the payment, then finally captures the Payment Intent
func CompleteTestPayment(ctx context.Context, tsCtx TerminalSessionContext) (TerminalSessionContext, error) {
	parentTraceID := SetParentTraceID(tsCtx.TransactionID, tsCtx.MethodID, "processPayment")
	err := CollectPaymentMethod(tsCtx, parentTraceID)

	if err != nil {
		return tsCtx, err
	}

	spinner := ansi.StartNewSpinner("Tap, swipe or dip your Stripe Test Card in the reader...", os.Stdout)
	paymentMethod, err := WaitForPaymentCollection(tsCtx, parentTraceID, 0)

	if err != nil {
		return tsCtx, err
	}

	ansi.StopSpinner(spinner, ansi.Faint("Payment Method collected"), os.Stdout)
	spinner = ansi.StartNewSpinner("Confirming payment...", os.Stdout)
	paymentMethodID, err := ConfirmPayment(tsCtx, paymentMethod, parentTraceID)

	if err != nil {
		return tsCtx, err
	}

	ansi.StopSpinner(spinner, ansi.Faint(fmt.Sprintf("Payment confirmed %s", paymentMethodID)), os.Stdout)

	// manually capturing payment intent as required by terminal flow
	spinner = ansi.StartNewSpinner("Capturing Payment Intent...", os.Stdout)
	err = CapturePaymentIntent(ctx, tsCtx)

	if err != nil {
		return tsCtx, err
	}

	ansi.StopSpinner(spinner, ansi.Faint(fmt.Sprintf("Payment Intent manually captured %s", tsCtx.PaymentIntentID)), os.Stdout)

	return tsCtx, nil
}

// SummarizeQuickstartCompletion is the success text that is output once the quickstart flow is completed. It lists the Payment Intent Dashboard URL, and the Terminal readers Dashboard URL
func SummarizeQuickstartCompletion(tsCtx TerminalSessionContext) error {
	color := ansi.Color(os.Stdout)
	successText := color.Green("✔ Test payment complete! Here are some example applications from Stripe to continue with your integration.")
	exampleAppURL := color.Cyan("https://stripe.com/docs/terminal/example-applications")
	paymentIntentURL := color.Cyan(fmt.Sprintf("https://dashboard.stripe.com/test/payments/%s", tsCtx.PaymentIntentID))
	readerURL := color.Cyan(fmt.Sprintf("https://dashboard.stripe.com/test/terminal/locations/%s", tsCtx.LocationID))
	successPrint := fmt.Sprintf("%s\n%s\n\n", successText, exampleAppURL)
	fmt.Print(successPrint)

	fmt.Printf("View your test payment: %s\n", paymentIntentURL)
	fmt.Printf("View your registered reader: %s\n", readerURL)

	return nil
}
