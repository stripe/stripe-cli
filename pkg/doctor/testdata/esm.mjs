import Stripe from 'stripe';
const stripe = new Stripe('sk_test_x');
await stripe.paymentIntents.create({
  payment_method_types: ['card'],
  amount: 1099,
});
