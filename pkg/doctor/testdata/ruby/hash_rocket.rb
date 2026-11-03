require 'stripe'

# payment_method_types in a comment must NOT match
Stripe::PaymentIntent.create(:payment_method_types => ['card'])
Stripe::SetupIntent.create("payment_method_types" => ['card'])
