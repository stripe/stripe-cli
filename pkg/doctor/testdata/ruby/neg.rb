# Not a Stripe resource: must NOT match
MyOrderModel.create(
  payment_method_types: ['card'],
)
