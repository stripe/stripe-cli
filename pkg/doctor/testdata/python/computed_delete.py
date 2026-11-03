import stripe

payment_method_types = 'description'
params = {'payment_method_types': ['card'], 'description': 'x'}
# deletes params['description'] at runtime, not the payment method key
del params[payment_method_types]
stripe.PaymentIntent.create(**params)
