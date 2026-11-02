import com.stripe.param.PaymentIntentCreateParams;
import java.util.Arrays;

class Demo {
  void go() {
    // payment_method_types in a comment must NOT match
    PaymentIntentCreateParams params = PaymentIntentCreateParams.builder()
        .setAmount(1099L)
        .setCurrency("eur")
        .addAllPaymentMethodType(Arrays.asList("card", "ideal"))
        .build();
  }
}
