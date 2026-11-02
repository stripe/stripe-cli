<?php
$stripe = new \Stripe\StripeClient('sk_test_x');
// "payment_method_types" is a string VALUE here, not a parameter name
$stripe->paymentIntents->create(['description' => 'payment_method_types']);
