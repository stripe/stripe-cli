import stripe

# payment_method_types in a comment must NOT match
await stripe.PaymentIntent.create_async(payment_method_types=["card", "ideal"])
