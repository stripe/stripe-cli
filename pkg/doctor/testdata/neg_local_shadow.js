const stripe = require('stripe')('sk_test_x');

// module-level params object — must NOT be confused with the function's
// own local re-declaration below, which shadows it for the rest of the
// function body (not a formal parameter, so declaresParam alone misses it)
const params = {
  payment_method_types: ['card'],
};

function createIntent() {
  const params = {};
  return stripe.paymentIntents.create(params);
}
