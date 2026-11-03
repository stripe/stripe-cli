import stripe

# the rule is scoped to POST /v1/payment_intents (create) only — a call
# to a different SDK method on the same resource must NOT match
stripe.PaymentIntent.modify(
    "pi_123",
    payment_method_types=["card", "ideal"],
)
