package main

import (
	"github.com/stripe/stripe-go/v79"
	"github.com/stripe/stripe-go/v79/paymentintent"
)

func main() {
	// legacy value literal, passed by address
	params := stripe.PaymentIntentParams{
		PaymentMethodTypes: stripe.StringSlice([]string{"card"}),
	}
	pi, _ := paymentintent.New(&params)
	_ = pi
}
