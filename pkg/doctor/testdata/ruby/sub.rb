require 'stripe'
# nested under payment_settings on subscriptions
Stripe::Subscription.create(
  customer: 'cus_x',
  payment_settings: {
    payment_method_types: ['card'],
  },
)
