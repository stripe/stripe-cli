// legacy Card Element frontend — must produce a frontend warning signal,
// but its paymentMethodTypes-style options are NOT server findings
const card = elements.create('card');
stripe.confirmCardPayment(clientSecret, { payment_method: { card } });
