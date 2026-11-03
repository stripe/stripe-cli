import stripe

params = {'payment_method_types': ['card'], 'amount': 1000}
del params['amount']
stripe.PaymentIntent.create(**params)
