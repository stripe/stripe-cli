require 'stripe'
# pre-migration: direct Subscription create — should ADVISE
Stripe::Subscription.create(
  customer: 'cus_x',
  items: [{price: 'price_x'}],
)
