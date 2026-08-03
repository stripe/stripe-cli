require 'stripe'
# nested payment_settings on a PAYMENT INTENT is not a valid rule pairing:
# the nested rule is scoped to subscriptions/invoices only. Must NOT match.
Stripe::PaymentIntent.create(
  amount: 1099,
  payment_settings: {
    payment_method_types: ['card'],
  },
)
# and a top-level param on a SUBSCRIPTION call is also not in the rule:
Stripe::Subscription.create(
  customer: 'cus_x',
  payment_method_types: ['card'],
)
