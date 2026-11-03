using Stripe;

class D {
  void Go() {
    // target-typed new(): the type is UnrelatedOptions; the resource
    // token only appears as a string value in the initializer
    UnrelatedOptions options = new()
    {
      Description = "PaymentIntentCreateOptions",
      PaymentMethodTypes = formatted,
    };
  }
}
