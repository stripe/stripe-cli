package main

import (
	"github.com/stripe/stripe-go/v79"
	"github.com/stripe/stripe-go/v79/paymentintent"
)

func main() {
	// PaymentIntentParams is shared by create and update — this is an
	// update, outside the create-only rule, and must NOT match
	params := &stripe.PaymentIntentParams{
		PaymentMethodTypes: stripe.StringSlice([]string{"card", "ideal"}),
	}
	pi, _ := paymentintent.Update("pi_123", params)
	_ = pi
}
