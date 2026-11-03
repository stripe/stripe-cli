require 'stripe'

params = { payment_method_types: ['card'] }
Stripe::PaymentIntent.create(params.merge(amount: 1000))
