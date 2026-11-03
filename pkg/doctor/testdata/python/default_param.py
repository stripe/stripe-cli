import stripe

params = {'payment_method_types': ['card']}


def create_intent(options=params):
    # only `options` is bound here; params is the module-level bag
    return stripe.PaymentIntent.create(**params)
