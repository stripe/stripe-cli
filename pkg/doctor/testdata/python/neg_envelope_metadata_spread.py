from stripe import StripeClient

client = StripeClient('sk_test_x')
# spread into an unrelated keyword, not the request envelope
params = {'payment_method_types': ['card']}
client.v1.payment_intents.create(metadata={**params}, params={'amount': 1000})
