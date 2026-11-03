const stripe = require('stripe')('sk_test_x');

async function create(c) {
  let params = {};
  if (c) {
    params = { payment_method_types: ['card'] };
  }
  else {
    params = { payment_method_types: ['card'] };
  }
  return stripe.paymentIntents.create(params);
}
