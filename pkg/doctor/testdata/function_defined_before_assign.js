const stripe = require('stripe')('sk_test_x');

// defined (and textually positioned) before params is assigned below,
// but only actually invoked afterward — the call legitimately sees the
// assigned value at runtime
function sendPayment() {
  return stripe.paymentIntents.create(params);
}

const params = {
  payment_method_types: ['card', 'ideal'],
};
sendPayment();
