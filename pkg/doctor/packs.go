package doctor

// packs.go — additional migration rule packs, authored from changelog and
// spec research (each match list grounded in docs.stripe.com/changelog and
// the vendored OpenAPI spec; see testdata-packs/<topic>/ for the per-pack
// fixture corpus and packs_test.go for exact expected counts).
//
// All four are action=advise: their remediations are renames or value
// rewrites, primitives the fixer does not have yet. `fix` refuses them with
// a pointer to the docs; `doctor` detects and explains.
//
// KNOWN ENGINE ISSUE (found during pack verification): resource-token
// matching is substring-based, so PascalSingular("subscription_schedules") =
// "SubscriptionSchedule" CONTAINS "Subscription" — a top-level param inside a
// SubscriptionSchedule call can spuriously satisfy a subscriptions-scoped
// match. Fix is word-boundary token matching in resolve.go (follow-up).

var taxPercentRule = Rule{
	ID:           "tax-migration/no-tax-percent",
	Severity:     "warn",
	Action:       "advise",
	IntroducedIn: "2020-08-27",
	Message:      "Replace `tax_percent` with `default_tax_rates` referencing a Stripe Tax Rate object \u2014 `tax_percent` was removed from Subscription and Invoice requests in API version 2020-08-27.",
	Docs:         "https://docs.stripe.com/billing/migration/taxes",
	Match: []ParamMatch{
		{Param: "tax_percent", Operations: []string{"POST /v1/subscriptions", "POST /v1/subscriptions/{subscription_exposed_id}", "POST /v1/invoices", "POST /v1/invoices/{invoice}"}},
	},
}

var collectionMethodRule = Rule{
	ID:           "billing-collection-method-rename/rename-billing",
	Severity:     "warn",
	Action:       "advise",
	IntroducedIn: "2019-10-17",
	Message:      "Rename `billing` to `collection_method`, its name as of API version 2019-10-17.",
	Docs:         "https://docs.stripe.com/changelog/2019-10-17/renames-billing-attribute",
	Match: []ParamMatch{
		{Param: "billing", Operations: []string{"GET /v1/invoices", "POST /v1/invoices", "POST /v1/invoices/{invoice}", "GET /v1/subscriptions", "POST /v1/subscriptions", "POST /v1/subscriptions/{subscription_exposed_id}"}},
		{Param: "phases.billing", Operations: []string{"POST /v1/subscription_schedules", "POST /v1/subscription_schedules/{schedule}"}},
		{Param: "default_settings.billing", Operations: []string{"POST /v1/subscription_schedules", "POST /v1/subscription_schedules/{schedule}"}},
	},
}

var prorateRule = Rule{
	ID:           "billing/prorate-proration-behavior",
	Severity:     "warn",
	Action:       "advise",
	IntroducedIn: "2020-08-27",
	Message:      "Replace `prorate` with `proration_behavior`: `prorate: true` becomes `proration_behavior: 'create_prorations'`, `prorate: false` becomes `proration_behavior: 'none'`.",
	Docs:         "https://docs.stripe.com/billing/subscriptions/prorations",
	Match: []ParamMatch{
		{Param: "prorate", Operations: []string{"POST /v1/subscriptions", "POST /v1/subscriptions/{subscription_exposed_id}", "POST /v1/subscription_items", "POST /v1/subscription_items/{item}"}},
	},
}

var sourceTypesRule = Rule{
	ID:           "pi-rename/allowed-source-types",
	Severity:     "warn",
	Action:       "advise",
	IntroducedIn: "2019-02-11",
	Message:      "Rename `allowed_source_types` to `payment_method_types` on PaymentIntents (create, update, confirm) per API version 2019-02-11 -- once renamed, the `dpm/no-payment-method-types` pack then applies to the resulting `payment_method_types` parameter.",
	Docs:         "https://docs.stripe.com/changelog/2019-02-11/renames-allowed-source-types-payment-method-types",
	Match: []ParamMatch{
		{Param: "allowed_source_types", Operations: []string{"POST /v1/payment_intents", "POST /v1/payment_intents/{intent}", "POST /v1/payment_intents/{intent}/confirm"}},
	},
}

