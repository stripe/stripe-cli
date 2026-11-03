import com.stripe.param.PaymentIntentCreateParams;

class Demo {
  void go() {
    // payment_method_types in a comment must NOT match
    PaymentIntentCreateParams params = PaymentIntentCreateParams.builder()
        .setAmount(1099L)
        .setCurrency("eur")
        .addPaymentMethodType("card")
        .build();
  }
}
