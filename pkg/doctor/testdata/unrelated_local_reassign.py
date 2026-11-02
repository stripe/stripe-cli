import stripe

# module-level bag, sent as-is below
params = {"payment_method_types": ["card"], "amount": 1099}


def never_called():
    # an unrelated, local "params" in a function that's never invoked —
    # must NOT be treated as overwriting the module-level binding above
    params = {}
    return params


stripe.PaymentIntent.create(**params)
