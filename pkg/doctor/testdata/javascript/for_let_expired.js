const stripe = require('stripe')('sk_test_x');

const params = { payment_method_types: ['card'] };

function create() {
  // this params is scoped to the for statement alone
  for (let params = {}; false; ) {}
  return stripe.paymentIntents.create(params);
}
