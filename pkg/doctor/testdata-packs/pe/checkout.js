// legacy per-payment-method Elements, client-side confirm
const elements = stripe.elements();
const card = elements.create('card');
card.mount('#card-element');
await stripe.confirmCardPayment(clientSecret, {payment_method: {card}});
