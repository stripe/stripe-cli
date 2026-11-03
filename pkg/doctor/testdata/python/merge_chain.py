import stripe

base = {'payment_method_types': ['card']}
params = dict(base, amount=1000)
stripe.PaymentIntent.create(**params)
