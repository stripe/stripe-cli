const stripe = require('stripe')('sk_test_x');

let params = { amount: 1000 };
Object.assign(params, { payment_method_types: ['card'] });
// replaced with a clean bag before the call
params = { amount: 1000 };
stripe.paymentIntents.create(params);
