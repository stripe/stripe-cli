const stripe = require('stripe')('sk_test_x');

stripe.paymentIntents.create(Object.assign({ amount: 1000 }, { payment_method_types: ['card'] }));
