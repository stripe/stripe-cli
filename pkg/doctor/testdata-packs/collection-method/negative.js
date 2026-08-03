// Local, non-Stripe object — never reaches a Stripe call. Must NOT match.
const order = { billing: 'net_30', total: 42 };
saveOrder(order);
