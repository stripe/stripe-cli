const stripe = require('stripe')('sk_test_x');
// confirm-at-create with client PaymentMethod id and manual mandate_data
const intent = await stripe.paymentIntents.create({
  confirm: true,
  amount: 1099,
  currency: 'usd',
  automatic_payment_methods: {enabled: true},
  payment_method: req.body.paymentMethodId,
  use_stripe_sdk: true,
  mandate_data: {customer_acceptance: {type: 'online'}},
});