// ---------- ewcs: Payment Element -> Elements with Checkout Sessions ----------
//
// Recommendation-class re-architecture (the from-state APIs are NOT
// deprecated and there is no version gate), so IntroducedIn is empty and the
// rule only anchors *inventory* advisories on the calls the migration
// replaces. The real value is the signal set: from-state client tokens,
// the expected checkout.session.* webhook events, and Stripe.js floors.
// Known limit (documented): zero-argument setup_intents.create() cannot be
// anchored by a (param, operations) match — needs the operation-level match
// primitive.
var ewcsRule = Rule{
	ID:       "ewcs/migrate-to-checkout-sessions",
	Severity: "info",
	Action:   "advise",
	Message:  "Stripe recommends migrating this create call to POST /v1/checkout/sessions with ui_mode:'elements' (Elements with Checkout Sessions); amount/currency move into line_items[].price_data.",
	Docs:     "https://docs.stripe.com/payments/payment-element/migration-ewcs",
	Match: []ParamMatch{{
		Param: "amount",
		Operations: []string{
			"POST /v1/payment_intents",
		},
	}, {
		Param: "items",
		Operations: []string{
			"POST /v1/subscriptions",
		},
	}},
}

var ewcsSignals = PackSignals{
	WebhookEvents: []string{
		"checkout.session.completed",
		"checkout.session.async_payment_succeeded",
		"checkout.session.async_payment_failed",
		"checkout.session.expired",
	},
	FrontendTokens: []FrontendToken{
		{Token: "stripe.elements(", Note: "replace with stripe.initCheckoutElementsSdk({clientSecret}) per the EWCS migration"},
		{Token: "confirmCardPayment", Note: "replace with actions.confirm() on the Checkout instance"},
		{Token: "confirmCardSetup", Note: "replace with actions.confirm() on the Checkout instance"},
		{Token: "confirmP24Payment", Note: "replace with actions.confirm() on the Checkout instance"},
		{Token: "confirmP24Setup", Note: "replace with actions.confirm() on the Checkout instance"},
		{Token: "elements.create('card'", Note: "replace with checkout.createPaymentElement()"},
		{Token: `elements.create("card"`, Note: "replace with checkout.createPaymentElement()"},
		{Token: "setup_future_usage", Note: "client-side save moves to server-side saved_payment_method_options + actions.confirm({savePaymentMethod:true})"},
		{Token: "js.stripe.com/v3", Note: "pin the script tag to js.stripe.com/dahlia/stripe.js (Stripe.js v8)"},
		{Token: "'@stripe/react-stripe-js'", Note: "imports move to '@stripe/react-stripe-js/checkout' (CheckoutElementsProvider / useCheckoutElements)"},
		{Token: `"@stripe/react-stripe-js"`, Note: "imports move to '@stripe/react-stripe-js/checkout' (CheckoutElementsProvider / useCheckoutElements)"},
	},
	ManifestFloors: []ManifestFloor{
		{Package: "@stripe/stripe-js", Min: "8.0.0"},
		{Package: "@stripe/react-stripe-js", Min: "5.0.0"},
	},
}

// dpmSignals is the DPM pack's declaration of what used to be hard-coded in
// doctor.go: the delayed-notification events and legacy Card Element tokens.
var dpmSignals = PackSignals{
	WebhookEvents: []string{
		"checkout.session.completed",
		"checkout.session.async_payment_succeeded",
		"checkout.session.async_payment_failed",
	},
	FrontendTokens: []FrontendToken{
		{Token: "confirmCardPayment", Note: "legacy Card Element flow — dashboard-managed methods cannot render there"},
		{Token: "createToken(", Note: "legacy Card Element flow — dashboard-managed methods cannot render there"},
		{Token: "elements.create('card'", Note: "legacy Card Element flow — dashboard-managed methods cannot render there"},
		{Token: `elements.create("card"`, Note: "legacy Card Element flow — dashboard-managed methods cannot render there"},
	},
}

// ---------- flex: flexible payment features beta -> GA ----------
//
// Beta->GA migration (account-gated, not version-gated: IntroducedIn cannot
// express beta enrollment and is left empty). Only two request-side anchors
// exist in the entire doc; the rest is param ADDITION (invisible to a
// presence matcher) and response/webhook semantics carried in the messages.
var flexRule = Rule{
	ID:       "flex/beta-to-ga",
	Severity: "warn",
	Action:   "advise",
	Message:  "Flexible payment features beta->GA: rename request_incremental_authorization_support=true to request_incremental_authorization='if_available'; overcapture/extended-auth/multicapture now require explicit payment_method_options[card][request_*]='if_available'; partial captures emit charge.updated (not charge.captured); final_capture=false while fully capturing now returns HTTP 400.",
	Docs:     "https://docs.stripe.com/payments/flexible-features-migration",
	Match: []ParamMatch{{
		Param: "payment_method_options.card.request_incremental_authorization_support",
		Operations: []string{
			"POST /v1/payment_intents",
			"POST /v1/payment_intents/{intent}",
			"POST /v1/payment_intents/{intent}/confirm",
		},
	}, {
		Param: "final_capture",
		Operations: []string{
			"POST /v1/payment_intents/{intent}/capture",
		},
	}},
}

