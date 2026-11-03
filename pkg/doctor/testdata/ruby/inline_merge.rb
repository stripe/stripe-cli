require 'stripe'

Stripe::PaymentIntent.create({ amount: 1000 }.merge(payment_method_types: ['card']))
