<?php
$stripe = new \Stripe\StripeClient('sk_test_x');

// the original bag is overwritten with a clean one before the call
$params = ['payment_method_types' => ['card'], 'amount' => 1099];
$params = [];
$stripe->paymentIntents->create($params);
