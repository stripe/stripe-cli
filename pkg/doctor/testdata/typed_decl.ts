// bag declared with an explicit params type, then assigned separately —
// exercises the same type: resolution path as C#'s target-typed new()
let params: Stripe.PaymentIntentCreateParams;
params = {
  payment_method_types: ['card', 'ideal'],
};
