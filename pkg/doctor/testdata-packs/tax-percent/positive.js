const stripe = require('stripe')('sk_test_x');

await stripe.invoices.create({
  customer: 'cus_123',
  tax_percent: 21.0,
});
