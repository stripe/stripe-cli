const stripe = require('stripe')('sk_test_x');

// params is nested under metadata, not merged into the create() call's
// own params — and "intent"'s untyped declarator must NOT be mistaken
// for a typed declaration just because its initializer mentions
// paymentIntents
const params = { payment_method_types: 'card' };
const intent = stripe.paymentIntents.create({ metadata: params });
