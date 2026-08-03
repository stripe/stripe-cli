<?php
// billing on a non-matching resource (customers). Must NOT match.
$stripe->customers->create([
  'email' => 'a@example.com',
  'billing' => 'charge_automatically',
]);
