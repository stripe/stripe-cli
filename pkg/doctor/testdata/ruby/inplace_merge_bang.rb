require 'stripe'

params = { amount: 1000 }
params.merge!(payment_method_types: ['card'])
Stripe::PaymentIntent.create(params)
