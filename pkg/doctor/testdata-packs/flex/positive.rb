require 'stripe'
# beta spelling, 2-level nested — should ADVISE (rename + value change)
Stripe::PaymentIntent.create(
  amount: 5000,
  currency: 'usd',
  capture_method: 'manual',
  payment_method_options: {
    card: {
      request_incremental_authorization_support: true,
    },
  },
)
