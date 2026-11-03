require 'stripe'

params = { payment_method_types: ['card'] }
[1].each do |_|
  # writes the captured outer params (merging it), not a new local
  params = params.merge(amount: 1000)
  Stripe::PaymentIntent.create(params)
end
