# oldest era: tokens + charges
token = Stripe::Token.create(card: params)
Stripe::Charge.create(amount: 1099, currency: 'usd', source: token.id)
