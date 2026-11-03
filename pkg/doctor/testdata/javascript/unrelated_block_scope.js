const stripe = require('stripe')('sk_test_x');

// module-level bag, sent as-is below
const params = { payment_method_types: ['card'], amount: 1099 };

if (someCondition) {
  // unrelated, block-scoped const in a totally different branch — must
  // NOT be treated as reassigning the outer params above
  const params = {};
  console.log(params);
}

stripe.paymentIntents.create(params);
