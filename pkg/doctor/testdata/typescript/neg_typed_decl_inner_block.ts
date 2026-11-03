let params: Stripe.PaymentIntentCreateParams;

{
  // a different, untyped params scoped to this block
  let params;
  params = { payment_method_types: ['card'] };
}
