package main

import (
	"github.com/stripe/stripe-go/v79"
	"github.com/stripe/stripe-go/v79/paymentintent"
)

var params = &stripe.PaymentIntentParams{
	PaymentMethodTypes: stripe.StringSlice([]string{"card"}),
}

func create(cond bool) {
	if cond {
		// expires at the end of this block
		params := &stripe.PaymentIntentParams{}
		_ = params
	}
	pi, _ := paymentintent.New(params)
	_ = pi
}
