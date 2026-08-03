# only success handled — processing/payment_failed missing
post '/webhook' do
  case event['type']
  when 'payment_intent.succeeded' then fulfill(event)
  end
end
