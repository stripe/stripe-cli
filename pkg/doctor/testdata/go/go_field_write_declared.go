package main

import (
	"github.com/stripe/stripe-go/v79"
	"github.com/stripe/stripe-go/v79/paymentintent"
)

func defaults() *stripe.PaymentIntentParams { return &stripe.PaymentIntentParams{} }

func main() {
	var params *stripe.PaymentIntentParams
	// the declared type survives this value assignment
	params = defaults()
	params.PaymentMethodTypes = stripe.StringSlice([]string{"card"})
	pi, _ := paymentintent.New(params)
	_ = pi
}