// ---------- pe: legacy per-payment-method Elements -> Payment Element ----------
//
// From-state: Card/iDEAL/P24 Elements with the confirm<PM>Payment/Setup call
// family on direct PaymentIntents/SetupIntents. The guide prints no server
// before-code, so the pack is signal-led: no reliable server anchor exists
// (payment_method_types-without-automatic_payment_methods is corroborating
// evidence only — and dpm's territory). Note the guide itself steers most
// users to ewcs; the triage topic carries that decision.
var peRule = Rule{
	ID:       "pe/legacy-elements-to-payment-element",
	Severity: "info",
	Action:   "advise",
	Message:  "Migrate legacy per-payment-method Elements to the single Payment Element: elements.create(\"payment\") + elements.submit() + stripe.confirmPayment/confirmSetup; server adds automatic_payment_methods[enabled]=true.",
	Docs:     "https://docs.stripe.com/payments/payment-element/migration",
	Match:    []ParamMatch{},
}

var peSignals = PackSignals{
	// The migrated flow is webhook-driven; the guide names these three.
	WebhookEvents: []string{
		"payment_intent.succeeded",
		"payment_intent.processing",
		"payment_intent.payment_failed",
	},
	FrontendTokens: []FrontendToken{
		{Token: "confirmCardPayment", Note: "replace with elements.submit() + stripe.confirmPayment({elements, clientSecret, confirmParams}) — EXEMPT if used with a 'cardCvc' Element for CVC recollection of saved cards"},
		{Token: "confirmP24Payment", Note: "whole confirm<PM>Payment family collapses into stripe.confirmPayment"},
		{Token: "confirmCardSetup", Note: "replace with elements.submit() + stripe.confirmSetup({elements, clientSecret, confirmParams})"},
		{Token: "confirmP24Setup", Note: "whole confirm<PM>Setup family collapses into stripe.confirmSetup"},
		{Token: "elements.create('card'", Note: "per-payment-method Elements are replaced by elements.create(\"payment\")"},
		{Token: `elements.create("card"`, Note: "per-payment-method Elements are replaced by elements.create(\"payment\")"},
	},
}

// ---------- ct: PaymentMethod server-handoff -> Confirmation Tokens ----------
//
// From-state: Payment Element, server-side confirmation, client
// stripe.createPaymentMethod({elements,...}) handing paymentMethod.id to the
// server which passes payment_method on a confirm:true PaymentIntent create.
// Stripe brands that from-state UNSUPPORTED, so this is effectively mandatory
// for its audience. Server anchor choice: `payment_method` is hopelessly
// broad and `use_stripe_sdk`/`confirm` PERSIST after migration; only
// `mandate_data` is from-state-specific (the ConfirmationToken carries it
// afterwards) — partial recall, honest precision. Detection burden is on the
// signals (createPaymentMethod is the discriminator).
//
// Doc trap recorded during mapping: migration-ct.md's markdown rendering
// DROPS the removed lines from its Before snippets — the authoritative
// from-state code lives on finalize-payments-on-the-server-legacy.md.
var ctRule = Rule{
	ID:       "ct/payment-method-handoff-to-confirmation-tokens",
	Severity: "warn",
	Action:   "advise",
	Message:  "Server-side confirmation with a manually assembled mandate_data marks the legacy PaymentMethod handoff (branded unsupported): switch the client to stripe.createConfirmationToken({elements,...}) and pass confirmation_token instead of payment_method — mandate_data/shipping/return_url then ride on the token.",
	Docs:     "https://docs.stripe.com/payments/payment-element/migration-ct",
	Match: []ParamMatch{{
		Param: "mandate_data",
		Operations: []string{
			"POST /v1/payment_intents",
			"POST /v1/payment_intents/{intent}/confirm",
		},
	}},
}

