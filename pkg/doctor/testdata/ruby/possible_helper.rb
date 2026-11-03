require 'stripe'

# the bag reaches Stripe through a helper the resolver doesn't follow:
# reported as possible rather than dropped
def send_intent(p)
  Stripe::PaymentIntent.create(p)
end

params = { payment_method_types: ['card'] }
send_intent(params)
