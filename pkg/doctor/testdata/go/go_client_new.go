package main

import (
	"github.com/stripe/stripe-go/v79"
	"github.com/stripe/stripe-go/v79/client"
)

func main() {
	sc := &client.API{}
	params := &stripe.PaymentIntentParams{
		PaymentMethodTypes: stripe.StringSlice([]string{"card"}),
	}
	pi, _ := sc.PaymentIntents.New(params)
	_ = pi
}
