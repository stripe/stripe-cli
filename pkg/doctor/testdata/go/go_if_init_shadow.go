package main

import (
	"github.com/stripe/stripe-go/v79"
	"github.com/stripe/stripe-go/v79/paymentintent"
)

func create(cond bool) {
	params := &stripe.PaymentIntentParams{
		PaymentMethodTypes: stripe.StringSlice([]string{"card"}),
	}
	// this params is scoped to the if statement alone
	if params := (&stripe.PaymentIntentParams{}); cond {
		_ = params
	}
	pi, _ := paymentintent.New(params)
	_ = pi
}
