const stripe = require('stripe')('sk_test_x');
// final_capture semantics changed in GA — should ADVISE
await stripe.paymentIntents.capture('pi_x', {
  amount_to_capture: 400,
  final_capture: false,
});
