const stripe = require('stripe')('sk_test_x');

// the callback's own local object is never passed to create — it's
// not a request argument at all, just something built inside the
// callback body
stripe.paymentIntents.create({ amount: 1099 }, function (err, intent) {
  const options = { payment_method_types: ['card'] };
  console.log(options);
});
