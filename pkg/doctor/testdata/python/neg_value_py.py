import stripe

# "payment_method_types" is a string VALUE here, not a parameter name
stripe.PaymentIntent.create(**{"description": "payment_method_types"})
