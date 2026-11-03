import stripe

params = {'amount': 1000}
params.update({'payment_method_types': ['card']})
stripe.PaymentIntent.create(**params)
