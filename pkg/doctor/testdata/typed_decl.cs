using Stripe;

class D {
  void Go() {
    // target-typed new(): the type lives on the declaration, not the literal
    PaymentIntentCreateOptions options;
    options = new()
    {
      Amount = 1099,
      PaymentMethodTypes = formatted,
    };
  }
}
