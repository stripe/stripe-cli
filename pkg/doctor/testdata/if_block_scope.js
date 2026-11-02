const stripe = require('stripe')('sk_test_x');

// module-level bag, sent as-is by the function below
const params = { payment_method_types: ['card'], amount: 1099 };

function createIntent(debug) {
  if (debug) {
    // block-scoped to the if-branch only — must NOT shadow the
    // module-level params reaching the call after this block
    const params = {};
    console.log(params);
  }
  return stripe.paymentIntents.create(params);
}
