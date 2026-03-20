// This file is generated; DO NOT EDIT.

package resources

import "github.com/stripe/stripe-cli/pkg/cmd/resource"

var V1CheckoutSessionsUpdate = resource.OperationSpec{
	Name:    "update",
	Path:    "/v1/checkout/sessions/{session}",
	Method:  "POST",
	Summary: "Update a Checkout Session",
	Params: map[string]*resource.ParamSpec{
		"collected_information.shipping_details.name": {
			Type:        "string",
			Description: "The name of customer",
			Required:    true,
		},
		"collected_information.shipping_details.address.city": {
			Type:        "string",
			Description: "City, district, suburb, town, or village.",
		},
		"collected_information.shipping_details.address.country": {
			Type:        "string",
			Description: "Two-letter country code ([ISO 3166-1 alpha-2](https://en.wikipedia.org/wiki/ISO_3166-1_alpha-2)).",
			Required:    true,
		},
		"collected_information.shipping_details.address.line1": {
			Type:        "string",
			Description: "Address line 1, such as the street, PO Box, or company name.",
			Required:    true,
		},
		"collected_information.shipping_details.address.line2": {
			Type:        "string",
			Description: "Address line 2, such as the apartment, suite, unit, or building.",
		},
		"collected_information.shipping_details.address.postal_code": {
			Type:        "string",
			Description: "ZIP or postal code.",
		},
		"collected_information.shipping_details.address.state": {
			Type:        "string",
			Description: "State, county, province, or region ([ISO 3166-2](https://en.wikipedia.org/wiki/ISO_3166-2)).",
		},
	},
}

var V1CheckoutSessionsExpire = resource.OperationSpec{
	Name:    "expire",
	Path:    "/v1/checkout/sessions/{session}/expire",
	Method:  "POST",
	Summary: "Expire a Checkout Session",
}

var V1CheckoutSessionsList = resource.OperationSpec{
	Name:    "list",
	Path:    "/v1/checkout/sessions",
	Method:  "GET",
	Summary: "List all Checkout Sessions",
	Params: map[string]*resource.ParamSpec{
		"created": {
			Type:        "integer",
			Description: "Only return Checkout Sessions that were created during the given date interval.",
		},
		"customer_account": {
			Type:        "string",
			Description: "Only return the Checkout Sessions for the Account specified.",
		},
		"ending_before": {
			Type:        "string",
			Description: "A cursor for use in pagination. `ending_before` is an object ID that defines your place in the list. For instance, if you make a list request and receive 100 objects, starting with `obj_bar`, your subsequent call can include `ending_before=obj_bar` in order to fetch the previous page of the list.",
		},
		"limit": {
			Type:        "integer",
			Description: "A limit on the number of objects to be returned. Limit can range between 1 and 100, and the default is 10.",
		},
		"payment_link": {
			Type:        "string",
			Description: "Only return the Checkout Sessions for the Payment Link specified.",
		},
		"starting_after": {
			Type:        "string",
			Description: "A cursor for use in pagination. `starting_after` is an object ID that defines your place in the list. For instance, if you make a list request and receive 100 objects, ending with `obj_foo`, your subsequent call can include `starting_after=obj_foo` in order to fetch the next page of the list.",
		},
		"subscription": {
			Type:        "string",
			Description: "Only return the Checkout Session for the subscription specified.",
		},
		"customer": {
			Type:        "string",
			Description: "Only return the Checkout Sessions for the Customer specified.",
		},
		"payment_intent": {
			Type:        "string",
			Description: "Only return the Checkout Session for the PaymentIntent specified.",
		},
		"status": {
			Type:        "string",
			Description: "Only return the Checkout Sessions matching the given status.",
			Enum: []resource.EnumSpec{
				{Value: "complete"},
				{Value: "expired"},
				{Value: "open"},
			},
		},
	},
}

var V1CheckoutSessionsRetrieve = resource.OperationSpec{
	Name:    "retrieve",
	Path:    "/v1/checkout/sessions/{session}",
	Method:  "GET",
	Summary: "Retrieve a Checkout Session",
}

var V1CheckoutSessionsListLineItems = resource.OperationSpec{
	Name:    "list_line_items",
	Path:    "/v1/checkout/sessions/{session}/line_items",
	Method:  "GET",
	Summary: "Retrieve a Checkout Session's line items",
	Params: map[string]*resource.ParamSpec{
		"ending_before": {
			Type:        "string",
			Description: "A cursor for use in pagination. `ending_before` is an object ID that defines your place in the list. For instance, if you make a list request and receive 100 objects, starting with `obj_bar`, your subsequent call can include `ending_before=obj_bar` in order to fetch the previous page of the list.",
		},
		"limit": {
			Type:        "integer",
			Description: "A limit on the number of objects to be returned. Limit can range between 1 and 100, and the default is 10.",
		},
		"starting_after": {
			Type:        "string",
			Description: "A cursor for use in pagination. `starting_after` is an object ID that defines your place in the list. For instance, if you make a list request and receive 100 objects, ending with `obj_foo`, your subsequent call can include `starting_after=obj_foo` in order to fetch the next page of the list.",
		},
	},
}

