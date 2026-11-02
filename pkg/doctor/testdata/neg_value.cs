// PaymentMethodTypes here is an unrelated local variable whose name
// happens to match the parameter spelling — it's used as a VALUE, not
// an initializer field name, so it must NOT match.
var PaymentMethodTypes = "some description";
var options = new PaymentIntentCreateOptions { Description = PaymentMethodTypes };
