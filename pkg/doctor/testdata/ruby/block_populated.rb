require 'stripe'

params = {}
[1].each do |_|
  # writes the outer params, which the call after the block sends
  params = { payment_method_types: ['card'] }
end
Stripe::PaymentIntent.create(params)