var V1CheckoutSessionsCreate = resource.OperationSpec{
	Name:    "create",
	Path:    "/v1/checkout/sessions",
	Method:  "POST",
	Summary: "Create a Checkout Session",
	Params: map[string]*resource.ParamSpec{
		"payment_method_options.payto.setup_future_usage": {
			Type:        "string",
			Description: "Indicates that you intend to make future payments with this PaymentIntent's payment method.\n\nIf you provide a Customer with the PaymentIntent, you can use this parameter to [attach the payment method](/payments/save-during-payment) to the Customer after the PaymentIntent is confirmed and the customer completes any required actions. If you don't provide a Customer, you can still [attach](/api/payment_methods/attach) the payment method to a Customer after the transaction completes.\n\nIf the payment method is `card_present` and isn't a digital wallet, Stripe creates and attaches a [generated_card](/api/charges/object#charge_object-payment_method_details-card_present-generated_card) payment method representing the card to the Customer instead.\n\nWhen processing card payments, Stripe uses `setup_future_usage` to help you comply with regional legislation and network rules, such as [SCA](/strong-customer-authentication).",
			Enum: []resource.EnumSpec{
				{Value: "none"},
				{Value: "off_session"},
			},
		},
		"payment_method_options.kr_card.setup_future_usage": {
			Type:        "string",
			Description: "Indicates that you intend to make future payments with this PaymentIntent's payment method.\n\nIf you provide a Customer with the PaymentIntent, you can use this parameter to [attach the payment method](/payments/save-during-payment) to the Customer after the PaymentIntent is confirmed and the customer completes any required actions. If you don't provide a Customer, you can still [attach](/api/payment_methods/attach) the payment method to a Customer after the transaction completes.\n\nIf the payment method is `card_present` and isn't a digital wallet, Stripe creates and attaches a [generated_card](/api/charges/object#charge_object-payment_method_details-card_present-generated_card) payment method representing the card to the Customer instead.\n\nWhen processing card payments, Stripe uses `setup_future_usage` to help you comply with regional legislation and network rules, such as [SCA](/strong-customer-authentication).",
			Enum: []resource.EnumSpec{
				{Value: "none"},
				{Value: "off_session"},
			},
		},
		"subscription_data.trial_settings.end_behavior.missing_payment_method": {
			Type:        "string",
			Description: "Indicates how the subscription should change when the trial ends if the user did not provide a payment method.",
			Required:    true,
			Enum: []resource.EnumSpec{
				{Value: "cancel"},
				{Value: "create_invoice"},
				{Value: "pause"},
			},
		},
		"payment_intent_data.shipping.phone": {
			Type:        "string",
			Description: "Recipient phone (including extension).",
		},
		"payment_method_options.wechat_pay.app_id": {
			Type:        "string",
			Description: "The app ID registered with WeChat Pay. Only required when client is ios or android.",
		},
		"payment_method_options.boleto.expires_after_days": {
			Type:        "integer",
			Description: "The number of calendar days before a Boleto voucher expires. For example, if you create a Boleto voucher on Monday and you set expires_after_days to 2, the Boleto invoice will expire on Wednesday at 23:59 America/Sao_Paulo time.",
		},
		"payment_method_options.alipay.setup_future_usage": {
			Type:        "string",
			Description: "Indicates that you intend to make future payments with this PaymentIntent's payment method.\n\nIf you provide a Customer with the PaymentIntent, you can use this parameter to [attach the payment method](/payments/save-during-payment) to the Customer after the PaymentIntent is confirmed and the customer completes any required actions. If you don't provide a Customer, you can still [attach](/api/payment_methods/attach) the payment method to a Customer after the transaction completes.\n\nIf the payment method is `card_present` and isn't a digital wallet, Stripe creates and attaches a [generated_card](/api/charges/object#charge_object-payment_method_details-card_present-generated_card) payment method representing the card to the Customer instead.\n\nWhen processing card payments, Stripe uses `setup_future_usage` to help you comply with regional legislation and network rules, such as [SCA](/strong-customer-authentication).",
			Enum: []resource.EnumSpec{
				{Value: "none"},
			},
		},
		"payment_method_options.link.setup_future_usage": {
			Type:        "string",
			Description: "Indicates that you intend to make future payments with this PaymentIntent's payment method.\n\nIf you provide a Customer with the PaymentIntent, you can use this parameter to [attach the payment method](/payments/save-during-payment) to the Customer after the PaymentIntent is confirmed and the customer completes any required actions. If you don't provide a Customer, you can still [attach](/api/payment_methods/attach) the payment method to a Customer after the transaction completes.\n\nIf the payment method is `card_present` and isn't a digital wallet, Stripe creates and attaches a [generated_card](/api/charges/object#charge_object-payment_method_details-card_present-generated_card) payment method representing the card to the Customer instead.\n\nWhen processing card payments, Stripe uses `setup_future_usage` to help you comply with regional legislation and network rules, such as [SCA](/strong-customer-authentication).",
			Enum: []resource.EnumSpec{
				{Value: "none"},
				{Value: "off_session"},
			},
		},
		"payment_method_options.paypal.preferred_locale": {
			Type:        "string",
			Description: "[Preferred locale](https://docs.stripe.com/payments/paypal/supported-locales) of the PayPal checkout page that the customer is redirected to.",
			Enum: []resource.EnumSpec{
				{Value: "cs-CZ"},
				{Value: "da-DK"},
				{Value: "de-AT"},
				{Value: "de-DE"},
				{Value: "de-LU"},
				{Value: "el-GR"},
				{Value: "en-GB"},
				{Value: "en-US"},
				{Value: "es-ES"},
				{Value: "fi-FI"},
				{Value: "fr-BE"},
				{Value: "fr-FR"},
				{Value: "fr-LU"},
				{Value: "hu-HU"},
				{Value: "it-IT"},
				{Value: "nl-BE"},
				{Value: "nl-NL"},
				{Value: "pl-PL"},
				{Value: "pt-PT"},
				{Value: "sk-SK"},
				{Value: "sv-SE"},
			},
		},
		"subscription_data.billing_cycle_anchor": {
			Type:        "integer",
			Description: "A future timestamp to anchor the subscription's billing cycle for new subscriptions. You can't set this parameter if `ui_mode` is `custom`.",
			Format:      "unix-time",
		},
		"payment_method_options.card.setup_future_usage": {
			Type:        "string",
			Description: "Indicates that you intend to make future payments with this PaymentIntent's payment method.\n\nIf you provide a Customer with the PaymentIntent, you can use this parameter to [attach the payment method](/payments/save-during-payment) to the Customer after the PaymentIntent is confirmed and the customer completes any required actions. If you don't provide a Customer, you can still [attach](/api/payment_methods/attach) the payment method to a Customer after the transaction completes.\n\nIf the payment method is `card_present` and isn't a digital wallet, Stripe creates and attaches a [generated_card](/api/charges/object#charge_object-payment_method_details-card_present-generated_card) payment method representing the card to the Customer instead.\n\nWhen processing card payments, Stripe uses `setup_future_usage` to help you comply with regional legislation and network rules, such as [SCA](/strong-customer-authentication).",
			Enum: []resource.EnumSpec{
				{Value: "off_session"},
				{Value: "on_session"},
			},
		},
		"payment_method_options.customer_balance.funding_type": {
			Type:        "string",
			Description: "The funding method type to be used when there are not enough funds in the customer balance. Permitted values include: `bank_transfer`.",
			Enum: []resource.EnumSpec{
				{Value: "bank_transfer"},
			},
		},
		"payment_method_options.acss_debit.currency": {
			Type:        "string",
			Description: "Three-letter [ISO currency code](https://www.iso.org/iso-4217-currency-codes.html), in lowercase. Must be a [supported currency](https://stripe.com/docs/currencies). This is only accepted for Checkout Sessions in `setup` mode.",
			Enum: []resource.EnumSpec{
				{Value: "cad"},
				{Value: "usd"},
			},
		},
		"payment_method_options.p24.tos_shown_and_accepted": {
			Type:        "boolean",
			Description: "Confirm that the payer has accepted the P24 terms and conditions.",
		},
		"payment_method_options.payto.mandate_options.amount_type": {
			Type:        "string",
			Description: "The type of amount that will be collected. The amount charged must be exact or up to the value of `amount` param for `fixed` or `maximum` type respectively. Defaults to `maximum`.",
			Enum: []resource.EnumSpec{
				{Value: "fixed"},
				{Value: "maximum"},
			},
		},
		"subscription_data.billing_mode.type": {
			Type:        "string",
			Description: "Controls the calculation and orchestration of prorations and invoices for subscriptions. If no value is passed, the default is `flexible`.",
			Required:    true,
			Enum: []resource.EnumSpec{
				{Value: "classic"},
				{Value: "flexible"},
			},
		},
		"branding_settings.logo.type": {
			Type:        "string",
			Description: "The type of image for the logo. Must be one of `file` or `url`.",
			Required:    true,
			Enum: []resource.EnumSpec{
				{Value: "file"},
				{Value: "url"},
			},
		},
		"payment_method_options.customer_balance.bank_transfer.eu_bank_transfer.country": {
			Type:        "string",
			Description: "The desired country code of the bank account information. Permitted values include: `DE`, `FR`, `IE`, or `NL`.",
			Required:    true,
		},
		"payment_method_options.payto.mandate_options.payment_schedule": {
			Type:        "string",
			Description: "The periodicity at which payments will be collected. Defaults to `adhoc`.",
			Enum: []resource.EnumSpec{
				{Value: "adhoc"},
				{Value: "annual"},
				{Value: "daily"},
				{Value: "fortnightly"},
				{Value: "monthly"},
				{Value: "quarterly"},
				{Value: "semi_annual"},
				{Value: "weekly"},
			},
		},
		"payment_method_options.oxxo.setup_future_usage": {
			Type:        "string",
			Description: "Indicates that you intend to make future payments with this PaymentIntent's payment method.\n\nIf you provide a Customer with the PaymentIntent, you can use this parameter to [attach the payment method](/payments/save-during-payment) to the Customer after the PaymentIntent is confirmed and the customer completes any required actions. If you don't provide a Customer, you can still [attach](/api/payment_methods/attach) the payment method to a Customer after the transaction completes.\n\nIf the payment method is `card_present` and isn't a digital wallet, Stripe creates and attaches a [generated_card](/api/charges/object#charge_object-payment_method_details-card_present-generated_card) payment method representing the card to the Customer instead.\n\nWhen processing card payments, Stripe uses `setup_future_usage` to help you comply with regional legislation and network rules, such as [SCA](/strong-customer-authentication).",
			Enum: []resource.EnumSpec{
				{Value: "none"},
			},
		},
		"branding_settings.icon.file": {
			Type:        "string",
			Description: "The ID of a [File upload](https://stripe.com/docs/api/files) representing the icon. Purpose must be `business_icon`. Required if `type` is `file` and disallowed otherwise.",
		},
		"payment_intent_data.statement_descriptor_suffix": {
			Type:        "string",
			Description: "Provides information about a card charge. Concatenated to the account's [statement descriptor prefix](https://docs.stripe.com/get-started/account/statement-descriptors#static) to form the complete statement descriptor that appears on the customer's statement.",
		},
		"payment_method_options.acss_debit.verification_method": {
			Type:        "string",
			Description: "Verification method for the intent",
			Enum: []resource.EnumSpec{
				{Value: "automatic"},
				{Value: "instant"},
				{Value: "microdeposits"},
			},
		},
		"payment_method_options.amazon_pay.capture_method": {
			Type:        "string",
			Description: "Controls when the funds will be captured from the customer's account.",
			Enum: []resource.EnumSpec{
				{Value: "manual"},
			},
		},
		"payment_method_options.paypal.setup_future_usage": {
			Type:        "string",
			Description: "Indicates that you intend to make future payments with this PaymentIntent's payment method.\n\nIf you provide a Customer with the PaymentIntent, you can use this parameter to [attach the payment method](/payments/save-during-payment) to the Customer after the PaymentIntent is confirmed and the customer completes any required actions. If you don't provide a Customer, you can still [attach](/api/payment_methods/attach) the payment method to a Customer after the transaction completes.\n\nIf the payment method is `card_present` and isn't a digital wallet, Stripe creates and attaches a [generated_card](/api/charges/object#charge_object-payment_method_details-card_present-generated_card) payment method representing the card to the Customer instead.\n\nWhen processing card payments, Stripe uses `setup_future_usage` to help you comply with regional legislation and network rules, such as [SCA](/strong-customer-authentication).\n\nIf you've already set `setup_future_usage` and you're performing a request using a publishable key, you can only update the value from `on_session` to `off_session`.",
			Enum: []resource.EnumSpec{
				{Value: "none"},
				{Value: "off_session"},
			},
		},
		"subscription_data.on_behalf_of": {
			Type:        "string",
			Description: "The account on behalf of which to charge, for each of the subscription's invoices.",
		},
		"name_collection.individual.optional": {
			Type:        "boolean",
			Description: "Whether the customer is required to provide their name before completing the Checkout Session. Defaults to `false`.",
		},
		"payment_intent_data.on_behalf_of": {
			Type:        "string",
			Description: "The Stripe account ID for which these funds are intended. For details,\nsee the PaymentIntents [use case for connected\naccounts](/docs/payments/connected-accounts).",
		},
		"payment_method_options.card.request_extended_authorization": {
			Type:        "string",
			Description: "Request ability to [capture beyond the standard authorization validity window](/payments/extended-authorization) for this CheckoutSession.",
			Enum: []resource.EnumSpec{
				{Value: "if_available"},
				{Value: "never"},
			},
		},
		"payment_method_options.wechat_pay.client": {
			Type:        "string",
			Description: "The client type that the end customer will pay from",
			Required:    true,
			Enum: []resource.EnumSpec{
				{Value: "android"},
				{Value: "ios"},
				{Value: "web"},
			},
		},
		"payment_method_options.mobilepay.setup_future_usage": {
			Type:        "string",
			Description: "Indicates that you intend to make future payments with this PaymentIntent's payment method.\n\nIf you provide a Customer with the PaymentIntent, you can use this parameter to [attach the payment method](/payments/save-during-payment) to the Customer after the PaymentIntent is confirmed and the customer completes any required actions. If you don't provide a Customer, you can still [attach](/api/payment_methods/attach) the payment method to a Customer after the transaction completes.\n\nIf the payment method is `card_present` and isn't a digital wallet, Stripe creates and attaches a [generated_card](/api/charges/object#charge_object-payment_method_details-card_present-generated_card) payment method representing the card to the Customer instead.\n\nWhen processing card payments, Stripe uses `setup_future_usage` to help you comply with regional legislation and network rules, such as [SCA](/strong-customer-authentication).",
			Enum: []resource.EnumSpec{
				{Value: "none"},
			},
		},
		"payment_method_options.naver_pay.setup_future_usage": {
			Type:        "string",
			Description: "Indicates that you intend to make future payments with this PaymentIntent's payment method.\n\nIf you provide a Customer with the PaymentIntent, you can use this parameter to [attach the payment method](/payments/save-during-payment) to the Customer after the PaymentIntent is confirmed and the customer completes any required actions. If you don't provide a Customer, you can still [attach](/api/payment_methods/attach) the payment method to a Customer after the transaction completes.\n\nIf the payment method is `card_present` and isn't a digital wallet, Stripe creates and attaches a [generated_card](/api/charges/object#charge_object-payment_method_details-card_present-generated_card) payment method representing the card to the Customer instead.\n\nWhen processing card payments, Stripe uses `setup_future_usage` to help you comply with regional legislation and network rules, such as [SCA](/strong-customer-authentication).",
			Enum: []resource.EnumSpec{
				{Value: "none"},
				{Value: "off_session"},
			},
		},
		"invoice_creation.invoice_data.account_tax_ids": {
			Type:        "array",
			Description: "The account tax IDs associated with the invoice.",
		},
		"setup_intent_data.description": {
			Type:        "string",
			Description: "An arbitrary string attached to the object. Often useful for displaying to users.",
		},
		"payment_intent_data.description": {
			Type:        "string",
			Description: "An arbitrary string attached to the object. Often useful for displaying to users.",
		},
		"payment_method_options.revolut_pay.setup_future_usage": {
			Type:        "string",
			Description: "Indicates that you intend to make future payments with this PaymentIntent's payment method.\n\nIf you provide a Customer with the PaymentIntent, you can use this parameter to [attach the payment method](/payments/save-during-payment) to the Customer after the PaymentIntent is confirmed and the customer completes any required actions. If you don't provide a Customer, you can still [attach](/api/payment_methods/attach) the payment method to a Customer after the transaction completes.\n\nIf the payment method is `card_present` and isn't a digital wallet, Stripe creates and attaches a [generated_card](/api/charges/object#charge_object-payment_method_details-card_present-generated_card) payment method representing the card to the Customer instead.\n\nWhen processing card payments, Stripe uses `setup_future_usage` to help you comply with regional legislation and network rules, such as [SCA](/strong-customer-authentication).",
			Enum: []resource.EnumSpec{
				{Value: "none"},
				{Value: "off_session"},
			},
		},
		"payment_method_options.giropay.setup_future_usage": {
			Type:        "string",
			Description: "Indicates that you intend to make future payments with this PaymentIntent's payment method.\n\nIf you provide a Customer with the PaymentIntent, you can use this parameter to [attach the payment method](/payments/save-during-payment) to the Customer after the PaymentIntent is confirmed and the customer completes any required actions. If you don't provide a Customer, you can still [attach](/api/payment_methods/attach) the payment method to a Customer after the transaction completes.\n\nIf the payment method is `card_present` and isn't a digital wallet, Stripe creates and attaches a [generated_card](/api/charges/object#charge_object-payment_method_details-card_present-generated_card) payment method representing the card to the Customer instead.\n\nWhen processing card payments, Stripe uses `setup_future_usage` to help you comply with regional legislation and network rules, such as [SCA](/strong-customer-authentication).",
			Enum: []resource.EnumSpec{
				{Value: "none"},
			},
		},
		"subscription_data.application_fee_percent": {
			Type:        "number",
			Description: "A non-negative decimal between 0 and 100, with at most two decimal places. This represents the percentage of the subscription invoice total that will be transferred to the application owner's Stripe account. To use an application fee percent, the request must be made on behalf of another account, using the `Stripe-Account` header or an OAuth key. For more information, see the application fees [documentation](https://stripe.com/docs/connect/subscriptions#collecting-fees-on-subscriptions).",
		},
		"payment_intent_data.shipping.address.line1": {
			Type:        "string",
			Description: "Address line 1, such as the street, PO Box, or company name.",
			Required:    true,
		},
		"payment_method_options.customer_balance.bank_transfer.type": {
			Type:        "string",
			Description: "The list of bank transfer types that this PaymentIntent is allowed to use for funding.",
			Required:    true,
			Enum: []resource.EnumSpec{
				{Value: "eu_bank_transfer"},
				{Value: "gb_bank_transfer"},
				{Value: "jp_bank_transfer"},
				{Value: "mx_bank_transfer"},
				{Value: "us_bank_transfer"},
			},
		},
		"payment_method_options.satispay.capture_method": {
			Type:        "string",
			Description: "Controls when the funds will be captured from the customer's account.",
			Enum: []resource.EnumSpec{
				{Value: "manual"},
			},
		},
		"payment_method_options.twint.setup_future_usage": {
			Type:        "string",
			Description: "Indicates that you intend to make future payments with this PaymentIntent's payment method.\n\nIf you provide a Customer with the PaymentIntent, you can use this parameter to [attach the payment method](/payments/save-during-payment) to the Customer after the PaymentIntent is confirmed and the customer completes any required actions. If you don't provide a Customer, you can still [attach](/api/payment_methods/attach) the payment method to a Customer after the transaction completes.\n\nIf the payment method is `card_present` and isn't a digital wallet, Stripe creates and attaches a [generated_card](/api/charges/object#charge_object-payment_method_details-card_present-generated_card) payment method representing the card to the Customer instead.\n\nWhen processing card payments, Stripe uses `setup_future_usage` to help you comply with regional legislation and network rules, such as [SCA](/strong-customer-authentication).",
			Enum: []resource.EnumSpec{
				{Value: "none"},
			},
		},
		"payment_method_options.sepa_debit.target_date": {
			Type:        "string",
			Description: "Controls when Stripe will attempt to debit the funds from the customer's account. The date must be a string in YYYY-MM-DD format. The date must be in the future and between 3 and 15 calendar days from now.",
		},
		"payment_method_options.swish.reference": {
			Type:        "string",
			Description: "The order reference that will be displayed to customers in the Swish application. Defaults to the `id` of the Payment Intent.",
		},
		"payment_method_options.paypal.capture_method": {
			Type:        "string",
			Description: "Controls when the funds will be captured from the customer's account.",
			Enum: []resource.EnumSpec{
				{Value: "manual"},
			},
		},
		"allow_promotion_codes": {
			Type:        "boolean",
			Description: "Enables user redeemable promotion codes.",
		},
		"branding_settings.icon.url": {
			Type:        "string",
			Description: "The URL of the image. Required if `type` is `url` and disallowed otherwise.",
		},
		"subscription_data.description": {
			Type:        "string",
			Description: "The subscription's description, meant to be displayable to the customer.\nUse this field to optionally store an explanation of the subscription\nfor rendering in the [customer portal](https://docs.stripe.com/customer-management).",
		},
		"name_collection.individual.enabled": {
			Type:        "boolean",
			Description: "Enable individual name collection on the Checkout Session. Defaults to `false`.",
			Required:    true,
		},
		"name_collection.business.optional": {
			Type:        "boolean",
			Description: "Whether the customer is required to provide a business name before completing the Checkout Session. Defaults to `false`.",
		},
		"payment_intent_data.shipping.tracking_number": {
			Type:        "string",
			Description: "The tracking number for a physical product, obtained from the delivery service. If multiple tracking numbers were generated for this purchase, please separate them with commas.",
		},
		"payment_intent_data.shipping.address.country": {
			Type:        "string",
			Description: "Two-letter country code ([ISO 3166-1 alpha-2](https://en.wikipedia.org/wiki/ISO_3166-1_alpha-2)).",
		},
		"payment_intent_data.shipping.carrier": {
			Type:        "string",
			Description: "The delivery service that shipped a physical product, such as Fedex, UPS, USPS, etc.",
		},
		"customer": {
			Type:        "string",
			Description: "ID of an existing Customer, if one exists. In `payment` mode, the customer’s most recently saved card\npayment method will be used to prefill the email, name, card details, and billing address\non the Checkout page. In `subscription` mode, the customer’s [default payment method](https://docs.stripe.com/api/customers/update#update_customer-invoice_settings-default_payment_method)\nwill be used if it’s a card, otherwise the most recently saved card will be used. A valid billing address, billing name and billing email are required on the payment method for Checkout to prefill the customer's card details.\n\nIf the Customer already has a valid [email](https://docs.stripe.com/api/customers/object#customer_object-email) set, the email will be prefilled and not editable in Checkout.\nIf the Customer does not have a valid `email`, Checkout will set the email entered during the session on the Customer.\n\nIf blank for Checkout Sessions in `subscription` mode or with `customer_creation` set as `always` in `payment` mode, Checkout will create a new Customer object based on information provided during the payment flow.\n\nYou can set [`payment_intent_data.setup_future_usage`](https://docs.stripe.com/api/checkout/sessions/create#create_checkout_session-payment_intent_data-setup_future_usage) to have Checkout automatically attach the payment method to the Customer you pass in for future reuse.",
		},
		"payment_method_options.affirm.capture_method": {
			Type:        "string",
			Description: "Controls when the funds will be captured from the customer's account.",
			Enum: []resource.EnumSpec{
				{Value: "manual"},
			},
		},
		"payment_method_options.cashapp.setup_future_usage": {
			Type:        "string",
			Description: "Indicates that you intend to make future payments with this PaymentIntent's payment method.\n\nIf you provide a Customer with the PaymentIntent, you can use this parameter to [attach the payment method](/payments/save-during-payment) to the Customer after the PaymentIntent is confirmed and the customer completes any required actions. If you don't provide a Customer, you can still [attach](/api/payment_methods/attach) the payment method to a Customer after the transaction completes.\n\nIf the payment method is `card_present` and isn't a digital wallet, Stripe creates and attaches a [generated_card](/api/charges/object#charge_object-payment_method_details-card_present-generated_card) payment method representing the card to the Customer instead.\n\nWhen processing card payments, Stripe uses `setup_future_usage` to help you comply with regional legislation and network rules, such as [SCA](/strong-customer-authentication).",
			Enum: []resource.EnumSpec{
				{Value: "none"},
				{Value: "off_session"},
				{Value: "on_session"},
			},
		},
		"shipping_address_collection.allowed_countries": {
			Type:        "array",
			Description: "An array of two-letter ISO country codes representing which countries Checkout should provide as options for\nshipping locations.",
			Required:    true,
		},
		"branding_settings.display_name": {
			Type:        "string",
			Description: "A string to override the business name shown on the Checkout Session. This only shows at the top of the Checkout page, and your business name still appears in terms, receipts, and other places.",
		},
		"payment_intent_data.shipping.address.city": {
			Type:        "string",
			Description: "City, district, suburb, town, or village.",
		},
		"payment_method_options.konbini.expires_after_days": {
			Type:        "integer",
			Description: "The number of calendar days (between 1 and 60) after which Konbini payment instructions will expire. For example, if a PaymentIntent is confirmed with Konbini and `expires_after_days` set to 2 on Monday JST, the instructions will expire on Wednesday 23:59:59 JST. Defaults to 3 days.",
		},
		"payment_method_options.us_bank_account.financial_connections.permissions": {
			Type:        "array",
			Description: "The list of permissions to request. If this parameter is passed, the `payment_method` permission must be included. Valid permissions include: `balances`, `ownership`, `payment_method`, and `transactions`.",
		},
		"payment_method_options.multibanco.setup_future_usage": {
			Type:        "string",
			Description: "Indicates that you intend to make future payments with this PaymentIntent's payment method.\n\nIf you provide a Customer with the PaymentIntent, you can use this parameter to [attach the payment method](/payments/save-during-payment) to the Customer after the PaymentIntent is confirmed and the customer completes any required actions. If you don't provide a Customer, you can still [attach](/api/payment_methods/attach) the payment method to a Customer after the transaction completes.\n\nIf the payment method is `card_present` and isn't a digital wallet, Stripe creates and attaches a [generated_card](/api/charges/object#charge_object-payment_method_details-card_present-generated_card) payment method representing the card to the Customer instead.\n\nWhen processing card payments, Stripe uses `setup_future_usage` to help you comply with regional legislation and network rules, such as [SCA](/strong-customer-authentication).",
			Enum: []resource.EnumSpec{
				{Value: "none"},
			},
		},
		"invoice_creation.invoice_data.footer": {
			Type:        "string",
			Description: "Default footer to be displayed on invoices for this customer.",
		},
		"saved_payment_method_options.payment_method_save": {
			Type:        "string",
			Description: "Enable customers to choose if they wish to save their payment method for future use. Disabled by default.",
			Enum: []resource.EnumSpec{
				{Value: "disabled"},
				{Value: "enabled"},
			},
		},
		"payment_intent_data.receipt_email": {
			Type:        "string",
			Description: "Email address that the receipt for the resulting payment will be sent to. If `receipt_email` is specified for a payment in live mode, a receipt will be sent regardless of your [email settings](https://dashboard.stripe.com/account/emails).",
		},
		"automatic_tax.enabled": {
			Type:        "boolean",
			Description: "Set to `true` to [calculate tax automatically](https://docs.stripe.com/tax) using the customer's location.\n\nEnabling this parameter causes Checkout to collect any billing address information necessary for tax calculation.",
			Required:    true,
		},
		"payment_method_options.acss_debit.mandate_options.payment_schedule": {
			Type:        "string",
			Description: "Payment schedule for the mandate.",
			Enum: []resource.EnumSpec{
				{Value: "combined"},
				{Value: "interval"},
				{Value: "sporadic"},
			},
		},
		"payment_method_options.boleto.setup_future_usage": {
			Type:        "string",
			Description: "Indicates that you intend to make future payments with this PaymentIntent's payment method.\n\nIf you provide a Customer with the PaymentIntent, you can use this parameter to [attach the payment method](/payments/save-during-payment) to the Customer after the PaymentIntent is confirmed and the customer completes any required actions. If you don't provide a Customer, you can still [attach](/api/payment_methods/attach) the payment method to a Customer after the transaction completes.\n\nIf the payment method is `card_present` and isn't a digital wallet, Stripe creates and attaches a [generated_card](/api/charges/object#charge_object-payment_method_details-card_present-generated_card) payment method representing the card to the Customer instead.\n\nWhen processing card payments, Stripe uses `setup_future_usage` to help you comply with regional legislation and network rules, such as [SCA](/strong-customer-authentication).",
			Enum: []resource.EnumSpec{
				{Value: "none"},
				{Value: "off_session"},
				{Value: "on_session"},
			},
		},
		"payment_method_options.payto.mandate_options.end_date": {
			Type:        "string",
			Description: "Date, in YYYY-MM-DD format, after which payments will not be collected. Defaults to no end date.",
		},
		"invoice_creation.invoice_data.issuer.type": {
			Type:        "string",
			Description: "Type of the account referenced in the request.",
			Required:    true,
			Enum: []resource.EnumSpec{
				{Value: "account"},
				{Value: "self"},
			},
		},
		"payment_intent_data.transfer_data.amount": {
			Type:        "integer",
			Description: "The amount that will be transferred automatically when a charge succeeds.",
		},
		"payment_method_options.bacs_debit.mandate_options.reference_prefix": {
			Type:        "string",
			Description: "Prefix used to generate the Mandate reference. Must be at most 12 characters long. Must consist of only uppercase letters, numbers, spaces, or the following special characters: '/', '_', '-', '&', '.'. Cannot begin with 'DDIC' or 'STRIPE'.",
		},
		"payment_method_options.card.request_multicapture": {
			Type:        "string",
			Description: "Request ability to make [multiple captures](/payments/multicapture) for this CheckoutSession.",
			Enum: []resource.EnumSpec{
				{Value: "if_available"},
				{Value: "never"},
			},
		},
		"payment_method_options.ideal.setup_future_usage": {
			Type:        "string",
			Description: "Indicates that you intend to make future payments with this PaymentIntent's payment method.\n\nIf you provide a Customer with the PaymentIntent, you can use this parameter to [attach the payment method](/payments/save-during-payment) to the Customer after the PaymentIntent is confirmed and the customer completes any required actions. If you don't provide a Customer, you can still [attach](/api/payment_methods/attach) the payment method to a Customer after the transaction completes.\n\nIf the payment method is `card_present` and isn't a digital wallet, Stripe creates and attaches a [generated_card](/api/charges/object#charge_object-payment_method_details-card_present-generated_card) payment method representing the card to the Customer instead.\n\nWhen processing card payments, Stripe uses `setup_future_usage` to help you comply with regional legislation and network rules, such as [SCA](/strong-customer-authentication).",
			Enum: []resource.EnumSpec{
				{Value: "none"},
			},
		},
		"payment_method_options.affirm.setup_future_usage": {
			Type:        "string",
			Description: "Indicates that you intend to make future payments with this PaymentIntent's payment method.\n\nIf you provide a Customer with the PaymentIntent, you can use this parameter to [attach the payment method](/payments/save-during-payment) to the Customer after the PaymentIntent is confirmed and the customer completes any required actions. If you don't provide a Customer, you can still [attach](/api/payment_methods/attach) the payment method to a Customer after the transaction completes.\n\nIf the payment method is `card_present` and isn't a digital wallet, Stripe creates and attaches a [generated_card](/api/charges/object#charge_object-payment_method_details-card_present-generated_card) payment method representing the card to the Customer instead.\n\nWhen processing card payments, Stripe uses `setup_future_usage` to help you comply with regional legislation and network rules, such as [SCA](/strong-customer-authentication).",
			Enum: []resource.EnumSpec{
				{Value: "none"},
			},
		},
		"payment_method_options.kakao_pay.capture_method": {
			Type:        "string",
			Description: "Controls when the funds will be captured from the customer's account.",
			Enum: []resource.EnumSpec{
				{Value: "manual"},
			},
		},
		"payment_method_options.payto.mandate_options.amount": {
			Type:        "integer",
			Description: "Amount that will be collected. It is required when `amount_type` is `fixed`.",
		},
		"payment_method_options.pix.expires_after_seconds": {
			Type:        "integer",
			Description: "The number of seconds (between 10 and 1209600) after which Pix payment will expire. Defaults to 86400 seconds.",
		},
		"branding_settings.border_style": {
			Type:        "string",
			Description: "The border style for the Checkout Session.",
			Enum: []resource.EnumSpec{
				{Value: "pill"},
				{Value: "rectangular"},
				{Value: "rounded"},
			},
		},
		"payment_intent_data.application_fee_amount": {
			Type:        "integer",
			Description: "The amount of the application fee (if any) that will be requested to be applied to the payment and transferred to the application owner's Stripe account. The amount of the application fee collected will be capped at the total amount captured. For more information, see the PaymentIntents [use case for connected accounts](https://docs.stripe.com/payments/connected-accounts).",
		},
		"payment_method_options.card.capture_method": {
			Type:        "string",
			Description: "Controls when the funds will be captured from the customer's account.",
			Enum: []resource.EnumSpec{
				{Value: "manual"},
			},
		},
		"payment_method_options.card.request_overcapture": {
			Type:        "string",
			Description: "Request ability to [overcapture](/payments/overcapture) for this CheckoutSession.",
			Enum: []resource.EnumSpec{
				{Value: "if_available"},
				{Value: "never"},
			},
		},
		"payment_method_options.revolut_pay.capture_method": {
			Type:        "string",
			Description: "Controls when the funds will be captured from the customer's account.",
			Enum: []resource.EnumSpec{
				{Value: "manual"},
			},
		},
		"payment_method_options.wechat_pay.setup_future_usage": {
			Type:        "string",
			Description: "Indicates that you intend to make future payments with this PaymentIntent's payment method.\n\nIf you provide a Customer with the PaymentIntent, you can use this parameter to [attach the payment method](/payments/save-during-payment) to the Customer after the PaymentIntent is confirmed and the customer completes any required actions. If you don't provide a Customer, you can still [attach](/api/payment_methods/attach) the payment method to a Customer after the transaction completes.\n\nIf the payment method is `card_present` and isn't a digital wallet, Stripe creates and attaches a [generated_card](/api/charges/object#charge_object-payment_method_details-card_present-generated_card) payment method representing the card to the Customer instead.\n\nWhen processing card payments, Stripe uses `setup_future_usage` to help you comply with regional legislation and network rules, such as [SCA](/strong-customer-authentication).",
			Enum: []resource.EnumSpec{
				{Value: "none"},
			},
		},
		"payment_method_options.afterpay_clearpay.capture_method": {
			Type:        "string",
			Description: "Controls when the funds will be captured from the customer's account.",
			Enum: []resource.EnumSpec{
				{Value: "manual"},
			},
		},
		"setup_intent_data.on_behalf_of": {
			Type:        "string",
			Description: "The Stripe account for which the setup is intended.",
		},
		"branding_settings.font_family": {
			Type:        "string",
			Description: "The font family for the Checkout Session corresponding to one of the [supported font families](https://docs.stripe.com/payments/checkout/customization/appearance?payment-ui=stripe-hosted#font-compatibility).",
			Enum: []resource.EnumSpec{
				{Value: "be_vietnam_pro"},
				{Value: "bitter"},
				{Value: "chakra_petch"},
				{Value: "default"},
				{Value: "hahmlet"},
				{Value: "inconsolata"},
				{Value: "inter"},
				{Value: "lato"},
				{Value: "lora"},
				{Value: "m_plus_1_code"},
				{Value: "montserrat"},
				{Value: "noto_sans"},
				{Value: "noto_sans_jp"},
				{Value: "noto_serif"},
				{Value: "nunito"},
				{Value: "open_sans"},
				{Value: "pridi"},
				{Value: "pt_sans"},
				{Value: "pt_serif"},
				{Value: "raleway"},
				{Value: "roboto"},
				{Value: "roboto_slab"},
				{Value: "source_sans_pro"},
				{Value: "titillium_web"},
				{Value: "ubuntu_mono"},
				{Value: "zen_maru_gothic"},
			},
		},
		"submit_type": {
			Type:        "string",
			Description: "Describes the type of transaction being performed by Checkout in order\nto customize relevant text on the page, such as the submit button.\n `submit_type` can only be specified on Checkout Sessions in\n`payment` or `subscription` mode. If blank or `auto`, `pay` is used.\nYou can't set this parameter if `ui_mode` is `custom`.",
			Enum: []resource.EnumSpec{
				{Value: "auto"},
				{Value: "book"},
				{Value: "donate"},
				{Value: "pay"},
				{Value: "subscribe"},
			},
		},
		"payment_method_options.card.restrictions.brands_blocked": {
			Type:        "array",
			Description: "Specify the card brands to block in the Checkout Session. If a customer enters or selects a card belonging to a blocked brand, they can't complete the Session.",
		},
		"payment_method_options.card.statement_descriptor_suffix_kanji": {
			Type:        "string",
			Description: "Provides information about a card payment that customers see on their statements. Concatenated with the Kanji prefix (shortened Kanji descriptor) or Kanji statement descriptor that’s set on the account to form the complete statement descriptor. Maximum 17 characters. On card statements, the *concatenation* of both prefix and suffix (including separators) will appear truncated to 17 characters.",
		},
		"payment_method_options.alma.capture_method": {
			Type:        "string",
			Description: "Controls when the funds will be captured from the customer's account.",
			Enum: []resource.EnumSpec{
				{Value: "manual"},
			},
		},
		"after_expiration.recovery.enabled": {
			Type:        "boolean",
			Description: "If `true`, a recovery URL will be generated to recover this Checkout Session if it\nexpires before a successful transaction is completed. It will be attached to the\nCheckout Session object upon expiration.",
			Required:    true,
		},
		"payment_intent_data.setup_future_usage": {
			Type:        "string",
			Description: "Indicates that you intend to [make future payments](https://docs.stripe.com/payments/payment-intents#future-usage) with the payment\nmethod collected by this Checkout Session.\n\nWhen setting this to `on_session`, Checkout will show a notice to the\ncustomer that their payment details will be saved.\n\nWhen setting this to `off_session`, Checkout will show a notice to the\ncustomer that their payment details will be saved and used for future\npayments.\n\nIf a Customer has been provided or Checkout creates a new Customer,\nCheckout will attach the payment method to the Customer.\n\nIf Checkout does not create a Customer, the payment method is not attached\nto a Customer. To reuse the payment method, you can retrieve it from the\nCheckout Session's PaymentIntent.\n\nWhen processing card payments, Checkout also uses `setup_future_usage`\nto dynamically optimize your payment flow and comply with regional\nlegislation and network rules, such as SCA.",
			Enum: []resource.EnumSpec{
				{Value: "off_session"},
				{Value: "on_session"},
			},
		},
		"ui_mode": {
			Type:        "string",
			Description: "The UI mode of the Session. Defaults to `hosted`.",
			Enum: []resource.EnumSpec{
				{Value: "custom"},
				{Value: "embedded"},
				{Value: "hosted"},
			},
		},
		"payment_method_options.bacs_debit.setup_future_usage": {
			Type:        "string",
			Description: "Indicates that you intend to make future payments with this PaymentIntent's payment method.\n\nIf you provide a Customer with the PaymentIntent, you can use this parameter to [attach the payment method](/payments/save-during-payment) to the Customer after the PaymentIntent is confirmed and the customer completes any required actions. If you don't provide a Customer, you can still [attach](/api/payment_methods/attach) the payment method to a Customer after the transaction completes.\n\nIf the payment method is `card_present` and isn't a digital wallet, Stripe creates and attaches a [generated_card](/api/charges/object#charge_object-payment_method_details-card_present-generated_card) payment method representing the card to the Customer instead.\n\nWhen processing card payments, Stripe uses `setup_future_usage` to help you comply with regional legislation and network rules, such as [SCA](/strong-customer-authentication).",
			Enum: []resource.EnumSpec{
				{Value: "none"},
				{Value: "off_session"},
				{Value: "on_session"},
			},
		},
		"payment_method_options.billie.capture_method": {
			Type:        "string",
			Description: "Controls when the funds will be captured from the customer's account.",
			Enum: []resource.EnumSpec{
				{Value: "manual"},
			},
		},
		"payment_method_options.grabpay.setup_future_usage": {
			Type:        "string",
			Description: "Indicates that you intend to make future payments with this PaymentIntent's payment method.\n\nIf you provide a Customer with the PaymentIntent, you can use this parameter to [attach the payment method](/payments/save-during-payment) to the Customer after the PaymentIntent is confirmed and the customer completes any required actions. If you don't provide a Customer, you can still [attach](/api/payment_methods/attach) the payment method to a Customer after the transaction completes.\n\nIf the payment method is `card_present` and isn't a digital wallet, Stripe creates and attaches a [generated_card](/api/charges/object#charge_object-payment_method_details-card_present-generated_card) payment method representing the card to the Customer instead.\n\nWhen processing card payments, Stripe uses `setup_future_usage` to help you comply with regional legislation and network rules, such as [SCA](/strong-customer-authentication).",
			Enum: []resource.EnumSpec{
				{Value: "none"},
			},
		},
		"payment_method_options.afterpay_clearpay.setup_future_usage": {
			Type:        "string",
			Description: "Indicates that you intend to make future payments with this PaymentIntent's payment method.\n\nIf you provide a Customer with the PaymentIntent, you can use this parameter to [attach the payment method](/payments/save-during-payment) to the Customer after the PaymentIntent is confirmed and the customer completes any required actions. If you don't provide a Customer, you can still [attach](/api/payment_methods/attach) the payment method to a Customer after the transaction completes.\n\nIf the payment method is `card_present` and isn't a digital wallet, Stripe creates and attaches a [generated_card](/api/charges/object#charge_object-payment_method_details-card_present-generated_card) payment method representing the card to the Customer instead.\n\nWhen processing card payments, Stripe uses `setup_future_usage` to help you comply with regional legislation and network rules, such as [SCA](/strong-customer-authentication).",
			Enum: []resource.EnumSpec{
				{Value: "none"},
			},
		},
		"payment_method_options.demo_pay.setup_future_usage": {
			Type:        "string",
			Description: "Indicates that you intend to make future payments with this PaymentIntent's payment method.\n\nIf you provide a Customer with the PaymentIntent, you can use this parameter to [attach the payment method](/payments/save-during-payment) to the Customer after the PaymentIntent is confirmed and the customer completes any required actions. If you don't provide a Customer, you can still [attach](/api/payment_methods/attach) the payment method to a Customer after the transaction completes.\n\nIf the payment method is `card_present` and isn't a digital wallet, Stripe creates and attaches a [generated_card](/api/charges/object#charge_object-payment_method_details-card_present-generated_card) payment method representing the card to the Customer instead.\n\nWhen processing card payments, Stripe uses `setup_future_usage` to help you comply with regional legislation and network rules, such as [SCA](/strong-customer-authentication).",
			Enum: []resource.EnumSpec{
				{Value: "none"},
				{Value: "off_session"},
			},
		},
		"payment_method_options.acss_debit.mandate_options.custom_mandate_url": {
			Type:        "string",
			Description: "A URL for custom mandate text to render during confirmation step.\nThe URL will be rendered with additional GET parameters `payment_intent` and `payment_intent_client_secret` when confirming a Payment Intent,\nor `setup_intent` and `setup_intent_client_secret` when confirming a Setup Intent.",
		},
		"payment_method_options.payto.mandate_options.purpose": {
			Type:        "string",
			Description: "The purpose for which payments are made. Has a default value based on your merchant category code.",
			Enum: []resource.EnumSpec{
				{Value: "dependant_support"},
				{Value: "government"},
				{Value: "loan"},
				{Value: "mortgage"},
				{Value: "other"},
				{Value: "pension"},
				{Value: "personal"},
				{Value: "retail"},
				{Value: "salary"},
				{Value: "tax"},
				{Value: "utility"},
			},
		},
		"payment_method_options.payto.mandate_options.payments_per_period": {
			Type:        "integer",
			Description: "The number of payments that will be made during a payment period. Defaults to 1 except for when `payment_schedule` is `adhoc`. In that case, it defaults to no limit.",
		},
		"payment_method_options.sepa_debit.setup_future_usage": {
			Type:        "string",
			Description: "Indicates that you intend to make future payments with this PaymentIntent's payment method.\n\nIf you provide a Customer with the PaymentIntent, you can use this parameter to [attach the payment method](/payments/save-during-payment) to the Customer after the PaymentIntent is confirmed and the customer completes any required actions. If you don't provide a Customer, you can still [attach](/api/payment_methods/attach) the payment method to a Customer after the transaction completes.\n\nIf the payment method is `card_present` and isn't a digital wallet, Stripe creates and attaches a [generated_card](/api/charges/object#charge_object-payment_method_details-card_present-generated_card) payment method representing the card to the Customer instead.\n\nWhen processing card payments, Stripe uses `setup_future_usage` to help you comply with regional legislation and network rules, such as [SCA](/strong-customer-authentication).",
			Enum: []resource.EnumSpec{
				{Value: "none"},
				{Value: "off_session"},
				{Value: "on_session"},
			},
		},
		"payment_method_options.us_bank_account.setup_future_usage": {
			Type:        "string",
			Description: "Indicates that you intend to make future payments with this PaymentIntent's payment method.\n\nIf you provide a Customer with the PaymentIntent, you can use this parameter to [attach the payment method](/payments/save-during-payment) to the Customer after the PaymentIntent is confirmed and the customer completes any required actions. If you don't provide a Customer, you can still [attach](/api/payment_methods/attach) the payment method to a Customer after the transaction completes.\n\nIf the payment method is `card_present` and isn't a digital wallet, Stripe creates and attaches a [generated_card](/api/charges/object#charge_object-payment_method_details-card_present-generated_card) payment method representing the card to the Customer instead.\n\nWhen processing card payments, Stripe uses `setup_future_usage` to help you comply with regional legislation and network rules, such as [SCA](/strong-customer-authentication).",
			Enum: []resource.EnumSpec{
				{Value: "none"},
				{Value: "off_session"},
				{Value: "on_session"},
			},
		},
		"payment_method_options.us_bank_account.verification_method": {
			Type:        "string",
			Description: "Verification method for the intent",
			Enum: []resource.EnumSpec{
				{Value: "automatic"},
				{Value: "instant"},
			},
		},
		"customer_update.shipping": {
			Type:        "string",
			Description: "Describes whether Checkout saves shipping information onto `customer.shipping`.\nTo collect shipping information, use `shipping_address_collection`. Defaults to `never`.",
			Enum: []resource.EnumSpec{
				{Value: "auto"},
				{Value: "never"},
			},
		},
		"tax_id_collection.enabled": {
			Type:        "boolean",
			Description: "Enable tax ID collection during checkout. Defaults to `false`.",
			Required:    true,
		},
		"payment_intent_data.capture_method": {
			Type:        "string",
			Description: "Controls when the funds will be captured from the customer's account.",
			Enum: []resource.EnumSpec{
				{Value: "automatic"},
				{Value: "automatic_async"},
				{Value: "manual"},
			},
		},
		"payment_method_options.card.installments.enabled": {
			Type:        "boolean",
			Description: "Setting to true enables installments for this Checkout Session.\nSetting to false will prevent any installment plan from applying to a payment.",
		},
		"payment_method_options.klarna.capture_method": {
			Type:        "string",
			Description: "Controls when the funds will be captured from the customer's account.",
			Enum: []resource.EnumSpec{
				{Value: "manual"},
			},
		},
		"payment_method_options.cashapp.capture_method": {
			Type:        "string",
			Description: "Controls when the funds will be captured from the customer's account.",
			Enum: []resource.EnumSpec{
				{Value: "manual"},
			},
		},
		"payment_method_options.p24.setup_future_usage": {
			Type:        "string",
			Description: "Indicates that you intend to make future payments with this PaymentIntent's payment method.\n\nIf you provide a Customer with the PaymentIntent, you can use this parameter to [attach the payment method](/payments/save-during-payment) to the Customer after the PaymentIntent is confirmed and the customer completes any required actions. If you don't provide a Customer, you can still [attach](/api/payment_methods/attach) the payment method to a Customer after the transaction completes.\n\nIf the payment method is `card_present` and isn't a digital wallet, Stripe creates and attaches a [generated_card](/api/charges/object#charge_object-payment_method_details-card_present-generated_card) payment method representing the card to the Customer instead.\n\nWhen processing card payments, Stripe uses `setup_future_usage` to help you comply with regional legislation and network rules, such as [SCA](/strong-customer-authentication).",
			Enum: []resource.EnumSpec{
				{Value: "none"},
			},
		},
		"payment_method_options.us_bank_account.target_date": {
			Type:        "string",
			Description: "Controls when Stripe will attempt to debit the funds from the customer's account. The date must be a string in YYYY-MM-DD format. The date must be in the future and between 3 and 15 calendar days from now.",
		},
		"payment_method_options.pix.setup_future_usage": {
			Type:        "string",
			Description: "Indicates that you intend to make future payments with this PaymentIntent's payment method.\n\nIf you provide a Customer with the PaymentIntent, you can use this parameter to [attach the payment method](/payments/save-during-payment) to the Customer after the PaymentIntent is confirmed and the customer completes any required actions. If you don't provide a Customer, you can still [attach](/api/payment_methods/attach) the payment method to a Customer after the transaction completes.\n\nIf the payment method is `card_present` and isn't a digital wallet, Stripe creates and attaches a [generated_card](/api/charges/object#charge_object-payment_method_details-card_present-generated_card) payment method representing the card to the Customer instead.\n\nWhen processing card payments, Stripe uses `setup_future_usage` to help you comply with regional legislation and network rules, such as [SCA](/strong-customer-authentication).",
			Enum: []resource.EnumSpec{
				{Value: "none"},
			},
		},
		"origin_context": {
			Type:        "string",
			Description: "Where the user is coming from. This informs the optimizations that are applied to the session. You can't set this parameter if `ui_mode` is `custom`.",
			Enum: []resource.EnumSpec{
				{Value: "mobile_app"},
				{Value: "web"},
			},
		},
		"payment_intent_data.transfer_group": {
			Type:        "string",
			Description: "A string that identifies the resulting payment as part of a group. See the PaymentIntents [use case for connected accounts](https://docs.stripe.com/connect/separate-charges-and-transfers) for details.",
		},
		"payment_method_options.customer_balance.setup_future_usage": {
			Type:        "string",
			Description: "Indicates that you intend to make future payments with this PaymentIntent's payment method.\n\nIf you provide a Customer with the PaymentIntent, you can use this parameter to [attach the payment method](/payments/save-during-payment) to the Customer after the PaymentIntent is confirmed and the customer completes any required actions. If you don't provide a Customer, you can still [attach](/api/payment_methods/attach) the payment method to a Customer after the transaction completes.\n\nIf the payment method is `card_present` and isn't a digital wallet, Stripe creates and attaches a [generated_card](/api/charges/object#charge_object-payment_method_details-card_present-generated_card) payment method representing the card to the Customer instead.\n\nWhen processing card payments, Stripe uses `setup_future_usage` to help you comply with regional legislation and network rules, such as [SCA](/strong-customer-authentication).",
			Enum: []resource.EnumSpec{
				{Value: "none"},
			},
		},
		"payment_method_options.mobilepay.capture_method": {
			Type:        "string",
			Description: "Controls when the funds will be captured from the customer's account.",
			Enum: []resource.EnumSpec{
				{Value: "manual"},
			},
		},
		"payment_method_options.sepa_debit.mandate_options.reference_prefix": {
			Type:        "string",
			Description: "Prefix used to generate the Mandate reference. Must be at most 12 characters long. Must consist of only uppercase letters, numbers, spaces, or the following special characters: '/', '_', '-', '&', '.'. Cannot begin with 'STRIPE'.",
		},
		"payment_method_options.paypal.risk_correlation_id": {
			Type:        "string",
			Description: "The risk correlation ID for an on-session payment using a saved PayPal payment method.",
		},
		"subscription_data.invoice_settings.issuer.type": {
			Type:        "string",
			Description: "Type of the account referenced in the request.",
			Required:    true,
			Enum: []resource.EnumSpec{
				{Value: "account"},
				{Value: "self"},
			},
		},
		"payment_method_types": {
			Type:        "array",
			Description: "A list of the types of payment methods (e.g., `card`) this Checkout Session can accept.\n\nYou can omit this attribute to manage your payment methods from the [Stripe Dashboard](https://dashboard.stripe.com/settings/payment_methods).\nSee [Dynamic Payment Methods](https://docs.stripe.com/payments/payment-methods/integration-options#using-dynamic-payment-methods) for more details.\n\nRead more about the supported payment methods and their requirements in our [payment\nmethod details guide](/docs/payments/checkout/payment-methods).\n\nIf multiple payment methods are passed, Checkout will dynamically reorder them to\nprioritize the most relevant payment methods based on the customer's location and\nother characteristics.",
		},
		"branding_settings.icon.type": {
			Type:        "string",
			Description: "The type of image for the icon. Must be one of `file` or `url`.",
			Required:    true,
			Enum: []resource.EnumSpec{
				{Value: "file"},
				{Value: "url"},
			},
		},
		"payment_intent_data.statement_descriptor": {
			Type:        "string",
			Description: "Text that appears on the customer's statement as the statement descriptor for a non-card charge. This value overrides the account's default statement descriptor. For information about requirements, including the 22-character limit, see [the Statement Descriptor docs](https://docs.stripe.com/get-started/account/statement-descriptors).\n\nSetting this value for a card charge returns an error. For card charges, set the [statement_descriptor_suffix](https://docs.stripe.com/get-started/account/statement-descriptors#dynamic) instead.",
		},
		"automatic_tax.liability.type": {
			Type:        "string",
			Description: "Type of the account referenced in the request.",
			Required:    true,
			Enum: []resource.EnumSpec{
				{Value: "account"},
				{Value: "self"},
			},
		},
		"payment_method_configuration": {
			Type:        "string",
			Description: "The ID of the payment method configuration to use with this Checkout session.",
		},
		"subscription_data.billing_mode.flexible.proration_discounts": {
			Type:        "string",
			Description: "Controls how invoices and invoice items display proration amounts and discount amounts.",
			Enum: []resource.EnumSpec{
				{Value: "included"},
				{Value: "itemized"},
			},
		},
		"subscription_data.proration_behavior": {
			Type:        "string",
			Description: "Determines how to handle prorations resulting from the `billing_cycle_anchor`. If no value is passed, the default is `create_prorations`.",
			Enum: []resource.EnumSpec{
				{Value: "create_prorations"},
				{Value: "none"},
			},
		},
		"name_collection.business.enabled": {
			Type:        "boolean",
			Description: "Enable business name collection on the Checkout Session. Defaults to `false`.",
			Required:    true,
		},
		"return_url": {
			Type:        "string",
			Description: "The URL to redirect your customer back to after they authenticate or cancel their payment on the\npayment method's app or site. This parameter is required if `ui_mode` is `embedded` or `custom`\nand redirect-based payment methods are enabled on the session.",
		},
		"payment_intent_data.shipping.address.line2": {
			Type:        "string",
			Description: "Address line 2, such as the apartment, suite, unit, or building.",
		},
		"success_url": {
			Type:        "string",
			Description: "The URL to which Stripe should send customers when payment or setup\nis complete.\nThis parameter is not allowed if ui_mode is `embedded` or `custom`. If you'd like to use\ninformation from the successful Checkout Session on your page, read the\nguide on [customizing your success page](https://docs.stripe.com/payments/checkout/custom-success-page).",
		},
		"excluded_payment_method_types": {
			Type:        "array",
			Description: "A list of the types of payment methods (e.g., `card`) that should be excluded from this Checkout Session. This should only be used when payment methods for this Checkout Session are managed through the [Stripe Dashboard](https://dashboard.stripe.com/settings/payment_methods).",
		},
		"payment_method_options.naver_pay.capture_method": {
			Type:        "string",
			Description: "Controls when the funds will be captured from the customer's account.",
			Enum: []resource.EnumSpec{
				{Value: "manual"},
			},
		},
		"payment_method_options.us_bank_account.financial_connections.prefetch": {
			Type:        "array",
			Description: "List of data features that you would like to retrieve upon account creation.",
		},
		"payment_method_options.paypal.reference": {
			Type:        "string",
			Description: "A reference of the PayPal transaction visible to customer which is mapped to PayPal's invoice ID. This must be a globally unique ID if you have configured in your PayPal settings to block multiple payments per invoice ID.",
		},
		"invoice_creation.invoice_data.issuer.account": {
			Type:        "string",
			Description: "The connected account being referenced when `type` is `account`.",
		},
		"cancel_url": {
			Type:        "string",
			Description: "If set, Checkout displays a back button and customers will be directed to this URL if they decide to cancel payment and return to your website. This parameter is not allowed if ui_mode is `embedded` or `custom`.",
		},
		"payment_intent_data.shipping.name": {
			Type:        "string",
			Description: "Recipient name.",
			Required:    true,
		},
		"consent_collection.terms_of_service": {
			Type:        "string",
			Description: "If set to `required`, it requires customers to check a terms of service checkbox before being able to pay.\nThere must be a valid terms of service URL set in your [Dashboard settings](https://dashboard.stripe.com/settings/public).",
			Enum: []resource.EnumSpec{
				{Value: "none"},
				{Value: "required"},
			},
		},
		"payment_method_options.acss_debit.mandate_options.interval_description": {
			Type:        "string",
			Description: "Description of the mandate interval. Only required if 'payment_schedule' parameter is 'interval' or 'combined'.",
		},
		"payment_method_options.acss_debit.setup_future_usage": {
			Type:        "string",
			Description: "Indicates that you intend to make future payments with this PaymentIntent's payment method.\n\nIf you provide a Customer with the PaymentIntent, you can use this parameter to [attach the payment method](/payments/save-during-payment) to the Customer after the PaymentIntent is confirmed and the customer completes any required actions. If you don't provide a Customer, you can still [attach](/api/payment_methods/attach) the payment method to a Customer after the transaction completes.\n\nIf the payment method is `card_present` and isn't a digital wallet, Stripe creates and attaches a [generated_card](/api/charges/object#charge_object-payment_method_details-card_present-generated_card) payment method representing the card to the Customer instead.\n\nWhen processing card payments, Stripe uses `setup_future_usage` to help you comply with regional legislation and network rules, such as [SCA](/strong-customer-authentication).",
			Enum: []resource.EnumSpec{
				{Value: "none"},
				{Value: "off_session"},
				{Value: "on_session"},
			},
		},
		"payment_method_options.konbini.setup_future_usage": {
			Type:        "string",
			Description: "Indicates that you intend to make future payments with this PaymentIntent's payment method.\n\nIf you provide a Customer with the PaymentIntent, you can use this parameter to [attach the payment method](/payments/save-during-payment) to the Customer after the PaymentIntent is confirmed and the customer completes any required actions. If you don't provide a Customer, you can still [attach](/api/payment_methods/attach) the payment method to a Customer after the transaction completes.\n\nIf the payment method is `card_present` and isn't a digital wallet, Stripe creates and attaches a [generated_card](/api/charges/object#charge_object-payment_method_details-card_present-generated_card) payment method representing the card to the Customer instead.\n\nWhen processing card payments, Stripe uses `setup_future_usage` to help you comply with regional legislation and network rules, such as [SCA](/strong-customer-authentication).",
			Enum: []resource.EnumSpec{
				{Value: "none"},
			},
		},
		"payment_method_options.link.capture_method": {
			Type:        "string",
			Description: "Controls when the funds will be captured from the customer's account.",
			Enum: []resource.EnumSpec{
				{Value: "manual"},
			},
		},
		"mode": {
			Type:        "string",
			Description: "The mode of the Checkout Session. Pass `subscription` if the Checkout Session includes at least one recurring item.",
			Enum: []resource.EnumSpec{
				{Value: "payment"},
				{Value: "setup"},
				{Value: "subscription"},
			},
		},
		"customer_account": {
			Type:        "string",
			Description: "ID of an existing Account, if one exists. Has the same behavior as `customer`.",
		},
		"branding_settings.background_color": {
			Type:        "string",
			Description: "A hex color value starting with `#` representing the background color for the Checkout Session.",
		},
		"payment_intent_data.shipping.address.postal_code": {
			Type:        "string",
			Description: "ZIP or postal code.",
		},
		"payment_method_options.kakao_pay.setup_future_usage": {
			Type:        "string",
			Description: "Indicates that you intend to make future payments with this PaymentIntent's payment method.\n\nIf you provide a Customer with the PaymentIntent, you can use this parameter to [attach the payment method](/payments/save-during-payment) to the Customer after the PaymentIntent is confirmed and the customer completes any required actions. If you don't provide a Customer, you can still [attach](/api/payment_methods/attach) the payment method to a Customer after the transaction completes.\n\nIf the payment method is `card_present` and isn't a digital wallet, Stripe creates and attaches a [generated_card](/api/charges/object#charge_object-payment_method_details-card_present-generated_card) payment method representing the card to the Customer instead.\n\nWhen processing card payments, Stripe uses `setup_future_usage` to help you comply with regional legislation and network rules, such as [SCA](/strong-customer-authentication).",
			Enum: []resource.EnumSpec{
				{Value: "none"},
				{Value: "off_session"},
			},
		},
		"payment_method_options.fpx.setup_future_usage": {
			Type:        "string",
			Description: "Indicates that you intend to make future payments with this PaymentIntent's payment method.\n\nIf you provide a Customer with the PaymentIntent, you can use this parameter to [attach the payment method](/payments/save-during-payment) to the Customer after the PaymentIntent is confirmed and the customer completes any required actions. If you don't provide a Customer, you can still [attach](/api/payment_methods/attach) the payment method to a Customer after the transaction completes.\n\nIf the payment method is `card_present` and isn't a digital wallet, Stripe creates and attaches a [generated_card](/api/charges/object#charge_object-payment_method_details-card_present-generated_card) payment method representing the card to the Customer instead.\n\nWhen processing card payments, Stripe uses `setup_future_usage` to help you comply with regional legislation and network rules, such as [SCA](/strong-customer-authentication).",
			Enum: []resource.EnumSpec{
				{Value: "none"},
			},
		},
		"payment_method_options.au_becs_debit.setup_future_usage": {
			Type:        "string",
			Description: "Indicates that you intend to make future payments with this PaymentIntent's payment method.\n\nIf you provide a Customer with the PaymentIntent, you can use this parameter to [attach the payment method](/payments/save-during-payment) to the Customer after the PaymentIntent is confirmed and the customer completes any required actions. If you don't provide a Customer, you can still [attach](/api/payment_methods/attach) the payment method to a Customer after the transaction completes.\n\nIf the payment method is `card_present` and isn't a digital wallet, Stripe creates and attaches a [generated_card](/api/charges/object#charge_object-payment_method_details-card_present-generated_card) payment method representing the card to the Customer instead.\n\nWhen processing card payments, Stripe uses `setup_future_usage` to help you comply with regional legislation and network rules, such as [SCA](/strong-customer-authentication).",
			Enum: []resource.EnumSpec{
				{Value: "none"},
			},
		},
		"branding_settings.button_color": {
			Type:        "string",
			Description: "A hex color value starting with `#` representing the button color for the Checkout Session.",
		},
		"phone_number_collection.enabled": {
			Type:        "boolean",
			Description: "Set to `true` to enable phone number collection.\n\nCan only be set in `payment` and `subscription` mode.",
			Required:    true,
		},
		"consent_collection.payment_method_reuse_agreement.position": {
			Type:        "string",
			Description: "Determines the position and visibility of the payment method reuse agreement in the UI. When set to `auto`, Stripe's\ndefaults will be used. When set to `hidden`, the payment method reuse agreement text will always be hidden in the UI.",
			Required:    true,
			Enum: []resource.EnumSpec{
				{Value: "auto"},
				{Value: "hidden"},
			},
		},
		"payment_method_options.bacs_debit.target_date": {
			Type:        "string",
			Description: "Controls when Stripe will attempt to debit the funds from the customer's account. The date must be a string in YYYY-MM-DD format. The date must be in the future and between 3 and 15 calendar days from now.",
		},
		"payment_method_options.card.request_incremental_authorization": {
			Type:        "string",
			Description: "Request ability to [increment the authorization](/payments/incremental-authorization) for this CheckoutSession.",
			Enum: []resource.EnumSpec{
				{Value: "if_available"},
				{Value: "never"},
			},
		},
		"payment_method_options.payto.mandate_options.start_date": {
			Type:        "string",
			Description: "Date, in YYYY-MM-DD format, from which payments will be collected. Defaults to confirmation time.",
		},
		"subscription_data.transfer_data.amount_percent": {
			Type:        "number",
			Description: "A non-negative decimal between 0 and 100, with at most two decimal places. This represents the percentage of the subscription invoice total that will be transferred to the destination account. By default, the entire amount is transferred to the destination.",
		},
		"subscription_data.trial_end": {
			Type:        "integer",
			Description: "Unix timestamp representing the end of the trial period the customer will get before being charged for the first time. Has to be at least 48 hours in the future.",
			Format:      "unix-time",
		},
		"wallet_options.link.display": {
			Type:        "string",
			Description: "Specifies whether Checkout should display Link as a payment option. By default, Checkout will display all the supported wallets that the Checkout Session was created with. This is the `auto` behavior, and it is the default choice.",
			Enum: []resource.EnumSpec{
				{Value: "auto"},
				{Value: "never"},
			},
		},
		"subscription_data.trial_period_days": {
			Type:        "integer",
			Description: "Integer representing the number of trial period days before the customer is charged for the first time. Has to be at least 1.",
		},
		"redirect_on_completion": {
			Type:        "string",
			Description: "This parameter applies to `ui_mode: embedded`. Learn more about the [redirect behavior](https://docs.stripe.com/payments/checkout/custom-success-page?payment-ui=embedded-form) of embedded sessions. Defaults to `always`.",
			Enum: []resource.EnumSpec{
				{Value: "always"},
				{Value: "if_required"},
				{Value: "never"},
			},
		},
		"saved_payment_method_options.allow_redisplay_filters": {
			Type:        "array",
			Description: "Uses the `allow_redisplay` value of each saved payment method to filter the set presented to a returning customer. By default, only saved payment methods with ’allow_redisplay: ‘always’ are shown in Checkout.",
		},
		"tax_id_collection.required": {
			Type:        "string",
			Description: "Describes whether a tax ID is required during checkout. Defaults to `never`. You can't set this parameter if `ui_mode` is `custom`.",
			Enum: []resource.EnumSpec{
				{Value: "if_supported"},
				{Value: "never"},
			},
		},
		"client_reference_id": {
			Type:        "string",
			Description: "A unique string to reference the Checkout Session. This can be a\ncustomer ID, a cart ID, or similar, and can be used to reconcile the\nsession with your internal systems.",
		},
		"payment_method_options.acss_debit.mandate_options.default_for": {
			Type:        "array",
			Description: "List of Stripe products where this mandate can be selected automatically. Only usable in `setup` mode.",
		},
		"payment_method_options.bancontact.setup_future_usage": {
			Type:        "string",
			Description: "Indicates that you intend to make future payments with this PaymentIntent's payment method.\n\nIf you provide a Customer with the PaymentIntent, you can use this parameter to [attach the payment method](/payments/save-during-payment) to the Customer after the PaymentIntent is confirmed and the customer completes any required actions. If you don't provide a Customer, you can still [attach](/api/payment_methods/attach) the payment method to a Customer after the transaction completes.\n\nIf the payment method is `card_present` and isn't a digital wallet, Stripe creates and attaches a [generated_card](/api/charges/object#charge_object-payment_method_details-card_present-generated_card) payment method representing the card to the Customer instead.\n\nWhen processing card payments, Stripe uses `setup_future_usage` to help you comply with regional legislation and network rules, such as [SCA](/strong-customer-authentication).",
			Enum: []resource.EnumSpec{
				{Value: "none"},
			},
		},
		"saved_payment_method_options.payment_method_remove": {
			Type:        "string",
			Description: "Enable customers to choose if they wish to remove their saved payment methods. Disabled by default.",
			Enum: []resource.EnumSpec{
				{Value: "disabled"},
				{Value: "enabled"},
			},
		},
		"currency": {
			Type:        "string",
			Description: "Three-letter [ISO currency code](https://www.iso.org/iso-4217-currency-codes.html), in lowercase. Must be a [supported currency](https://stripe.com/docs/currencies). Required in `setup` mode when `payment_method_types` is not set.",
			Format:      "currency",
		},
		"payment_intent_data.shipping.address.state": {
			Type:        "string",
			Description: "State, county, province, or region ([ISO 3166-2](https://en.wikipedia.org/wiki/ISO_3166-2)).",
		},
		"expires_at": {
			Type:        "integer",
			Description: "The Epoch time in seconds at which the Checkout Session will expire. It can be anywhere from 30 minutes to 24 hours after Checkout Session creation. By default, this value is 24 hours from creation.",
			Format:      "unix-time",
		},
		"payment_method_options.card.request_three_d_secure": {
			Type:        "string",
			Description: "We strongly recommend that you rely on our SCA Engine to automatically prompt your customers for authentication based on risk level and [other requirements](https://docs.stripe.com/strong-customer-authentication). However, if you wish to request 3D Secure based on logic from your own fraud engine, provide this option. If not provided, this value defaults to `automatic`. Read our guide on [manually requesting 3D Secure](https://docs.stripe.com/payments/3d-secure/authentication-flow#manual-three-ds) for more information on how this configuration interacts with Radar and our SCA Engine.",
			Enum: []resource.EnumSpec{
				{Value: "any"},
				{Value: "automatic"},
				{Value: "challenge"},
			},
		},
		"payment_method_options.acss_debit.target_date": {
			Type:        "string",
			Description: "Controls when Stripe will attempt to debit the funds from the customer's account. The date must be a string in YYYY-MM-DD format. The date must be in the future and between 3 and 15 calendar days from now.",
		},
		"payment_method_options.samsung_pay.capture_method": {
			Type:        "string",
			Description: "Controls when the funds will be captured from the customer's account.",
			Enum: []resource.EnumSpec{
				{Value: "manual"},
			},
		},
		"permissions.update_shipping_details": {
			Type:        "string",
			Description: "Determines which entity is allowed to update the shipping details.\n\nDefault is `client_only`. Stripe Checkout client will automatically update the shipping details. If set to `server_only`, only your server is allowed to update the shipping details.\n\nWhen set to `server_only`, you must add the onShippingDetailsChange event handler when initializing the Stripe Checkout client and manually update the shipping details from your server using the Stripe API.",
			Enum: []resource.EnumSpec{
				{Value: "client_only"},
				{Value: "server_only"},
			},
		},
		"customer_update.name": {
			Type:        "string",
			Description: "Describes whether Checkout saves the name onto `customer.name`. Defaults to `never`.",
			Enum: []resource.EnumSpec{
				{Value: "auto"},
				{Value: "never"},
			},
		},
		"after_expiration.recovery.allow_promotion_codes": {
			Type:        "boolean",
			Description: "Enables user redeemable promotion codes on the recovered Checkout Sessions. Defaults to `false`",
		},
		"branding_settings.logo.url": {
			Type:        "string",
			Description: "The URL of the image. Required if `type` is `url` and disallowed otherwise.",
		},
		"payment_method_options.acss_debit.mandate_options.transaction_type": {
			Type:        "string",
			Description: "Transaction type of the mandate.",
			Enum: []resource.EnumSpec{
				{Value: "business"},
				{Value: "personal"},
			},
		},
		"payment_method_options.eps.setup_future_usage": {
			Type:        "string",
			Description: "Indicates that you intend to make future payments with this PaymentIntent's payment method.\n\nIf you provide a Customer with the PaymentIntent, you can use this parameter to [attach the payment method](/payments/save-during-payment) to the Customer after the PaymentIntent is confirmed and the customer completes any required actions. If you don't provide a Customer, you can still [attach](/api/payment_methods/attach) the payment method to a Customer after the transaction completes.\n\nIf the payment method is `card_present` and isn't a digital wallet, Stripe creates and attaches a [generated_card](/api/charges/object#charge_object-payment_method_details-card_present-generated_card) payment method representing the card to the Customer instead.\n\nWhen processing card payments, Stripe uses `setup_future_usage` to help you comply with regional legislation and network rules, such as [SCA](/strong-customer-authentication).",
			Enum: []resource.EnumSpec{
				{Value: "none"},
			},
		},
		"payment_method_options.au_becs_debit.target_date": {
			Type:        "string",
			Description: "Controls when Stripe will attempt to debit the funds from the customer's account. The date must be a string in YYYY-MM-DD format. The date must be in the future and between 3 and 15 calendar days from now.",
		},
		"subscription_data.default_tax_rates": {
			Type:        "array",
			Description: "The tax rates that will apply to any subscription item that does not have\n`tax_rates` set. Invoices created will have their `default_tax_rates` populated\nfrom the subscription.",
		},
		"subscription_data.transfer_data.destination": {
			Type:        "string",
			Description: "ID of an existing, connected Stripe account.",
			Required:    true,
		},
		"payment_intent_data.transfer_data.destination": {
			Type:        "string",
			Description: "If specified, successful charges will be attributed to the destination\naccount for tax reporting, and the funds from charges will be transferred\nto the destination account. The ID of the resulting transfer will be\nreturned on the successful charge's `transfer` field.",
			Required:    true,
		},
		"automatic_tax.liability.account": {
			Type:        "string",
			Description: "The connected account being referenced when `type` is `account`.",
		},
		"payment_method_options.sofort.setup_future_usage": {
			Type:        "string",
			Description: "Indicates that you intend to make future payments with this PaymentIntent's payment method.\n\nIf you provide a Customer with the PaymentIntent, you can use this parameter to [attach the payment method](/payments/save-during-payment) to the Customer after the PaymentIntent is confirmed and the customer completes any required actions. If you don't provide a Customer, you can still [attach](/api/payment_methods/attach) the payment method to a Customer after the transaction completes.\n\nIf the payment method is `card_present` and isn't a digital wallet, Stripe creates and attaches a [generated_card](/api/charges/object#charge_object-payment_method_details-card_present-generated_card) payment method representing the card to the Customer instead.\n\nWhen processing card payments, Stripe uses `setup_future_usage` to help you comply with regional legislation and network rules, such as [SCA](/strong-customer-authentication).",
			Enum: []resource.EnumSpec{
				{Value: "none"},
			},
		},
		"payment_method_options.pix.amount_includes_iof": {
			Type:        "string",
			Description: "Determines if the amount includes the IOF tax. Defaults to `never`.",
			Enum: []resource.EnumSpec{
				{Value: "always"},
				{Value: "never"},
			},
		},
		"invoice_creation.enabled": {
			Type:        "boolean",
			Description: "Set to `true` to enable invoice creation.",
			Required:    true,
		},
		"customer_email": {
			Type:        "string",
			Description: "If provided, this value will be used when the Customer object is created.\nIf not provided, customers will be asked to enter their email address.\nUse this parameter to prefill customer data if you already have an email\non file. To access information about the customer once a session is\ncomplete, use the `customer` field.",
		},
		"payment_method_data.allow_redisplay": {
			Type:        "string",
			Description: "Allow redisplay will be set on the payment method on confirmation and indicates whether this payment method can be shown again to the customer in a checkout flow. Only set this field if you wish to override the allow_redisplay value determined by Checkout.",
			Enum: []resource.EnumSpec{
				{Value: "always"},
				{Value: "limited"},
				{Value: "unspecified"},
			},
		},
		"invoice_creation.invoice_data.description": {
			Type:        "string",
			Description: "An arbitrary string attached to the object. Often useful for displaying to users.",
		},
		"subscription_data.invoice_settings.issuer.account": {
			Type:        "string",
			Description: "The connected account being referenced when `type` is `account`.",
		},
		"customer_update.address": {
			Type:        "string",
			Description: "Describes whether Checkout saves the billing address onto `customer.address`.\nTo always collect a full billing address, use `billing_address_collection`. Defaults to `never`.",
			Enum: []resource.EnumSpec{
				{Value: "auto"},
				{Value: "never"},
			},
		},
		"adaptive_pricing.enabled": {
			Type:        "boolean",
			Description: "If set to `true`, Adaptive Pricing is available on [eligible sessions](https://docs.stripe.com/payments/currencies/localize-prices/adaptive-pricing?payment-ui=stripe-hosted#restrictions). Defaults to your [dashboard setting](https://dashboard.stripe.com/settings/adaptive-pricing).",
		},
		"branding_settings.logo.file": {
			Type:        "string",
			Description: "The ID of a [File upload](https://stripe.com/docs/api/files) representing the logo. Purpose must be `business_logo`. Required if `type` is `file` and disallowed otherwise.",
		},
		"locale": {
			Type:        "string",
			Description: "The IETF language tag of the locale Checkout is displayed in. If blank or `auto`, the browser's locale is used.",
			Enum: []resource.EnumSpec{
				{Value: "auto"},
				{Value: "bg"},
				{Value: "cs"},
				{Value: "da"},
				{Value: "de"},
				{Value: "el"},
				{Value: "en"},
				{Value: "en-GB"},
				{Value: "es"},
				{Value: "es-419"},
				{Value: "et"},
				{Value: "fi"},
				{Value: "fil"},
				{Value: "fr"},
				{Value: "fr-CA"},
				{Value: "hr"},
				{Value: "hu"},
				{Value: "id"},
				{Value: "it"},
				{Value: "ja"},
				{Value: "ko"},
				{Value: "lt"},
				{Value: "lv"},
				{Value: "ms"},
				{Value: "mt"},
				{Value: "nb"},
				{Value: "nl"},
				{Value: "pl"},
				{Value: "pt"},
				{Value: "pt-BR"},
				{Value: "ro"},
				{Value: "ru"},
				{Value: "sk"},
				{Value: "sl"},
				{Value: "sv"},
				{Value: "th"},
				{Value: "tr"},
				{Value: "vi"},
				{Value: "zh"},
				{Value: "zh-HK"},
				{Value: "zh-TW"},
			},
		},
		"payment_method_options.card.statement_descriptor_suffix_kana": {
			Type:        "string",
			Description: "Provides information about a card payment that customers see on their statements. Concatenated with the Kana prefix (shortened Kana descriptor) or Kana statement descriptor that’s set on the account to form the complete statement descriptor. Maximum 22 characters. On card statements, the *concatenation* of both prefix and suffix (including separators) will appear truncated to 22 characters.",
		},
		"payment_method_options.paynow.setup_future_usage": {
			Type:        "string",
			Description: "Indicates that you intend to make future payments with this PaymentIntent's payment method.\n\nIf you provide a Customer with the PaymentIntent, you can use this parameter to [attach the payment method](/payments/save-during-payment) to the Customer after the PaymentIntent is confirmed and the customer completes any required actions. If you don't provide a Customer, you can still [attach](/api/payment_methods/attach) the payment method to a Customer after the transaction completes.\n\nIf the payment method is `card_present` and isn't a digital wallet, Stripe creates and attaches a [generated_card](/api/charges/object#charge_object-payment_method_details-card_present-generated_card) payment method representing the card to the Customer instead.\n\nWhen processing card payments, Stripe uses `setup_future_usage` to help you comply with regional legislation and network rules, such as [SCA](/strong-customer-authentication).",
			Enum: []resource.EnumSpec{
				{Value: "none"},
			},
		},
		"payment_method_options.klarna.setup_future_usage": {
			Type:        "string",
			Description: "Indicates that you intend to make future payments with this PaymentIntent's payment method.\n\nIf you provide a Customer with the PaymentIntent, you can use this parameter to [attach the payment method](/payments/save-during-payment) to the Customer after the PaymentIntent is confirmed and the customer completes any required actions. If you don't provide a Customer, you can still [attach](/api/payment_methods/attach) the payment method to a Customer after the transaction completes.\n\nIf the payment method is `card_present` and isn't a digital wallet, Stripe creates and attaches a [generated_card](/api/charges/object#charge_object-payment_method_details-card_present-generated_card) payment method representing the card to the Customer instead.\n\nWhen processing card payments, Stripe uses `setup_future_usage` to help you comply with regional legislation and network rules, such as [SCA](/strong-customer-authentication).",
			Enum: []resource.EnumSpec{
				{Value: "none"},
			},
		},
		"payment_method_options.payco.capture_method": {
			Type:        "string",
			Description: "Controls when the funds will be captured from the customer's account.",
			Enum: []resource.EnumSpec{
				{Value: "manual"},
			},
		},
		"payment_method_options.oxxo.expires_after_days": {
			Type:        "integer",
			Description: "The number of calendar days before an OXXO voucher expires. For example, if you create an OXXO voucher on Monday and you set expires_after_days to 2, the OXXO invoice will expire on Wednesday at 23:59 America/Mexico_City time.",
		},
		"payment_method_options.customer_balance.bank_transfer.requested_address_types": {
			Type:        "array",
			Description: "List of address types that should be returned in the financial_addresses response. If not specified, all valid types will be returned.\n\nPermitted values include: `sort_code`, `zengin`, `iban`, or `spei`.",
		},
		"payment_method_options.kr_card.capture_method": {
			Type:        "string",
			Description: "Controls when the funds will be captured from the customer's account.",
			Enum: []resource.EnumSpec{
				{Value: "manual"},
			},
		},
		"customer_creation": {
			Type:        "string",
			Description: "Configure whether a Checkout Session creates a [Customer](https://docs.stripe.com/api/customers) during Session confirmation.\n\nWhen a Customer is not created, you can still retrieve email, address, and other customer data entered in Checkout\nwith [customer_details](https://docs.stripe.com/api/checkout/sessions/object#checkout_session_object-customer_details).\n\nSessions that don't create Customers instead are grouped by [guest customers](https://docs.stripe.com/payments/checkout/guest-customers)\nin the Dashboard. Promotion codes limited to first time customers will return invalid for these Sessions.\n\nCan only be set in `payment` and `setup` mode.",
			Enum: []resource.EnumSpec{
				{Value: "always"},
				{Value: "if_required"},
			},
		},
		"payment_method_collection": {
			Type:        "string",
			Description: "Specify whether Checkout should collect a payment method. When set to `if_required`, Checkout will not collect a payment method when the total due for the session is 0.\nThis may occur if the Checkout Session includes a free trial or a discount.\n\nCan only be set in `subscription` mode. Defaults to `always`.\n\nIf you'd like information on how to collect a payment method outside of Checkout, read the guide on configuring [subscriptions with a free trial](https://docs.stripe.com/payments/checkout/free-trials).",
			Enum: []resource.EnumSpec{
				{Value: "always"},
				{Value: "if_required"},
			},
		},
		"billing_address_collection": {
			Type:        "string",
			Description: "Specify whether Checkout should collect the customer's billing address. Defaults to `auto`.",
			Enum: []resource.EnumSpec{
				{Value: "auto"},
				{Value: "required"},
			},
		},
		"consent_collection.promotions": {
			Type:        "string",
			Description: "If set to `auto`, enables the collection of customer consent for promotional communications. The Checkout\nSession will determine whether to display an option to opt into promotional communication\nfrom the merchant depending on the customer's locale. Only available to US merchants.",
			Enum: []resource.EnumSpec{
				{Value: "auto"},
				{Value: "none"},
			},
		},
		"payment_method_options.amazon_pay.setup_future_usage": {
			Type:        "string",
			Description: "Indicates that you intend to make future payments with this PaymentIntent's payment method.\n\nIf you provide a Customer with the PaymentIntent, you can use this parameter to [attach the payment method](/payments/save-during-payment) to the Customer after the PaymentIntent is confirmed and the customer completes any required actions. If you don't provide a Customer, you can still [attach](/api/payment_methods/attach) the payment method to a Customer after the transaction completes.\n\nIf the payment method is `card_present` and isn't a digital wallet, Stripe creates and attaches a [generated_card](/api/charges/object#charge_object-payment_method_details-card_present-generated_card) payment method representing the card to the Customer instead.\n\nWhen processing card payments, Stripe uses `setup_future_usage` to help you comply with regional legislation and network rules, such as [SCA](/strong-customer-authentication).",
			Enum: []resource.EnumSpec{
				{Value: "none"},
				{Value: "off_session"},
			},
		},
	},
}
