package main

import (
	"github.com/stripe/stripe-go/v79"
	"github.com/stripe/stripe-go/v79/paymentintent"
)

func main() {
	// legacy shared type, sent only to Update
	params := &stripe.PaymentIntentParams{}
	params.PaymentMethodTypes = stripe.StringSlice([]string{"card"})
	pi, _ := paymentintent.Update("pi_123", params)
	_ = pi
}
