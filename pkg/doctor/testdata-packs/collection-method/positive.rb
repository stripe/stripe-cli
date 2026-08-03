require 'stripe'

# Subscription create — top-level billing
Stripe::Subscription.create(
  customer: 'cus_x',
  billing: 'charge_automatically',
)

# Subscription update — top-level billing
Stripe::Subscription.update(
  'sub_x',
  billing: 'send_invoice',
)

# SubscriptionSchedule create — billing nested under default_settings AND under a phase
Stripe::SubscriptionSchedule.create(
  customer: 'cus_x',
  default_settings: {
    billing: 'charge_automatically',
  },
  phases: [
    {
      billing: 'charge_automatically',
      items: [{price: 'price_x'}],
    },
  ],
)
