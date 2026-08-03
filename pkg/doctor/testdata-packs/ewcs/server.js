const stripe = require('stripe')('sk_test_x');
// pre-migration: direct PaymentIntent create — should ADVISE
await stripe.paymentIntents.create({
  amount: 1099,
  currency: 'eur',
});
