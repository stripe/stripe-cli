// one corner already migrated
await stripe.checkout.sessions.create({mode: 'payment', automatic_payment_methods: {enabled: true}});
