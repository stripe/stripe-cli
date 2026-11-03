package main

import (
	"github.com/stripe/stripe-go/v79"
	"github.com/stripe/stripe-go/v79/paymentintent"
)

func main() {
	// legacy, create-and-update-shared PaymentIntentParams, bound with
	// var (not :=), then passed to New — must still resolve
	var params = &stripe.PaymentIntentParams{
		PaymentMethodTypes: stripe.StringSlice([]string{"card", "ideal"}),
	}
	pi, _ := paymentintent.New(params)
	_ = pi
}
