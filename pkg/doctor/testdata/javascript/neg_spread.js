const stripe = require('stripe')('sk_test_x');

const base = {
  payment_method_types: ['card'],
};
// base is nested under metadata, not spread into the top-level params — must NOT match
await stripe.paymentIntents.create({ metadata: base, amount: 1000 });