var ctSignals = PackSignals{
	FrontendTokens: []FrontendToken{
		{Token: "stripe.createPaymentMethod", Note: "replace with stripe.createConfirmationToken({elements, params:{payment_method_data,...}}) — the from-state is branded unsupported"},
		{Token: "paymentMethodId", Note: "the client->server handoff key becomes confirmationTokenId (confirmationToken.id)"},
	},
}

// ---------- elements: the "which of the three guides applies?" triage ----------
//
// The three Payment Element migration guides are disjoint by from-state plus
// one orthogonal axis; a repo can be in several states at once. This topic
// detects each from-state with file-level evidence and names the guide.
var elementsRule = Rule{
	ID:       "elements/triage",
	Severity: "info",
	Action:   "advise",
	Message:  "Payment Element migration triage — see the 'Which migration applies' section of this report.",
	Docs:     "https://docs.stripe.com/payments/payment-element",
	Match:    []ParamMatch{},
}

var elementsTriage = []TriageBranch{
	{
		Detected:  "Charges-era integration (tokens / direct charge creation)",
		Recommend: "PREREQUISITE: migrate Charges -> PaymentIntents first (docs.stripe.com/payments/payment-intents/migration) — none of the Payment Element guides apply until then",
		Tokens:    []string{"stripe.createToken(", "createToken(", "/v1/charges", "Stripe::Charge.create", "stripe.charges.create", "Charge.create("},
	},
	{
		Detected:  "legacy per-payment-method Elements with CLIENT-side confirmation (Card Element era)",
		Recommend: "migrate the UI — topic `ewcs` (Stripe-recommended: Checkout Sessions manage tax/discounts/subscriptions/shipping/currency) or topic `pe` (minimal change, keep your own PaymentIntent orchestration)",
		Tokens:    []string{"confirmCardPayment", "confirmCardSetup", "confirmP24Payment", "confirmP24Setup", "elements.create('card'", `elements.create("card"`},
	},
	{
		Detected:  "Payment Element with SERVER-side confirmation via PaymentMethod handoff",
		Recommend: "topic `ct`: adopt Confirmation Tokens — this from-state is branded unsupported by Stripe, making the migration effectively mandatory (orthogonal to pe/ewcs; can compose with either)",
		Tokens:    []string{"stripe.createPaymentMethod", "paymentMethodCreation: 'manual'", `paymentMethodCreation: "manual"`, "handleNextAction"},
	},
	{
		Detected:  "already-migrated markers (to-state code present)",
		Recommend: "these files are on a CURRENT integration shape — cross-check against the from-states above; mixed states usually mean a migration is partially done",
		Tokens:    []string{"initCheckoutElementsSdk", "createConfirmationToken", "confirmation_token", `elements.create("payment"`, "elements.create('payment'", "automatic_payment_methods"},
	},
}

var dpmRule = Rule{
	ID:           "dpm/no-payment-method-types",
	Severity:     "warn",
	Action:       "remove",
	IntroducedIn: "2023-08-16",
	Message:      "Remove `payment_method_types` so payment methods are managed in the Dashboard.",
	Docs:         "https://docs.stripe.com/payments/payment-methods/dynamic-payment-methods",
	// Below the cutoff, a bare PaymentIntent/SetupIntent defaults to
	// card-only (verified live) — the doc's migration there is
	// remove-and-replace, not remove. Checkout Sessions and Payment Links
	// have no automatic_payment_methods parameter, so no companion there.
	Companion: &Companion{
		Param:     "automatic_payment_methods",
		ForParam:  "payment_method_types",
		Resources: []string{"payment_intents", "setup_intents"},
	},
	Match: []ParamMatch{{
		Param: "payment_method_types",
		Operations: []string{
			"POST /v1/payment_intents",
			"POST /v1/payment_intents/{intent}",
			"POST /v1/payment_intents/{intent}/confirm",
			"POST /v1/setup_intents",
			"POST /v1/setup_intents/{intent}",
			"POST /v1/checkout/sessions",
			"POST /v1/payment_links",
			"POST /v1/payment_links/{payment_link}",
		},
	}, {
		Param: "payment_settings.payment_method_types",
		Operations: []string{
			"POST /v1/subscriptions",
			"POST /v1/subscriptions/{subscription_exposed_id}",
			"POST /v1/invoices",
			"POST /v1/invoices/{invoice}",
		},
	}},
}

// ---------- Findings ----------
