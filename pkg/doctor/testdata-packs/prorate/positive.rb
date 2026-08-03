require 'stripe'

# prorate is deprecated; migrate to proration_behavior
Stripe::Subscription.update(
  'sub_1MnGA42eZvKYlo2ClPYPojkP',
  prorate: false,
)

Stripe::SubscriptionItem.create(
  subscription: 'sub_1MnGA42eZvKYlo2ClPYPojkP',
  price: 'price_1MoBy5LkdIwHu7ixZhnattbh',
  prorate: true,
)
