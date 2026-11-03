require 'stripe'
# nested payment_settings.payment_method_types is not a leaf param the rule
# tracks at all (dpm only covers the top-level param). Must NOT match.
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
