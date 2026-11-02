import stripe

params = {'payment_method_types': ['card'], 'amount': 1099}


def create_intent():
    def helper():
        # an unrelated local to this nested helper, never visible in
        # create_intent's own body — must NOT shadow the module-level
        # params reaching the call below
        params = {}
        return params

    helper()
    return stripe.PaymentIntent.create(**params)
