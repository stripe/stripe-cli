package main

import (
	"github.com/stripe/stripe-go/v79"
	"github.com/stripe/stripe-go/v79/paymentintent"
)

func create(cond bool) {
	params := &stripe.PaymentIntentParams{
		PaymentMethodTypes: stripe.StringSlice([]string{"card"}),
	}
	// New sees the initializer's clean shadow; the dirty bag only reaches Update
	if params := (&stripe.PaymentIntentParams{}); cond {
		paymentintent.New(params)
	}
	paymentintent.Update("pi_123", params)
}
