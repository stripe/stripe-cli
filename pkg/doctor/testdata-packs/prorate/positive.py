import stripe

stripe.Subscription.modify(
    "sub_1MnGA42eZvKYlo2ClPYPojkP",
    prorate=True,
)
