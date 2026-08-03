import stripe

stripe.Subscription.create(
    customer="cus_123",
    items=[{"price": "price_123"}],
    tax_percent=15.0,
)
