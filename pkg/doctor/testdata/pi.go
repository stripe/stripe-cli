package main

import (
	"github.com/stripe/stripe-go/v79"
	"github.com/stripe/stripe-go/v79/paymentintent"
)

func main() {
	// payment_method_types in a comment must NOT match
	params := &stripe.PaymentIntentParams{
		Amount:             stripe.Int64(1099),
		Currency:           stripe.String("eur"),
		PaymentMethodTypes: stripe.StringSlice([]string{"card", "ideal"}),
	}
	pi, _ := paymentintent.New(params)
	_ = pi
}
