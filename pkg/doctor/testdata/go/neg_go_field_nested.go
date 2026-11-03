package main

import "github.com/stripe/stripe-go/v79"

func main() {
	// a nested field, not the rule's top-level payment_method_types
	params := &stripe.PaymentIntentCreateParams{}
	params.PaymentSettings.PaymentMethodTypes = stripe.StringSlice([]string{"card"})
	_ = params
}
