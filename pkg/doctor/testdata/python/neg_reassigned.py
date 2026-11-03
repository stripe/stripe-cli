import stripe

# the original bag is overwritten with a clean one before the call —
# the call only ever sends the second, clean bag, not this one
params = {
    'payment_method_types': ['card'],
    'amount': 1099,
}
params = {
    'amount': 1099,
}
stripe.PaymentIntent.create(**params)
