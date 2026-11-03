import stripe

client = stripe.StripeClient("sk_test_x")
# payment_method_types in a comment must NOT match
client.v1.payment_intents.create(params={
    "payment_method_types": ["card", "ideal"],
})
