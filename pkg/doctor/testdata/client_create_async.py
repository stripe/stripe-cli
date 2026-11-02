import stripe

client = stripe.StripeClient("sk_test_x")
# payment_method_types in a comment must NOT match
await client.v1.payment_intents.create_async(
    params={"payment_method_types": ["card", "ideal"]},
)
