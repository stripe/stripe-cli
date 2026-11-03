package main

import "github.com/stripe/stripe-go/v79"

func main() {
	var params stripe.PaymentIntentCreateParams
	params.PaymentMethodTypes = stripe.StringSlice([]string{"card"})
	_ = params
}
