import com.stripe.param.PaymentIntentCreateParams;

class Recv {
  void go() {
    PaymentIntentCreateParams.Builder paramsBuilder = new PaymentIntentCreateParams.Builder()
        .setAmount(1099L);
    // resolved through the receiver's declared type, not the chain
    paramsBuilder.addPaymentMethodType("card");
  }
}
