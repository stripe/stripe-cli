import stripe

params = {'payment_method_types': ['card']}
params = dict(params, amount=1000)
stripe.PaymentIntent.create(**params)
