require 'stripe'

# merged into metadata, not the request's own params
params = { payment_method_types: ['card'] }
Stripe::PaymentIntent.create(amount: 1000, metadata: params.merge(source: 'x'))
