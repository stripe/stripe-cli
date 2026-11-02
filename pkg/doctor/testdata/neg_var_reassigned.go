package main

import (
	"github.com/stripe/stripe-go/v79"
	"github.com/stripe/stripe-go/v79/paymentintent"
)

func main() {
	// the original var-bound bag is overwritten with a clean one (via
	// plain assignment, not another var) before the call that runs —
	// despite the mismatched var_spec/assignment_statement wrapper
	// depths, these are still unconditional sibling statements
	var params = &stripe.PaymentIntentParams{
		PaymentMethodTypes: stripe.StringSlice([]string{"card", "ideal"}),
	}
	params = &stripe.PaymentIntentParams{}
	pi, _ := paymentintent.New(params)
	_ = pi
}
