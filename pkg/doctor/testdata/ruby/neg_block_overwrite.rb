require 'stripe'

params = { payment_method_types: ['card'] }
[1].each do |_|
  # overwrites the captured outer params with a clean bag before the call
  params = { amount: 1000 }
  Stripe::PaymentIntent.create(params)
end
