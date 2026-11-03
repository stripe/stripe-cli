class NegBuilderArg {
  void go() {
    // the resource token is only a builder argument, not the builder's type
    new UnrelatedBuilder().setDescription("PaymentIntentCreateParams").addPaymentMethodType("card");
  }
}
