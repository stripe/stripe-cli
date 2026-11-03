const stripe = require('stripe')('sk_test_x');

let params = { payment_method_types: ['card'], amount: 1099 };

function createIntent(useDefaults) {
  // braced form of closure_conditional_write.js: a write to the captured
  // params inside an if block, which may not run
  if (useDefaults) {
    params = {};
  }
  return stripe.paymentIntents.create(params);
}
