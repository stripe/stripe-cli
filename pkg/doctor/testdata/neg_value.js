const stripe = require('stripe')('sk_test_x');

// "payment_method_types" is a string VALUE here, not a parameter name
await stripe.paymentIntents.create({ description: 'payment_method_types' });
