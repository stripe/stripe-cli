require 'stripe'

Stripe::PaymentIntent.create(
  amount: 1099,
  currency: 'eur',
  allowed_source_types: ['card'],
)
