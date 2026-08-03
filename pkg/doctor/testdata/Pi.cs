using Stripe;

class Demo {
  void Go() {
    // payment_method_types in a comment must NOT match
    var options = new PaymentIntentCreateOptions
    {
      Amount = 1099,
      Currency = "eur",
      PaymentMethodTypes = new List<string> { "card", "ideal" },
    };
  }
}
