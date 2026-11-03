import stripe

def unrelated_gateway():
    # same variable name, different function: must NOT resolve
    params = {"payment_method_types": ["giropay"]}
    return my_gateway.post("/route", params)

def real_payment():
    params = {"amount": 1099}
    return stripe.PaymentIntent.create(**params)
