package main

import (
	"github.com/stripe/stripe-go/v79"
	"github.com/stripe/stripe-go/v79/paymentintent"
)

func main() {
	// outer bag, used only by Update below — the create-only rule
	// must not flag it through the inner New call, which uses its
	// own, block-scoped clean params instead
	params := &stripe.PaymentIntentParams{
		PaymentMethodTypes: stripe.StringSlice([]string{"card", "ideal"}),
	}
	if someCondition {
		params := &stripe.PaymentIntentParams{}
		pi, _ := paymentintent.New(params)
		_ = pi
	}
	pi2, _ := paymentintent.Update("pi_123", params)
	_ = pi2
}
