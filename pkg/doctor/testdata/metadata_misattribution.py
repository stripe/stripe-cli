import stripe

# bound to metadata, not merged into the call's own params — must NOT match
params = {"payment_method_types": "card"}
stripe.PaymentIntent.create(metadata=params)
