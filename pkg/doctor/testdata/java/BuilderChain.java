import com.stripe.param.PaymentIntentCreateParams;

class BuilderChain {
  void go() {
    PaymentIntentCreateParams.Builder b = PaymentIntentCreateParams.builder();
    b.addPaymentMethodType("card").setAmount(1000L);
  }
}
