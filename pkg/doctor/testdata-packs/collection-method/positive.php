<?php
// Subscription create — top-level billing
$stripe->subscriptions->create([
  'customer' => 'cus_x',
  'billing' => 'charge_automatically',
]);
