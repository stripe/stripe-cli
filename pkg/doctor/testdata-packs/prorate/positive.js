const stripe = require('stripe')('sk_test_x');

await stripe.subscriptions.update('sub_1MnGA42eZvKYlo2ClPYPojkP', {
  prorate: false,
});

await stripe.subscriptionItems.create({
  subscription: 'sub_1MnGA42eZvKYlo2ClPYPojkP',
  price: 'price_1MoBy5LkdIwHu7ixZhnattbh',
  prorate: true,
});
