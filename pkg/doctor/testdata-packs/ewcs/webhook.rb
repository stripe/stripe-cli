# old-world fulfillment: PaymentIntent events only — the four
# checkout.session.* events the pack expects are all absent
post '/webhook' do
  case event['type']
  when 'payment_intent.succeeded' then fulfill(event)
  end
end
