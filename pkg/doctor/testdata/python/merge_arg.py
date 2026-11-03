import stripe

params = {'payment_method_types': ['card']}
stripe.PaymentIntent.create(**dict(params, amount=1000))
