require 'stripe'

# "payment_method_types" is a string VALUE here, not a parameter name
Stripe::PaymentIntent.create(description: 'payment_method_types')
