const stripe = require('stripe')('sk_test_x');

let params = {};
[1].forEach(() => {
  params = { payment_method_types: ['card'] };
});
stripe.paymentIntents.create(params);
