# the doc's three delayed-notification cases — handler-signal fixture
post '/webhook' do
  case event['type']
  when 'checkout.session.completed' then create_order(event)
  when 'checkout.session.async_payment_succeeded' then fulfill_order(event)
  when 'checkout.session.async_payment_failed' then email_customer(event)
  end
end
