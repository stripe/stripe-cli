# beta param at the WRONG nesting (top-level) must not match,
# and final_capture on a non-capture resource must not match
import stripe
stripe.PaymentIntent.create(
    amount=5000,
    currency="usd",
    request_incremental_authorization_support=True,
)
config = {"final_capture": False}
