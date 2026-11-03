package main

import (
	"github.com/stripe/stripe-go/v79"
	"github.com/stripe/stripe-go/v79/paymentintent"
)

func main() {
	params := &stripe.PaymentIntentParams{}
	pi, _ := paymentintent.New(params)
	// set only after the create already ran
	params.PaymentMethodTypes = stripe.StringSlice([]string{"card"})
	_ = pi
}
