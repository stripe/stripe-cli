import stripe

client = stripe.StripeClient("sk_test_x")

# param bag built as a variable, passed via the reserved "params" keyword
params = {
    'payment_method_types': ['card'],
    'amount': 1099,
}
intent = client.v1.payment_intents.create(params=params)
