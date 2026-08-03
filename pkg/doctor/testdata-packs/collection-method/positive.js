const stripe = require('stripe')(apiKey);

// Subscription create — top-level billing
stripe.subscriptions.create({
  customer: 'cus_x',
  billing: 'charge_automatically',
});

// Invoice update — top-level billing
stripe.invoices.update('in_x', {
  billing: 'send_invoice',
});
