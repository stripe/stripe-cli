const stripe = require('stripe')('sk_test_x');

const params = { payment_method_types: ['card'], amount: 1000 };
delete params.payment_method_types;
stripe.paymentIntents.create(params);
