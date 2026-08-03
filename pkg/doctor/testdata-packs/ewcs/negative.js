// already migrated: checkout session create carries no top-level amount,
// and amount on a non-matching resource must not fire either
await stripe.checkout.sessions.create({
  line_items: [{price_data: {currency: 'eur', unit_amount: 1099, product_data: {name: 'T'}}, quantity: 1}],
  mode: 'payment', ui_mode: 'elements', return_url: 'https://x/return',
});
await stripe.refunds.create({ amount: 500 });
