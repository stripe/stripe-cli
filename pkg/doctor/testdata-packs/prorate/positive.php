<?php
$stripe = new \Stripe\StripeClient('sk_test_x');

$stripe->subscriptions->update('sub_1MnGA42eZvKYlo2ClPYPojkP', [
  'prorate' => false,
]);
