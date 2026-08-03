<?php
$stripe = new \Stripe\StripeClient('sk_test_x');

$stripe->subscriptions->update('sub_123', [
  'tax_percent' => 19.0,
]);

// legacy static-call SDK style
\Stripe\Invoice::create([
  'customer' => 'cus_123',
  'tax_percent' => 10.0,
]);
