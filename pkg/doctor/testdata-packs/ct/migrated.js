// already on Confirmation Tokens — must produce no findings and no signals
const {error, confirmationToken} = await stripe.createConfirmationToken({elements});
const intent = await stripe.paymentIntents.create({
  confirm: true,
  amount: 1099,
  currency: 'usd',
  use_stripe_sdk: true,
  confirmation_token: req.body.confirmationTokenId,
});
