// an outer, untyped bag reaching nothing itself must NOT be confirmed
// through an unrelated inner block's own typed re-declaration of the
// same name — only the clean inner object actually reaches Stripe
let options = {
  payment_method_types: ['card'],
};

{
  let options: Stripe.PaymentIntentCreateParams;
  options = {};
}
