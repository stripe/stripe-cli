import stripe
from stripe import StripeClient

stripe.PaymentIntent.create(**dict(payment_method_types=['card']))
StripeClient('sk_test_x').v1.payment_intents.create(params=dict(payment_method_types=['card']))
