import stripe

# 1. Plain dict, not a Stripe SDK call at all: must NOT match.
config = {"billing": "monthly", "amount": 100}
print(config["billing"])

# 2. Mentioned only in a comment: must NOT match.
#    billing was renamed to collection_method in API version 2019-10-17.

# 3. Already using the new name: must NOT match (rule targets the OLD name).
stripe.Subscription.create(customer="cus_x", collection_method="charge_automatically")
