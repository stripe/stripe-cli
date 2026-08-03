import stripe

# prorate is deprecated on Subscription/SubscriptionItem; noted here for
# reference only -- neither line below is ever passed to a Stripe call.
prorate = True
settings = {"prorate": True}
