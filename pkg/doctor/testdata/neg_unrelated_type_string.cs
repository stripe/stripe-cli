using Stripe;

class Demo {
  void Go() {
    // the resource token only appears inside a string VALUE, never as
    // the constructed type itself — UnrelatedOptions is not a Stripe
    // params type at all
    var options = new UnrelatedOptions
    {
      Description = "PaymentIntentCreateOptions",
      PaymentMethodTypes = new List<string> { "card", "ideal" },
    };
  }
}
