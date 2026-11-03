require 'stripe'

# the merge builds the metadata value, not the request's own params
Stripe::PaymentIntent.create(amount: 1000, metadata: {}.merge(payment_method_types: 'card'))
