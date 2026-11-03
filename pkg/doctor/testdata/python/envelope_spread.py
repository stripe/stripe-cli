from stripe import StripeClient

client = StripeClient('sk_test_x')
params = {'payment_method_types': ['card']}
client.v1.payment_intents.create(params={**params, 'amount': 1000})
