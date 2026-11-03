package main

import (
	"github.com/stripe/stripe-go/v79"
	"github.com/stripe/stripe-go/v79/paymentintent"
)

func main() {
	params := &stripe.PaymentIntentParams{
		PaymentMethodTypes: stripe.StringSlice([]string{"card"}),
	}
	// a slice literal is not an operation; the only call is Update
	batch := []*stripe.PaymentIntentParams{params}
	_ = batch
	pi, _ := paymentintent.Update("pi_123", params)
	_ = pi
}
