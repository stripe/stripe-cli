const stripe = require('stripe')('sk_test_x');

// prorate only ever applied to Subscription and SubscriptionItem, never
// PaymentIntent -- must NOT match
await stripe.paymentIntents.update('pi_3MtwBwLkdIwHu7ix28a3tqPa', {
  prorate: true,
});
