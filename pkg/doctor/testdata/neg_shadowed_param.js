const stripe = require('stripe')('sk_test_x');

// module-level params object — must NOT be confused with the function
// parameter of the same name below, which shadows it
const params = {
  payment_method_types: ['card'],
};

function createIntent(params) {
  return stripe.paymentIntents.create(params);
}
