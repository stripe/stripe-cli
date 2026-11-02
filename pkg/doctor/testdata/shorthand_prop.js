const stripe = require('stripe')('sk_test_x');

const payment_method_types = ['card'];
stripe.paymentIntents.create({ payment_method_types, amount: 1099 });
