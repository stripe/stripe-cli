import stripe

def real_payment():
    params = {"payment_method_types": ["card"], "amount": 1099}
    return stripe.PaymentIntent.create(**params)
