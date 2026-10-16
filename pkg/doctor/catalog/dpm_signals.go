package catalog

// DpmSignals is a stopgap: the live doctor_rules API (see apirule.go,
// judge.FetchDPMRule) doesn't return a "signals" block yet, so this stays
// hardcoded here and is patched onto the Rule regardless of whether it
// came from the live fetch or the embedded dpm.json fallback. Remove this
// file once the API grows real signals support, and read WebhookEvents
// from the response instead.
var DpmSignals = PackSignals{
	WebhookEvents: []WebhookEvent{
		{Event: "checkout.session.completed", Note: "fires when checkout finishes — for a delayed method (OXXO, bank transfer, ...) that means the customer committed to pay, not that payment is confirmed yet"},
		{Event: "checkout.session.async_payment_succeeded", Note: "fires later, out of band, when a delayed payment method actually clears — without this handler, those orders never get fulfilled"},
		{Event: "checkout.session.async_payment_failed", Note: "fires later if a delayed payment method expires or fails — without this handler, those orders never get reconciled or refunded"},
	},
}
