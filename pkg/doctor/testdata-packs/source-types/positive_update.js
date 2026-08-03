const stripe = require('stripe')('sk_test_x');

await stripe.paymentIntents.update(intentId, {
  allowed_source_types: ['card'],
});
