const stripe = require('stripe')('sk_test_x');

const params = { payment_method_types: ['card'], amount: 1099 };

function createIntent() {
  function helper() {
    // an unrelated local to this nested helper, never visible in
    // createIntent's own body — must NOT shadow the module-level
    // params reaching the call below
    const params = {};
    return params;
  }
  helper();
  return stripe.paymentIntents.create(params);
}
