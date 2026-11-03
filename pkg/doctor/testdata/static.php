<?php
// legacy static-call SDK style
$setup_intent = \Stripe\SetupIntent::create([
    'payment_method_types' => ['card', 'au_becs_debit'],
    'customer' => $customer_id,
]);
