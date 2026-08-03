# tax_percent must NOT match here: Charge is not in this rule's operations
Stripe::Charge.create(
  amount: 1099,
  currency: 'usd',
  tax_percent: 20.0,
)
