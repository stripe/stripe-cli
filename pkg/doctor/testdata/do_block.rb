require 'stripe'

# the trailing do...end block must NOT be mistaken for the real
# argument list — payment_method_types is in the actual arguments
Stripe::PaymentIntent.create(payment_method_types: ['card', 'ideal']) do |x|
  puts x
end
