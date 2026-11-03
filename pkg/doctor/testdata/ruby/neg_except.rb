require 'stripe'

params = { payment_method_types: ['card'], amount: 1000 }
# removes the outgoing parameter; not a merge
params = params.except(:payment_method_types)
Stripe::PaymentIntent.create(params)
