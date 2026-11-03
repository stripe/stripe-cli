const stripe = require('stripe')('sk_test_x');

let params = { payment_method_types: ['card'], amount: 1099 };
// brace-less form: only overwrites params when the condition is true,
// so the original bag still reaches Stripe when it's false
if (useDefaults) params = { amount: 1000 };
stripe.paymentIntents.create(params);
