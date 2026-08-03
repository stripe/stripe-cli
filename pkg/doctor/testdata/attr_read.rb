# reading the field off a response is NOT passing it as a param
intent = Stripe::PaymentIntent.retrieve('pi_123')
puts intent.payment_method_types.length
