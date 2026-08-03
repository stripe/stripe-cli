require 'stripe'

Stripe::Subscription.create(
  customer: 'cus_123',
  items: [{price: 'price_123'}],
  tax_percent: 20.0,
)

Stripe::Invoice.update(
  'in_123',
  tax_percent: 7.5,
)
