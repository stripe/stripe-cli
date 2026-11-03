import stripe

params = {'payment_method_types': ['card'], 'amount': 1000}
del params['payment_method_types']
stripe.PaymentIntent.create(**params)
