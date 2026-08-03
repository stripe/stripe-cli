<?php
$stripe = new \Stripe\StripeClient('sk_test_x');
$stripe->paymentIntents->create([
  'amount' => 1099,
  'currency' => 'eur',
  'allowed_source_types' => ['card'],
]);
