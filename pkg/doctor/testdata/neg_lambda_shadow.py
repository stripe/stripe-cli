import stripe

# module-level params dict — must NOT be confused with the lambda's own
# parameter of the same name, which shadows it for the lambda's body
params = {"payment_method_types": ["card"]}

f = lambda params: stripe.PaymentIntent.create(**params)
