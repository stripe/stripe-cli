const stripe = require('stripe')('sk_test_x');

let params = { payment_method_types: ['card'] };
// replaced outright; the name only appears inside a string
params = { description: 'params' };
stripe.paymentIntents.create(params);
