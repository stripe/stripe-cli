const stripe = require('stripe')('sk_test_x');

// the original bag is overwritten with a clean one before the call —
// despite the mismatched lexical_declaration/expression_statement
// wrapper node types, these are still unconditional sibling statements
let params = { payment_method_types: ['card'], amount: 1099 };
params = {};
stripe.paymentIntents.create(params);
