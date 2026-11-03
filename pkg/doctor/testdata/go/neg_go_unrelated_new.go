package main

import (
	"github.com/stripe/stripe-go/v79"
	"github.com/stripe/stripe-go/v79/paymentintent"
)

func viaVariable() {
	// reaches an unrelated New and a PaymentIntent Update — never a create
	params := &stripe.PaymentIntentParams{
		PaymentMethodTypes: stripe.StringSlice([]string{"card"}),
	}
	req, _ := request.New(params)
	_ = req
	pi, _ := paymentintent.Update("pi_123", params)
	_ = pi
}

func inline() {
	req, _ := request.New(&stripe.PaymentIntentParams{
		PaymentMethodTypes: stripe.StringSlice([]string{"card"}),
	})
	_ = req
}
