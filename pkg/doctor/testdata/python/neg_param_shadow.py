import stripe

params = {'payment_method_types': ['card']}


def create_intent(params=None):
    # the function's own parameter shadows the module-level bag
    return stripe.PaymentIntent.create(**params)
