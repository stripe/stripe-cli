const stripe = require('stripe')('sk_test_x');

let params = { payment_method_types: ['card'], amount: 1099 };

function createIntent(useDefaults) {
  // a write (not a declaration) to the outer, captured params — only
  // replaces it when useDefaults is true, so the original bag still
  // reaches Stripe when it's false
  useDefaults && (params = {});
  return stripe.paymentIntents.create(params);
}
