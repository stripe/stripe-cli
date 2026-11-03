import stripe

# dict(...) nested under metadata, not the request's own params
stripe.PaymentIntent.create(amount=1000, metadata=dict(payment_method_types='card'))
