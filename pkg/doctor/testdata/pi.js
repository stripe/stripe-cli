const stripe = require('stripe')('sk_test_x');
// payment_method_types in a comment must NOT match
await stripe.paymentIntents.create({
  amount: 1099,
  currency: 'eur',
  payment_method_types: ['card', 'ideal'],
});
