const stripe = require('stripe')('sk_test_x');

// the comment between the method name and the argument list must NOT
// be swallowed into the extracted method name
stripe.paymentIntents.create /* comment */ ({
  payment_method_types: ['card', 'ideal'],
});
