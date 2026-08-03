require 'stripe'
# Not a Stripe resource: must NOT match
MyBillingModel.update(
  prorate: true,
)
