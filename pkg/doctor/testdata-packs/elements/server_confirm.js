// server-confirmation flow, PaymentMethod handoff (legacy, unsupported)
const elements = stripe.elements({mode: 'payment', amount: 1099, currency: 'usd', paymentMethodCreation: 'manual'});
await elements.submit();
const {error, paymentMethod} = await stripe.createPaymentMethod({elements});
await fetch('/create-confirm-intent', {method: 'POST', body: JSON.stringify({paymentMethodId: paymentMethod.id})});
