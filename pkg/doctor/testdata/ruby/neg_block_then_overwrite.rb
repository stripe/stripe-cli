require 'stripe'

params = {}
[1].each do |_|
  params = { payment_method_types: ['card'] }
end
# the block's bag is discarded before the call
params = { amount: 1000 }
Stripe::PaymentIntent.create(params)
