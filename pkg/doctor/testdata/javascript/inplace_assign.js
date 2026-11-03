const stripe = require('stripe')('sk_test_x');

const params = { amount: 1000 };
Object.assign(params, { payment_method_types: ['card'] });
stripe.paymentIntents.create(params);
