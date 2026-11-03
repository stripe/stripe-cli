const stripe = require('stripe')('sk_test_x');

function create(params) {
  // params is a function parameter: no prior assignment in the file
  Object.assign(params, { payment_method_types: ['card'] });
  return stripe.paymentIntents.create(params);
}
