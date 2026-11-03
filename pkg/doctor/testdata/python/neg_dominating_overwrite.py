import stripe


def create(flag):
    if flag:
        params = {'payment_method_types': ['card']}
    # every path to the call passes through this clean assignment
    params = {}
    return stripe.PaymentIntent.create(**params)
