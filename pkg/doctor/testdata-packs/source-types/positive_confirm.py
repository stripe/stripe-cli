import stripe

stripe.PaymentIntent.confirm(
    "pi_123",
    allowed_source_types=["card"],
)
