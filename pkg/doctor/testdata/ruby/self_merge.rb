require 'stripe'

params = { payment_method_types: ['card'] }
params = params.merge(amount: 1000)
Stripe::PaymentIntent.create(params)
