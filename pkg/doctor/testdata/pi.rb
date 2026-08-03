require 'stripe'

# payment_method_types in a comment must NOT match
Stripe::PaymentIntent.create(
  amount: 1099,
  currency: 'eur',
  payment_method_types: ['card', 'ideal'],
)

log("payment_method_types")
