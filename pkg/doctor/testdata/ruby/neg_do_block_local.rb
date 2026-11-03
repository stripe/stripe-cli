require 'stripe'

# the block's own local hash is never passed to create — it's not a
# request argument at all, just something built inside the callback
Stripe::PaymentIntent.create(amount: 1099, currency: 'usd') do |intent|
  options = { payment_method_types: ['card'] }
  puts options
end
