require 'stripe'

params = {}
[1].each do |_|
  # the call runs before this write in the block
  Stripe::PaymentIntent.create(params)
  params = { payment_method_types: ['card'] }
end
