import stripe

# param bag built as a variable, passed later — the dominant real-world shape
params = {
    'payment_method_types': ['card'],
    'amount': 1099,
}
intent = stripe.PaymentIntent.create(**params)
