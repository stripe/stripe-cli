const stripe = require('stripe')('sk_test_x');

// param bag built as a variable, merged into the request via object spread
const base = {
  payment_method_types: ['card', 'ideal'],
};
await stripe.paymentIntents.create({ ...base, amount: 1000, currency: 'usd' });
