require 'stripe'

params = { amount: 1000 }
# non-mutating merge whose result is discarded
params.merge(payment_method_types: ['card'])
Stripe::PaymentIntent.create(params)
