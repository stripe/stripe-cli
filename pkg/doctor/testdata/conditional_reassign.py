import stripe

# the original bag is still sent whenever use_defaults is false — a
# conditional reassignment isn't a definite overwrite on every path
params = {"payment_method_types": ["card"], "amount": 1099}
if use_defaults:
    params = {"amount": 1099}
stripe.PaymentIntent.create(**params)
