import stripe

# the call already ran with a clean bag before params was ever
# reassigned to one containing payment_method_types — that later bag
# was never actually sent anywhere
params = {"amount": 1099}
stripe.PaymentIntent.create(**params)
params = {"payment_method_types": ["card"]}
