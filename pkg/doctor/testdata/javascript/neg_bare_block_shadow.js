const stripe = require('stripe')('sk_test_x');

// module-level bag, must NOT be used below
const params = { payment_method_types: ['card'], amount: 1099 };

{
  // a bare block (no if/for/function) still introduces its own
  // let/const scope — the call here binds to THIS params, not the
  // module-level one above
  const params = {};
  stripe.paymentIntents.create(params);
}
