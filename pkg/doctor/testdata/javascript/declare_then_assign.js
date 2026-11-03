const stripe = require('stripe')('sk_test_x');

// declared empty, populated later — the declaration is the binding the
// later write fills in, not a separate one shadowing it
let params = {};
params = { payment_method_types: ['card'] };
stripe.paymentIntents.create(params);
