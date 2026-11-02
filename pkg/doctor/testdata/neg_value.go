package main

import "github.com/stripe/stripe-go/v79"

func main() {
	// PaymentMethodTypes here is an unrelated local variable whose name
	// happens to match the parameter spelling — it's used as a VALUE,
	// not a struct field name, so it must NOT match.
	PaymentMethodTypes := "some description"
	params := &stripe.PaymentIntentParams{
		Description: PaymentMethodTypes,
	}
	_ = params
}
