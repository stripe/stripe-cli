const stripe = require('stripe')('sk_test_x');

let params;
if (flag) {
  params = { payment_method_types: ['card'] };
}
params = {};
stripe.paymentIntents.create(params);
