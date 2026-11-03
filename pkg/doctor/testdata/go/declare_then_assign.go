package main

import (
	"github.com/stripe/stripe-go/v79"
	"github.com/stripe/stripe-go/v79/paymentintent"
)

func main() {
	// declared first, populated by a later plain assignment
	var params *stripe.PaymentIntentParams
	params = &stripe.PaymentIntentParams{
		PaymentMethodTypes: stripe.StringSlice([]string{"card"}),
	}
	pi, _ := paymentintent.New(params)
	_ = pi
}
