import Stripe from 'stripe';
const stripe = new Stripe('sk_test_x');

export async function createIntent() {
  await stripe.paymentIntents.create({
    amount: 1099,
    payment_method_types: ['card', 'link'],
  });
  return <div>ok</div>;
}
