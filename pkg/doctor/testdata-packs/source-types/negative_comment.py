import stripe

# allowed_source_types was renamed to payment_method_types in API 2019-02-11
stripe.PaymentIntent.create(
    amount=1099,
    currency="eur",
    payment_method_types=["card"],
)

log("allowed_source_types")
