import stripe

# construct_from builds a local object from already-received data — it
# makes no API request at all, so its fields must NOT generate removal
# findings the way an actual create() call would.
intent = stripe.PaymentIntent.construct_from(
    {"payment_method_types": ["card", "ideal"]},
    "sk_test_x",
)
