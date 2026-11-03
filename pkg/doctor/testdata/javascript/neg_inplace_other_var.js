const stripe = require('stripe')('sk_test_x');

const params = { amount: 1000 };
const other = {};
// mutates a different object than the one sent
Object.assign(other, { payment_method_types: ['card'] });
stripe.paymentIntents.create(params);
