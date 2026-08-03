require 'stripe'

# allowed_source_types here targets the Sources resource, not PaymentIntents -- must NOT match this pack
Stripe::Source.create(
  type: 'card',
  allowed_source_types: ['card'],
)
