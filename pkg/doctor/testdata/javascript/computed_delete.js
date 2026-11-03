const stripe = require('stripe')('sk_test_x');

const payment_method_types = 'description';
const params = { payment_method_types: ['card'], description: 'x' };
// deletes params.description at runtime, not the payment method key
delete params[payment_method_types];
stripe.paymentIntents.create(params);
