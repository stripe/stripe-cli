require 'stripe'

# billing here is a real request param, but on Customer — not one of the
# renamed resources (subscriptions / subscription_schedules / invoices).
# Must NOT match.
Stripe::Customer.create(
  email: 'a@example.com',
  billing: 'charge_automatically',
)
