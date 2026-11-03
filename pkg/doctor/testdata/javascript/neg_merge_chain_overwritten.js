const stripe = require('stripe')('sk_test_x');

let base = { payment_method_types: ['card'] };
// replaced before it's merged into params
base = {};
const params = { ...base, amount: 1000 };
stripe.paymentIntents.create(params);
