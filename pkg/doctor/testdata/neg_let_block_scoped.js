const stripe = require('stripe')('sk_test_x');

function createIntent(debug) {
  if (debug) {
    // block-scoped to the if-branch only — never reaches the call
    // below, which sits outside the branch entirely
    const params = { payment_method_types: ['card'] };
    console.log(params);
  }
  return stripe.paymentIntents.create(params);
}
