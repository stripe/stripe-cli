const stripe = require('stripe')('sk_test_x');

const base = { payment_method_types: ['card'] };
const params = { ...base, amount: 1000 };
stripe.paymentIntents.create(params);
