package main

import (
	"github.com/stripe/stripe-go/v79"
	"github.com/stripe/stripe-go/v79/paymentintent"
)

func main() {
	// clean, outer params actually reaches New below
	params := &stripe.PaymentIntentParams{}
	if someCondition {
		// an unrelated, inner-block-scoped params carrying the dirty
		// key — it never escapes this if-branch, so it must NOT be
		// associated with the New call after the branch closes
		params := &stripe.PaymentIntentParams{
			PaymentMethodTypes: stripe.StringSlice([]string{"card", "ideal"}),
		}
		_ = params
	}
	pi, _ := paymentintent.New(params)
	_ = pi
}
