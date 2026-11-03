const stripe = require('stripe')('sk_test_x');

// the reassignment merges the original bag back in, so
// payment_method_types still reaches Stripe
let params = { payment_method_types: ['card'] };
params = { ...params, amount: 1000 };
stripe.paymentIntents.create(params);
