// an unrelated nested function's own typed declaration of the same
// name must not satisfy a lookup from the outer, module-level scope
function helper() {
  let params: Stripe.PaymentIntentCreateParams;
  params = {};
}

let params = {
  payment_method_types: ['card'],
};
