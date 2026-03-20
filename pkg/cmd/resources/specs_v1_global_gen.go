// This file is generated; DO NOT EDIT.

package resources

import "github.com/stripe/stripe-cli/pkg/cmd/resource"

var V1MandatesRetrieve = resource.OperationSpec{
	Name:    "retrieve",
	Path:    "/v1/mandates/{mandate}",
	Method:  "GET",
	Summary: "Retrieve a Mandate",
}

var V1AccountSessionsCreate = resource.OperationSpec{
	Name:    "create",
	Path:    "/v1/account_sessions",
	Method:  "POST",
	Summary: "Create an Account Session",
	Params: map[string]*resource.ParamSpec{
		"components.balances.features.standard_payouts": {
			Type:        "boolean",
			Description: "Whether to allow creation of standard payouts. Defaults to `true` when `controller.losses.payments` is set to `stripe` for the account, otherwise `false`.",
		},
		"components.payments.features.dispute_management": {
			Type:        "boolean",
			Description: "Whether responding to disputes is enabled, including submitting evidence and accepting disputes. This is `true` by default.",
		},
		"components.payment_details.features.destination_on_behalf_of_charge_management": {
			Type:        "boolean",
			Description: "Whether connected accounts can manage destination charges that are created on behalf of them. This is `false` by default.",
		},
		"components.issuing_card.features.cardholder_management": {
			Type:        "boolean",
			Description: "Whether to allow cardholder management features.",
		},
		"components.financial_account.features.transfer_balance": {
			Type:        "boolean",
			Description: "Whether to allow transferring balance.",
		},
		"components.payment_disputes.features.destination_on_behalf_of_charge_management": {
			Type:        "boolean",
			Description: "Whether connected accounts can manage destination charges that are created on behalf of them. This is `false` by default.",
		},
		"components.payment_details.enabled": {
			Type:        "boolean",
			Description: "Whether the embedded component is enabled.",
			Required:    true,
		},
		"components.instant_payouts_promotion.enabled": {
			Type:        "boolean",
			Description: "Whether the embedded component is enabled.",
			Required:    true,
		},
		"components.issuing_cards_list.features.disable_stripe_user_authentication": {
			Type:        "boolean",
			Description: "Whether Stripe user authentication is disabled. This value can only be `true` for accounts where `controller.requirement_collection` is `application` for the account. The default value is the opposite of the `external_account_collection` value. For example, if you don't set `external_account_collection`, it defaults to `true` and `disable_stripe_user_authentication` defaults to `false`.",
		},
		"components.tax_settings.enabled": {
			Type:        "boolean",
			Description: "Whether the embedded component is enabled.",
			Required:    true,
		},
		"components.financial_account_transactions.enabled": {
			Type:        "boolean",
			Description: "Whether the embedded component is enabled.",
			Required:    true,
		},
		"components.payment_details.features.capture_payments": {
			Type:        "boolean",
			Description: "Whether to allow capturing and cancelling payment intents. This is `true` by default.",
		},
		"components.account_onboarding.features.disable_stripe_user_authentication": {
			Type:        "boolean",
			Description: "Whether Stripe user authentication is disabled. This value can only be `true` for accounts where `controller.requirement_collection` is `application` for the account. The default value is the opposite of the `external_account_collection` value. For example, if you don't set `external_account_collection`, it defaults to `true` and `disable_stripe_user_authentication` defaults to `false`.",
		},
		"components.issuing_card.features.card_spend_dispute_management": {
			Type:        "boolean",
			Description: "Whether to allow card spend dispute management features.",
		},
		"components.issuing_cards_list.features.card_spend_dispute_management": {
			Type:        "boolean",
			Description: "Whether to allow card spend dispute management features.",
		},
		"components.issuing_card.enabled": {
			Type:        "boolean",
			Description: "Whether the embedded component is enabled.",
			Required:    true,
		},
		"components.notification_banner.enabled": {
			Type:        "boolean",
			Description: "Whether the embedded component is enabled.",
			Required:    true,
		},
		"components.disputes_list.features.dispute_management": {
			Type:        "boolean",
			Description: "Whether responding to disputes is enabled, including submitting evidence and accepting disputes. This is `true` by default.",
		},
		"components.tax_registrations.enabled": {
			Type:        "boolean",
			Description: "Whether the embedded component is enabled.",
			Required:    true,
		},
		"components.issuing_card.features.card_management": {
			Type:        "boolean",
			Description: "Whether to allow card management features.",
		},
		"components.documents.enabled": {
			Type:        "boolean",
			Description: "Whether the embedded component is enabled.",
			Required:    true,
		},
		"components.financial_account.features.send_money": {
			Type:        "boolean",
			Description: "Whether to allow sending money.",
		},
		"components.issuing_cards_list.features.card_management": {
			Type:        "boolean",
			Description: "Whether to allow card management features.",
		},
		"components.payment_details.features.dispute_management": {
			Type:        "boolean",
			Description: "Whether responding to disputes is enabled, including submitting evidence and accepting disputes. This is `true` by default.",
		},
		"components.balances.features.external_account_collection": {
			Type:        "boolean",
			Description: "Whether external account collection is enabled. This feature can only be `false` for accounts where you’re responsible for collecting updated information when requirements are due or change, like Custom accounts. The default value for this feature is `true`.",
		},
		"components.payments.features.refund_management": {
			Type:        "boolean",
			Description: "Whether sending refunds is enabled. This is `true` by default.",
		},
		"components.payouts.features.external_account_collection": {
			Type:        "boolean",
			Description: "Whether external account collection is enabled. This feature can only be `false` for accounts where you’re responsible for collecting updated information when requirements are due or change, like Custom accounts. The default value for this feature is `true`.",
		},
		"components.issuing_cards_list.features.cardholder_management": {
			Type:        "boolean",
			Description: "Whether to allow cardholder management features.",
		},
		"components.payouts.features.edit_payout_schedule": {
			Type:        "boolean",
			Description: "Whether to allow payout schedule to be changed. Defaults to `true` when `controller.losses.payments` is set to `stripe` for the account, otherwise `false`.",
		},
		"components.financial_account.features.external_account_collection": {
			Type:        "boolean",
			Description: "Whether external account collection is enabled. This feature can only be `false` for accounts where you’re responsible for collecting updated information when requirements are due or change, like Custom accounts. The default value for this feature is `true`.",
		},
		"components.financial_account_transactions.features.card_spend_dispute_management": {
			Type:        "boolean",
			Description: "Whether to allow card spend dispute management features.",
		},
		"components.account_management.enabled": {
			Type:        "boolean",
			Description: "Whether the embedded component is enabled.",
			Required:    true,
		},
		"components.instant_payouts_promotion.features.external_account_collection": {
			Type:        "boolean",
			Description: "Whether external account collection is enabled. This feature can only be `false` for accounts where you’re responsible for collecting updated information when requirements are due or change, like Custom accounts. The default value for this feature is `true`.",
		},
		"components.payments.enabled": {
			Type:        "boolean",
			Description: "Whether the embedded component is enabled.",
			Required:    true,
		},
		"components.payments.features.capture_payments": {
			Type:        "boolean",
			Description: "Whether to allow capturing and cancelling payment intents. This is `true` by default.",
		},
		"components.issuing_cards_list.enabled": {
			Type:        "boolean",
			Description: "Whether the embedded component is enabled.",
			Required:    true,
		},
		"components.notification_banner.features.disable_stripe_user_authentication": {
			Type:        "boolean",
			Description: "Whether Stripe user authentication is disabled. This value can only be `true` for accounts where `controller.requirement_collection` is `application` for the account. The default value is the opposite of the `external_account_collection` value. For example, if you don't set `external_account_collection`, it defaults to `true` and `disable_stripe_user_authentication` defaults to `false`.",
		},
		"components.payment_details.features.refund_management": {
			Type:        "boolean",
			Description: "Whether sending refunds is enabled. This is `true` by default.",
		},
		"components.disputes_list.features.destination_on_behalf_of_charge_management": {
			Type:        "boolean",
			Description: "Whether connected accounts can manage destination charges that are created on behalf of them. This is `false` by default.",
		},
		"components.account_onboarding.enabled": {
			Type:        "boolean",
			Description: "Whether the embedded component is enabled.",
			Required:    true,
		},
		"components.issuing_card.features.spend_control_management": {
			Type:        "boolean",
			Description: "Whether to allow spend control management features.",
		},
		"components.payment_disputes.features.refund_management": {
			Type:        "boolean",
			Description: "Whether sending refunds is enabled. This is `true` by default.",
		},
		"components.balances.features.disable_stripe_user_authentication": {
			Type:        "boolean",
			Description: "Whether Stripe user authentication is disabled. This value can only be `true` for accounts where `controller.requirement_collection` is `application` for the account. The default value is the opposite of the `external_account_collection` value. For example, if you don't set `external_account_collection`, it defaults to `true` and `disable_stripe_user_authentication` defaults to `false`.",
		},
		"components.instant_payouts_promotion.features.instant_payouts": {
			Type:        "boolean",
			Description: "Whether instant payouts are enabled for this component.",
		},
		"components.payouts.features.instant_payouts": {
			Type:        "boolean",
			Description: "Whether instant payouts are enabled for this component.",
		},
		"components.financial_account.features.disable_stripe_user_authentication": {
			Type:        "boolean",
			Description: "Whether Stripe user authentication is disabled. This value can only be `true` for accounts where `controller.requirement_collection` is `application` for the account. The default value is the opposite of the `external_account_collection` value. For example, if you don't set `external_account_collection`, it defaults to `true` and `disable_stripe_user_authentication` defaults to `false`.",
		},
		"components.notification_banner.features.external_account_collection": {
			Type:        "boolean",
			Description: "Whether external account collection is enabled. This feature can only be `false` for accounts where you’re responsible for collecting updated information when requirements are due or change, like Custom accounts. The default value for this feature is `true`.",
		},
		"components.account_management.features.external_account_collection": {
			Type:        "boolean",
			Description: "Whether external account collection is enabled. This feature can only be `false` for accounts where you’re responsible for collecting updated information when requirements are due or change, like Custom accounts. The default value for this feature is `true`.",
		},
		"components.disputes_list.enabled": {
			Type:        "boolean",
			Description: "Whether the embedded component is enabled.",
			Required:    true,
		},
		"components.disputes_list.features.capture_payments": {
			Type:        "boolean",
			Description: "Whether to allow capturing and cancelling payment intents. This is `true` by default.",
		},
		"components.balances.enabled": {
			Type:        "boolean",
			Description: "Whether the embedded component is enabled.",
			Required:    true,
		},
		"components.payouts.features.disable_stripe_user_authentication": {
			Type:        "boolean",
			Description: "Whether Stripe user authentication is disabled. This value can only be `true` for accounts where `controller.requirement_collection` is `application` for the account. The default value is the opposite of the `external_account_collection` value. For example, if you don't set `external_account_collection`, it defaults to `true` and `disable_stripe_user_authentication` defaults to `false`.",
		},
		"components.payment_disputes.features.dispute_management": {
			Type:        "boolean",
			Description: "Whether responding to disputes is enabled, including submitting evidence and accepting disputes. This is `true` by default.",
		},
		"components.account_onboarding.features.external_account_collection": {
			Type:        "boolean",
			Description: "Whether external account collection is enabled. This feature can only be `false` for accounts where you’re responsible for collecting updated information when requirements are due or change, like Custom accounts. The default value for this feature is `true`.",
		},
		"components.payments.features.destination_on_behalf_of_charge_management": {
			Type:        "boolean",
			Description: "Whether connected accounts can manage destination charges that are created on behalf of them. This is `false` by default.",
		},
		"components.payment_disputes.enabled": {
			Type:        "boolean",
			Description: "Whether the embedded component is enabled.",
			Required:    true,
		},
		"account": {
			Type:        "string",
			Description: "The identifier of the account to create an Account Session for.",
			Required:    true,
		},
		"components.payouts_list.enabled": {
			Type:        "boolean",
			Description: "Whether the embedded component is enabled.",
			Required:    true,
		},
		"components.disputes_list.features.refund_management": {
			Type:        "boolean",
			Description: "Whether sending refunds is enabled. This is `true` by default.",
		},
		"components.payouts.enabled": {
			Type:        "boolean",
			Description: "Whether the embedded component is enabled.",
			Required:    true,
		},
		"components.payout_details.enabled": {
			Type:        "boolean",
			Description: "Whether the embedded component is enabled.",
			Required:    true,
		},
		"components.balances.features.edit_payout_schedule": {
			Type:        "boolean",
			Description: "Whether to allow payout schedule to be changed. Defaults to `true` when `controller.losses.payments` is set to `stripe` for the account, otherwise `false`.",
		},
		"components.financial_account.enabled": {
			Type:        "boolean",
			Description: "Whether the embedded component is enabled.",
			Required:    true,
		},
		"components.issuing_cards_list.features.spend_control_management": {
			Type:        "boolean",
			Description: "Whether to allow spend control management features.",
		},
		"components.account_management.features.disable_stripe_user_authentication": {
			Type:        "boolean",
			Description: "Whether Stripe user authentication is disabled. This value can only be `true` for accounts where `controller.requirement_collection` is `application` for the account. The default value is the opposite of the `external_account_collection` value. For example, if you don't set `external_account_collection`, it defaults to `true` and `disable_stripe_user_authentication` defaults to `false`.",
		},
		"components.balances.features.instant_payouts": {
			Type:        "boolean",
			Description: "Whether instant payouts are enabled for this component.",
		},
		"components.instant_payouts_promotion.features.disable_stripe_user_authentication": {
			Type:        "boolean",
			Description: "Whether Stripe user authentication is disabled. This value can only be `true` for accounts where `controller.requirement_collection` is `application` for the account. The default value is the opposite of the `external_account_collection` value. For example, if you don't set `external_account_collection`, it defaults to `true` and `disable_stripe_user_authentication` defaults to `false`.",
		},
		"components.payouts.features.standard_payouts": {
			Type:        "boolean",
			Description: "Whether to allow creation of standard payouts. Defaults to `true` when `controller.losses.payments` is set to `stripe` for the account, otherwise `false`.",
		},
	},
}

var V1CustomersList = resource.OperationSpec{
	Name:    "list",
	Path:    "/v1/customers",
	Method:  "GET",
	Summary: "List all customers",
	Params: map[string]*resource.ParamSpec{
		"test_clock": {
			Type:        "string",
			Description: "Provides a list of customers that are associated with the specified test clock. The response will not include customers with test clocks if this parameter is not set.",
		},
		"created": {
			Type:        "integer",
			Description: "Only return customers that were created during the given date interval.",
		},
		"email": {
			Type:        "string",
			Description: "A case-sensitive filter on the list based on the customer's `email` field. The value must be a string.",
		},
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

var V1CustomersRetrieve = resource.OperationSpec{
	Name:    "retrieve",
	Path:    "/v1/customers/{customer}",
	Method:  "GET",
	Summary: "Retrieve a customer",
}

var V1CustomersBalanceTransactions = resource.OperationSpec{
	Name:    "balance_transactions",
	Path:    "/v1/customers/{customer}/balance_transactions",
	Method:  "GET",
	Summary: "List customer balance transactions",
	Params: map[string]*resource.ParamSpec{
		"created": {
			Type:        "integer",
			Description: "Only return customer balance transactions that were created during the given date interval.",
		},
		"ending_before": {
			Type:        "string",
			Description: "A cursor for use in pagination. `ending_before` is an object ID that defines your place in the list. For instance, if you make a list request and receive 100 objects, starting with `obj_bar`, your subsequent call can include `ending_before=obj_bar` in order to fetch the previous page of the list.",
		},
		"invoice": {
			Type:        "string",
			Description: "Only return transactions that are related to the specified invoice.",
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

var V1CustomersRetrievePaymentMethod = resource.OperationSpec{
	Name:    "retrieve_payment_method",
	Path:    "/v1/customers/{customer}/payment_methods/{payment_method}",
	Method:  "GET",
	Summary: "Retrieve a Customer's PaymentMethod",
}

var V1CustomersSearch = resource.OperationSpec{
	Name:    "search",
	Path:    "/v1/customers/search",
	Method:  "GET",
	Summary: "Search customers",
	Params: map[string]*resource.ParamSpec{
		"limit": {
			Type:        "integer",
			Description: "A limit on the number of objects to be returned. Limit can range between 1 and 100, and the default is 10.",
		},
		"page": {
			Type:        "string",
			Description: "A cursor for pagination across multiple pages of results. Don't include this parameter on the first call. Use the next_page value returned in a previous response to request subsequent results.",
		},
		"query": {
			Type:        "string",
			Description: "The search query string. See [search query language](https://docs.stripe.com/search#search-query-language) and the list of supported [query fields for customers](https://docs.stripe.com/search#query-fields-for-customers).",
			Required:    true,
		},
	},
}

var V1CustomersUpdate = resource.OperationSpec{
	Name:    "update",
	Path:    "/v1/customers/{customer}",
	Method:  "POST",
	Summary: "Update a customer",
	Params: map[string]*resource.ParamSpec{
		"tax.validate_location": {
			Type:        "string",
			Description: "A flag that indicates when Stripe should validate the customer tax location. Defaults to `auto`.",
			Enum: []resource.EnumSpec{
				{Value: "auto"},
				{Value: "deferred"},
				{Value: "immediately"},
			},
		},
		"tax_exempt": {
			Type:        "string",
			Description: "The customer's tax exemption. One of `none`, `exempt`, or `reverse`.",
			Enum: []resource.EnumSpec{
				{Value: "exempt"},
				{Value: "none"},
				{Value: "reverse"},
			},
		},
		"individual_name": {
			Type:        "string",
			Description: "The customer's full name. This may be up to *150 characters*.",
		},
		"invoice_prefix": {
			Type:        "string",
			Description: "The prefix for the customer used to generate unique invoice numbers. Must be 3–12 uppercase letters or numbers.",
		},
		"next_invoice_sequence": {
			Type:        "integer",
			Description: "The sequence to be used on the customer's next invoice. Defaults to 1.",
		},
		"email": {
			Type:        "string",
			Description: "Customer's email address. It's displayed alongside the customer in your dashboard and can be useful for searching and tracking. This may be up to *512 characters*.",
		},
		"tax.ip_address": {
			Type:        "string",
			Description: "A recent IP address of the customer used for tax reporting and tax location inference. Stripe recommends updating the IP address when a new PaymentMethod is attached or the address field on the customer is updated. We recommend against updating this field more frequently since it could result in unexpected tax location/reporting outcomes.",
		},
		"business_name": {
			Type:        "string",
			Description: "The customer's business name. This may be up to *150 characters*.",
		},
		"invoice_settings.default_payment_method": {
			Type:        "string",
			Description: "ID of a payment method that's attached to the customer, to be used as the customer's default payment method for subscriptions and invoices.",
		},
		"cash_balance.settings.reconciliation_mode": {
			Type:        "string",
			Description: "Controls how funds transferred by the customer are applied to payment intents and invoices. Valid options are `automatic`, `manual`, or `merchant_default`. For more information about these reconciliation modes, see [Reconciliation](https://docs.stripe.com/payments/customer-balance/reconciliation).",
			Enum: []resource.EnumSpec{
				{Value: "automatic"},
				{Value: "manual"},
				{Value: "merchant_default"},
			},
		},
		"source": {
			Type: "string",
		},
		"default_source": {
			Type:        "string",
			Description: "If you are using payment methods created via the PaymentMethods API, see the [invoice_settings.default_payment_method](https://docs.stripe.com/api/customers/update#update_customer-invoice_settings-default_payment_method) parameter.\n\nProvide the ID of a payment source already attached to this customer to make it this customer's default payment source.\n\nIf you want to add a new payment source and make it the default, see the [source](https://docs.stripe.com/api/customers/update#update_customer-source) property.",
		},
		"balance": {
			Type:        "integer",
			Description: "An integer amount in cents (or local equivalent) that represents the customer's current balance, which affect the customer's future invoices. A negative amount represents a credit that decreases the amount due on an invoice; a positive amount increases the amount due on an invoice.",
		},
		"name": {
			Type:        "string",
			Description: "The customer's full name or business name.",
		},
		"description": {
			Type:        "string",
			Description: "An arbitrary string that you can attach to a customer object. It is displayed alongside the customer in the dashboard.",
		},
		"invoice_settings.footer": {
			Type:        "string",
			Description: "Default footer to be displayed on invoices for this customer.",
		},
		"phone": {
			Type:        "string",
			Description: "The customer's phone number.",
		},
		"validate": {
			Type: "boolean",
		},
		"preferred_locales": {
			Type:        "array",
			Description: "Customer's preferred languages, ordered by preference.",
		},
	},
}

var V1CustomersCreateFundingInstructions = resource.OperationSpec{
	Name:    "create_funding_instructions",
	Path:    "/v1/customers/{customer}/funding_instructions",
	Method:  "POST",
	Summary: "Create or retrieve funding instructions for a customer cash balance",
	Params: map[string]*resource.ParamSpec{
		"bank_transfer.type": {
			Type:        "string",
			Description: "The type of the `bank_transfer`",
			Required:    true,
			Enum: []resource.EnumSpec{
				{Value: "eu_bank_transfer"},
				{Value: "gb_bank_transfer"},
				{Value: "jp_bank_transfer"},
				{Value: "mx_bank_transfer"},
				{Value: "us_bank_transfer"},
			},
		},
		"bank_transfer.eu_bank_transfer.country": {
			Type:        "string",
			Description: "The desired country code of the bank account information. Permitted values include: `DE`, `FR`, `IE`, or `NL`.",
			Required:    true,
		},
		"bank_transfer.requested_address_types": {
			Type:        "array",
			Description: "List of address types that should be returned in the financial_addresses response. If not specified, all valid types will be returned.\n\nPermitted values include: `sort_code`, `zengin`, `iban`, or `spei`.",
		},
		"currency": {
			Type:        "string",
			Description: "Three-letter [ISO currency code](https://www.iso.org/iso-4217-currency-codes.html), in lowercase. Must be a [supported currency](https://stripe.com/docs/currencies).",
			Required:    true,
			Format:      "currency",
		},
		"funding_type": {
			Type:        "string",
			Description: "The `funding_type` to get the instructions for.",
			Required:    true,
			Enum: []resource.EnumSpec{
				{Value: "bank_transfer"},
			},
		},
	},
}

var V1CustomersDelete = resource.OperationSpec{
	Name:    "delete",
	Path:    "/v1/customers/{customer}",
	Method:  "DELETE",
	Summary: "Delete a customer",
}

var V1CustomersDeleteDiscount = resource.OperationSpec{
	Name:    "delete_discount",
	Path:    "/v1/customers/{customer}/discount",
	Method:  "DELETE",
	Summary: "Delete a customer discount",
}

var V1CustomersListPaymentMethods = resource.OperationSpec{
	Name:    "list_payment_methods",
	Path:    "/v1/customers/{customer}/payment_methods",
	Method:  "GET",
	Summary: "List a Customer's PaymentMethods",
	Params: map[string]*resource.ParamSpec{
		"allow_redisplay": {
			Type:        "string",
			Description: "This field indicates whether this payment method can be shown again to its customer in a checkout flow. Stripe products such as Checkout and Elements use this field to determine whether a payment method can be shown as a saved payment method in a checkout flow.",
			Enum: []resource.EnumSpec{
				{Value: "always"},
				{Value: "limited"},
				{Value: "unspecified"},
			},
		},
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
		"type": {
			Type:        "string",
			Description: "An optional filter on the list, based on the object `type` field. Without the filter, the list includes all current and future payment method types. If your integration expects only one type of payment method in the response, make sure to provide a type value in the request.",
			Enum: []resource.EnumSpec{
				{Value: "acss_debit"},
				{Value: "affirm"},
				{Value: "afterpay_clearpay"},
				{Value: "alipay"},
				{Value: "alma"},
				{Value: "amazon_pay"},
				{Value: "au_becs_debit"},
				{Value: "bacs_debit"},
				{Value: "bancontact"},
				{Value: "billie"},
				{Value: "blik"},
				{Value: "boleto"},
				{Value: "card"},
				{Value: "cashapp"},
				{Value: "crypto"},
				{Value: "custom"},
				{Value: "customer_balance"},
				{Value: "eps"},
				{Value: "fpx"},
				{Value: "giropay"},
				{Value: "grabpay"},
				{Value: "ideal"},
				{Value: "kakao_pay"},
				{Value: "klarna"},
				{Value: "konbini"},
				{Value: "kr_card"},
				{Value: "link"},
				{Value: "mb_way"},
				{Value: "mobilepay"},
				{Value: "multibanco"},
				{Value: "naver_pay"},
				{Value: "nz_bank_account"},
				{Value: "oxxo"},
				{Value: "p24"},
				{Value: "pay_by_bank"},
				{Value: "payco"},
				{Value: "paynow"},
				{Value: "paypal"},
				{Value: "payto"},
				{Value: "pix"},
				{Value: "promptpay"},
				{Value: "revolut_pay"},
				{Value: "samsung_pay"},
				{Value: "satispay"},
				{Value: "sepa_debit"},
				{Value: "sofort"},
				{Value: "swish"},
				{Value: "twint"},
				{Value: "us_bank_account"},
				{Value: "wechat_pay"},
				{Value: "zip"},
			},
		},
	},
}

var V1CustomersCreate = resource.OperationSpec{
	Name:    "create",
	Path:    "/v1/customers",
	Method:  "POST",
	Summary: "Create a customer",
	Params: map[string]*resource.ParamSpec{
		"email": {
			Type:        "string",
			Description: "Customer's email address. It's displayed alongside the customer in your dashboard and can be useful for searching and tracking. This may be up to *512 characters*.",
		},
		"balance": {
			Type:        "integer",
			Description: "An integer amount in cents (or local equivalent) that represents the customer's current balance, which affect the customer's future invoices. A negative amount represents a credit that decreases the amount due on an invoice; a positive amount increases the amount due on an invoice.",
		},
		"tax.ip_address": {
			Type:        "string",
			Description: "A recent IP address of the customer used for tax reporting and tax location inference. Stripe recommends updating the IP address when a new PaymentMethod is attached or the address field on the customer is updated. We recommend against updating this field more frequently since it could result in unexpected tax location/reporting outcomes.",
		},
		"invoice_settings.default_payment_method": {
			Type:        "string",
			Description: "ID of a payment method that's attached to the customer, to be used as the customer's default payment method for subscriptions and invoices.",
		},
		"cash_balance.settings.reconciliation_mode": {
			Type:        "string",
			Description: "Controls how funds transferred by the customer are applied to payment intents and invoices. Valid options are `automatic`, `manual`, or `merchant_default`. For more information about these reconciliation modes, see [Reconciliation](https://docs.stripe.com/payments/customer-balance/reconciliation).",
			Enum: []resource.EnumSpec{
				{Value: "automatic"},
				{Value: "manual"},
				{Value: "merchant_default"},
			},
		},
		"invoice_prefix": {
			Type:        "string",
			Description: "The prefix for the customer used to generate unique invoice numbers. Must be 3–12 uppercase letters or numbers.",
		},
		"tax.validate_location": {
			Type:        "string",
			Description: "A flag that indicates when Stripe should validate the customer tax location. Defaults to `deferred`.",
			Enum: []resource.EnumSpec{
				{Value: "deferred"},
				{Value: "immediately"},
			},
		},
		"business_name": {
			Type:        "string",
			Description: "The customer's business name. This may be up to *150 characters*.",
		},
		"invoice_settings.footer": {
			Type:        "string",
			Description: "Default footer to be displayed on invoices for this customer.",
		},
		"tax_exempt": {
			Type:        "string",
			Description: "The customer's tax exemption. One of `none`, `exempt`, or `reverse`.",
			Enum: []resource.EnumSpec{
				{Value: "exempt"},
				{Value: "none"},
				{Value: "reverse"},
			},
		},
		"test_clock": {
			Type:        "string",
			Description: "ID of the test clock to attach to the customer.",
		},
		"validate": {
			Type: "boolean",
		},
		"name": {
			Type:        "string",
			Description: "The customer's full name or business name.",
		},
		"phone": {
			Type:        "string",
			Description: "The customer's phone number.",
		},
		"individual_name": {
			Type:        "string",
			Description: "The customer's full name. This may be up to *150 characters*.",
		},
		"next_invoice_sequence": {
			Type:        "integer",
			Description: "The sequence to be used on the customer's next invoice. Defaults to 1.",
		},
		"preferred_locales": {
			Type:        "array",
			Description: "Customer's preferred languages, ordered by preference.",
		},
		"source": {
			Type: "string",
		},
		"payment_method": {
			Type: "string",
		},
		"description": {
			Type:        "string",
			Description: "An arbitrary string that you can attach to a customer object. It is displayed alongside the customer in the dashboard.",
		},
	},
}

var V1CustomersTestHelpersFundCashBalance = resource.OperationSpec{
	Name:    "fund_cash_balance",
	Path:    "/v1/test_helpers/customers/{customer}/fund_cash_balance",
	Method:  "POST",
	Summary: "Fund a test mode cash balance",
	Params: map[string]*resource.ParamSpec{
		"reference": {
			Type:        "string",
			Description: "A description of the test funding. This simulates free-text references supplied by customers when making bank transfers to their cash balance. You can use this to test how Stripe's [reconciliation algorithm](https://docs.stripe.com/payments/customer-balance/reconciliation) applies to different user inputs.",
		},
		"amount": {
			Type:        "integer",
			Description: "Amount to be used for this test cash balance transaction. A positive integer representing how much to fund in the [smallest currency unit](https://docs.stripe.com/currencies#zero-decimal) (e.g., 100 cents to fund $1.00 or 100 to fund ¥100, a zero-decimal currency).",
			Required:    true,
		},
		"currency": {
			Type:        "string",
			Description: "Three-letter [ISO currency code](https://www.iso.org/iso-4217-currency-codes.html), in lowercase. Must be a [supported currency](https://stripe.com/docs/currencies).",
			Required:    true,
			Format:      "currency",
		},
	},
}

var V1InvoiceRenderingTemplatesList = resource.OperationSpec{
	Name:    "list",
	Path:    "/v1/invoice_rendering_templates",
	Method:  "GET",
	Summary: "List all invoice rendering templates",
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
		"status": {
			Type: "string",
			Enum: []resource.EnumSpec{
				{Value: "active"},
				{Value: "archived"},
			},
		},
	},
}

var V1InvoiceRenderingTemplatesRetrieve = resource.OperationSpec{
	Name:    "retrieve",
	Path:    "/v1/invoice_rendering_templates/{template}",
	Method:  "GET",
	Summary: "Retrieve an invoice rendering template",
	Params: map[string]*resource.ParamSpec{
		"version": {
			Type: "integer",
		},
	},
}

var V1InvoiceRenderingTemplatesArchive = resource.OperationSpec{
	Name:    "archive",
	Path:    "/v1/invoice_rendering_templates/{template}/archive",
	Method:  "POST",
	Summary: "Archive an invoice rendering template",
}

var V1InvoiceRenderingTemplatesUnarchive = resource.OperationSpec{
	Name:    "unarchive",
	Path:    "/v1/invoice_rendering_templates/{template}/unarchive",
	Method:  "POST",
	Summary: "Unarchive an invoice rendering template",
}

var V1ExternalAccountsDelete = resource.OperationSpec{
	Name:    "delete",
	Path:    "/v1/accounts/{account}/external_accounts/{id}",
	Method:  "DELETE",
	Summary: "Delete an external account",
}

var V1ExternalAccountsList = resource.OperationSpec{
	Name:    "list",
	Path:    "/v1/accounts/{account}/external_accounts",
	Method:  "GET",
	Summary: "List all external accounts",
	Params: map[string]*resource.ParamSpec{
		"ending_before": {
			Type:        "string",
			Description: "A cursor for use in pagination. `ending_before` is an object ID that defines your place in the list. For instance, if you make a list request and receive 100 objects, starting with `obj_bar`, your subsequent call can include `ending_before=obj_bar` in order to fetch the previous page of the list.",
		},
		"limit": {
			Type:        "integer",
			Description: "A limit on the number of objects to be returned. Limit can range between 1 and 100, and the default is 10.",
		},
		"object": {
			Type:        "string",
			Description: "Filter external accounts according to a particular object type.",
			Enum: []resource.EnumSpec{
				{Value: "bank_account"},
				{Value: "card"},
			},
		},
		"starting_after": {
			Type:        "string",
			Description: "A cursor for use in pagination. `starting_after` is an object ID that defines your place in the list. For instance, if you make a list request and receive 100 objects, ending with `obj_foo`, your subsequent call can include `starting_after=obj_foo` in order to fetch the next page of the list.",
		},
	},
}

var V1ExternalAccountsRetrieve = resource.OperationSpec{
	Name:    "retrieve",
	Path:    "/v1/accounts/{account}/external_accounts/{id}",
	Method:  "GET",
	Summary: "Retrieve an external account",
}

var V1ExternalAccountsCreate = resource.OperationSpec{
	Name:    "create",
	Path:    "/v1/accounts/{account}/external_accounts",
	Method:  "POST",
	Summary: "Create an external account",
	Params: map[string]*resource.ParamSpec{
		"external_account": {
			Type:        "string",
			Description: "A token, like the ones returned by [Stripe.js](https://docs.stripe.com/js) or a dictionary containing a user's external account details (with the options shown below). Please refer to full [documentation](https://stripe.com/docs/api/external_accounts) instead.",
			Required:    true,
		},
		"default_for_currency": {
			Type:        "boolean",
			Description: "When set to true, or if this is the first external account added in this currency, this account becomes the default external account for its currency.",
		},
	},
}

var V1ExternalAccountsUpdate = resource.OperationSpec{
	Name:   "update",
	Path:   "/v1/accounts/{account}/external_accounts/{id}",
	Method: "POST",
	Params: map[string]*resource.ParamSpec{
		"address_line2": {
			Type:        "string",
			Description: "Address line 2 (Apartment/Suite/Unit/Building).",
		},
		"address_line1": {
			Type:        "string",
			Description: "Address line 1 (Street address/PO Box/Company name).",
		},
		"address_zip": {
			Type:        "string",
			Description: "ZIP or postal code.",
		},
		"default_for_currency": {
			Type:        "boolean",
			Description: "When set to true, this becomes the default external account for its currency.",
		},
		"account_holder_type": {
			Type:        "string",
			Description: "The type of entity that holds the account. This can be either `individual` or `company`.",
			Enum: []resource.EnumSpec{
				{Value: "company"},
				{Value: "individual"},
			},
		},
		"address_city": {
			Type:        "string",
			Description: "City/District/Suburb/Town/Village.",
		},
		"documents.bank_account_ownership_verification.files": {
			Type:        "array",
			Description: "One or more document ids returned by a [file upload](https://api.stripe.com#create_file) with a `purpose` value of `account_requirement`.",
		},
		"account_holder_name": {
			Type:        "string",
			Description: "The name of the person or business that owns the bank account.",
		},
		"address_country": {
			Type:        "string",
			Description: "Billing address country, if provided when creating card.",
		},
		"exp_month": {
			Type:        "string",
			Description: "Two digit number representing the card’s expiration month.",
		},
		"account_type": {
			Type:        "string",
			Description: "The bank account type. This can only be `checking` or `savings` in most countries. In Japan, this can only be `futsu` or `toza`.",
			Enum: []resource.EnumSpec{
				{Value: "checking"},
				{Value: "futsu"},
				{Value: "savings"},
				{Value: "toza"},
			},
		},
		"address_state": {
			Type:        "string",
			Description: "State/County/Province/Region.",
		},
		"name": {
			Type:        "string",
			Description: "Cardholder name.",
		},
		"exp_year": {
			Type:        "string",
			Description: "Four digit number representing the card’s expiration year.",
		},
	},
}

var V1TransfersRetrieve = resource.OperationSpec{
	Name:    "retrieve",
	Path:    "/v1/transfers/{transfer}",
	Method:  "GET",
	Summary: "Retrieve a transfer",
}

var V1TransfersCreate = resource.OperationSpec{
	Name:    "create",
	Path:    "/v1/transfers",
	Method:  "POST",
	Summary: "Create a transfer",
	Params: map[string]*resource.ParamSpec{
		"amount": {
			Type:        "integer",
			Description: "A positive integer in cents (or local equivalent) representing how much to transfer.",
		},
		"description": {
			Type:        "string",
			Description: "An arbitrary string attached to the object. Often useful for displaying to users.",
		},
		"source_transaction": {
			Type:        "string",
			Description: "You can use this parameter to transfer funds from a charge before they are added to your available balance. A pending balance will transfer immediately but the funds will not become available until the original charge becomes available. [See the Connect documentation](https://docs.stripe.com/connect/separate-charges-and-transfers#transfer-availability) for details.",
		},
		"transfer_group": {
			Type:        "string",
			Description: "A string that identifies this transaction as part of a group. See the [Connect documentation](https://docs.stripe.com/connect/separate-charges-and-transfers#transfer-options) for details.",
		},
		"currency": {
			Type:        "string",
			Description: "Three-letter [ISO code for currency](https://www.iso.org/iso-4217-currency-codes.html) in lowercase. Must be a [supported currency](https://docs.stripe.com/currencies).",
			Required:    true,
			Format:      "currency",
		},
		"destination": {
			Type:        "string",
			Description: "The ID of a connected Stripe account. <a href=\"/docs/connect/separate-charges-and-transfers\">See the Connect documentation</a> for details.",
			Required:    true,
		},
		"source_type": {
			Type:        "string",
			Description: "The source balance to use for this transfer. One of `bank_account`, `card`, or `fpx`. For most users, this will default to `card`.",
			Enum: []resource.EnumSpec{
				{Value: "bank_account"},
				{Value: "card"},
				{Value: "fpx"},
			},
		},
	},
}

var V1TransfersUpdate = resource.OperationSpec{
	Name:    "update",
	Path:    "/v1/transfers/{transfer}",
	Method:  "POST",
	Summary: "Update a transfer",
	Params: map[string]*resource.ParamSpec{
		"description": {
			Type:        "string",
			Description: "An arbitrary string attached to the object. Often useful for displaying to users.",
		},
	},
}

var V1TransfersList = resource.OperationSpec{
	Name:    "list",
	Path:    "/v1/transfers",
	Method:  "GET",
	Summary: "List all transfers",
	Params: map[string]*resource.ParamSpec{
		"starting_after": {
			Type:        "string",
			Description: "A cursor for use in pagination. `starting_after` is an object ID that defines your place in the list. For instance, if you make a list request and receive 100 objects, ending with `obj_foo`, your subsequent call can include `starting_after=obj_foo` in order to fetch the next page of the list.",
		},
		"transfer_group": {
			Type:        "string",
			Description: "Only return transfers with the specified transfer group.",
		},
		"created": {
			Type:        "integer",
			Description: "Only return transfers that were created during the given date interval.",
		},
		"destination": {
			Type:        "string",
			Description: "Only return transfers for the destination specified by this account ID.",
		},
		"ending_before": {
			Type:        "string",
			Description: "A cursor for use in pagination. `ending_before` is an object ID that defines your place in the list. For instance, if you make a list request and receive 100 objects, starting with `obj_bar`, your subsequent call can include `ending_before=obj_bar` in order to fetch the previous page of the list.",
		},
		"limit": {
			Type:        "integer",
			Description: "A limit on the number of objects to be returned. Limit can range between 1 and 100, and the default is 10.",
		},
	},
}

var V1CountrySpecsList = resource.OperationSpec{
	Name:    "list",
	Path:    "/v1/country_specs",
	Method:  "GET",
	Summary: "List Country Specs",
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

var V1CountrySpecsRetrieve = resource.OperationSpec{
	Name:    "retrieve",
	Path:    "/v1/country_specs/{country}",
	Method:  "GET",
	Summary: "Retrieve a Country Spec",
}

var V1ApplicationFeesList = resource.OperationSpec{
	Name:    "list",
	Path:    "/v1/application_fees",
	Method:  "GET",
	Summary: "List all application fees",
	Params: map[string]*resource.ParamSpec{
		"charge": {
			Type:        "string",
			Description: "Only return application fees for the charge specified by this charge ID.",
		},
		"created": {
			Type:        "integer",
			Description: "Only return applications fees that were created during the given date interval.",
		},
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

var V1ApplicationFeesRetrieve = resource.OperationSpec{
	Name:    "retrieve",
	Path:    "/v1/application_fees/{id}",
	Method:  "GET",
	Summary: "Retrieve an application fee",
}

var V1FileLinksUpdate = resource.OperationSpec{
	Name:    "update",
	Path:    "/v1/file_links/{link}",
	Method:  "POST",
	Summary: "Update a file link",
	Params: map[string]*resource.ParamSpec{
		"expires_at": {
			Type:        "string",
			Description: "A future timestamp after which the link will no longer be usable, or `now` to expire the link immediately.",
		},
	},
}

var V1FileLinksList = resource.OperationSpec{
	Name:    "list",
	Path:    "/v1/file_links",
	Method:  "GET",
	Summary: "List all file links",
	Params: map[string]*resource.ParamSpec{
		"limit": {
			Type:        "integer",
			Description: "A limit on the number of objects to be returned. Limit can range between 1 and 100, and the default is 10.",
		},
		"starting_after": {
			Type:        "string",
			Description: "A cursor for use in pagination. `starting_after` is an object ID that defines your place in the list. For instance, if you make a list request and receive 100 objects, ending with `obj_foo`, your subsequent call can include `starting_after=obj_foo` in order to fetch the next page of the list.",
		},
		"created": {
			Type:        "integer",
			Description: "Only return links that were created during the given date interval.",
		},
		"ending_before": {
			Type:        "string",
			Description: "A cursor for use in pagination. `ending_before` is an object ID that defines your place in the list. For instance, if you make a list request and receive 100 objects, starting with `obj_bar`, your subsequent call can include `ending_before=obj_bar` in order to fetch the previous page of the list.",
		},
		"expired": {
			Type:        "boolean",
			Description: "Filter links by their expiration status. By default, Stripe returns all links.",
		},
		"file": {
			Type:        "string",
			Description: "Only return links for the given file.",
		},
	},
}

var V1FileLinksRetrieve = resource.OperationSpec{
	Name:    "retrieve",
	Path:    "/v1/file_links/{link}",
	Method:  "GET",
	Summary: "Retrieve a file link",
}

var V1FileLinksCreate = resource.OperationSpec{
	Name:    "create",
	Path:    "/v1/file_links",
	Method:  "POST",
	Summary: "Create a file link",
	Params: map[string]*resource.ParamSpec{
		"expires_at": {
			Type:        "integer",
			Description: "The link isn't usable after this future timestamp.",
			Format:      "unix-time",
		},
		"file": {
			Type:        "string",
			Description: "The ID of the file. The file's `purpose` must be one of the following: `business_icon`, `business_logo`, `customer_signature`, `dispute_evidence`, `finance_report_run`, `financial_account_statement`, `identity_document_downloadable`, `issuing_regulatory_reporting`, `pci_document`, `selfie`, `sigma_scheduled_query`, `tax_document_user_upload`, `terminal_android_apk`, or `terminal_reader_splashscreen`.",
			Required:    true,
		},
	},
}

var V1LineItemsList = resource.OperationSpec{
	Name:    "list",
	Path:    "/v1/invoices/{invoice}/lines",
	Method:  "GET",
	Summary: "Retrieve an invoice's line items",
	Params: map[string]*resource.ParamSpec{
		"limit": {
			Type:        "integer",
			Description: "A limit on the number of objects to be returned. Limit can range between 1 and 100, and the default is 10.",
		},
		"starting_after": {
			Type:        "string",
			Description: "A cursor for use in pagination. `starting_after` is an object ID that defines your place in the list. For instance, if you make a list request and receive 100 objects, ending with `obj_foo`, your subsequent call can include `starting_after=obj_foo` in order to fetch the next page of the list.",
		},
		"ending_before": {
			Type:        "string",
			Description: "A cursor for use in pagination. `ending_before` is an object ID that defines your place in the list. For instance, if you make a list request and receive 100 objects, starting with `obj_bar`, your subsequent call can include `ending_before=obj_bar` in order to fetch the previous page of the list.",
		},
	},
}

var V1LineItemsUpdate = resource.OperationSpec{
	Name:    "update",
	Path:    "/v1/invoices/{invoice}/lines/{line_item_id}",
	Method:  "POST",
	Summary: "Update an invoice's line item",
	Params: map[string]*resource.ParamSpec{
		"period.start": {
			Type:        "integer",
			Description: "The start of the period. This value is inclusive.",
			Required:    true,
			Format:      "unix-time",
		},
		"price_data.product_data.name": {
			Type:        "string",
			Description: "The product's name, meant to be displayable to the customer.",
			Required:    true,
		},
		"discountable": {
			Type:        "boolean",
			Description: "Controls whether discounts apply to this line item. Defaults to false for prorations or negative line items, and true for all other line items. Cannot be set to true for prorations.",
		},
		"price_data.unit_amount": {
			Type:        "integer",
			Description: "A non-negative integer in cents (or local equivalent) representing how much to charge. One of `unit_amount` or `unit_amount_decimal` is required.",
		},
		"price_data.product": {
			Type:        "string",
			Description: "The ID of the [Product](https://docs.stripe.com/api/products) that this [Price](https://docs.stripe.com/api/prices) will belong to. One of `product` or `product_data` is required.",
		},
		"price_data.product_data.description": {
			Type:        "string",
			Description: "The product's description, meant to be displayable to the customer. Use this field to optionally store a long form explanation of the product being sold for your own rendering purposes.",
		},
		"price_data.product_data.images": {
			Type:        "array",
			Description: "A list of up to 8 URLs of images for this product, meant to be displayable to the customer.",
		},
		"amount": {
			Type:        "integer",
			Description: "The integer amount in cents (or local equivalent) of the charge to be applied to the upcoming invoice. If you want to apply a credit to the customer's account, pass a negative amount.",
		},
		"pricing.price": {
			Type:        "string",
			Description: "The ID of the price object.",
		},
		"quantity": {
			Type:        "integer",
			Description: "Non-negative integer. The quantity of units for the line item.",
		},
		"tax_rates": {
			Type:        "array",
			Description: "The tax rates which apply to the line item. When set, the `default_tax_rates` on the invoice do not apply to this line item. Pass an empty string to remove previously-defined tax rates.",
		},
		"price_data.product_data.tax_code": {
			Type:        "string",
			Description: "A [tax code](https://docs.stripe.com/tax/tax-categories) ID.",
		},
		"price_data.product_data.unit_label": {
			Type:        "string",
			Description: "A label that represents units of this product. When set, this will be included in customers' receipts, invoices, Checkout, and the customer portal.",
		},
		"period.end": {
			Type:        "integer",
			Description: "The end of the period, which must be greater than or equal to the start. This value is inclusive.",
			Required:    true,
			Format:      "unix-time",
		},
		"description": {
			Type:        "string",
			Description: "An arbitrary string which you can attach to the invoice item. The description is displayed in the invoice for easy tracking.",
		},
		"price_data.tax_behavior": {
			Type:        "string",
			Description: "Only required if a [default tax behavior](https://docs.stripe.com/tax/products-prices-tax-categories-tax-behavior#setting-a-default-tax-behavior-(recommended)) was not provided in the Stripe Tax settings. Specifies whether the price is considered inclusive of taxes or exclusive of taxes. One of `inclusive`, `exclusive`, or `unspecified`. Once specified as either `inclusive` or `exclusive`, it cannot be changed.",
			Enum: []resource.EnumSpec{
				{Value: "exclusive"},
				{Value: "inclusive"},
				{Value: "unspecified"},
			},
		},
		"price_data.unit_amount_decimal": {
			Type:        "string",
			Description: "Same as `unit_amount`, but accepts a decimal value in cents (or local equivalent) with at most 12 decimal places. Only one of `unit_amount` and `unit_amount_decimal` can be set.",
			Format:      "decimal",
		},
		"price_data.currency": {
			Type:        "string",
			Description: "Three-letter [ISO currency code](https://www.iso.org/iso-4217-currency-codes.html), in lowercase. Must be a [supported currency](https://stripe.com/docs/currencies).",
			Required:    true,
			Format:      "currency",
		},
	},
}

var V1BalanceRetrieve = resource.OperationSpec{
	Name:    "retrieve",
	Path:    "/v1/balance",
	Method:  "GET",
	Summary: "Retrieve balance",
}

var V1SetupAttemptsList = resource.OperationSpec{
	Name:    "list",
	Path:    "/v1/setup_attempts",
	Method:  "GET",
	Summary: "List all SetupAttempts",
	Params: map[string]*resource.ParamSpec{
		"starting_after": {
			Type:        "string",
			Description: "A cursor for use in pagination. `starting_after` is an object ID that defines your place in the list. For instance, if you make a list request and receive 100 objects, ending with `obj_foo`, your subsequent call can include `starting_after=obj_foo` in order to fetch the next page of the list.",
		},
		"created": {
			Type:        "integer",
			Description: "A filter on the list, based on the object `created` field. The value\ncan be a string with an integer Unix timestamp or a\ndictionary with a number of different query options.",
		},
		"ending_before": {
			Type:        "string",
			Description: "A cursor for use in pagination. `ending_before` is an object ID that defines your place in the list. For instance, if you make a list request and receive 100 objects, starting with `obj_bar`, your subsequent call can include `ending_before=obj_bar` in order to fetch the previous page of the list.",
		},
		"limit": {
			Type:        "integer",
			Description: "A limit on the number of objects to be returned. Limit can range between 1 and 100, and the default is 10.",
		},
		"setup_intent": {
			Type:        "string",
			Description: "Only return SetupAttempts created by the SetupIntent specified by\nthis ID.",
			Required:    true,
		},
	},
}

var V1FilesList = resource.OperationSpec{
	Name:    "list",
	Path:    "/v1/files",
	Method:  "GET",
	Summary: "List all files",
	Params: map[string]*resource.ParamSpec{
		"created": {
			Type:        "integer",
			Description: "Only return files that were created during the given date interval.",
		},
		"ending_before": {
			Type:        "string",
			Description: "A cursor for use in pagination. `ending_before` is an object ID that defines your place in the list. For instance, if you make a list request and receive 100 objects, starting with `obj_bar`, your subsequent call can include `ending_before=obj_bar` in order to fetch the previous page of the list.",
		},
		"limit": {
			Type:        "integer",
			Description: "A limit on the number of objects to be returned. Limit can range between 1 and 100, and the default is 10.",
		},
		"purpose": {
			Type:        "string",
			Description: "Filter queries by the file purpose. If you don't provide a purpose, the queries return unfiltered files.",
			Enum: []resource.EnumSpec{
				{Value: "account_requirement"},
				{Value: "additional_verification"},
				{Value: "business_icon"},
				{Value: "business_logo"},
				{Value: "customer_signature"},
				{Value: "dispute_evidence"},
				{Value: "document_provider_identity_document"},
				{Value: "finance_report_run"},
				{Value: "financial_account_statement"},
				{Value: "identity_document"},
				{Value: "identity_document_downloadable"},
				{Value: "issuing_regulatory_reporting"},
				{Value: "pci_document"},
				{Value: "platform_terms_of_service"},
				{Value: "selfie"},
				{Value: "sigma_scheduled_query"},
				{Value: "tax_document_user_upload"},
				{Value: "terminal_android_apk"},
				{Value: "terminal_reader_splashscreen"},
				{Value: "terminal_wifi_certificate"},
				{Value: "terminal_wifi_private_key"},
			},
		},
		"starting_after": {
			Type:        "string",
			Description: "A cursor for use in pagination. `starting_after` is an object ID that defines your place in the list. For instance, if you make a list request and receive 100 objects, ending with `obj_foo`, your subsequent call can include `starting_after=obj_foo` in order to fetch the next page of the list.",
		},
	},
}

var V1FilesRetrieve = resource.OperationSpec{
	Name:    "retrieve",
	Path:    "/v1/files/{file}",
	Method:  "GET",
	Summary: "Retrieve a file",
}

var V1FilesCreate = resource.OperationSpec{
	Name:      "create",
	Path:      "/v1/files",
	Method:    "POST",
	ServerURL: "https://files.stripe.com/",
	Summary:   "Create a file",
}

var V1PersonsDelete = resource.OperationSpec{
	Name:    "delete",
	Path:    "/v1/accounts/{account}/persons/{person}",
	Method:  "DELETE",
	Summary: "Delete a person",
}

var V1PersonsList = resource.OperationSpec{
	Name:    "list",
	Path:    "/v1/accounts/{account}/persons",
	Method:  "GET",
	Summary: "List all persons",
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

var V1PersonsRetrieve = resource.OperationSpec{
	Name:    "retrieve",
	Path:    "/v1/accounts/{account}/persons/{person}",
	Method:  "GET",
	Summary: "Retrieve a person",
}

var V1PersonsCreate = resource.OperationSpec{
	Name:    "create",
	Path:    "/v1/accounts/{account}/persons",
	Method:  "POST",
	Summary: "Create a person",
	Params: map[string]*resource.ParamSpec{
		"gender": {
			Type:        "string",
			Description: "The person's gender (International regulations require either \"male\" or \"female\").",
		},
		"address_kana.state": {
			Type:        "string",
			Description: "Prefecture.",
		},
		"nationality": {
			Type:        "string",
			Description: "The country where the person is a national. Two-letter country code ([ISO 3166-1 alpha-2](https://en.wikipedia.org/wiki/ISO_3166-1_alpha-2)), or \"XX\" if unavailable.",
		},
		"documents.company_authorization.files": {
			Type:        "array",
			Description: "One or more document ids returned by a [file upload](https://api.stripe.com#create_file) with a `purpose` value of `account_requirement`.",
		},
		"registered_address.country": {
			Type:        "string",
			Description: "Two-letter country code ([ISO 3166-1 alpha-2](https://en.wikipedia.org/wiki/ISO_3166-1_alpha-2)).",
		},
		"last_name_kana": {
			Type:        "string",
			Description: "The Kana variation of the person's last name (Japan only).",
		},
		"address.city": {
			Type:        "string",
			Description: "City, district, suburb, town, or village.",
		},
		"full_name_aliases": {
			Type:        "array",
			Description: "A list of alternate names or aliases that the person is known by.",
		},
		"registered_address.state": {
			Type:        "string",
			Description: "State, county, province, or region ([ISO 3166-2](https://en.wikipedia.org/wiki/ISO_3166-2)).",
		},
		"address_kanji.town": {
			Type:        "string",
			Description: "Town or cho-me.",
		},
		"id_number": {
			Type:        "string",
			Description: "The person's ID number, as appropriate for their country. For example, a social security number in the U.S., social insurance number in Canada, etc. Instead of the number itself, you can also provide a [PII token provided by Stripe.js](https://docs.stripe.com/js/tokens/create_token?type=pii).",
		},
		"email": {
			Type:        "string",
			Description: "The person's email address.",
		},
		"additional_tos_acceptances.account.date": {
			Type:        "integer",
			Description: "The Unix timestamp marking when the account representative accepted the service agreement.",
			Format:      "unix-time",
		},
		"last_name_kanji": {
			Type:        "string",
			Description: "The Kanji variation of the person's last name (Japan only).",
		},
		"relationship.owner": {
			Type:        "boolean",
			Description: "Whether the person is an owner of the account’s legal entity.",
		},
		"relationship.legal_guardian": {
			Type:        "boolean",
			Description: "Whether the person is the legal guardian of the account's representative.",
		},
		"us_cfpb_data.ethnicity_details.ethnicity": {
			Type:        "array",
			Description: "The persons ethnicity",
		},
		"us_cfpb_data.race_details.race": {
			Type:        "array",
			Description: "The persons race.",
		},
		"us_cfpb_data.self_identified_gender": {
			Type:        "string",
			Description: "The persons self-identified gender",
		},
		"address_kana.postal_code": {
			Type:        "string",
			Description: "Postal code.",
		},
		"ssn_last_4": {
			Type:        "string",
			Description: "The last four digits of the person's Social Security number (U.S. only).",
		},
		"documents.visa.files": {
			Type:        "array",
			Description: "One or more document ids returned by a [file upload](https://api.stripe.com#create_file) with a `purpose` value of `account_requirement`.",
		},
		"address.line1": {
			Type:        "string",
			Description: "Address line 1, such as the street, PO Box, or company name.",
		},
		"verification.additional_document.back": {
			Type:        "string",
			Description: "The back of an ID returned by a [file upload](https://api.stripe.com#create_file) with a `purpose` value of `identity_document`. The uploaded file needs to be a color image (smaller than 8,000px by 8,000px), in JPG, PNG, or PDF format, and less than 10 MB in size.",
		},
		"address_kanji.line2": {
			Type:        "string",
			Description: "Building details.",
		},
		"address.postal_code": {
			Type:        "string",
			Description: "ZIP or postal code.",
		},
		"registered_address.line1": {
			Type:        "string",
			Description: "Address line 1, such as the street, PO Box, or company name.",
		},
		"address_kana.country": {
			Type:        "string",
			Description: "Two-letter country code ([ISO 3166-1 alpha-2](https://en.wikipedia.org/wiki/ISO_3166-1_alpha-2)).",
		},
		"address_kana.line1": {
			Type:        "string",
			Description: "Block or building number.",
		},
		"first_name_kana": {
			Type:        "string",
			Description: "The Kana variation of the person's first name (Japan only).",
		},
		"first_name": {
			Type:        "string",
			Description: "The person's first name.",
		},
		"verification.document.front": {
			Type:        "string",
			Description: "The front of an ID returned by a [file upload](https://api.stripe.com#create_file) with a `purpose` value of `identity_document`. The uploaded file needs to be a color image (smaller than 8,000px by 8,000px), in JPG, PNG, or PDF format, and less than 10 MB in size.",
		},
		"maiden_name": {
			Type:        "string",
			Description: "The person's maiden name.",
		},
		"relationship.percent_ownership": {
			Type:        "number",
			Description: "The percent owned by the person of the account's legal entity.",
		},
		"address_kana.line2": {
			Type:        "string",
			Description: "Building details.",
		},
		"political_exposure": {
			Type:        "string",
			Description: "Indicates if the person or any of their representatives, family members, or other closely related persons, declares that they hold or have held an important public job or function, in any jurisdiction.",
			Enum: []resource.EnumSpec{
				{Value: "existing"},
				{Value: "none"},
			},
		},
		"additional_tos_acceptances.account.ip": {
			Type:        "string",
			Description: "The IP address from which the account representative accepted the service agreement.",
		},
		"id_number_secondary": {
			Type:        "string",
			Description: "The person's secondary ID number, as appropriate for their country, will be used for enhanced verification checks. In Thailand, this would be the laser code found on the back of an ID card. Instead of the number itself, you can also provide a [PII token provided by Stripe.js](https://docs.stripe.com/js/tokens/create_token?type=pii).",
		},
		"registered_address.postal_code": {
			Type:        "string",
			Description: "ZIP or postal code.",
		},
		"address_kanji.postal_code": {
			Type:        "string",
			Description: "Postal code.",
		},
		"address_kanji.line1": {
			Type:        "string",
			Description: "Block or building number.",
		},
		"phone": {
			Type:        "string",
			Description: "The person's phone number.",
		},
		"relationship.executive": {
			Type:        "boolean",
			Description: "Whether the person has significant responsibility to control, manage, or direct the organization.",
		},
		"us_cfpb_data.ethnicity_details.ethnicity_other": {
			Type:        "string",
			Description: "Please specify your origin, when other is selected.",
		},
		"first_name_kanji": {
			Type:        "string",
			Description: "The Kanji variation of the person's first name (Japan only).",
		},
		"verification.document.back": {
			Type:        "string",
			Description: "The back of an ID returned by a [file upload](https://api.stripe.com#create_file) with a `purpose` value of `identity_document`. The uploaded file needs to be a color image (smaller than 8,000px by 8,000px), in JPG, PNG, or PDF format, and less than 10 MB in size.",
		},
		"address.line2": {
			Type:        "string",
			Description: "Address line 2, such as the apartment, suite, unit, or building.",
		},
		"registered_address.line2": {
			Type:        "string",
			Description: "Address line 2, such as the apartment, suite, unit, or building.",
		},
		"verification.additional_document.front": {
			Type:        "string",
			Description: "The front of an ID returned by a [file upload](https://api.stripe.com#create_file) with a `purpose` value of `identity_document`. The uploaded file needs to be a color image (smaller than 8,000px by 8,000px), in JPG, PNG, or PDF format, and less than 10 MB in size.",
		},
		"address_kanji.state": {
			Type:        "string",
			Description: "Prefecture.",
		},
		"documents.passport.files": {
			Type:        "array",
			Description: "One or more document ids returned by a [file upload](https://api.stripe.com#create_file) with a `purpose` value of `account_requirement`.",
		},
		"registered_address.city": {
			Type:        "string",
			Description: "City, district, suburb, town, or village.",
		},
		"relationship.representative": {
			Type:        "boolean",
			Description: "Whether the person is authorized as the primary representative of the account. This is the person nominated by the business to provide information about themselves, and general information about the account. There can only be one representative at any given time. At the time the account is created, this person should be set to the person responsible for opening the account.",
		},
		"address_kana.town": {
			Type:        "string",
			Description: "Town or cho-me.",
		},
		"address.country": {
			Type:        "string",
			Description: "Two-letter country code ([ISO 3166-1 alpha-2](https://en.wikipedia.org/wiki/ISO_3166-1_alpha-2)).",
		},
		"last_name": {
			Type:        "string",
			Description: "The person's last name.",
		},
		"address_kanji.city": {
			Type:        "string",
			Description: "City or ward.",
		},
		"address_kanji.country": {
			Type:        "string",
			Description: "Two-letter country code ([ISO 3166-1 alpha-2](https://en.wikipedia.org/wiki/ISO_3166-1_alpha-2)).",
		},
		"relationship.title": {
			Type:        "string",
			Description: "The person's title (e.g., CEO, Support Engineer).",
		},
		"relationship.authorizer": {
			Type:        "boolean",
			Description: "Whether the person is the authorizer of the account's representative.",
		},
		"relationship.director": {
			Type:        "boolean",
			Description: "Whether the person is a director of the account's legal entity. Directors are typically members of the governing board of the company, or responsible for ensuring the company meets its regulatory obligations.",
		},
		"us_cfpb_data.race_details.race_other": {
			Type:        "string",
			Description: "Please specify your race, when other is selected.",
		},
		"address.state": {
			Type:        "string",
			Description: "State, county, province, or region ([ISO 3166-2](https://en.wikipedia.org/wiki/ISO_3166-2)).",
		},
		"additional_tos_acceptances.account.user_agent": {
			Type:        "string",
			Description: "The user agent of the browser from which the account representative accepted the service agreement.",
		},
		"address_kana.city": {
			Type:        "string",
			Description: "City or ward.",
		},
		"person_token": {
			Type:        "string",
			Description: "A [person token](https://docs.stripe.com/connect/account-tokens), used to securely provide details to the person.",
		},
	},
}

var V1PersonsUpdate = resource.OperationSpec{
	Name:    "update",
	Path:    "/v1/accounts/{account}/persons/{person}",
	Method:  "POST",
	Summary: "Update a person",
	Params: map[string]*resource.ParamSpec{
		"address_kana.postal_code": {
			Type:        "string",
			Description: "Postal code.",
		},
		"additional_tos_acceptances.account.date": {
			Type:        "integer",
			Description: "The Unix timestamp marking when the account representative accepted the service agreement.",
			Format:      "unix-time",
		},
		"address_kanji.town": {
			Type:        "string",
			Description: "Town or cho-me.",
		},
		"political_exposure": {
			Type:        "string",
			Description: "Indicates if the person or any of their representatives, family members, or other closely related persons, declares that they hold or have held an important public job or function, in any jurisdiction.",
			Enum: []resource.EnumSpec{
				{Value: "existing"},
				{Value: "none"},
			},
		},
		"us_cfpb_data.self_identified_gender": {
			Type:        "string",
			Description: "The persons self-identified gender",
		},
		"documents.passport.files": {
			Type:        "array",
			Description: "One or more document ids returned by a [file upload](https://api.stripe.com#create_file) with a `purpose` value of `account_requirement`.",
		},
		"id_number_secondary": {
			Type:        "string",
			Description: "The person's secondary ID number, as appropriate for their country, will be used for enhanced verification checks. In Thailand, this would be the laser code found on the back of an ID card. Instead of the number itself, you can also provide a [PII token provided by Stripe.js](https://docs.stripe.com/js/tokens/create_token?type=pii).",
		},
		"person_token": {
			Type:        "string",
			Description: "A [person token](https://docs.stripe.com/connect/account-tokens), used to securely provide details to the person.",
		},
		"relationship.owner": {
			Type:        "boolean",
			Description: "Whether the person is an owner of the account’s legal entity.",
		},
		"address_kana.town": {
			Type:        "string",
			Description: "Town or cho-me.",
		},
		"additional_tos_acceptances.account.ip": {
			Type:        "string",
			Description: "The IP address from which the account representative accepted the service agreement.",
		},
		"relationship.executive": {
			Type:        "boolean",
			Description: "Whether the person has significant responsibility to control, manage, or direct the organization.",
		},
		"ssn_last_4": {
			Type:        "string",
			Description: "The last four digits of the person's Social Security number (U.S. only).",
		},
		"last_name_kana": {
			Type:        "string",
			Description: "The Kana variation of the person's last name (Japan only).",
		},
		"address_kanji.line1": {
			Type:        "string",
			Description: "Block or building number.",
		},
		"address_kanji.line2": {
			Type:        "string",
			Description: "Building details.",
		},
		"nationality": {
			Type:        "string",
			Description: "The country where the person is a national. Two-letter country code ([ISO 3166-1 alpha-2](https://en.wikipedia.org/wiki/ISO_3166-1_alpha-2)), or \"XX\" if unavailable.",
		},
		"registered_address.country": {
			Type:        "string",
			Description: "Two-letter country code ([ISO 3166-1 alpha-2](https://en.wikipedia.org/wiki/ISO_3166-1_alpha-2)).",
		},
		"registered_address.line1": {
			Type:        "string",
			Description: "Address line 1, such as the street, PO Box, or company name.",
		},
		"last_name_kanji": {
			Type:        "string",
			Description: "The Kanji variation of the person's last name (Japan only).",
		},
		"us_cfpb_data.race_details.race_other": {
			Type:        "string",
			Description: "Please specify your race, when other is selected.",
		},
		"documents.company_authorization.files": {
			Type:        "array",
			Description: "One or more document ids returned by a [file upload](https://api.stripe.com#create_file) with a `purpose` value of `account_requirement`.",
		},
		"phone": {
			Type:        "string",
			Description: "The person's phone number.",
		},
		"address_kana.line1": {
			Type:        "string",
			Description: "Block or building number.",
		},
		"address_kanji.city": {
			Type:        "string",
			Description: "City or ward.",
		},
		"full_name_aliases": {
			Type:        "array",
			Description: "A list of alternate names or aliases that the person is known by.",
		},
		"registered_address.city": {
			Type:        "string",
			Description: "City, district, suburb, town, or village.",
		},
		"registered_address.line2": {
			Type:        "string",
			Description: "Address line 2, such as the apartment, suite, unit, or building.",
		},
		"verification.additional_document.front": {
			Type:        "string",
			Description: "The front of an ID returned by a [file upload](https://api.stripe.com#create_file) with a `purpose` value of `identity_document`. The uploaded file needs to be a color image (smaller than 8,000px by 8,000px), in JPG, PNG, or PDF format, and less than 10 MB in size.",
		},
		"address_kana.country": {
			Type:        "string",
			Description: "Two-letter country code ([ISO 3166-1 alpha-2](https://en.wikipedia.org/wiki/ISO_3166-1_alpha-2)).",
		},
		"address.city": {
			Type:        "string",
			Description: "City, district, suburb, town, or village.",
		},
		"address_kana.state": {
			Type:        "string",
			Description: "Prefecture.",
		},
		"us_cfpb_data.ethnicity_details.ethnicity": {
			Type:        "array",
			Description: "The persons ethnicity",
		},
		"verification.additional_document.back": {
			Type:        "string",
			Description: "The back of an ID returned by a [file upload](https://api.stripe.com#create_file) with a `purpose` value of `identity_document`. The uploaded file needs to be a color image (smaller than 8,000px by 8,000px), in JPG, PNG, or PDF format, and less than 10 MB in size.",
		},
		"first_name_kanji": {
			Type:        "string",
			Description: "The Kanji variation of the person's first name (Japan only).",
		},
		"address_kana.city": {
			Type:        "string",
			Description: "City or ward.",
		},
		"address.state": {
			Type:        "string",
			Description: "State, county, province, or region ([ISO 3166-2](https://en.wikipedia.org/wiki/ISO_3166-2)).",
		},
		"id_number": {
			Type:        "string",
			Description: "The person's ID number, as appropriate for their country. For example, a social security number in the U.S., social insurance number in Canada, etc. Instead of the number itself, you can also provide a [PII token provided by Stripe.js](https://docs.stripe.com/js/tokens/create_token?type=pii).",
		},
		"gender": {
			Type:        "string",
			Description: "The person's gender (International regulations require either \"male\" or \"female\").",
		},
		"email": {
			Type:        "string",
			Description: "The person's email address.",
		},
		"relationship.percent_ownership": {
			Type:        "number",
			Description: "The percent owned by the person of the account's legal entity.",
		},
		"address_kana.line2": {
			Type:        "string",
			Description: "Building details.",
		},
		"documents.visa.files": {
			Type:        "array",
			Description: "One or more document ids returned by a [file upload](https://api.stripe.com#create_file) with a `purpose` value of `account_requirement`.",
		},
		"registered_address.postal_code": {
			Type:        "string",
			Description: "ZIP or postal code.",
		},
		"address.line1": {
			Type:        "string",
			Description: "Address line 1, such as the street, PO Box, or company name.",
		},
		"first_name": {
			Type:        "string",
			Description: "The person's first name.",
		},
		"first_name_kana": {
			Type:        "string",
			Description: "The Kana variation of the person's first name (Japan only).",
		},
		"registered_address.state": {
			Type:        "string",
			Description: "State, county, province, or region ([ISO 3166-2](https://en.wikipedia.org/wiki/ISO_3166-2)).",
		},
		"us_cfpb_data.ethnicity_details.ethnicity_other": {
			Type:        "string",
			Description: "Please specify your origin, when other is selected.",
		},
		"verification.document.back": {
			Type:        "string",
			Description: "The back of an ID returned by a [file upload](https://api.stripe.com#create_file) with a `purpose` value of `identity_document`. The uploaded file needs to be a color image (smaller than 8,000px by 8,000px), in JPG, PNG, or PDF format, and less than 10 MB in size.",
		},
		"last_name": {
			Type:        "string",
			Description: "The person's last name.",
		},
		"maiden_name": {
			Type:        "string",
			Description: "The person's maiden name.",
		},
		"address.country": {
			Type:        "string",
			Description: "Two-letter country code ([ISO 3166-1 alpha-2](https://en.wikipedia.org/wiki/ISO_3166-1_alpha-2)).",
		},
		"address_kanji.state": {
			Type:        "string",
			Description: "Prefecture.",
		},
		"relationship.representative": {
			Type:        "boolean",
			Description: "Whether the person is authorized as the primary representative of the account. This is the person nominated by the business to provide information about themselves, and general information about the account. There can only be one representative at any given time. At the time the account is created, this person should be set to the person responsible for opening the account.",
		},
		"relationship.authorizer": {
			Type:        "boolean",
			Description: "Whether the person is the authorizer of the account's representative.",
		},
		"relationship.legal_guardian": {
			Type:        "boolean",
			Description: "Whether the person is the legal guardian of the account's representative.",
		},
		"us_cfpb_data.race_details.race": {
			Type:        "array",
			Description: "The persons race.",
		},
		"address.line2": {
			Type:        "string",
			Description: "Address line 2, such as the apartment, suite, unit, or building.",
		},
		"address.postal_code": {
			Type:        "string",
			Description: "ZIP or postal code.",
		},
		"address_kanji.country": {
			Type:        "string",
			Description: "Two-letter country code ([ISO 3166-1 alpha-2](https://en.wikipedia.org/wiki/ISO_3166-1_alpha-2)).",
		},
		"relationship.title": {
			Type:        "string",
			Description: "The person's title (e.g., CEO, Support Engineer).",
		},
		"relationship.director": {
			Type:        "boolean",
			Description: "Whether the person is a director of the account's legal entity. Directors are typically members of the governing board of the company, or responsible for ensuring the company meets its regulatory obligations.",
		},
		"additional_tos_acceptances.account.user_agent": {
			Type:        "string",
			Description: "The user agent of the browser from which the account representative accepted the service agreement.",
		},
		"address_kanji.postal_code": {
			Type:        "string",
			Description: "Postal code.",
		},
		"verification.document.front": {
			Type:        "string",
			Description: "The front of an ID returned by a [file upload](https://api.stripe.com#create_file) with a `purpose` value of `identity_document`. The uploaded file needs to be a color image (smaller than 8,000px by 8,000px), in JPG, PNG, or PDF format, and less than 10 MB in size.",
		},
	},
}

var V1PlansRetrieve = resource.OperationSpec{
	Name:    "retrieve",
	Path:    "/v1/plans/{plan}",
	Method:  "GET",
	Summary: "Retrieve a plan",
}

var V1PlansCreate = resource.OperationSpec{
	Name:    "create",
	Path:    "/v1/plans",
	Method:  "POST",
	Summary: "Create a plan",
	Params: map[string]*resource.ParamSpec{
		"amount_decimal": {
			Type:        "string",
			Description: "Same as `amount`, but accepts a decimal value with at most 12 decimal places. Only one of `amount` and `amount_decimal` can be set.",
			Format:      "decimal",
		},
		"meter": {
			Type:        "string",
			Description: "The meter tracking the usage of a metered price",
		},
		"active": {
			Type:        "boolean",
			Description: "Whether the plan is currently available for new subscriptions. Defaults to `true`.",
		},
		"id": {
			Type:        "string",
			Description: "An identifier randomly generated by Stripe. Used to identify this plan when subscribing a customer. You can optionally override this ID, but the ID must be unique across all plans in your Stripe account. You can, however, use the same plan ID in both live and test modes.",
		},
		"nickname": {
			Type:        "string",
			Description: "A brief description of the plan, hidden from customers.",
		},
		"usage_type": {
			Type:        "string",
			Description: "Configures how the quantity per period should be determined. Can be either `metered` or `licensed`. `licensed` automatically bills the `quantity` set when adding it to a subscription. `metered` aggregates the total usage based on usage records. Defaults to `licensed`.",
			Enum: []resource.EnumSpec{
				{Value: "licensed"},
				{Value: "metered"},
			},
		},
		"interval_count": {
			Type:        "integer",
			Description: "The number of intervals between subscription billings. For example, `interval=month` and `interval_count=3` bills every 3 months. Maximum of three years interval allowed (3 years, 36 months, or 156 weeks).",
		},
		"tiers_mode": {
			Type:        "string",
			Description: "Defines if the tiering price should be `graduated` or `volume` based. In `volume`-based tiering, the maximum quantity within a period determines the per unit price, in `graduated` tiering pricing can successively change as the quantity grows.",
			Enum: []resource.EnumSpec{
				{Value: "graduated"},
				{Value: "volume"},
			},
		},
		"amount": {
			Type:        "integer",
			Description: "A positive integer in cents (or local equivalent) (or 0 for a free plan) representing how much to charge on a recurring basis.",
		},
		"transform_usage.divide_by": {
			Type:        "integer",
			Description: "Divide usage by this number.",
			Required:    true,
		},
		"transform_usage.round": {
			Type:        "string",
			Description: "After division, either round the result `up` or `down`.",
			Required:    true,
			Enum: []resource.EnumSpec{
				{Value: "down"},
				{Value: "up"},
			},
		},
		"currency": {
			Type:        "string",
			Description: "Three-letter [ISO currency code](https://www.iso.org/iso-4217-currency-codes.html), in lowercase. Must be a [supported currency](https://stripe.com/docs/currencies).",
			Required:    true,
			Format:      "currency",
		},
		"trial_period_days": {
			Type:        "integer",
			Description: "Default number of trial days when subscribing a customer to this plan using [`trial_from_plan=true`](https://docs.stripe.com/api#create_subscription-trial_from_plan).",
		},
		"product": {
			Type: "string",
		},
		"billing_scheme": {
			Type:        "string",
			Description: "Describes how to compute the price per period. Either `per_unit` or `tiered`. `per_unit` indicates that the fixed amount (specified in `amount`) will be charged per unit in `quantity` (for plans with `usage_type=licensed`), or per unit of total usage (for plans with `usage_type=metered`). `tiered` indicates that the unit pricing will be computed using a tiering strategy as defined using the `tiers` and `tiers_mode` attributes.",
			Enum: []resource.EnumSpec{
				{Value: "per_unit"},
				{Value: "tiered"},
			},
		},
		"interval": {
			Type:        "string",
			Description: "Specifies billing frequency. Either `day`, `week`, `month` or `year`.",
			Required:    true,
			Enum: []resource.EnumSpec{
				{Value: "day"},
				{Value: "month"},
				{Value: "week"},
				{Value: "year"},
			},
		},
	},
}

var V1PlansUpdate = resource.OperationSpec{
	Name:    "update",
	Path:    "/v1/plans/{plan}",
	Method:  "POST",
	Summary: "Update a plan",
	Params: map[string]*resource.ParamSpec{
		"nickname": {
			Type:        "string",
			Description: "A brief description of the plan, hidden from customers.",
		},
		"product": {
			Type:        "string",
			Description: "The product the plan belongs to. This cannot be changed once it has been used in a subscription or subscription schedule.",
		},
		"trial_period_days": {
			Type:        "integer",
			Description: "Default number of trial days when subscribing a customer to this plan using [`trial_from_plan=true`](https://docs.stripe.com/api#create_subscription-trial_from_plan).",
		},
		"active": {
			Type:        "boolean",
			Description: "Whether the plan is currently available for new subscriptions.",
		},
	},
}

var V1PlansDelete = resource.OperationSpec{
	Name:    "delete",
	Path:    "/v1/plans/{plan}",
	Method:  "DELETE",
	Summary: "Delete a plan",
}

var V1PlansList = resource.OperationSpec{
	Name:    "list",
	Path:    "/v1/plans",
	Method:  "GET",
	Summary: "List all plans",
	Params: map[string]*resource.ParamSpec{
		"ending_before": {
			Type:        "string",
			Description: "A cursor for use in pagination. `ending_before` is an object ID that defines your place in the list. For instance, if you make a list request and receive 100 objects, starting with `obj_bar`, your subsequent call can include `ending_before=obj_bar` in order to fetch the previous page of the list.",
		},
		"limit": {
			Type:        "integer",
			Description: "A limit on the number of objects to be returned. Limit can range between 1 and 100, and the default is 10.",
		},
		"product": {
			Type:        "string",
			Description: "Only return plans for the given product.",
		},
		"starting_after": {
			Type:        "string",
			Description: "A cursor for use in pagination. `starting_after` is an object ID that defines your place in the list. For instance, if you make a list request and receive 100 objects, ending with `obj_foo`, your subsequent call can include `starting_after=obj_foo` in order to fetch the next page of the list.",
		},
		"active": {
			Type:        "boolean",
			Description: "Only return plans that are active or inactive (e.g., pass `false` to list all inactive plans).",
		},
		"created": {
			Type:        "integer",
			Description: "A filter on the list, based on the object `created` field. The value can be a string with an integer Unix timestamp, or it can be a dictionary with a number of different query options.",
		},
	},
}

var V1SourcesDetach = resource.OperationSpec{
	Name:    "detach",
	Path:    "/v1/customers/{customer}/sources/{id}",
	Method:  "DELETE",
	Summary: "Delete a customer source",
}

var V1SourcesRetrieve = resource.OperationSpec{
	Name:    "retrieve",
	Path:    "/v1/sources/{source}",
	Method:  "GET",
	Summary: "Retrieve a source",
	Params: map[string]*resource.ParamSpec{
		"client_secret": {
			Type:        "string",
			Description: "The client secret of the source. Required if a publishable key is used to retrieve the source.",
		},
	},
}

var V1SourcesSourceTransactions = resource.OperationSpec{
	Name:   "source_transactions",
	Path:   "/v1/sources/{source}/source_transactions",
	Method: "GET",
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

var V1SourcesCreate = resource.OperationSpec{
	Name:    "create",
	Path:    "/v1/sources",
	Method:  "POST",
	Summary: "Shares a source",
	Params: map[string]*resource.ParamSpec{
		"owner.phone": {
			Type:        "string",
			Description: "Owner's phone number.",
		},
		"redirect.return_url": {
			Type:        "string",
			Description: "The URL you provide to redirect the customer back to you after they authenticated their payment. It can use your application URI scheme in the context of a mobile application.",
			Required:    true,
		},
		"source_order.shipping.address.city": {
			Type:        "string",
			Description: "City, district, suburb, town, or village.",
		},
		"mandate.acceptance.offline.contact_email": {
			Type:        "string",
			Description: "An email to contact you with if a copy of the mandate is requested, required if `type` is `offline`.",
			Required:    true,
		},
		"mandate.acceptance.type": {
			Type:        "string",
			Description: "The type of acceptance information included with the mandate. Either `online` or `offline`",
			Enum: []resource.EnumSpec{
				{Value: "offline"},
				{Value: "online"},
			},
		},
		"source_order.shipping.name": {
			Type:        "string",
			Description: "Recipient name.",
		},
		"owner.email": {
			Type:        "string",
			Description: "Owner's email address.",
		},
		"flow": {
			Type:        "string",
			Description: "The authentication `flow` of the source to create. `flow` is one of `redirect`, `receiver`, `code_verification`, `none`. It is generally inferred unless a type supports multiple flows.",
			Enum: []resource.EnumSpec{
				{Value: "code_verification"},
				{Value: "none"},
				{Value: "receiver"},
				{Value: "redirect"},
			},
		},
		"owner.address.city": {
			Type:        "string",
			Description: "City, district, suburb, town, or village.",
		},
		"owner.address.state": {
			Type:        "string",
			Description: "State, county, province, or region ([ISO 3166-2](https://en.wikipedia.org/wiki/ISO_3166-2)).",
		},
		"type": {
			Type:        "string",
			Description: "The `type` of the source to create. Required unless `customer` and `original_source` are specified (see the [Cloning card Sources](https://docs.stripe.com/sources/connect#cloning-card-sources) guide)",
		},
		"usage": {
			Type: "string",
			Enum: []resource.EnumSpec{
				{Value: "reusable"},
				{Value: "single_use"},
			},
		},
		"receiver.refund_attributes_method": {
			Type:        "string",
			Description: "The method Stripe should use to request information needed to process a refund or mispayment. Either `email` (an email is sent directly to the customer) or `manual` (a `source.refund_attributes_required` event is sent to your webhooks endpoint). Refer to each payment method's documentation to learn which refund attributes may be required.",
			Enum: []resource.EnumSpec{
				{Value: "email"},
				{Value: "manual"},
				{Value: "none"},
			},
		},
		"mandate.acceptance.online.user_agent": {
			Type:        "string",
			Description: "The user agent of the browser from which the mandate was accepted or refused by the customer.",
		},
		"owner.address.country": {
			Type:        "string",
			Description: "Two-letter country code ([ISO 3166-1 alpha-2](https://en.wikipedia.org/wiki/ISO_3166-1_alpha-2)).",
		},
		"owner.address.line2": {
			Type:        "string",
			Description: "Address line 2, such as the apartment, suite, unit, or building.",
		},
		"source_order.shipping.address.state": {
			Type:        "string",
			Description: "State, county, province, or region ([ISO 3166-2](https://en.wikipedia.org/wiki/ISO_3166-2)).",
		},
		"source_order.shipping.carrier": {
			Type:        "string",
			Description: "The delivery service that shipped a physical product, such as Fedex, UPS, USPS, etc.",
		},
		"owner.address.postal_code": {
			Type:        "string",
			Description: "ZIP or postal code.",
		},
		"token": {
			Type:        "string",
			Description: "An optional token used to create the source. When passed, token properties will override source parameters.",
		},
		"source_order.shipping.address.line1": {
			Type:        "string",
			Description: "Address line 1, such as the street, PO Box, or company name.",
			Required:    true,
		},
		"source_order.shipping.address.postal_code": {
			Type:        "string",
			Description: "ZIP or postal code.",
		},
		"mandate.notification_method": {
			Type:        "string",
			Description: "The method Stripe should use to notify the customer of upcoming debit instructions and/or mandate confirmation as required by the underlying debit network. Either `email` (an email is sent directly to the customer), `manual` (a `source.mandate_notification` event is sent to your webhooks endpoint and you should handle the notification) or `none` (the underlying debit network does not require any notification).",
			Enum: []resource.EnumSpec{
				{Value: "deprecated_none"},
				{Value: "email"},
				{Value: "manual"},
				{Value: "none"},
				{Value: "stripe_email"},
			},
		},
		"mandate.acceptance.date": {
			Type:        "integer",
			Description: "The Unix timestamp (in seconds) when the mandate was accepted or refused by the customer.",
			Format:      "unix-time",
		},
		"mandate.acceptance.online.date": {
			Type:        "integer",
			Description: "The Unix timestamp (in seconds) when the mandate was accepted or refused by the customer.",
			Format:      "unix-time",
		},
		"source_order.shipping.address.country": {
			Type:        "string",
			Description: "Two-letter country code ([ISO 3166-1 alpha-2](https://en.wikipedia.org/wiki/ISO_3166-1_alpha-2)).",
		},
		"amount": {
			Type:        "integer",
			Description: "Amount associated with the source. This is the amount for which the source will be chargeable once ready. Required for `single_use` sources. Not supported for `receiver` type sources, where charge amount may not be specified until funds land.",
		},
		"customer": {
			Type:        "string",
			Description: "The `Customer` to whom the original source is attached to. Must be set when the original source is not a `Source` (e.g., `Card`).",
		},
		"mandate.currency": {
			Type:        "string",
			Description: "The currency specified by the mandate. (Must match `currency` of the source)",
			Format:      "currency",
		},
		"mandate.acceptance.ip": {
			Type:        "string",
			Description: "The IP address from which the mandate was accepted or refused by the customer.",
		},
		"mandate.acceptance.status": {
			Type:        "string",
			Description: "The status of the mandate acceptance. Either `accepted` (the mandate was accepted) or `refused` (the mandate was refused).",
			Required:    true,
			Enum: []resource.EnumSpec{
				{Value: "accepted"},
				{Value: "pending"},
				{Value: "refused"},
				{Value: "revoked"},
			},
		},
		"mandate.acceptance.user_agent": {
			Type:        "string",
			Description: "The user agent of the browser from which the mandate was accepted or refused by the customer.",
		},
		"mandate.amount": {
			Type:        "integer",
			Description: "The amount specified by the mandate. (Leave null for a mandate covering all amounts)",
		},
		"owner.address.line1": {
			Type:        "string",
			Description: "Address line 1, such as the street, PO Box, or company name.",
		},
		"original_source": {
			Type:        "string",
			Description: "The source to share.",
		},
		"mandate.interval": {
			Type:        "string",
			Description: "The interval of debits permitted by the mandate. Either `one_time` (just permitting a single debit), `scheduled` (with debits on an agreed schedule or for clearly-defined events), or `variable`(for debits with any frequency)",
			Enum: []resource.EnumSpec{
				{Value: "one_time"},
				{Value: "scheduled"},
				{Value: "variable"},
			},
		},
		"statement_descriptor": {
			Type:        "string",
			Description: "An arbitrary string to be displayed on your customer's statement. As an example, if your website is `RunClub` and the item you're charging for is a race ticket, you may want to specify a `statement_descriptor` of `RunClub 5K race ticket.` While many payment types will display this information, some may not display it at all.",
		},
		"currency": {
			Type:        "string",
			Description: "Three-letter [ISO code for the currency](https://stripe.com/docs/currencies) associated with the source. This is the currency for which the source will be chargeable once ready.",
			Format:      "currency",
		},
		"source_order.shipping.tracking_number": {
			Type:        "string",
			Description: "The tracking number for a physical product, obtained from the delivery service. If multiple tracking numbers were generated for this purchase, please separate them with commas.",
		},
		"source_order.shipping.address.line2": {
			Type:        "string",
			Description: "Address line 2, such as the apartment, suite, unit, or building.",
		},
		"source_order.shipping.phone": {
			Type:        "string",
			Description: "Recipient phone (including extension).",
		},
		"mandate.acceptance.online.ip": {
			Type:        "string",
			Description: "The IP address from which the mandate was accepted or refused by the customer.",
		},
		"owner.name": {
			Type:        "string",
			Description: "Owner's full name.",
		},
	},
}

var V1SourcesUpdate = resource.OperationSpec{
	Name:    "update",
	Path:    "/v1/sources/{source}",
	Method:  "POST",
	Summary: "Update a source",
	Params: map[string]*resource.ParamSpec{
		"source_order.shipping.address.line2": {
			Type:        "string",
			Description: "Address line 2, such as the apartment, suite, unit, or building.",
		},
		"source_order.shipping.tracking_number": {
			Type:        "string",
			Description: "The tracking number for a physical product, obtained from the delivery service. If multiple tracking numbers were generated for this purchase, please separate them with commas.",
		},
		"mandate.acceptance.user_agent": {
			Type:        "string",
			Description: "The user agent of the browser from which the mandate was accepted or refused by the customer.",
		},
		"owner.address.state": {
			Type:        "string",
			Description: "State, county, province, or region ([ISO 3166-2](https://en.wikipedia.org/wiki/ISO_3166-2)).",
		},
		"owner.name": {
			Type:        "string",
			Description: "Owner's full name.",
		},
		"mandate.acceptance.online.date": {
			Type:        "integer",
			Description: "The Unix timestamp (in seconds) when the mandate was accepted or refused by the customer.",
			Format:      "unix-time",
		},
		"mandate.acceptance.online.user_agent": {
			Type:        "string",
			Description: "The user agent of the browser from which the mandate was accepted or refused by the customer.",
		},
		"mandate.notification_method": {
			Type:        "string",
			Description: "The method Stripe should use to notify the customer of upcoming debit instructions and/or mandate confirmation as required by the underlying debit network. Either `email` (an email is sent directly to the customer), `manual` (a `source.mandate_notification` event is sent to your webhooks endpoint and you should handle the notification) or `none` (the underlying debit network does not require any notification).",
			Enum: []resource.EnumSpec{
				{Value: "deprecated_none"},
				{Value: "email"},
				{Value: "manual"},
				{Value: "none"},
				{Value: "stripe_email"},
			},
		},
		"owner.address.city": {
			Type:        "string",
			Description: "City, district, suburb, town, or village.",
		},
		"amount": {
			Type:        "integer",
			Description: "Amount associated with the source.",
		},
		"mandate.amount": {
			Type:        "integer",
			Description: "The amount specified by the mandate. (Leave null for a mandate covering all amounts)",
		},
		"owner.address.postal_code": {
			Type:        "string",
			Description: "ZIP or postal code.",
		},
		"owner.phone": {
			Type:        "string",
			Description: "Owner's phone number.",
		},
		"source_order.shipping.address.city": {
			Type:        "string",
			Description: "City, district, suburb, town, or village.",
		},
		"source_order.shipping.carrier": {
			Type:        "string",
			Description: "The delivery service that shipped a physical product, such as Fedex, UPS, USPS, etc.",
		},
		"mandate.acceptance.online.ip": {
			Type:        "string",
			Description: "The IP address from which the mandate was accepted or refused by the customer.",
		},
		"mandate.acceptance.type": {
			Type:        "string",
			Description: "The type of acceptance information included with the mandate. Either `online` or `offline`",
			Enum: []resource.EnumSpec{
				{Value: "offline"},
				{Value: "online"},
			},
		},
		"owner.address.country": {
			Type:        "string",
			Description: "Two-letter country code ([ISO 3166-1 alpha-2](https://en.wikipedia.org/wiki/ISO_3166-1_alpha-2)).",
		},
		"source_order.shipping.address.state": {
			Type:        "string",
			Description: "State, county, province, or region ([ISO 3166-2](https://en.wikipedia.org/wiki/ISO_3166-2)).",
		},
		"mandate.acceptance.offline.contact_email": {
			Type:        "string",
			Description: "An email to contact you with if a copy of the mandate is requested, required if `type` is `offline`.",
			Required:    true,
		},
		"mandate.acceptance.status": {
			Type:        "string",
			Description: "The status of the mandate acceptance. Either `accepted` (the mandate was accepted) or `refused` (the mandate was refused).",
			Required:    true,
			Enum: []resource.EnumSpec{
				{Value: "accepted"},
				{Value: "pending"},
				{Value: "refused"},
				{Value: "revoked"},
			},
		},
		"mandate.acceptance.date": {
			Type:        "integer",
			Description: "The Unix timestamp (in seconds) when the mandate was accepted or refused by the customer.",
			Format:      "unix-time",
		},
		"owner.address.line1": {
			Type:        "string",
			Description: "Address line 1, such as the street, PO Box, or company name.",
		},
		"owner.email": {
			Type:        "string",
			Description: "Owner's email address.",
		},
		"source_order.shipping.address.postal_code": {
			Type:        "string",
			Description: "ZIP or postal code.",
		},
		"source_order.shipping.address.line1": {
			Type:        "string",
			Description: "Address line 1, such as the street, PO Box, or company name.",
			Required:    true,
		},
		"source_order.shipping.name": {
			Type:        "string",
			Description: "Recipient name.",
		},
		"source_order.shipping.phone": {
			Type:        "string",
			Description: "Recipient phone (including extension).",
		},
		"mandate.acceptance.ip": {
			Type:        "string",
			Description: "The IP address from which the mandate was accepted or refused by the customer.",
		},
		"mandate.currency": {
			Type:        "string",
			Description: "The currency specified by the mandate. (Must match `currency` of the source)",
			Format:      "currency",
		},
		"mandate.interval": {
			Type:        "string",
			Description: "The interval of debits permitted by the mandate. Either `one_time` (just permitting a single debit), `scheduled` (with debits on an agreed schedule or for clearly-defined events), or `variable`(for debits with any frequency)",
			Enum: []resource.EnumSpec{
				{Value: "one_time"},
				{Value: "scheduled"},
				{Value: "variable"},
			},
		},
		"owner.address.line2": {
			Type:        "string",
			Description: "Address line 2, such as the apartment, suite, unit, or building.",
		},
		"source_order.shipping.address.country": {
			Type:        "string",
			Description: "Two-letter country code ([ISO 3166-1 alpha-2](https://en.wikipedia.org/wiki/ISO_3166-1_alpha-2)).",
		},
	},
}

var V1SourcesVerify = resource.OperationSpec{
	Name:   "verify",
	Path:   "/v1/sources/{source}/verify",
	Method: "POST",
	Params: map[string]*resource.ParamSpec{
		"values": {
			Type:        "array",
			Description: "The values needed to verify the source.",
			Required:    true,
		},
	},
}

var V1PromotionCodesList = resource.OperationSpec{
	Name:    "list",
	Path:    "/v1/promotion_codes",
	Method:  "GET",
	Summary: "List all promotion codes",
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
		"active": {
			Type:        "boolean",
			Description: "Filter promotion codes by whether they are active.",
		},
		"code": {
			Type:        "string",
			Description: "Only return promotion codes that have this case-insensitive code.",
		},
		"customer": {
			Type:        "string",
			Description: "Only return promotion codes that are restricted to this customer.",
		},
		"coupon": {
			Type:        "string",
			Description: "Only return promotion codes for this coupon.",
		},
		"created": {
			Type:        "integer",
			Description: "A filter on the list, based on the object `created` field. The value can be a string with an integer Unix timestamp, or it can be a dictionary with a number of different query options.",
		},
		"customer_account": {
			Type:        "string",
			Description: "Only return promotion codes that are restricted to this account representing the customer.",
		},
	},
}

var V1PromotionCodesRetrieve = resource.OperationSpec{
	Name:    "retrieve",
	Path:    "/v1/promotion_codes/{promotion_code}",
	Method:  "GET",
	Summary: "Retrieve a promotion code",
}

var V1PromotionCodesCreate = resource.OperationSpec{
	Name:    "create",
	Path:    "/v1/promotion_codes",
	Method:  "POST",
	Summary: "Create a promotion code",
	Params: map[string]*resource.ParamSpec{
		"restrictions.minimum_amount_currency": {
			Type:        "string",
			Description: "Three-letter [ISO code](https://stripe.com/docs/currencies) for minimum_amount",
			Format:      "currency",
		},
		"active": {
			Type:        "boolean",
			Description: "Whether the promotion code is currently active.",
		},
		"code": {
			Type:        "string",
			Description: "The customer-facing code. Regardless of case, this code must be unique across all active promotion codes for a specific customer. Valid characters are lower case letters (a-z), upper case letters (A-Z), digits (0-9), and dashes (-).\n\nIf left blank, we will generate one automatically.",
		},
		"customer": {
			Type:        "string",
			Description: "The customer who can use this promotion code. If not set, all customers can use the promotion code.",
		},
		"customer_account": {
			Type:        "string",
			Description: "The account representing the customer who can use this promotion code. If not set, all customers can use the promotion code.",
		},
		"expires_at": {
			Type:        "integer",
			Description: "The timestamp at which this promotion code will expire. If the coupon has specified a `redeems_by`, then this value cannot be after the coupon's `redeems_by`.",
			Format:      "unix-time",
		},
		"max_redemptions": {
			Type:        "integer",
			Description: "A positive integer specifying the number of times the promotion code can be redeemed. If the coupon has specified a `max_redemptions`, then this value cannot be greater than the coupon's `max_redemptions`.",
		},
		"promotion.type": {
			Type:        "string",
			Description: "Specifies the type of promotion.",
			Required:    true,
			Enum: []resource.EnumSpec{
				{Value: "coupon"},
			},
		},
		"restrictions.first_time_transaction": {
			Type:        "boolean",
			Description: "A Boolean indicating if the Promotion Code should only be redeemed for Customers without any successful payments or invoices",
		},
		"promotion.coupon": {
			Type:        "string",
			Description: "If promotion `type` is `coupon`, the coupon for this promotion code.",
		},
		"restrictions.minimum_amount": {
			Type:        "integer",
			Description: "Minimum amount required to redeem this Promotion Code into a Coupon (e.g., a purchase must be $100 or more to work).",
		},
	},
}

var V1PromotionCodesUpdate = resource.OperationSpec{
	Name:    "update",
	Path:    "/v1/promotion_codes/{promotion_code}",
	Method:  "POST",
	Summary: "Update a promotion code",
	Params: map[string]*resource.ParamSpec{
		"active": {
			Type:        "boolean",
			Description: "Whether the promotion code is currently active. A promotion code can only be reactivated when the coupon is still valid and the promotion code is otherwise redeemable.",
		},
	},
}

var V1BalanceTransactionsList = resource.OperationSpec{
	Name:    "list",
	Path:    "/v1/balance_transactions",
	Method:  "GET",
	Summary: "List all balance transactions",
	Params: map[string]*resource.ParamSpec{
		"source": {
			Type:        "string",
			Description: "Only returns transactions associated with the given object.",
		},
		"starting_after": {
			Type:        "string",
			Description: "A cursor for use in pagination. `starting_after` is an object ID that defines your place in the list. For instance, if you make a list request and receive 100 objects, ending with `obj_foo`, your subsequent call can include `starting_after=obj_foo` in order to fetch the next page of the list.",
		},
		"type": {
			Type:        "string",
			Description: "Only returns transactions of the given type. One of: `adjustment`, `advance`, `advance_funding`, `anticipation_repayment`, `application_fee`, `application_fee_refund`, `charge`, `climate_order_purchase`, `climate_order_refund`, `connect_collection_transfer`, `contribution`, `issuing_authorization_hold`, `issuing_authorization_release`, `issuing_dispute`, `issuing_transaction`, `obligation_outbound`, `obligation_reversal_inbound`, `payment`, `payment_failure_refund`, `payment_network_reserve_hold`, `payment_network_reserve_release`, `payment_refund`, `payment_reversal`, `payment_unreconciled`, `payout`, `payout_cancel`, `payout_failure`, `payout_minimum_balance_hold`, `payout_minimum_balance_release`, `refund`, `refund_failure`, `reserve_transaction`, `reserved_funds`, `reserve_hold`, `reserve_release`, `stripe_fee`, `stripe_fx_fee`, `stripe_balance_payment_debit`, `stripe_balance_payment_debit_reversal`, `tax_fee`, `topup`, `topup_reversal`, `transfer`, `transfer_cancel`, `transfer_failure`, or `transfer_refund`.",
		},
		"created": {
			Type:        "integer",
			Description: "Only return transactions that were created during the given date interval.",
		},
		"currency": {
			Type:        "string",
			Description: "Only return transactions in a certain currency. Three-letter [ISO currency code](https://www.iso.org/iso-4217-currency-codes.html), in lowercase. Must be a [supported currency](https://stripe.com/docs/currencies).",
			Format:      "currency",
		},
		"ending_before": {
			Type:        "string",
			Description: "A cursor for use in pagination. `ending_before` is an object ID that defines your place in the list. For instance, if you make a list request and receive 100 objects, starting with `obj_bar`, your subsequent call can include `ending_before=obj_bar` in order to fetch the previous page of the list.",
		},
		"limit": {
			Type:        "integer",
			Description: "A limit on the number of objects to be returned. Limit can range between 1 and 100, and the default is 10.",
		},
		"payout": {
			Type:        "string",
			Description: "For automatic Stripe payouts only, only returns transactions that were paid out on the specified payout ID.",
		},
	},
}

var V1BalanceTransactionsRetrieve = resource.OperationSpec{
	Name:    "retrieve",
	Path:    "/v1/balance_transactions/{id}",
	Method:  "GET",
	Summary: "Retrieve a balance transaction",
}

var V1BankAccountsDelete = resource.OperationSpec{
	Name:    "delete",
	Path:    "/v1/accounts/{account}/external_accounts/{id}",
	Method:  "DELETE",
	Summary: "Delete an external account",
}

var V1BankAccountsUpdate = resource.OperationSpec{
	Name:   "update",
	Path:   "/v1/accounts/{account}/external_accounts/{id}",
	Method: "POST",
	Params: map[string]*resource.ParamSpec{
		"address_zip": {
			Type:        "string",
			Description: "ZIP or postal code.",
		},
		"default_for_currency": {
			Type:        "boolean",
			Description: "When set to true, this becomes the default external account for its currency.",
		},
		"name": {
			Type:        "string",
			Description: "Cardholder name.",
		},
		"account_holder_type": {
			Type:        "string",
			Description: "The type of entity that holds the account. This can be either `individual` or `company`.",
			Enum: []resource.EnumSpec{
				{Value: "company"},
				{Value: "individual"},
			},
		},
		"address_city": {
			Type:        "string",
			Description: "City/District/Suburb/Town/Village.",
		},
		"documents.bank_account_ownership_verification.files": {
			Type:        "array",
			Description: "One or more document ids returned by a [file upload](https://api.stripe.com#create_file) with a `purpose` value of `account_requirement`.",
		},
		"exp_year": {
			Type:        "string",
			Description: "Four digit number representing the card’s expiration year.",
		},
		"address_country": {
			Type:        "string",
			Description: "Billing address country, if provided when creating card.",
		},
		"exp_month": {
			Type:        "string",
			Description: "Two digit number representing the card’s expiration month.",
		},
		"account_type": {
			Type:        "string",
			Description: "The bank account type. This can only be `checking` or `savings` in most countries. In Japan, this can only be `futsu` or `toza`.",
			Enum: []resource.EnumSpec{
				{Value: "checking"},
				{Value: "futsu"},
				{Value: "savings"},
				{Value: "toza"},
			},
		},
		"address_state": {
			Type:        "string",
			Description: "State/County/Province/Region.",
		},
		"address_line2": {
			Type:        "string",
			Description: "Address line 2 (Apartment/Suite/Unit/Building).",
		},
		"account_holder_name": {
			Type:        "string",
			Description: "The name of the person or business that owns the bank account.",
		},
		"address_line1": {
			Type:        "string",
			Description: "Address line 1 (Street address/PO Box/Company name).",
		},
	},
}

var V1BankAccountsVerify = resource.OperationSpec{
	Name:    "verify",
	Path:    "/v1/customers/{customer}/sources/{id}/verify",
	Method:  "POST",
	Summary: "Verify a bank account",
	Params: map[string]*resource.ParamSpec{
		"amounts": {
			Type:        "array",
			Description: "Two positive integers, in *cents*, equal to the values of the microdeposits sent to the bank account.",
		},
	},
}

var V1CustomerCashBalanceTransactionsList = resource.OperationSpec{
	Name:    "list",
	Path:    "/v1/customers/{customer}/cash_balance_transactions",
	Method:  "GET",
	Summary: "List cash balance transactions",
	Params: map[string]*resource.ParamSpec{
		"starting_after": {
			Type:        "string",
			Description: "A cursor for use in pagination. `starting_after` is an object ID that defines your place in the list. For instance, if you make a list request and receive 100 objects, ending with `obj_foo`, your subsequent call can include `starting_after=obj_foo` in order to fetch the next page of the list.",
		},
		"ending_before": {
			Type:        "string",
			Description: "A cursor for use in pagination. `ending_before` is an object ID that defines your place in the list. For instance, if you make a list request and receive 100 objects, starting with `obj_bar`, your subsequent call can include `ending_before=obj_bar` in order to fetch the previous page of the list.",
		},
		"limit": {
			Type:        "integer",
			Description: "A limit on the number of objects to be returned. Limit can range between 1 and 100, and the default is 10.",
		},
	},
}

var V1CustomerCashBalanceTransactionsRetrieve = resource.OperationSpec{
	Name:    "retrieve",
	Path:    "/v1/customers/{customer}/cash_balance_transactions/{transaction}",
	Method:  "GET",
	Summary: "Retrieve a cash balance transaction",
}

var V1EventsList = resource.OperationSpec{
	Name:    "list",
	Path:    "/v1/events",
	Method:  "GET",
	Summary: "List all events",
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
		"type": {
			Type:        "string",
			Description: "A string containing a specific event name, or group of events using * as a wildcard. The list will be filtered to include only events with a matching event property.",
		},
		"types": {
			Type:        "array",
			Description: "An array of up to 20 strings containing specific event names. The list will be filtered to include only events with a matching event property. You may pass either `type` or `types`, but not both.",
		},
		"created": {
			Type:        "integer",
			Description: "Only return events that were created during the given date interval.",
		},
		"delivery_success": {
			Type:        "boolean",
			Description: "Filter events by whether all webhooks were successfully delivered. If false, events which are still pending or have failed all delivery attempts to a webhook endpoint will be returned.",
		},
	},
}

var V1EventsRetrieve = resource.OperationSpec{
	Name:    "retrieve",
	Path:    "/v1/events/{id}",
	Method:  "GET",
	Summary: "Retrieve an event",
}

var V1PaymentMethodsList = resource.OperationSpec{
	Name:    "list",
	Path:    "/v1/payment_methods",
	Method:  "GET",
	Summary: "List PaymentMethods",
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
		"type": {
			Type:        "string",
			Description: "Filters the list by the object `type` field. Unfiltered, the list returns all payment method types except `custom`. If your integration expects only one type of payment method in the response, specify that type value in the request to reduce your payload.",
			Enum: []resource.EnumSpec{
				{Value: "acss_debit"},
				{Value: "affirm"},
				{Value: "afterpay_clearpay"},
				{Value: "alipay"},
				{Value: "alma"},
				{Value: "amazon_pay"},
				{Value: "au_becs_debit"},
				{Value: "bacs_debit"},
				{Value: "bancontact"},
				{Value: "billie"},
				{Value: "blik"},
				{Value: "boleto"},
				{Value: "card"},
				{Value: "cashapp"},
				{Value: "crypto"},
				{Value: "custom"},
				{Value: "customer_balance"},
				{Value: "eps"},
				{Value: "fpx"},
				{Value: "giropay"},
				{Value: "grabpay"},
				{Value: "ideal"},
				{Value: "kakao_pay"},
				{Value: "klarna"},
				{Value: "konbini"},
				{Value: "kr_card"},
				{Value: "link"},
				{Value: "mb_way"},
				{Value: "mobilepay"},
				{Value: "multibanco"},
				{Value: "naver_pay"},
				{Value: "nz_bank_account"},
				{Value: "oxxo"},
				{Value: "p24"},
				{Value: "pay_by_bank"},
				{Value: "payco"},
				{Value: "paynow"},
				{Value: "paypal"},
				{Value: "payto"},
				{Value: "pix"},
				{Value: "promptpay"},
				{Value: "revolut_pay"},
				{Value: "samsung_pay"},
				{Value: "satispay"},
				{Value: "sepa_debit"},
				{Value: "sofort"},
				{Value: "swish"},
				{Value: "twint"},
				{Value: "us_bank_account"},
				{Value: "wechat_pay"},
				{Value: "zip"},
			},
		},
		"allow_redisplay": {
			Type:        "string",
			Description: "This field indicates whether this payment method can be shown again to its customer in a checkout flow. Stripe products such as Checkout and Elements use this field to determine whether a payment method can be shown as a saved payment method in a checkout flow.",
			Enum: []resource.EnumSpec{
				{Value: "always"},
				{Value: "limited"},
				{Value: "unspecified"},
			},
		},
		"customer": {
			Type:        "string",
			Description: "The ID of the customer whose PaymentMethods will be retrieved.",
		},
		"customer_account": {
			Type:        "string",
			Description: "The ID of the Account whose PaymentMethods will be retrieved.",
		},
	},
}

var V1PaymentMethodsRetrieve = resource.OperationSpec{
	Name:    "retrieve",
	Path:    "/v1/payment_methods/{payment_method}",
	Method:  "GET",
	Summary: "Retrieve a PaymentMethod",
}

var V1PaymentMethodsCreate = resource.OperationSpec{
	Name:    "create",
	Path:    "/v1/payment_methods",
	Method:  "POST",
	Summary: "Shares a PaymentMethod",
	Params: map[string]*resource.ParamSpec{
		"nz_bank_account.account_number": {
			Type:        "string",
			Description: "The account number for the bank account.",
			Required:    true,
		},
		"p24.bank": {
			Type:        "string",
			Description: "The customer's bank.",
			Enum: []resource.EnumSpec{
				{Value: "alior_bank"},
				{Value: "bank_millennium"},
				{Value: "bank_nowy_bfg_sa"},
				{Value: "bank_pekao_sa"},
				{Value: "banki_spbdzielcze"},
				{Value: "blik"},
				{Value: "bnp_paribas"},
				{Value: "boz"},
				{Value: "citi_handlowy"},
				{Value: "credit_agricole"},
				{Value: "envelobank"},
				{Value: "etransfer_pocztowy24"},
				{Value: "getin_bank"},
				{Value: "ideabank"},
				{Value: "ing"},
				{Value: "inteligo"},
				{Value: "mbank_mtransfer"},
				{Value: "nest_przelew"},
				{Value: "noble_pay"},
				{Value: "pbac_z_ipko"},
				{Value: "plus_bank"},
				{Value: "santander_przelew24"},
				{Value: "tmobile_usbugi_bankowe"},
				{Value: "toyota_bank"},
				{Value: "velobank"},
				{Value: "volkswagen_bank"},
			},
		},
		"ideal.bank": {
			Type:        "string",
			Description: "The customer's bank. Only use this parameter for existing customers. Don't use it for new customers.",
			Enum: []resource.EnumSpec{
				{Value: "abn_amro"},
				{Value: "adyen"},
				{Value: "asn_bank"},
				{Value: "bunq"},
				{Value: "buut"},
				{Value: "finom"},
				{Value: "handelsbanken"},
				{Value: "ing"},
				{Value: "knab"},
				{Value: "mollie"},
				{Value: "moneyou"},
				{Value: "n26"},
				{Value: "nn"},
				{Value: "rabobank"},
				{Value: "regiobank"},
				{Value: "revolut"},
				{Value: "sns_bank"},
				{Value: "triodos_bank"},
				{Value: "van_lanschot"},
				{Value: "yoursafe"},
			},
		},
		"us_bank_account.account_holder_type": {
			Type:        "string",
			Description: "Account holder type: individual or company.",
			Enum: []resource.EnumSpec{
				{Value: "company"},
				{Value: "individual"},
			},
		},
		"us_bank_account.account_number": {
			Type:        "string",
			Description: "Account number of the bank account.",
		},
		"klarna.dob.year": {
			Type:        "integer",
			Description: "The four-digit year of birth.",
			Required:    true,
		},
		"us_bank_account.account_type": {
			Type:        "string",
			Description: "Account type: checkings or savings. Defaults to checking if omitted.",
			Enum: []resource.EnumSpec{
				{Value: "checking"},
				{Value: "savings"},
			},
		},
		"us_bank_account.financial_connections_account": {
			Type:        "string",
			Description: "The ID of a Financial Connections Account to use as a payment method.",
		},
		"billing_details.phone": {
			Type:        "string",
			Description: "Billing phone number (including extension).",
		},
		"fpx.account_holder_type": {
			Type:        "string",
			Description: "Account holder type for FPX transaction",
			Enum: []resource.EnumSpec{
				{Value: "company"},
				{Value: "individual"},
			},
		},
		"payto.pay_id": {
			Type:        "string",
			Description: "The PayID alias for the bank account.",
		},
		"klarna.dob.month": {
			Type:        "integer",
			Description: "The month of birth, between 1 and 12.",
			Required:    true,
		},
		"billing_details.tax_id": {
			Type:        "string",
			Description: "Taxpayer identification number. Used only for transactions between LATAM buyers and non-LATAM sellers.",
		},
		"nz_bank_account.suffix": {
			Type:        "string",
			Description: "The suffix of the bank account number.",
			Required:    true,
		},
		"nz_bank_account.branch_code": {
			Type:        "string",
			Description: "The numeric code for the bank account's bank branch.",
			Required:    true,
		},
		"payment_method": {
			Type:        "string",
			Description: "The PaymentMethod to share.",
		},
		"payto.bsb_number": {
			Type:        "string",
			Description: "Bank-State-Branch number of the bank account.",
		},
		"au_becs_debit.account_number": {
			Type:        "string",
			Description: "The account number for the bank account.",
			Required:    true,
		},
		"acss_debit.transit_number": {
			Type:        "string",
			Description: "Transit number of the customer's bank.",
			Required:    true,
		},
		"nz_bank_account.reference": {
			Type: "string",
		},
		"sepa_debit.iban": {
			Type:        "string",
			Description: "IBAN of the bank account.",
			Required:    true,
		},
		"us_bank_account.routing_number": {
			Type:        "string",
			Description: "Routing number of the bank account.",
		},
		"boleto.tax_id": {
			Type:        "string",
			Description: "The tax ID of the customer (CPF for individual consumers or CNPJ for businesses consumers)",
			Required:    true,
		},
		"billing_details.email": {
			Type:        "string",
			Description: "Email address.",
		},
		"billing_details.name": {
			Type:        "string",
			Description: "Full name.",
		},
		"custom.type": {
			Type:        "string",
			Description: "ID of the Dashboard-only CustomPaymentMethodType. This field is used by Stripe products' internal code to support CPMs.",
			Required:    true,
		},
		"type": {
			Type:        "string",
			Description: "The type of the PaymentMethod. An additional hash is included on the PaymentMethod with a name matching this value. It contains additional information specific to the PaymentMethod type.",
			Enum: []resource.EnumSpec{
				{Value: "acss_debit"},
				{Value: "affirm"},
				{Value: "afterpay_clearpay"},
				{Value: "alipay"},
				{Value: "alma"},
				{Value: "amazon_pay"},
				{Value: "au_becs_debit"},
				{Value: "bacs_debit"},
				{Value: "bancontact"},
				{Value: "billie"},
				{Value: "blik"},
				{Value: "boleto"},
				{Value: "card"},
				{Value: "cashapp"},
				{Value: "crypto"},
				{Value: "custom"},
				{Value: "customer_balance"},
				{Value: "eps"},
				{Value: "fpx"},
				{Value: "giropay"},
				{Value: "grabpay"},
				{Value: "ideal"},
				{Value: "kakao_pay"},
				{Value: "klarna"},
				{Value: "konbini"},
				{Value: "kr_card"},
				{Value: "link"},
				{Value: "mb_way"},
				{Value: "mobilepay"},
				{Value: "multibanco"},
				{Value: "naver_pay"},
				{Value: "nz_bank_account"},
				{Value: "oxxo"},
				{Value: "p24"},
				{Value: "pay_by_bank"},
				{Value: "payco"},
				{Value: "paynow"},
				{Value: "paypal"},
				{Value: "payto"},
				{Value: "pix"},
				{Value: "promptpay"},
				{Value: "revolut_pay"},
				{Value: "samsung_pay"},
				{Value: "satispay"},
				{Value: "sepa_debit"},
				{Value: "sofort"},
				{Value: "swish"},
				{Value: "twint"},
				{Value: "us_bank_account"},
				{Value: "wechat_pay"},
				{Value: "zip"},
			},
		},
		"eps.bank": {
			Type:        "string",
			Description: "The customer's bank.",
			Enum: []resource.EnumSpec{
				{Value: "arzte_und_apotheker_bank"},
				{Value: "austrian_anadi_bank_ag"},
				{Value: "bank_austria"},
				{Value: "bankhaus_carl_spangler"},
				{Value: "bankhaus_schelhammer_und_schattera_ag"},
				{Value: "bawag_psk_ag"},
				{Value: "bks_bank_ag"},
				{Value: "brull_kallmus_bank_ag"},
				{Value: "btv_vier_lander_bank"},
				{Value: "capital_bank_grawe_gruppe_ag"},
				{Value: "deutsche_bank_ag"},
				{Value: "dolomitenbank"},
				{Value: "easybank_ag"},
				{Value: "erste_bank_und_sparkassen"},
				{Value: "hypo_alpeadriabank_international_ag"},
				{Value: "hypo_bank_burgenland_aktiengesellschaft"},
				{Value: "hypo_noe_lb_fur_niederosterreich_u_wien"},
				{Value: "hypo_oberosterreich_salzburg_steiermark"},
				{Value: "hypo_tirol_bank_ag"},
				{Value: "hypo_vorarlberg_bank_ag"},
				{Value: "marchfelder_bank"},
				{Value: "oberbank_ag"},
				{Value: "raiffeisen_bankengruppe_osterreich"},
				{Value: "schoellerbank_ag"},
				{Value: "sparda_bank_wien"},
				{Value: "volksbank_gruppe"},
				{Value: "volkskreditbank_ag"},
				{Value: "vr_bank_braunau"},
			},
		},
		"bacs_debit.account_number": {
			Type:        "string",
			Description: "Account number of the bank account that the funds will be debited from.",
		},
		"fpx.bank": {
			Type:        "string",
			Description: "The customer's bank.",
			Required:    true,
			Enum: []resource.EnumSpec{
				{Value: "affin_bank"},
				{Value: "agrobank"},
				{Value: "alliance_bank"},
				{Value: "ambank"},
				{Value: "bank_islam"},
				{Value: "bank_muamalat"},
				{Value: "bank_of_china"},
				{Value: "bank_rakyat"},
				{Value: "bsn"},
				{Value: "cimb"},
				{Value: "deutsche_bank"},
				{Value: "hong_leong_bank"},
				{Value: "hsbc"},
				{Value: "kfh"},
				{Value: "maybank2e"},
				{Value: "maybank2u"},
				{Value: "ocbc"},
				{Value: "pb_enterprise"},
				{Value: "public_bank"},
				{Value: "rhb"},
				{Value: "standard_chartered"},
				{Value: "uob"},
			},
		},
		"nz_bank_account.bank_code": {
			Type:        "string",
			Description: "The numeric code for the bank account's bank.",
			Required:    true,
		},
		"naver_pay.funding": {
			Type:        "string",
			Description: "Whether to use Naver Pay points or a card to fund this transaction. If not provided, this defaults to `card`.",
			Enum: []resource.EnumSpec{
				{Value: "card"},
				{Value: "points"},
			},
		},
		"acss_debit.account_number": {
			Type:        "string",
			Description: "Customer's bank account number.",
			Required:    true,
		},
		"acss_debit.institution_number": {
			Type:        "string",
			Description: "Institution number of the customer's bank.",
			Required:    true,
		},
		"radar_options.session": {
			Type:        "string",
			Description: "A [Radar Session](https://docs.stripe.com/radar/radar-session) is a snapshot of the browser metadata and device details that help Radar make more accurate predictions on your payments.",
		},
		"au_becs_debit.bsb_number": {
			Type:        "string",
			Description: "Bank-State-Branch number of the bank account.",
			Required:    true,
		},
		"klarna.dob.day": {
			Type:        "integer",
			Description: "The day of birth, between 1 and 31.",
			Required:    true,
		},
		"allow_redisplay": {
			Type:        "string",
			Description: "This field indicates whether this payment method can be shown again to its customer in a checkout flow. Stripe products such as Checkout and Elements use this field to determine whether a payment method can be shown as a saved payment method in a checkout flow. The field defaults to `unspecified`.",
			Enum: []resource.EnumSpec{
				{Value: "always"},
				{Value: "limited"},
				{Value: "unspecified"},
			},
		},
		"payto.account_number": {
			Type:        "string",
			Description: "The account number for the bank account.",
		},
		"customer": {
			Type:        "string",
			Description: "The `Customer` to whom the original PaymentMethod is attached.",
		},
		"bacs_debit.sort_code": {
			Type:        "string",
			Description: "Sort code of the bank account. (e.g., `10-20-30`)",
		},
		"sofort.country": {
			Type:        "string",
			Description: "Two-letter ISO code representing the country the bank account is located in.",
			Required:    true,
			Enum: []resource.EnumSpec{
				{Value: "AT"},
				{Value: "BE"},
				{Value: "DE"},
				{Value: "ES"},
				{Value: "IT"},
				{Value: "NL"},
			},
		},
		"nz_bank_account.account_holder_name": {
			Type:        "string",
			Description: "The name on the bank account. Only required if the account holder name is different from the name of the authorized signatory collected in the PaymentMethod’s billing details.",
		},
	},
}

var V1PaymentMethodsUpdate = resource.OperationSpec{
	Name:    "update",
	Path:    "/v1/payment_methods/{payment_method}",
	Method:  "POST",
	Summary: "Update a PaymentMethod",
	Params: map[string]*resource.ParamSpec{
		"payto.account_number": {
			Type:        "string",
			Description: "The account number for the bank account.",
		},
		"us_bank_account.account_holder_type": {
			Type:        "string",
			Description: "Bank account holder type.",
			Enum: []resource.EnumSpec{
				{Value: "company"},
				{Value: "individual"},
			},
		},
		"us_bank_account.account_type": {
			Type:        "string",
			Description: "Bank account type.",
			Enum: []resource.EnumSpec{
				{Value: "checking"},
				{Value: "savings"},
			},
		},
		"billing_details.name": {
			Type:        "string",
			Description: "Full name.",
		},
		"billing_details.phone": {
			Type:        "string",
			Description: "Billing phone number (including extension).",
		},
		"billing_details.tax_id": {
			Type:        "string",
			Description: "Taxpayer identification number. Used only for transactions between LATAM buyers and non-LATAM sellers.",
		},
		"card.exp_year": {
			Type:        "integer",
			Description: "Four-digit number representing the card's expiration year.",
		},
		"card.networks.preferred": {
			Type:        "string",
			Description: "The customer's preferred card network for co-branded cards. Supports `cartes_bancaires`, `mastercard`, or `visa`. Selection of a network that does not apply to the card will be stored as `invalid_preference` on the card.",
			Enum: []resource.EnumSpec{
				{Value: "cartes_bancaires"},
				{Value: "mastercard"},
				{Value: "visa"},
			},
		},
		"payto.bsb_number": {
			Type:        "string",
			Description: "Bank-State-Branch number of the bank account.",
		},
		"payto.pay_id": {
			Type:        "string",
			Description: "The PayID alias for the bank account.",
		},
		"allow_redisplay": {
			Type:        "string",
			Description: "This field indicates whether this payment method can be shown again to its customer in a checkout flow. Stripe products such as Checkout and Elements use this field to determine whether a payment method can be shown as a saved payment method in a checkout flow. The field defaults to `unspecified`.",
			Enum: []resource.EnumSpec{
				{Value: "always"},
				{Value: "limited"},
				{Value: "unspecified"},
			},
		},
		"billing_details.email": {
			Type:        "string",
			Description: "Email address.",
		},
		"card.exp_month": {
			Type:        "integer",
			Description: "Two-digit number representing the card's expiration month.",
		},
	},
}

var V1PaymentMethodsAttach = resource.OperationSpec{
	Name:    "attach",
	Path:    "/v1/payment_methods/{payment_method}/attach",
	Method:  "POST",
	Summary: "Attach a PaymentMethod to a Customer",
	Params: map[string]*resource.ParamSpec{
		"customer_account": {
			Type:        "string",
			Description: "The ID of the Account representing the customer to which to attach the PaymentMethod.",
		},
		"customer": {
			Type:        "string",
			Description: "The ID of the customer to which to attach the PaymentMethod.",
		},
	},
}

var V1PaymentMethodsDetach = resource.OperationSpec{
	Name:    "detach",
	Path:    "/v1/payment_methods/{payment_method}/detach",
	Method:  "POST",
	Summary: "Detach a PaymentMethod from a Customer",
}

var V1CapabilitiesList = resource.OperationSpec{
	Name:    "list",
	Path:    "/v1/accounts/{account}/capabilities",
	Method:  "GET",
	Summary: "List all account capabilities",
}

var V1CapabilitiesRetrieve = resource.OperationSpec{
	Name:    "retrieve",
	Path:    "/v1/accounts/{account}/capabilities/{capability}",
	Method:  "GET",
	Summary: "Retrieve an Account Capability",
}

var V1CapabilitiesUpdate = resource.OperationSpec{
	Name:    "update",
	Path:    "/v1/accounts/{account}/capabilities/{capability}",
	Method:  "POST",
	Summary: "Update an Account Capability",
	Params: map[string]*resource.ParamSpec{
		"requested": {
			Type:        "boolean",
			Description: "To request a new capability for an account, pass true. There can be a delay before the requested capability becomes active. If the capability has any activation requirements, the response includes them in the `requirements` arrays.\n\nIf a capability isn't permanent, you can remove it from the account by passing false. Some capabilities are permanent after they've been requested. Attempting to remove a permanent capability returns an error.",
		},
	},
}

var V1ExchangeRatesList = resource.OperationSpec{
	Name:    "list",
	Path:    "/v1/exchange_rates",
	Method:  "GET",
	Summary: "List all exchange rates",
	Params: map[string]*resource.ParamSpec{
		"ending_before": {
			Type:        "string",
			Description: "A cursor for use in pagination. `ending_before` is the currency that defines your place in the list. For instance, if you make a list request and receive 100 objects, starting with the exchange rate for currency X your subsequent call can include `ending_before=obj_bar` in order to fetch the previous page of the list.",
		},
		"limit": {
			Type:        "integer",
			Description: "A limit on the number of objects to be returned. Limit can range between 1 and total number of supported payout currencies, and the default is the max.",
		},
		"starting_after": {
			Type:        "string",
			Description: "A cursor for use in pagination. `starting_after` is the currency that defines your place in the list. For instance, if you make a list request and receive 100 objects, ending with the exchange rate for currency X, your subsequent call can include `starting_after=X` in order to fetch the next page of the list.",
		},
	},
}

var V1ExchangeRatesRetrieve = resource.OperationSpec{
	Name:    "retrieve",
	Path:    "/v1/exchange_rates/{rate_id}",
	Method:  "GET",
	Summary: "Retrieve an exchange rate",
}

var V1ProductFeaturesDelete = resource.OperationSpec{
	Name:    "delete",
	Path:    "/v1/products/{product}/features/{id}",
	Method:  "DELETE",
	Summary: "Remove a feature from a product",
}

var V1ProductFeaturesList = resource.OperationSpec{
	Name:    "list",
	Path:    "/v1/products/{product}/features",
	Method:  "GET",
	Summary: "List all features attached to a product",
	Params: map[string]*resource.ParamSpec{
		"starting_after": {
			Type:        "string",
			Description: "A cursor for use in pagination. `starting_after` is an object ID that defines your place in the list. For instance, if you make a list request and receive 100 objects, ending with `obj_foo`, your subsequent call can include `starting_after=obj_foo` in order to fetch the next page of the list.",
		},
		"ending_before": {
			Type:        "string",
			Description: "A cursor for use in pagination. `ending_before` is an object ID that defines your place in the list. For instance, if you make a list request and receive 100 objects, starting with `obj_bar`, your subsequent call can include `ending_before=obj_bar` in order to fetch the previous page of the list.",
		},
		"limit": {
			Type:        "integer",
			Description: "A limit on the number of objects to be returned. Limit can range between 1 and 100, and the default is 10.",
		},
	},
}

var V1ProductFeaturesRetrieve = resource.OperationSpec{
	Name:    "retrieve",
	Path:    "/v1/products/{product}/features/{id}",
	Method:  "GET",
	Summary: "Retrieve a product_feature",
}

var V1ProductFeaturesCreate = resource.OperationSpec{
	Name:    "create",
	Path:    "/v1/products/{product}/features",
	Method:  "POST",
	Summary: "Attach a feature to a product",
	Params: map[string]*resource.ParamSpec{
		"entitlement_feature": {
			Type:        "string",
			Description: "The ID of the [Feature](https://docs.stripe.com/api/entitlements/feature) object attached to this product.",
			Required:    true,
		},
	},
}

var V1ProductsSearch = resource.OperationSpec{
	Name:    "search",
	Path:    "/v1/products/search",
	Method:  "GET",
	Summary: "Search products",
	Params: map[string]*resource.ParamSpec{
		"page": {
			Type:        "string",
			Description: "A cursor for pagination across multiple pages of results. Don't include this parameter on the first call. Use the next_page value returned in a previous response to request subsequent results.",
		},
		"query": {
			Type:        "string",
			Description: "The search query string. See [search query language](https://docs.stripe.com/search#search-query-language) and the list of supported [query fields for products](https://docs.stripe.com/search#query-fields-for-products).",
			Required:    true,
		},
		"limit": {
			Type:        "integer",
			Description: "A limit on the number of objects to be returned. Limit can range between 1 and 100, and the default is 10.",
		},
	},
}

var V1ProductsCreate = resource.OperationSpec{
	Name:    "create",
	Path:    "/v1/products",
	Method:  "POST",
	Summary: "Create a product",
	Params: map[string]*resource.ParamSpec{
		"images": {
			Type:        "array",
			Description: "A list of up to 8 URLs of images for this product, meant to be displayable to the customer.",
		},
		"default_price_data.tax_behavior": {
			Type:        "string",
			Description: "Only required if a [default tax behavior](https://docs.stripe.com/tax/products-prices-tax-categories-tax-behavior#setting-a-default-tax-behavior-(recommended)) was not provided in the Stripe Tax settings. Specifies whether the price is considered inclusive of taxes or exclusive of taxes. One of `inclusive`, `exclusive`, or `unspecified`. Once specified as either `inclusive` or `exclusive`, it cannot be changed.",
			Enum: []resource.EnumSpec{
				{Value: "exclusive"},
				{Value: "inclusive"},
				{Value: "unspecified"},
			},
		},
		"default_price_data.unit_amount": {
			Type:        "integer",
			Description: "A positive integer in cents (or local equivalent) (or 0 for a free price) representing how much to charge. One of `unit_amount`, `unit_amount_decimal`, or `custom_unit_amount` is required.",
		},
		"package_dimensions.height": {
			Type:        "number",
			Description: "Height, in inches. Maximum precision is 2 decimal places.",
			Required:    true,
		},
		"statement_descriptor": {
			Type:        "string",
			Description: "An arbitrary string to be displayed on your customer's credit card or bank statement. While most banks display this information consistently, some may display it incorrectly or not at all.\n\nThis may be up to 22 characters. The statement description may not include `<`, `>`, `\\`, `\"`, `'` characters, and will appear on your customer's statement in capital letters. Non-ASCII characters are automatically stripped.\n It must contain at least one letter. Only used for subscription payments.",
		},
		"description": {
			Type:        "string",
			Description: "The product's description, meant to be displayable to the customer. Use this field to optionally store a long form explanation of the product being sold for your own rendering purposes.",
		},
		"active": {
			Type:        "boolean",
			Description: "Whether the product is currently available for purchase. Defaults to `true`.",
		},
		"default_price_data.custom_unit_amount.preset": {
			Type:        "integer",
			Description: "The starting unit amount which can be updated by the customer.",
		},
		"default_price_data.recurring.interval_count": {
			Type:        "integer",
			Description: "The number of intervals between subscription billings. For example, `interval=month` and `interval_count=3` bills every 3 months. Maximum of three years interval allowed (3 years, 36 months, or 156 weeks).",
		},
		"id": {
			Type:        "string",
			Description: "An identifier will be randomly generated by Stripe. You can optionally override this ID, but the ID must be unique across all products in your Stripe account.",
		},
		"type": {
			Type:        "string",
			Description: "The type of the product. Defaults to `service` if not explicitly specified, enabling use of this product with Subscriptions and Plans. Set this parameter to `good` to use this product with Orders and SKUs. On API versions before `2018-02-05`, this field defaults to `good` for compatibility reasons.",
			Enum: []resource.EnumSpec{
				{Value: "good"},
				{Value: "service"},
			},
		},
		"unit_label": {
			Type:        "string",
			Description: "A label that represents units of this product. When set, this will be included in customers' receipts, invoices, Checkout, and the customer portal.",
		},
		"tax_code": {
			Type:        "string",
			Description: "A [tax code](https://docs.stripe.com/tax/tax-categories) ID.",
		},
		"name": {
			Type:        "string",
			Description: "The product's name, meant to be displayable to the customer.",
			Required:    true,
		},
		"default_price_data.custom_unit_amount.enabled": {
			Type:        "boolean",
			Description: "Pass in `true` to enable `custom_unit_amount`, otherwise omit `custom_unit_amount`.",
			Required:    true,
		},
		"default_price_data.recurring.interval": {
			Type:        "string",
			Description: "Specifies billing frequency. Either `day`, `week`, `month` or `year`.",
			Required:    true,
			Enum: []resource.EnumSpec{
				{Value: "day"},
				{Value: "month"},
				{Value: "week"},
				{Value: "year"},
			},
		},
		"package_dimensions.length": {
			Type:        "number",
			Description: "Length, in inches. Maximum precision is 2 decimal places.",
			Required:    true,
		},
		"url": {
			Type:        "string",
			Description: "A URL of a publicly-accessible webpage for this product.",
		},
		"default_price_data.unit_amount_decimal": {
			Type:        "string",
			Description: "Same as `unit_amount`, but accepts a decimal value in cents (or local equivalent) with at most 12 decimal places. Only one of `unit_amount` and `unit_amount_decimal` can be set.",
			Format:      "decimal",
		},
		"default_price_data.currency": {
			Type:        "string",
			Description: "Three-letter [ISO currency code](https://www.iso.org/iso-4217-currency-codes.html), in lowercase. Must be a [supported currency](https://stripe.com/docs/currencies).",
			Required:    true,
			Format:      "currency",
		},
		"default_price_data.custom_unit_amount.maximum": {
			Type:        "integer",
			Description: "The maximum unit amount the customer can specify for this item.",
		},
		"default_price_data.custom_unit_amount.minimum": {
			Type:        "integer",
			Description: "The minimum unit amount the customer can specify for this item. Must be at least the minimum charge amount.",
		},
		"package_dimensions.weight": {
			Type:        "number",
			Description: "Weight, in ounces. Maximum precision is 2 decimal places.",
			Required:    true,
		},
		"package_dimensions.width": {
			Type:        "number",
			Description: "Width, in inches. Maximum precision is 2 decimal places.",
			Required:    true,
		},
		"shippable": {
			Type:        "boolean",
			Description: "Whether this product is shipped (i.e., physical goods).",
		},
	},
}

var V1ProductsUpdate = resource.OperationSpec{
	Name:    "update",
	Path:    "/v1/products/{id}",
	Method:  "POST",
	Summary: "Update a product",
	Params: map[string]*resource.ParamSpec{
		"description": {
			Type:        "string",
			Description: "The product's description, meant to be displayable to the customer. Use this field to optionally store a long form explanation of the product being sold for your own rendering purposes.",
		},
		"shippable": {
			Type:        "boolean",
			Description: "Whether this product is shipped (i.e., physical goods).",
		},
		"url": {
			Type:        "string",
			Description: "A URL of a publicly-accessible webpage for this product.",
		},
		"active": {
			Type:        "boolean",
			Description: "Whether the product is available for purchase.",
		},
		"statement_descriptor": {
			Type:        "string",
			Description: "An arbitrary string to be displayed on your customer's credit card or bank statement. While most banks display this information consistently, some may display it incorrectly or not at all.\n\nThis may be up to 22 characters. The statement description may not include `<`, `>`, `\\`, `\"`, `'` characters, and will appear on your customer's statement in capital letters. Non-ASCII characters are automatically stripped.\n It must contain at least one letter. May only be set if `type=service`. Only used for subscription payments.",
		},
		"name": {
			Type:        "string",
			Description: "The product's name, meant to be displayable to the customer.",
		},
		"tax_code": {
			Type:        "string",
			Description: "A [tax code](https://docs.stripe.com/tax/tax-categories) ID.",
		},
		"images": {
			Type:        "array",
			Description: "A list of up to 8 URLs of images for this product, meant to be displayable to the customer.",
		},
		"unit_label": {
			Type:        "string",
			Description: "A label that represents units of this product. When set, this will be included in customers' receipts, invoices, Checkout, and the customer portal. May only be set if `type=service`.",
		},
		"default_price": {
			Type:        "string",
			Description: "The ID of the [Price](https://docs.stripe.com/api/prices) object that is the default price for this product.",
		},
	},
}

var V1ProductsDelete = resource.OperationSpec{
	Name:    "delete",
	Path:    "/v1/products/{id}",
	Method:  "DELETE",
	Summary: "Delete a product",
}

var V1ProductsList = resource.OperationSpec{
	Name:    "list",
	Path:    "/v1/products",
	Method:  "GET",
	Summary: "List all products",
	Params: map[string]*resource.ParamSpec{
		"created": {
			Type:        "integer",
			Description: "Only return products that were created during the given date interval.",
		},
		"ending_before": {
			Type:        "string",
			Description: "A cursor for use in pagination. `ending_before` is an object ID that defines your place in the list. For instance, if you make a list request and receive 100 objects, starting with `obj_bar`, your subsequent call can include `ending_before=obj_bar` in order to fetch the previous page of the list.",
		},
		"ids": {
			Type:        "array",
			Description: "Only return products with the given IDs. Cannot be used with [starting_after](https://api.stripe.com#list_products-starting_after) or [ending_before](https://api.stripe.com#list_products-ending_before).",
		},
		"shippable": {
			Type:        "boolean",
			Description: "Only return products that can be shipped (i.e., physical, not digital products).",
		},
		"starting_after": {
			Type:        "string",
			Description: "A cursor for use in pagination. `starting_after` is an object ID that defines your place in the list. For instance, if you make a list request and receive 100 objects, ending with `obj_foo`, your subsequent call can include `starting_after=obj_foo` in order to fetch the next page of the list.",
		},
		"type": {
			Type:        "string",
			Description: "Only return products of this type.",
			Enum: []resource.EnumSpec{
				{Value: "good"},
				{Value: "service"},
			},
		},
		"url": {
			Type:        "string",
			Description: "Only return products with the given url.",
		},
		"active": {
			Type:        "boolean",
			Description: "Only return products that are active or inactive (e.g., pass `false` to list all inactive products).",
		},
		"limit": {
			Type:        "integer",
			Description: "A limit on the number of objects to be returned. Limit can range between 1 and 100, and the default is 10.",
		},
	},
}

var V1ProductsRetrieve = resource.OperationSpec{
	Name:    "retrieve",
	Path:    "/v1/products/{id}",
	Method:  "GET",
	Summary: "Retrieve a product",
}

var V1TaxRatesList = resource.OperationSpec{
	Name:    "list",
	Path:    "/v1/tax_rates",
	Method:  "GET",
	Summary: "List all tax rates",
	Params: map[string]*resource.ParamSpec{
		"active": {
			Type:        "boolean",
			Description: "Optional flag to filter by tax rates that are either active or inactive (archived).",
		},
		"created": {
			Type:        "integer",
			Description: "Optional range for filtering created date.",
		},
		"ending_before": {
			Type:        "string",
			Description: "A cursor for use in pagination. `ending_before` is an object ID that defines your place in the list. For instance, if you make a list request and receive 100 objects, starting with `obj_bar`, your subsequent call can include `ending_before=obj_bar` in order to fetch the previous page of the list.",
		},
		"inclusive": {
			Type:        "boolean",
			Description: "Optional flag to filter by tax rates that are inclusive (or those that are not inclusive).",
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

var V1TaxRatesRetrieve = resource.OperationSpec{
	Name:    "retrieve",
	Path:    "/v1/tax_rates/{tax_rate}",
	Method:  "GET",
	Summary: "Retrieve a tax rate",
}

var V1TaxRatesCreate = resource.OperationSpec{
	Name:    "create",
	Path:    "/v1/tax_rates",
	Method:  "POST",
	Summary: "Create a tax rate",
	Params: map[string]*resource.ParamSpec{
		"description": {
			Type:        "string",
			Description: "An arbitrary string attached to the tax rate for your internal use only. It will not be visible to your customers.",
		},
		"inclusive": {
			Type:        "boolean",
			Description: "This specifies if the tax rate is inclusive or exclusive.",
			Required:    true,
		},
		"jurisdiction": {
			Type:        "string",
			Description: "The jurisdiction for the tax rate. You can use this label field for tax reporting purposes. It also appears on your customer’s invoice.",
		},
		"display_name": {
			Type:        "string",
			Description: "The display name of the tax rate, which will be shown to users.",
			Required:    true,
		},
		"tax_type": {
			Type:        "string",
			Description: "The high-level tax type, such as `vat` or `sales_tax`.",
			Enum: []resource.EnumSpec{
				{Value: "amusement_tax"},
				{Value: "communications_tax"},
				{Value: "gst"},
				{Value: "hst"},
				{Value: "igst"},
				{Value: "jct"},
				{Value: "lease_tax"},
				{Value: "pst"},
				{Value: "qst"},
				{Value: "retail_delivery_fee"},
				{Value: "rst"},
				{Value: "sales_tax"},
				{Value: "service_tax"},
				{Value: "vat"},
			},
		},
		"country": {
			Type:        "string",
			Description: "Two-letter country code ([ISO 3166-1 alpha-2](https://en.wikipedia.org/wiki/ISO_3166-1_alpha-2)).",
		},
		"percentage": {
			Type:        "number",
			Description: "This represents the tax rate percent out of 100.",
			Required:    true,
		},
		"state": {
			Type:        "string",
			Description: "[ISO 3166-2 subdivision code](https://en.wikipedia.org/wiki/ISO_3166-2), without country prefix. For example, \"NY\" for New York, United States.",
		},
		"active": {
			Type:        "boolean",
			Description: "Flag determining whether the tax rate is active or inactive (archived). Inactive tax rates cannot be used with new applications or Checkout Sessions, but will still work for subscriptions and invoices that already have it set.",
		},
	},
}

var V1TaxRatesUpdate = resource.OperationSpec{
	Name:    "update",
	Path:    "/v1/tax_rates/{tax_rate}",
	Method:  "POST",
	Summary: "Update a tax rate",
	Params: map[string]*resource.ParamSpec{
		"state": {
			Type:        "string",
			Description: "[ISO 3166-2 subdivision code](https://en.wikipedia.org/wiki/ISO_3166-2), without country prefix. For example, \"NY\" for New York, United States.",
		},
		"active": {
			Type:        "boolean",
			Description: "Flag determining whether the tax rate is active or inactive (archived). Inactive tax rates cannot be used with new applications or Checkout Sessions, but will still work for subscriptions and invoices that already have it set.",
		},
		"jurisdiction": {
			Type:        "string",
			Description: "The jurisdiction for the tax rate. You can use this label field for tax reporting purposes. It also appears on your customer’s invoice.",
		},
		"tax_type": {
			Type:        "string",
			Description: "The high-level tax type, such as `vat` or `sales_tax`.",
			Enum: []resource.EnumSpec{
				{Value: "amusement_tax"},
				{Value: "communications_tax"},
				{Value: "gst"},
				{Value: "hst"},
				{Value: "igst"},
				{Value: "jct"},
				{Value: "lease_tax"},
				{Value: "pst"},
				{Value: "qst"},
				{Value: "retail_delivery_fee"},
				{Value: "rst"},
				{Value: "sales_tax"},
				{Value: "service_tax"},
				{Value: "vat"},
			},
		},
		"country": {
			Type:        "string",
			Description: "Two-letter country code ([ISO 3166-1 alpha-2](https://en.wikipedia.org/wiki/ISO_3166-1_alpha-2)).",
		},
		"description": {
			Type:        "string",
			Description: "An arbitrary string attached to the tax rate for your internal use only. It will not be visible to your customers.",
		},
		"display_name": {
			Type:        "string",
			Description: "The display name of the tax rate, which will be shown to users.",
		},
	},
}

var V1InvoicesSearch = resource.OperationSpec{
	Name:    "search",
	Path:    "/v1/invoices/search",
	Method:  "GET",
	Summary: "Search invoices",
	Params: map[string]*resource.ParamSpec{
		"limit": {
			Type:        "integer",
			Description: "A limit on the number of objects to be returned. Limit can range between 1 and 100, and the default is 10.",
		},
		"page": {
			Type:        "string",
			Description: "A cursor for pagination across multiple pages of results. Don't include this parameter on the first call. Use the next_page value returned in a previous response to request subsequent results.",
		},
		"query": {
			Type:        "string",
			Description: "The search query string. See [search query language](https://docs.stripe.com/search#search-query-language) and the list of supported [query fields for invoices](https://docs.stripe.com/search#query-fields-for-invoices).",
			Required:    true,
		},
	},
}

var V1InvoicesUpdate = resource.OperationSpec{
	Name:    "update",
	Path:    "/v1/invoices/{invoice}",
	Method:  "POST",
	Summary: "Update an invoice",
	Params: map[string]*resource.ParamSpec{
		"footer": {
			Type:        "string",
			Description: "Footer to be displayed on the invoice.",
		},
		"default_payment_method": {
			Type:        "string",
			Description: "ID of the default payment method for the invoice. It must belong to the customer associated with the invoice. If not set, defaults to the subscription's default payment method, if any, or to the default payment method in the customer's invoice settings.",
		},
		"on_behalf_of": {
			Type:        "string",
			Description: "The account (if any) for which the funds of the invoice payment are intended. If set, the invoice will be presented with the branding and support information of the specified account. See the [Invoices with Connect](https://docs.stripe.com/billing/invoices/connect) documentation for details.",
		},
		"payment_settings.payment_method_types": {
			Type:        "array",
			Description: "The list of payment method types (e.g. card) to provide to the invoice’s PaymentIntent. If not set, Stripe attempts to automatically determine the types to use by looking at the invoice’s default payment method, the subscription’s default payment method, the customer’s default payment method, and your [invoice template settings](https://dashboard.stripe.com/settings/billing/invoice).",
		},
		"due_date": {
			Type:        "integer",
			Description: "The date on which payment for this invoice is due. Only valid for invoices where `collection_method=send_invoice`. This field can only be updated on `draft` invoices.",
			Format:      "unix-time",
		},
		"automatic_tax.liability.account": {
			Type:        "string",
			Description: "The connected account being referenced when `type` is `account`.",
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
		"effective_at": {
			Type:        "integer",
			Description: "The date when this invoice is in effect. Same as `finalized_at` unless overwritten. When defined, this value replaces the system-generated 'Date of issue' printed on the invoice PDF and receipt.",
		},
		"automatically_finalizes_at": {
			Type:        "integer",
			Description: "The time when this invoice should be scheduled to finalize (up to 5 years in the future). The invoice is finalized at this time if it's still in draft state. To turn off automatic finalization, set `auto_advance` to false.",
			Format:      "unix-time",
		},
		"days_until_due": {
			Type:        "integer",
			Description: "The number of days from which the invoice is created until it is due. Only valid for invoices where `collection_method=send_invoice`. This field can only be updated on `draft` invoices.",
		},
		"default_tax_rates": {
			Type:        "array",
			Description: "The tax rates that will apply to any line item that does not have `tax_rates` set. Pass an empty string to remove previously-defined tax rates.",
		},
		"account_tax_ids": {
			Type:        "array",
			Description: "The account tax IDs associated with the invoice. Only editable when the invoice is a draft.",
		},
		"collection_method": {
			Type:        "string",
			Description: "Either `charge_automatically` or `send_invoice`. This field can be updated only on `draft` invoices.",
			Enum: []resource.EnumSpec{
				{Value: "charge_automatically"},
				{Value: "send_invoice"},
			},
		},
		"payment_settings.default_mandate": {
			Type:        "string",
			Description: "ID of the mandate to be used for this invoice. It must correspond to the payment method used to pay the invoice, including the invoice's default_payment_method or default_source, if set.",
		},
		"rendering.pdf.page_size": {
			Type:        "string",
			Description: "Page size for invoice PDF. Can be set to `a4`, `letter`, or `auto`.\n If set to `auto`, invoice PDF page size defaults to `a4` for customers with\n Japanese locale and `letter` for customers with other locales.",
			Enum: []resource.EnumSpec{
				{Value: "a4"},
				{Value: "auto"},
				{Value: "letter"},
			},
		},
		"issuer.account": {
			Type:        "string",
			Description: "The connected account being referenced when `type` is `account`.",
		},
		"issuer.type": {
			Type:        "string",
			Description: "Type of the account referenced in the request.",
			Required:    true,
			Enum: []resource.EnumSpec{
				{Value: "account"},
				{Value: "self"},
			},
		},
		"statement_descriptor": {
			Type:        "string",
			Description: "Extra information about a charge for the customer's credit card statement. It must contain at least one letter. If not specified and this invoice is part of a subscription, the default `statement_descriptor` will be set to the first subscription item's product's `statement_descriptor`.",
		},
		"application_fee_amount": {
			Type:        "integer",
			Description: "A fee in cents (or local equivalent) that will be applied to the invoice and transferred to the application owner's Stripe account. The request must be made with an OAuth key or the Stripe-Account header in order to take an application fee. For more information, see the application fees [documentation](https://docs.stripe.com/billing/invoices/connect#collecting-fees).",
		},
		"auto_advance": {
			Type:        "boolean",
			Description: "Controls whether Stripe performs [automatic collection](https://docs.stripe.com/invoicing/integration/automatic-advancement-collection) of the invoice.",
		},
		"default_source": {
			Type:        "string",
			Description: "ID of the default payment source for the invoice. It must belong to the customer associated with the invoice and be in a chargeable state. If not set, defaults to the subscription's default source, if any, or to the customer's default source.",
		},
		"rendering.amount_tax_display": {
			Type:        "string",
			Description: "How line-item prices and amounts will be displayed with respect to tax on invoice PDFs. One of `exclude_tax` or `include_inclusive_tax`. `include_inclusive_tax` will include inclusive tax (and exclude exclusive tax) in invoice PDF amounts. `exclude_tax` will exclude all tax (inclusive and exclusive alike) from invoice PDF amounts.",
			Enum: []resource.EnumSpec{
				{Value: "exclude_tax"},
				{Value: "include_inclusive_tax"},
			},
		},
		"rendering.template": {
			Type:        "string",
			Description: "ID of the invoice rendering template to use for this invoice.",
		},
		"number": {
			Type:        "string",
			Description: "Set the number for this invoice. If no number is present then a number will be assigned automatically when the invoice is finalized. In many markets, regulations require invoices to be unique, sequential and / or gapless. You are responsible for ensuring this is true across all your different invoicing systems in the event that you edit the invoice number using our API. If you use only Stripe for your invoices and do not change invoice numbers, Stripe handles this aspect of compliance for you automatically.",
		},
		"rendering.template_version": {
			Type:        "integer",
			Description: "The specific version of invoice rendering template to use for this invoice.",
		},
		"automatic_tax.enabled": {
			Type:        "boolean",
			Description: "Whether Stripe automatically computes tax on this invoice. Note that incompatible invoice items (invoice items with manually specified [tax rates](https://docs.stripe.com/api/tax_rates), negative amounts, or `tax_behavior=unspecified`) cannot be added to automatic tax invoices.",
			Required:    true,
		},
		"description": {
			Type:        "string",
			Description: "An arbitrary string attached to the object. Often useful for displaying to users. Referenced as 'memo' in the Dashboard.",
		},
	},
}

var V1InvoicesList = resource.OperationSpec{
	Name:    "list",
	Path:    "/v1/invoices",
	Method:  "GET",
	Summary: "List all invoices",
	Params: map[string]*resource.ParamSpec{
		"collection_method": {
			Type:        "string",
			Description: "The collection method of the invoice to retrieve. Either `charge_automatically` or `send_invoice`.",
			Enum: []resource.EnumSpec{
				{Value: "charge_automatically"},
				{Value: "send_invoice"},
			},
		},
		"customer": {
			Type:        "string",
			Description: "Only return invoices for the customer specified by this customer ID.",
		},
		"ending_before": {
			Type:        "string",
			Description: "A cursor for use in pagination. `ending_before` is an object ID that defines your place in the list. For instance, if you make a list request and receive 100 objects, starting with `obj_bar`, your subsequent call can include `ending_before=obj_bar` in order to fetch the previous page of the list.",
		},
		"status": {
			Type:        "string",
			Description: "The status of the invoice, one of `draft`, `open`, `paid`, `uncollectible`, or `void`. [Learn more](https://docs.stripe.com/billing/invoices/workflow#workflow-overview)",
			Enum: []resource.EnumSpec{
				{Value: "draft"},
				{Value: "open"},
				{Value: "paid"},
				{Value: "uncollectible"},
				{Value: "void"},
			},
		},
		"subscription": {
			Type:        "string",
			Description: "Only return invoices for the subscription specified by this subscription ID.",
		},
		"created": {
			Type:        "integer",
			Description: "Only return invoices that were created during the given date interval.",
		},
		"customer_account": {
			Type:        "string",
			Description: "Only return invoices for the account representing the customer specified by this account ID.",
		},
		"due_date": {
			Type: "integer",
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

var V1InvoicesPay = resource.OperationSpec{
	Name:    "pay",
	Path:    "/v1/invoices/{invoice}/pay",
	Method:  "POST",
	Summary: "Pay an invoice",
	Params: map[string]*resource.ParamSpec{
		"off_session": {
			Type:        "boolean",
			Description: "Indicates if a customer is on or off-session while an invoice payment is attempted. Defaults to `true` (off-session).",
		},
		"paid_out_of_band": {
			Type:        "boolean",
			Description: "Boolean representing whether an invoice is paid outside of Stripe. This will result in no charge being made. Defaults to `false`.",
		},
		"payment_method": {
			Type:        "string",
			Description: "A PaymentMethod to be charged. The PaymentMethod must be the ID of a PaymentMethod belonging to the customer associated with the invoice being paid.",
		},
		"source": {
			Type:        "string",
			Description: "A payment source to be charged. The source must be the ID of a source belonging to the customer associated with the invoice being paid.",
		},
		"forgive": {
			Type:        "boolean",
			Description: "In cases where the source used to pay the invoice has insufficient funds, passing `forgive=true` controls whether a charge should be attempted for the full amount available on the source, up to the amount to fully pay the invoice. This effectively forgives the difference between the amount available on the source and the amount due. \n\nPassing `forgive=false` will fail the charge if the source hasn't been pre-funded with the right amount. An example for this case is with ACH Credit Transfers and wires: if the amount wired is less than the amount due by a small amount, you might want to forgive the difference. Defaults to `false`.",
		},
		"mandate": {
			Type:        "string",
			Description: "ID of the mandate to be used for this invoice. It must correspond to the payment method used to pay the invoice, including the payment_method param or the invoice's default_payment_method or default_source, if set.",
		},
	},
}

var V1InvoicesUpdateLines = resource.OperationSpec{
	Name:    "update_lines",
	Path:    "/v1/invoices/{invoice}/update_lines",
	Method:  "POST",
	Summary: "Bulk update invoice line items",
}

var V1InvoicesDelete = resource.OperationSpec{
	Name:    "delete",
	Path:    "/v1/invoices/{invoice}",
	Method:  "DELETE",
	Summary: "Delete a draft invoice",
}

var V1InvoicesRetrieve = resource.OperationSpec{
	Name:    "retrieve",
	Path:    "/v1/invoices/{invoice}",
	Method:  "GET",
	Summary: "Retrieve an invoice",
}

var V1InvoicesCreate = resource.OperationSpec{
	Name:    "create",
	Path:    "/v1/invoices",
	Method:  "POST",
	Summary: "Create an invoice",
	Params: map[string]*resource.ParamSpec{
		"default_tax_rates": {
			Type:        "array",
			Description: "The tax rates that will apply to any line item that does not have `tax_rates` set.",
		},
		"payment_settings.default_mandate": {
			Type:        "string",
			Description: "ID of the mandate to be used for this invoice. It must correspond to the payment method used to pay the invoice, including the invoice's default_payment_method or default_source, if set.",
		},
		"rendering.template": {
			Type:        "string",
			Description: "ID of the invoice rendering template to use for this invoice.",
		},
		"days_until_due": {
			Type:        "integer",
			Description: "The number of days from when the invoice is created until it is due. Valid only for invoices where `collection_method=send_invoice`.",
		},
		"due_date": {
			Type:        "integer",
			Description: "The date on which payment for this invoice is due. Valid only for invoices where `collection_method=send_invoice`.",
			Format:      "unix-time",
		},
		"issuer.type": {
			Type:        "string",
			Description: "Type of the account referenced in the request.",
			Required:    true,
			Enum: []resource.EnumSpec{
				{Value: "account"},
				{Value: "self"},
			},
		},
		"pending_invoice_items_behavior": {
			Type:        "string",
			Description: "How to handle pending invoice items on invoice creation. Defaults to `exclude` if the parameter is omitted.",
			Enum: []resource.EnumSpec{
				{Value: "exclude"},
				{Value: "include"},
			},
		},
		"account_tax_ids": {
			Type:        "array",
			Description: "The account tax IDs associated with the invoice. Only editable when the invoice is a draft.",
		},
		"transfer_data.amount": {
			Type:        "integer",
			Description: "The amount that will be transferred automatically when the invoice is paid. If no amount is set, the full amount is transferred.",
		},
		"currency": {
			Type:        "string",
			Description: "The currency to create this invoice in. Defaults to that of `customer` if not specified.",
			Format:      "currency",
		},
		"shipping_cost.shipping_rate_data.fixed_amount.currency": {
			Type:        "string",
			Description: "Three-letter [ISO currency code](https://www.iso.org/iso-4217-currency-codes.html), in lowercase. Must be a [supported currency](https://stripe.com/docs/currencies).",
			Required:    true,
			Format:      "currency",
		},
		"shipping_cost.shipping_rate_data.tax_behavior": {
			Type:        "string",
			Description: "Specifies whether the rate is considered inclusive of taxes or exclusive of taxes. One of `inclusive`, `exclusive`, or `unspecified`.",
			Enum: []resource.EnumSpec{
				{Value: "exclusive"},
				{Value: "inclusive"},
				{Value: "unspecified"},
			},
		},
		"rendering.pdf.page_size": {
			Type:        "string",
			Description: "Page size for invoice PDF. Can be set to `a4`, `letter`, or `auto`.\n If set to `auto`, invoice PDF page size defaults to `a4` for customers with\n Japanese locale and `letter` for customers with other locales.",
			Enum: []resource.EnumSpec{
				{Value: "a4"},
				{Value: "auto"},
				{Value: "letter"},
			},
		},
		"shipping_cost.shipping_rate": {
			Type:        "string",
			Description: "The ID of the shipping rate to use for this order.",
		},
		"shipping_details.address.city": {
			Type:        "string",
			Description: "City, district, suburb, town, or village.",
		},
		"description": {
			Type:        "string",
			Description: "An arbitrary string attached to the object. Often useful for displaying to users. Referenced as 'memo' in the Dashboard.",
		},
		"on_behalf_of": {
			Type:        "string",
			Description: "The account (if any) for which the funds of the invoice payment are intended. If set, the invoice will be presented with the branding and support information of the specified account. See the [Invoices with Connect](https://docs.stripe.com/billing/invoices/connect) documentation for details.",
		},
		"shipping_details.address.postal_code": {
			Type:        "string",
			Description: "ZIP or postal code.",
		},
		"subscription": {
			Type:        "string",
			Description: "The ID of the subscription to invoice, if any. If set, the created invoice will only include pending invoice items for that subscription. The subscription's billing cycle and regular subscription events won't be affected.",
		},
		"automatic_tax.liability.account": {
			Type:        "string",
			Description: "The connected account being referenced when `type` is `account`.",
		},
		"customer_account": {
			Type:        "string",
			Description: "The ID of the account to bill.",
		},
		"transfer_data.destination": {
			Type:        "string",
			Description: "ID of an existing, connected Stripe account.",
			Required:    true,
		},
		"shipping_details.address.state": {
			Type:        "string",
			Description: "State, county, province, or region ([ISO 3166-2](https://en.wikipedia.org/wiki/ISO_3166-2)).",
		},
		"shipping_details.name": {
			Type:        "string",
			Description: "Recipient name.",
			Required:    true,
		},
		"shipping_details.phone": {
			Type:        "string",
			Description: "Recipient phone (including extension)",
		},
		"automatically_finalizes_at": {
			Type:        "integer",
			Description: "The time when this invoice should be scheduled to finalize (up to 5 years in the future). The invoice is finalized at this time if it's still in draft state.",
			Format:      "unix-time",
		},
		"shipping_cost.shipping_rate_data.tax_code": {
			Type:        "string",
			Description: "A [tax code](https://docs.stripe.com/tax/tax-categories) ID. The Shipping tax code is `txcd_92010001`.",
		},
		"shipping_cost.shipping_rate_data.delivery_estimate.maximum.unit": {
			Type:        "string",
			Description: "A unit of time.",
			Required:    true,
			Enum: []resource.EnumSpec{
				{Value: "business_day"},
				{Value: "day"},
				{Value: "hour"},
				{Value: "month"},
				{Value: "week"},
			},
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
		"application_fee_amount": {
			Type:        "integer",
			Description: "A fee in cents (or local equivalent) that will be applied to the invoice and transferred to the application owner's Stripe account. The request must be made with an OAuth key or the Stripe-Account header in order to take an application fee. For more information, see the application fees [documentation](https://docs.stripe.com/billing/invoices/connect#collecting-fees).",
		},
		"shipping_details.address.line1": {
			Type:        "string",
			Description: "Address line 1, such as the street, PO Box, or company name.",
		},
		"collection_method": {
			Type:        "string",
			Description: "Either `charge_automatically`, or `send_invoice`. When charging automatically, Stripe will attempt to pay this invoice using the default source attached to the customer. When sending an invoice, Stripe will email this invoice to the customer with payment instructions. Defaults to `charge_automatically`.",
			Enum: []resource.EnumSpec{
				{Value: "charge_automatically"},
				{Value: "send_invoice"},
			},
		},
		"shipping_cost.shipping_rate_data.display_name": {
			Type:        "string",
			Description: "The name of the shipping rate, meant to be displayable to the customer. This will appear on CheckoutSessions.",
			Required:    true,
		},
		"shipping_cost.shipping_rate_data.fixed_amount.amount": {
			Type:        "integer",
			Description: "A non-negative integer in cents representing how much to charge.",
			Required:    true,
		},
		"automatic_tax.enabled": {
			Type:        "boolean",
			Description: "Whether Stripe automatically computes tax on this invoice. Note that incompatible invoice items (invoice items with manually specified [tax rates](https://docs.stripe.com/api/tax_rates), negative amounts, or `tax_behavior=unspecified`) cannot be added to automatic tax invoices.",
			Required:    true,
		},
		"rendering.template_version": {
			Type:        "integer",
			Description: "The specific version of invoice rendering template to use for this invoice.",
		},
		"statement_descriptor": {
			Type:        "string",
			Description: "Extra information about a charge for the customer's credit card statement. It must contain at least one letter. If not specified and this invoice is part of a subscription, the default `statement_descriptor` will be set to the first subscription item's product's `statement_descriptor`.",
		},
		"shipping_cost.shipping_rate_data.delivery_estimate.maximum.value": {
			Type:        "integer",
			Description: "Must be greater than 0.",
			Required:    true,
		},
		"shipping_cost.shipping_rate_data.delivery_estimate.minimum.unit": {
			Type:        "string",
			Description: "A unit of time.",
			Required:    true,
			Enum: []resource.EnumSpec{
				{Value: "business_day"},
				{Value: "day"},
				{Value: "hour"},
				{Value: "month"},
				{Value: "week"},
			},
		},
		"rendering.amount_tax_display": {
			Type:        "string",
			Description: "How line-item prices and amounts will be displayed with respect to tax on invoice PDFs. One of `exclude_tax` or `include_inclusive_tax`. `include_inclusive_tax` will include inclusive tax (and exclude exclusive tax) in invoice PDF amounts. `exclude_tax` will exclude all tax (inclusive and exclusive alike) from invoice PDF amounts.",
			Enum: []resource.EnumSpec{
				{Value: "exclude_tax"},
				{Value: "include_inclusive_tax"},
			},
		},
		"footer": {
			Type:        "string",
			Description: "Footer to be displayed on the invoice.",
		},
		"auto_advance": {
			Type:        "boolean",
			Description: "Controls whether Stripe performs [automatic collection](https://docs.stripe.com/invoicing/integration/automatic-advancement-collection) of the invoice. If `false`, the invoice's state doesn't automatically advance without an explicit action. Defaults to false.",
		},
		"default_payment_method": {
			Type:        "string",
			Description: "ID of the default payment method for the invoice. It must belong to the customer associated with the invoice. If not set, defaults to the subscription's default payment method, if any, or to the default payment method in the customer's invoice settings.",
		},
		"from_invoice.invoice": {
			Type:        "string",
			Description: "The `id` of the invoice that will be cloned.",
			Required:    true,
		},
		"issuer.account": {
			Type:        "string",
			Description: "The connected account being referenced when `type` is `account`.",
		},
		"shipping_cost.shipping_rate_data.delivery_estimate.minimum.value": {
			Type:        "integer",
			Description: "Must be greater than 0.",
			Required:    true,
		},
		"effective_at": {
			Type:        "integer",
			Description: "The date when this invoice is in effect. Same as `finalized_at` unless overwritten. When defined, this value replaces the system-generated 'Date of issue' printed on the invoice PDF and receipt.",
			Format:      "unix-time",
		},
		"shipping_details.address.country": {
			Type:        "string",
			Description: "Two-letter country code ([ISO 3166-1 alpha-2](https://en.wikipedia.org/wiki/ISO_3166-1_alpha-2)).",
		},
		"shipping_details.address.line2": {
			Type:        "string",
			Description: "Address line 2, such as the apartment, suite, unit, or building.",
		},
		"number": {
			Type:        "string",
			Description: "Set the number for this invoice. If no number is present then a number will be assigned automatically when the invoice is finalized. In many markets, regulations require invoices to be unique, sequential and / or gapless. You are responsible for ensuring this is true across all your different invoicing systems in the event that you edit the invoice number using our API. If you use only Stripe for your invoices and do not change invoice numbers, Stripe handles this aspect of compliance for you automatically.",
		},
		"payment_settings.payment_method_types": {
			Type:        "array",
			Description: "The list of payment method types (e.g. card) to provide to the invoice’s PaymentIntent. If not set, Stripe attempts to automatically determine the types to use by looking at the invoice’s default payment method, the subscription’s default payment method, the customer’s default payment method, and your [invoice template settings](https://dashboard.stripe.com/settings/billing/invoice).",
		},
		"shipping_cost.shipping_rate_data.type": {
			Type:        "string",
			Description: "The type of calculation to use on the shipping rate.",
			Enum: []resource.EnumSpec{
				{Value: "fixed_amount"},
			},
		},
		"default_source": {
			Type:        "string",
			Description: "ID of the default payment source for the invoice. It must belong to the customer associated with the invoice and be in a chargeable state. If not set, defaults to the subscription's default source, if any, or to the customer's default source.",
		},
		"customer": {
			Type:        "string",
			Description: "The ID of the customer to bill.",
		},
		"from_invoice.action": {
			Type:        "string",
			Description: "The relation between the new invoice and the original invoice. Currently, only 'revision' is permitted",
			Required:    true,
			Enum: []resource.EnumSpec{
				{Value: "revision"},
			},
		},
	},
}

var V1InvoicesAddLines = resource.OperationSpec{
	Name:    "add_lines",
	Path:    "/v1/invoices/{invoice}/add_lines",
	Method:  "POST",
	Summary: "Bulk add invoice line items",
}

var V1InvoicesAttachPayment = resource.OperationSpec{
	Name:    "attach_payment",
	Path:    "/v1/invoices/{invoice}/attach_payment",
	Method:  "POST",
	Summary: "Attach a payment to an Invoice",
	Params: map[string]*resource.ParamSpec{
		"payment_intent": {
			Type:        "string",
			Description: "The ID of the PaymentIntent to attach to the invoice.",
		},
		"payment_record": {
			Type:        "string",
			Description: "The ID of the PaymentRecord to attach to the invoice.",
		},
	},
}

var V1InvoicesVoidInvoice = resource.OperationSpec{
	Name:    "void_invoice",
	Path:    "/v1/invoices/{invoice}/void",
	Method:  "POST",
	Summary: "Void an invoice",
}

var V1InvoicesCreatePreview = resource.OperationSpec{
	Name:    "create_preview",
	Path:    "/v1/invoices/create_preview",
	Method:  "POST",
	Summary: "Create a preview invoice",
	Params: map[string]*resource.ParamSpec{
		"subscription_details.proration_behavior": {
			Type:        "string",
			Description: "Determines how to handle [prorations](https://docs.stripe.com/billing/subscriptions/prorations) when the billing cycle changes (e.g., when switching plans, resetting `billing_cycle_anchor=now`, or starting a trial), or if an item's `quantity` changes. The default value is `create_prorations`.",
			Enum: []resource.EnumSpec{
				{Value: "always_invoice"},
				{Value: "create_prorations"},
				{Value: "none"},
			},
		},
		"subscription_details.trial_end": {
			Type:        "string",
			Description: "If provided, the invoice returned will preview updating or creating a subscription with that trial end. If set, one of `subscription_details.items` or `subscription` is required.",
		},
		"schedule": {
			Type:        "string",
			Description: "The identifier of the schedule whose upcoming invoice you'd like to retrieve. Cannot be used with subscription or subscription fields.",
		},
		"subscription": {
			Type:        "string",
			Description: "The identifier of the subscription for which you'd like to retrieve the upcoming invoice. If not provided, but a `subscription_details.items` is provided, you will preview creating a subscription with those items. If neither `subscription` nor `subscription_details.items` is provided, you will retrieve the next upcoming invoice from among the customer's subscriptions.",
		},
		"automatic_tax.liability.account": {
			Type:        "string",
			Description: "The connected account being referenced when `type` is `account`.",
		},
		"customer_details.tax.ip_address": {
			Type:        "string",
			Description: "A recent IP address of the customer used for tax reporting and tax location inference. Stripe recommends updating the IP address when a new PaymentMethod is attached or the address field on the customer is updated. We recommend against updating this field more frequently since it could result in unexpected tax location/reporting outcomes.",
		},
		"preview_mode": {
			Type:        "string",
			Description: "Customizes the types of values to include when calculating the invoice. Defaults to `next` if unspecified.",
			Enum: []resource.EnumSpec{
				{Value: "next"},
				{Value: "recurring"},
			},
		},
		"subscription_details.resume_at": {
			Type:        "string",
			Description: "For paused subscriptions, setting `subscription_details.resume_at` to `now` will preview the invoice that will be generated if the subscription is resumed.",
			Enum: []resource.EnumSpec{
				{Value: "now"},
			},
		},
		"schedule_details.billing_mode.type": {
			Type:        "string",
			Description: "Controls the calculation and orchestration of prorations and invoices for subscriptions. If no value is passed, the default is `flexible`.",
			Required:    true,
			Enum: []resource.EnumSpec{
				{Value: "classic"},
				{Value: "flexible"},
			},
		},
		"customer_details.tax_exempt": {
			Type:        "string",
			Description: "The customer's tax exemption. One of `none`, `exempt`, or `reverse`.",
			Enum: []resource.EnumSpec{
				{Value: "exempt"},
				{Value: "none"},
				{Value: "reverse"},
			},
		},
		"on_behalf_of": {
			Type:        "string",
			Description: "The account (if any) for which the funds of the invoice payment are intended. If set, the invoice will be presented with the branding and support information of the specified account. See the [Invoices with Connect](https://docs.stripe.com/billing/invoices/connect) documentation for details.",
		},
		"subscription_details.billing_mode.type": {
			Type:        "string",
			Description: "Controls the calculation and orchestration of prorations and invoices for subscriptions. If no value is passed, the default is `flexible`.",
			Required:    true,
			Enum: []resource.EnumSpec{
				{Value: "classic"},
				{Value: "flexible"},
			},
		},
		"subscription_details.billing_mode.flexible.proration_discounts": {
			Type:        "string",
			Description: "Controls how invoices and invoice items display proration amounts and discount amounts.",
			Enum: []resource.EnumSpec{
				{Value: "included"},
				{Value: "itemized"},
			},
		},
		"subscription_details.cancel_at_period_end": {
			Type:        "boolean",
			Description: "Indicate whether this subscription should cancel at the end of the current period (`current_period_end`). Defaults to `false`.",
		},
		"schedule_details.end_behavior": {
			Type:        "string",
			Description: "Behavior of the subscription schedule and underlying subscription when it ends. Possible values are `release` or `cancel` with the default being `release`. `release` will end the subscription schedule and keep the underlying subscription running. `cancel` will end the subscription schedule and cancel the underlying subscription.",
			Enum: []resource.EnumSpec{
				{Value: "cancel"},
				{Value: "release"},
			},
		},
		"customer_account": {
			Type:        "string",
			Description: "The identifier of the account representing the customer whose upcoming invoice you're retrieving. If `automatic_tax` is enabled then one of `customer`, `customer_account`, `customer_details`, `subscription`, or `schedule` must be set.",
		},
		"issuer.account": {
			Type:        "string",
			Description: "The connected account being referenced when `type` is `account`.",
		},
		"subscription_details.default_tax_rates": {
			Type:        "array",
			Description: "If provided, the invoice returned will preview updating or creating a subscription with these default tax rates. The default tax rates will apply to any line item that does not have `tax_rates` set.",
		},
		"currency": {
			Type:        "string",
			Description: "The currency to preview this invoice in. Defaults to that of `customer` if not specified.",
			Format:      "currency",
		},
		"subscription_details.cancel_now": {
			Type:        "boolean",
			Description: "This simulates the subscription being canceled or expired immediately.",
		},
		"subscription_details.start_date": {
			Type:        "integer",
			Description: "Date a subscription is intended to start (can be future or past).",
			Format:      "unix-time",
		},
		"subscription_details.billing_cycle_anchor": {
			Type:        "string",
			Description: "For new subscriptions, a future timestamp to anchor the subscription's [billing cycle](https://docs.stripe.com/subscriptions/billing-cycle). This is used to determine the date of the first full invoice, and, for plans with `month` or `year` intervals, the day of the month for subsequent invoices. For existing subscriptions, the value can only be set to `now` or `unchanged`.",
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
		"schedule_details.billing_mode.flexible.proration_discounts": {
			Type:        "string",
			Description: "Controls how invoices and invoice items display proration amounts and discount amounts.",
			Enum: []resource.EnumSpec{
				{Value: "included"},
				{Value: "itemized"},
			},
		},
		"customer": {
			Type:        "string",
			Description: "The identifier of the customer whose upcoming invoice you're retrieving. If `automatic_tax` is enabled then one of `customer`, `customer_details`, `subscription`, or `schedule` must be set.",
		},
		"issuer.type": {
			Type:        "string",
			Description: "Type of the account referenced in the request.",
			Required:    true,
			Enum: []resource.EnumSpec{
				{Value: "account"},
				{Value: "self"},
			},
		},
		"subscription_details.proration_date": {
			Type:        "integer",
			Description: "If previewing an update to a subscription, and doing proration, `subscription_details.proration_date` forces the proration to be calculated as though the update was done at the specified time. The time given must be within the current subscription period and within the current phase of the schedule backing this subscription, if the schedule exists. If set, `subscription`, and one of `subscription_details.items`, or `subscription_details.trial_end` are required. Also, `subscription_details.proration_behavior` cannot be set to 'none'.",
			Format:      "unix-time",
		},
		"automatic_tax.enabled": {
			Type:        "boolean",
			Description: "Whether Stripe automatically computes tax on this invoice. Note that incompatible invoice items (invoice items with manually specified [tax rates](https://docs.stripe.com/api/tax_rates), negative amounts, or `tax_behavior=unspecified`) cannot be added to automatic tax invoices.",
			Required:    true,
		},
		"subscription_details.cancel_at": {
			Type:        "integer",
			Description: "A timestamp at which the subscription should cancel. If set to a date before the current period ends, this will cause a proration if prorations have been enabled using `proration_behavior`. If set during a future period, this will always cause a proration for that period.",
		},
		"schedule_details.proration_behavior": {
			Type:        "string",
			Description: "In cases where the `schedule_details` params update the currently active phase, specifies if and how to prorate at the time of the request.",
			Enum: []resource.EnumSpec{
				{Value: "always_invoice"},
				{Value: "create_prorations"},
				{Value: "none"},
			},
		},
	},
}

var V1InvoicesFinalizeInvoice = resource.OperationSpec{
	Name:    "finalize_invoice",
	Path:    "/v1/invoices/{invoice}/finalize",
	Method:  "POST",
	Summary: "Finalize an invoice",
	Params: map[string]*resource.ParamSpec{
		"auto_advance": {
			Type:        "boolean",
			Description: "Controls whether Stripe performs [automatic collection](https://docs.stripe.com/invoicing/integration/automatic-advancement-collection) of the invoice. If `false`, the invoice's state doesn't automatically advance without an explicit action.",
		},
	},
}

var V1InvoicesMarkUncollectible = resource.OperationSpec{
	Name:    "mark_uncollectible",
	Path:    "/v1/invoices/{invoice}/mark_uncollectible",
	Method:  "POST",
	Summary: "Mark an invoice as uncollectible",
}

var V1InvoicesRemoveLines = resource.OperationSpec{
	Name:    "remove_lines",
	Path:    "/v1/invoices/{invoice}/remove_lines",
	Method:  "POST",
	Summary: "Bulk remove invoice line items",
}

var V1InvoicesSendInvoice = resource.OperationSpec{
	Name:    "send_invoice",
	Path:    "/v1/invoices/{invoice}/send",
	Method:  "POST",
	Summary: "Send an invoice for manual payment",
}

var V1SubscriptionItemsDelete = resource.OperationSpec{
	Name:    "delete",
	Path:    "/v1/subscription_items/{item}",
	Method:  "DELETE",
	Summary: "Delete a subscription item",
}

var V1SubscriptionItemsList = resource.OperationSpec{
	Name:    "list",
	Path:    "/v1/subscription_items",
	Method:  "GET",
	Summary: "List all subscription items",
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
		"subscription": {
			Type:        "string",
			Description: "The ID of the subscription whose items will be retrieved.",
			Required:    true,
		},
	},
}

var V1SubscriptionItemsRetrieve = resource.OperationSpec{
	Name:    "retrieve",
	Path:    "/v1/subscription_items/{item}",
	Method:  "GET",
	Summary: "Retrieve a subscription item",
}

var V1SubscriptionItemsCreate = resource.OperationSpec{
	Name:    "create",
	Path:    "/v1/subscription_items",
	Method:  "POST",
	Summary: "Create a subscription item",
	Params: map[string]*resource.ParamSpec{
		"price_data.unit_amount_decimal": {
			Type:        "string",
			Description: "Same as `unit_amount`, but accepts a decimal value in cents (or local equivalent) with at most 12 decimal places. Only one of `unit_amount` and `unit_amount_decimal` can be set.",
			Format:      "decimal",
		},
		"price_data.recurring.interval_count": {
			Type:        "integer",
			Description: "The number of intervals between subscription billings. For example, `interval=month` and `interval_count=3` bills every 3 months. Maximum of three years interval allowed (3 years, 36 months, or 156 weeks).",
		},
		"price_data.tax_behavior": {
			Type:        "string",
			Description: "Only required if a [default tax behavior](https://docs.stripe.com/tax/products-prices-tax-categories-tax-behavior#setting-a-default-tax-behavior-(recommended)) was not provided in the Stripe Tax settings. Specifies whether the price is considered inclusive of taxes or exclusive of taxes. One of `inclusive`, `exclusive`, or `unspecified`. Once specified as either `inclusive` or `exclusive`, it cannot be changed.",
			Enum: []resource.EnumSpec{
				{Value: "exclusive"},
				{Value: "inclusive"},
				{Value: "unspecified"},
			},
		},
		"proration_behavior": {
			Type:        "string",
			Description: "Determines how to handle [prorations](https://docs.stripe.com/billing/subscriptions/prorations) when the billing cycle changes (e.g., when switching plans, resetting `billing_cycle_anchor=now`, or starting a trial), or if an item's `quantity` changes. The default value is `create_prorations`.",
			Enum: []resource.EnumSpec{
				{Value: "always_invoice"},
				{Value: "create_prorations"},
				{Value: "none"},
			},
		},
		"quantity": {
			Type:        "integer",
			Description: "The quantity you'd like to apply to the subscription item you're creating.",
		},
		"payment_behavior": {
			Type:        "string",
			Description: "Use `allow_incomplete` to transition the subscription to `status=past_due` if a payment is required but cannot be paid. This allows you to manage scenarios where additional user actions are needed to pay a subscription's invoice. For example, SCA regulation may require 3DS authentication to complete payment. See the [SCA Migration Guide](https://docs.stripe.com/billing/migration/strong-customer-authentication) for Billing to learn more. This is the default behavior.\n\nUse `default_incomplete` to transition the subscription to `status=past_due` when payment is required and await explicit confirmation of the invoice's payment intent. This allows simpler management of scenarios where additional user actions are needed to pay a subscription’s invoice. Such as failed payments, [SCA regulation](https://docs.stripe.com/billing/migration/strong-customer-authentication), or collecting a mandate for a bank debit payment method.\n\nUse `pending_if_incomplete` to update the subscription using [pending updates](https://docs.stripe.com/billing/subscriptions/pending-updates). When you use `pending_if_incomplete` you can only pass the parameters [supported by pending updates](https://docs.stripe.com/billing/pending-updates-reference#supported-attributes).\n\nUse `error_if_incomplete` if you want Stripe to return an HTTP 402 status code if a subscription's invoice cannot be paid. For example, if a payment method requires 3DS authentication due to SCA regulation and further user action is needed, this parameter does not update the subscription and returns an error instead. This was the default behavior for API versions prior to 2019-03-14. See the [changelog](https://docs.stripe.com/changelog/2019-03-14) to learn more.",
			Enum: []resource.EnumSpec{
				{Value: "allow_incomplete"},
				{Value: "default_incomplete"},
				{Value: "error_if_incomplete"},
				{Value: "pending_if_incomplete"},
			},
		},
		"price_data.unit_amount": {
			Type:        "integer",
			Description: "A positive integer in cents (or local equivalent) (or 0 for a free price) representing how much to charge.",
		},
		"price_data.currency": {
			Type:        "string",
			Description: "Three-letter [ISO currency code](https://www.iso.org/iso-4217-currency-codes.html), in lowercase. Must be a [supported currency](https://stripe.com/docs/currencies).",
			Required:    true,
			Format:      "currency",
		},
		"price_data.product": {
			Type:        "string",
			Description: "The ID of the [Product](https://docs.stripe.com/api/products) that this [Price](https://docs.stripe.com/api/prices) will belong to.",
			Required:    true,
		},
		"price_data.recurring.interval": {
			Type:        "string",
			Description: "Specifies billing frequency. Either `day`, `week`, `month` or `year`.",
			Required:    true,
			Enum: []resource.EnumSpec{
				{Value: "day"},
				{Value: "month"},
				{Value: "week"},
				{Value: "year"},
			},
		},
		"tax_rates": {
			Type:        "array",
			Description: "A list of [Tax Rate](https://docs.stripe.com/api/tax_rates) ids. These Tax Rates will override the [`default_tax_rates`](https://docs.stripe.com/api/subscriptions/create#create_subscription-default_tax_rates) on the Subscription. When updating, pass an empty string to remove previously-defined tax rates.",
		},
		"plan": {
			Type:        "string",
			Description: "The identifier of the plan to add to the subscription.",
		},
		"subscription": {
			Type:        "string",
			Description: "The identifier of the subscription to modify.",
			Required:    true,
		},
		"proration_date": {
			Type:        "integer",
			Description: "If set, the proration will be calculated as though the subscription was updated at the given time. This can be used to apply the same proration that was previewed with the [upcoming invoice](https://api.stripe.com#retrieve_customer_invoice) endpoint.",
			Format:      "unix-time",
		},
		"price": {
			Type:        "string",
			Description: "The ID of the price object.",
		},
	},
}

var V1SubscriptionItemsUpdate = resource.OperationSpec{
	Name:    "update",
	Path:    "/v1/subscription_items/{item}",
	Method:  "POST",
	Summary: "Update a subscription item",
	Params: map[string]*resource.ParamSpec{
		"payment_behavior": {
			Type:        "string",
			Description: "Use `allow_incomplete` to transition the subscription to `status=past_due` if a payment is required but cannot be paid. This allows you to manage scenarios where additional user actions are needed to pay a subscription's invoice. For example, SCA regulation may require 3DS authentication to complete payment. See the [SCA Migration Guide](https://docs.stripe.com/billing/migration/strong-customer-authentication) for Billing to learn more. This is the default behavior.\n\nUse `default_incomplete` to transition the subscription to `status=past_due` when payment is required and await explicit confirmation of the invoice's payment intent. This allows simpler management of scenarios where additional user actions are needed to pay a subscription’s invoice. Such as failed payments, [SCA regulation](https://docs.stripe.com/billing/migration/strong-customer-authentication), or collecting a mandate for a bank debit payment method.\n\nUse `pending_if_incomplete` to update the subscription using [pending updates](https://docs.stripe.com/billing/subscriptions/pending-updates). When you use `pending_if_incomplete` you can only pass the parameters [supported by pending updates](https://docs.stripe.com/billing/pending-updates-reference#supported-attributes).\n\nUse `error_if_incomplete` if you want Stripe to return an HTTP 402 status code if a subscription's invoice cannot be paid. For example, if a payment method requires 3DS authentication due to SCA regulation and further user action is needed, this parameter does not update the subscription and returns an error instead. This was the default behavior for API versions prior to 2019-03-14. See the [changelog](https://docs.stripe.com/changelog/2019-03-14) to learn more.",
			Enum: []resource.EnumSpec{
				{Value: "allow_incomplete"},
				{Value: "default_incomplete"},
				{Value: "error_if_incomplete"},
				{Value: "pending_if_incomplete"},
			},
		},
		"price_data.currency": {
			Type:        "string",
			Description: "Three-letter [ISO currency code](https://www.iso.org/iso-4217-currency-codes.html), in lowercase. Must be a [supported currency](https://stripe.com/docs/currencies).",
			Required:    true,
			Format:      "currency",
		},
		"quantity": {
			Type:        "integer",
			Description: "The quantity you'd like to apply to the subscription item you're creating.",
		},
		"tax_rates": {
			Type:        "array",
			Description: "A list of [Tax Rate](https://docs.stripe.com/api/tax_rates) ids. These Tax Rates will override the [`default_tax_rates`](https://docs.stripe.com/api/subscriptions/create#create_subscription-default_tax_rates) on the Subscription. When updating, pass an empty string to remove previously-defined tax rates.",
		},
		"plan": {
			Type:        "string",
			Description: "The identifier of the new plan for this subscription item.",
		},
		"price_data.product": {
			Type:        "string",
			Description: "The ID of the [Product](https://docs.stripe.com/api/products) that this [Price](https://docs.stripe.com/api/prices) will belong to.",
			Required:    true,
		},
		"price_data.recurring.interval_count": {
			Type:        "integer",
			Description: "The number of intervals between subscription billings. For example, `interval=month` and `interval_count=3` bills every 3 months. Maximum of three years interval allowed (3 years, 36 months, or 156 weeks).",
		},
		"proration_date": {
			Type:        "integer",
			Description: "If set, the proration will be calculated as though the subscription was updated at the given time. This can be used to apply the same proration that was previewed with the [upcoming invoice](https://api.stripe.com#retrieve_customer_invoice) endpoint.",
			Format:      "unix-time",
		},
		"price": {
			Type:        "string",
			Description: "The ID of the price object. One of `price` or `price_data` is required. When changing a subscription item's price, `quantity` is set to 1 unless a `quantity` parameter is provided.",
		},
		"price_data.unit_amount": {
			Type:        "integer",
			Description: "A positive integer in cents (or local equivalent) (or 0 for a free price) representing how much to charge.",
		},
		"price_data.tax_behavior": {
			Type:        "string",
			Description: "Only required if a [default tax behavior](https://docs.stripe.com/tax/products-prices-tax-categories-tax-behavior#setting-a-default-tax-behavior-(recommended)) was not provided in the Stripe Tax settings. Specifies whether the price is considered inclusive of taxes or exclusive of taxes. One of `inclusive`, `exclusive`, or `unspecified`. Once specified as either `inclusive` or `exclusive`, it cannot be changed.",
			Enum: []resource.EnumSpec{
				{Value: "exclusive"},
				{Value: "inclusive"},
				{Value: "unspecified"},
			},
		},
		"proration_behavior": {
			Type:        "string",
			Description: "Determines how to handle [prorations](https://docs.stripe.com/billing/subscriptions/prorations) when the billing cycle changes (e.g., when switching plans, resetting `billing_cycle_anchor=now`, or starting a trial), or if an item's `quantity` changes. The default value is `create_prorations`.",
			Enum: []resource.EnumSpec{
				{Value: "always_invoice"},
				{Value: "create_prorations"},
				{Value: "none"},
			},
		},
		"price_data.unit_amount_decimal": {
			Type:        "string",
			Description: "Same as `unit_amount`, but accepts a decimal value in cents (or local equivalent) with at most 12 decimal places. Only one of `unit_amount` and `unit_amount_decimal` can be set.",
			Format:      "decimal",
		},
		"price_data.recurring.interval": {
			Type:        "string",
			Description: "Specifies billing frequency. Either `day`, `week`, `month` or `year`.",
			Required:    true,
			Enum: []resource.EnumSpec{
				{Value: "day"},
				{Value: "month"},
				{Value: "week"},
				{Value: "year"},
			},
		},
		"off_session": {
			Type:        "boolean",
			Description: "Indicates if a customer is on or off-session while an invoice payment is attempted. Defaults to `false` (on-session).",
		},
	},
}

var V1PayoutsUpdate = resource.OperationSpec{
	Name:    "update",
	Path:    "/v1/payouts/{payout}",
	Method:  "POST",
	Summary: "Update a payout",
}

var V1PayoutsCancel = resource.OperationSpec{
	Name:    "cancel",
	Path:    "/v1/payouts/{payout}/cancel",
	Method:  "POST",
	Summary: "Cancel a payout",
}

var V1PayoutsReverse = resource.OperationSpec{
	Name:    "reverse",
	Path:    "/v1/payouts/{payout}/reverse",
	Method:  "POST",
	Summary: "Reverse a payout",
}

var V1PayoutsList = resource.OperationSpec{
	Name:    "list",
	Path:    "/v1/payouts",
	Method:  "GET",
	Summary: "List all payouts",
	Params: map[string]*resource.ParamSpec{
		"limit": {
			Type:        "integer",
			Description: "A limit on the number of objects to be returned. Limit can range between 1 and 100, and the default is 10.",
		},
		"starting_after": {
			Type:        "string",
			Description: "A cursor for use in pagination. `starting_after` is an object ID that defines your place in the list. For instance, if you make a list request and receive 100 objects, ending with `obj_foo`, your subsequent call can include `starting_after=obj_foo` in order to fetch the next page of the list.",
		},
		"status": {
			Type:        "string",
			Description: "Only return payouts that have the given status: `pending`, `paid`, `failed`, or `canceled`.",
		},
		"arrival_date": {
			Type:        "integer",
			Description: "Only return payouts that are expected to arrive during the given date interval.",
		},
		"created": {
			Type:        "integer",
			Description: "Only return payouts that were created during the given date interval.",
		},
		"destination": {
			Type:        "string",
			Description: "The ID of an external account - only return payouts sent to this external account.",
		},
		"ending_before": {
			Type:        "string",
			Description: "A cursor for use in pagination. `ending_before` is an object ID that defines your place in the list. For instance, if you make a list request and receive 100 objects, starting with `obj_bar`, your subsequent call can include `ending_before=obj_bar` in order to fetch the previous page of the list.",
		},
	},
}

var V1PayoutsRetrieve = resource.OperationSpec{
	Name:    "retrieve",
	Path:    "/v1/payouts/{payout}",
	Method:  "GET",
	Summary: "Retrieve a payout",
}

var V1PayoutsCreate = resource.OperationSpec{
	Name:    "create",
	Path:    "/v1/payouts",
	Method:  "POST",
	Summary: "Create a payout",
	Params: map[string]*resource.ParamSpec{
		"description": {
			Type:        "string",
			Description: "An arbitrary string attached to the object. Often useful for displaying to users.",
		},
		"payout_method": {
			Type:        "string",
			Description: "The ID of a v2 FinancialAccount to send funds to.",
		},
		"source_type": {
			Type:        "string",
			Description: "The balance type of your Stripe balance to draw this payout from. Balances for different payment sources are kept separately. You can find the amounts with the Balances API. One of `bank_account`, `card`, or `fpx`.",
			Enum: []resource.EnumSpec{
				{Value: "bank_account"},
				{Value: "card"},
				{Value: "fpx"},
			},
		},
		"amount": {
			Type:        "integer",
			Description: "A positive integer in cents representing how much to payout.",
			Required:    true,
		},
		"destination": {
			Type:        "string",
			Description: "The ID of a bank account or a card to send the payout to. If you don't provide a destination, we use the default external account for the specified currency.",
		},
		"method": {
			Type:        "string",
			Description: "The method used to send this payout, which is `standard` or `instant`. We support `instant` for payouts to debit cards and bank accounts in certain countries. Learn more about [bank support for Instant Payouts](https://stripe.com/docs/payouts/instant-payouts-banks).",
			Enum: []resource.EnumSpec{
				{Value: "instant"},
				{Value: "standard"},
			},
		},
		"statement_descriptor": {
			Type:        "string",
			Description: "A string that displays on the recipient's bank or card statement (up to 22 characters). A `statement_descriptor` that's longer than 22 characters return an error. Most banks truncate this information and display it inconsistently. Some banks might not display it at all.",
		},
		"currency": {
			Type:        "string",
			Description: "Three-letter [ISO currency code](https://www.iso.org/iso-4217-currency-codes.html), in lowercase. Must be a [supported currency](https://stripe.com/docs/currencies).",
			Required:    true,
			Format:      "currency",
		},
	},
}

var V1ShippingRatesList = resource.OperationSpec{
	Name:    "list",
	Path:    "/v1/shipping_rates",
	Method:  "GET",
	Summary: "List all shipping rates",
	Params: map[string]*resource.ParamSpec{
		"active": {
			Type:        "boolean",
			Description: "Only return shipping rates that are active or inactive.",
		},
		"created": {
			Type:        "integer",
			Description: "A filter on the list, based on the object `created` field. The value can be a string with an integer Unix timestamp, or it can be a dictionary with a number of different query options.",
		},
		"currency": {
			Type:        "string",
			Description: "Only return shipping rates for the given currency.",
			Format:      "currency",
		},
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

var V1ShippingRatesRetrieve = resource.OperationSpec{
	Name:    "retrieve",
	Path:    "/v1/shipping_rates/{shipping_rate_token}",
	Method:  "GET",
	Summary: "Retrieve a shipping rate",
}

var V1ShippingRatesCreate = resource.OperationSpec{
	Name:    "create",
	Path:    "/v1/shipping_rates",
	Method:  "POST",
	Summary: "Create a shipping rate",
	Params: map[string]*resource.ParamSpec{
		"fixed_amount.amount": {
			Type:        "integer",
			Description: "A non-negative integer in cents representing how much to charge.",
			Required:    true,
		},
		"fixed_amount.currency": {
			Type:        "string",
			Description: "Three-letter [ISO currency code](https://www.iso.org/iso-4217-currency-codes.html), in lowercase. Must be a [supported currency](https://stripe.com/docs/currencies).",
			Required:    true,
			Format:      "currency",
		},
		"tax_behavior": {
			Type:        "string",
			Description: "Specifies whether the rate is considered inclusive of taxes or exclusive of taxes. One of `inclusive`, `exclusive`, or `unspecified`.",
			Enum: []resource.EnumSpec{
				{Value: "exclusive"},
				{Value: "inclusive"},
				{Value: "unspecified"},
			},
		},
		"delivery_estimate.maximum.unit": {
			Type:        "string",
			Description: "A unit of time.",
			Required:    true,
			Enum: []resource.EnumSpec{
				{Value: "business_day"},
				{Value: "day"},
				{Value: "hour"},
				{Value: "month"},
				{Value: "week"},
			},
		},
		"delivery_estimate.maximum.value": {
			Type:        "integer",
			Description: "Must be greater than 0.",
			Required:    true,
		},
		"delivery_estimate.minimum.value": {
			Type:        "integer",
			Description: "Must be greater than 0.",
			Required:    true,
		},
		"display_name": {
			Type:        "string",
			Description: "The name of the shipping rate, meant to be displayable to the customer. This will appear on CheckoutSessions.",
			Required:    true,
		},
		"tax_code": {
			Type:        "string",
			Description: "A [tax code](https://docs.stripe.com/tax/tax-categories) ID. The Shipping tax code is `txcd_92010001`.",
		},
		"type": {
			Type:        "string",
			Description: "The type of calculation to use on the shipping rate.",
			Enum: []resource.EnumSpec{
				{Value: "fixed_amount"},
			},
		},
		"delivery_estimate.minimum.unit": {
			Type:        "string",
			Description: "A unit of time.",
			Required:    true,
			Enum: []resource.EnumSpec{
				{Value: "business_day"},
				{Value: "day"},
				{Value: "hour"},
				{Value: "month"},
				{Value: "week"},
			},
		},
	},
}

var V1ShippingRatesUpdate = resource.OperationSpec{
	Name:    "update",
	Path:    "/v1/shipping_rates/{shipping_rate_token}",
	Method:  "POST",
	Summary: "Update a shipping rate",
	Params: map[string]*resource.ParamSpec{
		"tax_behavior": {
			Type:        "string",
			Description: "Specifies whether the rate is considered inclusive of taxes or exclusive of taxes. One of `inclusive`, `exclusive`, or `unspecified`.",
			Enum: []resource.EnumSpec{
				{Value: "exclusive"},
				{Value: "inclusive"},
				{Value: "unspecified"},
			},
		},
		"active": {
			Type:        "boolean",
			Description: "Whether the shipping rate can be used for new purchases. Defaults to `true`.",
		},
	},
}

var V1CreditNotesPreview = resource.OperationSpec{
	Name:    "preview",
	Path:    "/v1/credit_notes/preview",
	Method:  "GET",
	Summary: "Preview a credit note",
	Params: map[string]*resource.ParamSpec{
		"effective_at": {
			Type:        "integer",
			Description: "The date when this credit note is in effect. Same as `created` unless overwritten. When defined, this value replaces the system-generated 'Date of issue' printed on the credit note PDF.",
			Format:      "unix-time",
		},
		"email_type": {
			Type:        "string",
			Description: "Type of email to send to the customer, one of `credit_note` or `none` and the default is `credit_note`.",
			Enum: []resource.EnumSpec{
				{Value: "credit_note"},
				{Value: "none"},
			},
		},
		"memo": {
			Type:        "string",
			Description: "The credit note's memo appears on the credit note PDF.",
		},
		"out_of_band_amount": {
			Type:        "integer",
			Description: "The integer amount in cents (or local equivalent) representing the amount that is credited outside of Stripe.",
		},
		"amount": {
			Type:        "integer",
			Description: "The integer amount in cents (or local equivalent) representing the total amount of the credit note. One of `amount`, `lines`, or `shipping_cost` must be provided.",
		},
		"credit_amount": {
			Type:        "integer",
			Description: "The integer amount in cents (or local equivalent) representing the amount to credit the customer's balance, which will be automatically applied to their next invoice.",
		},
		"invoice": {
			Type:        "string",
			Description: "ID of the invoice.",
			Required:    true,
		},
		"reason": {
			Type:        "string",
			Description: "Reason for issuing this credit note, one of `duplicate`, `fraudulent`, `order_change`, or `product_unsatisfactory`",
			Enum: []resource.EnumSpec{
				{Value: "duplicate"},
				{Value: "fraudulent"},
				{Value: "order_change"},
				{Value: "product_unsatisfactory"},
			},
		},
		"refund_amount": {
			Type:        "integer",
			Description: "The integer amount in cents (or local equivalent) representing the amount to refund. If set, a refund will be created for the charge associated with the invoice.",
		},
	},
}

var V1CreditNotesPreviewLines = resource.OperationSpec{
	Name:    "preview_lines",
	Path:    "/v1/credit_notes/preview/lines",
	Method:  "GET",
	Summary: "Retrieve a credit note preview's line items",
	Params: map[string]*resource.ParamSpec{
		"amount": {
			Type:        "integer",
			Description: "The integer amount in cents (or local equivalent) representing the total amount of the credit note. One of `amount`, `lines`, or `shipping_cost` must be provided.",
		},
		"credit_amount": {
			Type:        "integer",
			Description: "The integer amount in cents (or local equivalent) representing the amount to credit the customer's balance, which will be automatically applied to their next invoice.",
		},
		"effective_at": {
			Type:        "integer",
			Description: "The date when this credit note is in effect. Same as `created` unless overwritten. When defined, this value replaces the system-generated 'Date of issue' printed on the credit note PDF.",
			Format:      "unix-time",
		},
		"email_type": {
			Type:        "string",
			Description: "Type of email to send to the customer, one of `credit_note` or `none` and the default is `credit_note`.",
			Enum: []resource.EnumSpec{
				{Value: "credit_note"},
				{Value: "none"},
			},
		},
		"limit": {
			Type:        "integer",
			Description: "A limit on the number of objects to be returned. Limit can range between 1 and 100, and the default is 10.",
		},
		"ending_before": {
			Type:        "string",
			Description: "A cursor for use in pagination. `ending_before` is an object ID that defines your place in the list. For instance, if you make a list request and receive 100 objects, starting with `obj_bar`, your subsequent call can include `ending_before=obj_bar` in order to fetch the previous page of the list.",
		},
		"invoice": {
			Type:        "string",
			Description: "ID of the invoice.",
			Required:    true,
		},
		"memo": {
			Type:        "string",
			Description: "The credit note's memo appears on the credit note PDF.",
		},
		"out_of_band_amount": {
			Type:        "integer",
			Description: "The integer amount in cents (or local equivalent) representing the amount that is credited outside of Stripe.",
		},
		"reason": {
			Type:        "string",
			Description: "Reason for issuing this credit note, one of `duplicate`, `fraudulent`, `order_change`, or `product_unsatisfactory`",
			Enum: []resource.EnumSpec{
				{Value: "duplicate"},
				{Value: "fraudulent"},
				{Value: "order_change"},
				{Value: "product_unsatisfactory"},
			},
		},
		"refund_amount": {
			Type:        "integer",
			Description: "The integer amount in cents (or local equivalent) representing the amount to refund. If set, a refund will be created for the charge associated with the invoice.",
		},
		"starting_after": {
			Type:        "string",
			Description: "A cursor for use in pagination. `starting_after` is an object ID that defines your place in the list. For instance, if you make a list request and receive 100 objects, ending with `obj_foo`, your subsequent call can include `starting_after=obj_foo` in order to fetch the next page of the list.",
		},
	},
}

var V1CreditNotesCreate = resource.OperationSpec{
	Name:    "create",
	Path:    "/v1/credit_notes",
	Method:  "POST",
	Summary: "Create a credit note",
	Params: map[string]*resource.ParamSpec{
		"invoice": {
			Type:        "string",
			Description: "ID of the invoice.",
			Required:    true,
		},
		"memo": {
			Type:        "string",
			Description: "The credit note's memo appears on the credit note PDF.",
		},
		"credit_amount": {
			Type:        "integer",
			Description: "The integer amount in cents (or local equivalent) representing the amount to credit the customer's balance, which will be automatically applied to their next invoice.",
		},
		"effective_at": {
			Type:        "integer",
			Description: "The date when this credit note is in effect. Same as `created` unless overwritten. When defined, this value replaces the system-generated 'Date of issue' printed on the credit note PDF.",
			Format:      "unix-time",
		},
		"refund_amount": {
			Type:        "integer",
			Description: "The integer amount in cents (or local equivalent) representing the amount to refund. If set, a refund will be created for the charge associated with the invoice.",
		},
		"shipping_cost.shipping_rate": {
			Type:        "string",
			Description: "The ID of the shipping rate to use for this order.",
		},
		"email_type": {
			Type:        "string",
			Description: "Type of email to send to the customer, one of `credit_note` or `none` and the default is `credit_note`.",
			Enum: []resource.EnumSpec{
				{Value: "credit_note"},
				{Value: "none"},
			},
		},
		"reason": {
			Type:        "string",
			Description: "Reason for issuing this credit note, one of `duplicate`, `fraudulent`, `order_change`, or `product_unsatisfactory`",
			Enum: []resource.EnumSpec{
				{Value: "duplicate"},
				{Value: "fraudulent"},
				{Value: "order_change"},
				{Value: "product_unsatisfactory"},
			},
		},
		"amount": {
			Type:        "integer",
			Description: "The integer amount in cents (or local equivalent) representing the total amount of the credit note. One of `amount`, `lines`, or `shipping_cost` must be provided.",
		},
		"out_of_band_amount": {
			Type:        "integer",
			Description: "The integer amount in cents (or local equivalent) representing the amount that is credited outside of Stripe.",
		},
	},
}

var V1CreditNotesUpdate = resource.OperationSpec{
	Name:    "update",
	Path:    "/v1/credit_notes/{id}",
	Method:  "POST",
	Summary: "Update a credit note",
	Params: map[string]*resource.ParamSpec{
		"memo": {
			Type:        "string",
			Description: "Credit note memo.",
		},
	},
}

var V1CreditNotesVoidCreditNote = resource.OperationSpec{
	Name:    "void_credit_note",
	Path:    "/v1/credit_notes/{id}/void",
	Method:  "POST",
	Summary: "Void a credit note",
}

var V1CreditNotesList = resource.OperationSpec{
	Name:    "list",
	Path:    "/v1/credit_notes",
	Method:  "GET",
	Summary: "List all credit notes",
	Params: map[string]*resource.ParamSpec{
		"invoice": {
			Type:        "string",
			Description: "Only return credit notes for the invoice specified by this invoice ID.",
		},
		"limit": {
			Type:        "integer",
			Description: "A limit on the number of objects to be returned. Limit can range between 1 and 100, and the default is 10.",
		},
		"starting_after": {
			Type:        "string",
			Description: "A cursor for use in pagination. `starting_after` is an object ID that defines your place in the list. For instance, if you make a list request and receive 100 objects, ending with `obj_foo`, your subsequent call can include `starting_after=obj_foo` in order to fetch the next page of the list.",
		},
		"created": {
			Type:        "integer",
			Description: "Only return credit notes that were created during the given date interval.",
		},
		"customer": {
			Type:        "string",
			Description: "Only return credit notes for the customer specified by this customer ID.",
		},
		"customer_account": {
			Type:        "string",
			Description: "Only return credit notes for the account representing the customer specified by this account ID.",
		},
		"ending_before": {
			Type:        "string",
			Description: "A cursor for use in pagination. `ending_before` is an object ID that defines your place in the list. For instance, if you make a list request and receive 100 objects, starting with `obj_bar`, your subsequent call can include `ending_before=obj_bar` in order to fetch the previous page of the list.",
		},
	},
}

var V1CreditNotesRetrieve = resource.OperationSpec{
	Name:    "retrieve",
	Path:    "/v1/credit_notes/{id}",
	Method:  "GET",
	Summary: "Retrieve a credit note",
}

var V1InvoiceLineItemsList = resource.OperationSpec{
	Name:    "list",
	Path:    "/v1/invoices/{invoice}/lines",
	Method:  "GET",
	Summary: "Retrieve an invoice's line items",
	Params: map[string]*resource.ParamSpec{
		"starting_after": {
			Type:        "string",
			Description: "A cursor for use in pagination. `starting_after` is an object ID that defines your place in the list. For instance, if you make a list request and receive 100 objects, ending with `obj_foo`, your subsequent call can include `starting_after=obj_foo` in order to fetch the next page of the list.",
		},
		"ending_before": {
			Type:        "string",
			Description: "A cursor for use in pagination. `ending_before` is an object ID that defines your place in the list. For instance, if you make a list request and receive 100 objects, starting with `obj_bar`, your subsequent call can include `ending_before=obj_bar` in order to fetch the previous page of the list.",
		},
		"limit": {
			Type:        "integer",
			Description: "A limit on the number of objects to be returned. Limit can range between 1 and 100, and the default is 10.",
		},
	},
}

var V1InvoiceLineItemsUpdate = resource.OperationSpec{
	Name:    "update",
	Path:    "/v1/invoices/{invoice}/lines/{line_item_id}",
	Method:  "POST",
	Summary: "Update an invoice's line item",
	Params: map[string]*resource.ParamSpec{
		"price_data.product_data.name": {
			Type:        "string",
			Description: "The product's name, meant to be displayable to the customer.",
			Required:    true,
		},
		"period.end": {
			Type:        "integer",
			Description: "The end of the period, which must be greater than or equal to the start. This value is inclusive.",
			Required:    true,
			Format:      "unix-time",
		},
		"period.start": {
			Type:        "integer",
			Description: "The start of the period. This value is inclusive.",
			Required:    true,
			Format:      "unix-time",
		},
		"discountable": {
			Type:        "boolean",
			Description: "Controls whether discounts apply to this line item. Defaults to false for prorations or negative line items, and true for all other line items. Cannot be set to true for prorations.",
		},
		"price_data.currency": {
			Type:        "string",
			Description: "Three-letter [ISO currency code](https://www.iso.org/iso-4217-currency-codes.html), in lowercase. Must be a [supported currency](https://stripe.com/docs/currencies).",
			Required:    true,
			Format:      "currency",
		},
		"price_data.tax_behavior": {
			Type:        "string",
			Description: "Only required if a [default tax behavior](https://docs.stripe.com/tax/products-prices-tax-categories-tax-behavior#setting-a-default-tax-behavior-(recommended)) was not provided in the Stripe Tax settings. Specifies whether the price is considered inclusive of taxes or exclusive of taxes. One of `inclusive`, `exclusive`, or `unspecified`. Once specified as either `inclusive` or `exclusive`, it cannot be changed.",
			Enum: []resource.EnumSpec{
				{Value: "exclusive"},
				{Value: "inclusive"},
				{Value: "unspecified"},
			},
		},
		"price_data.unit_amount": {
			Type:        "integer",
			Description: "A non-negative integer in cents (or local equivalent) representing how much to charge. One of `unit_amount` or `unit_amount_decimal` is required.",
		},
		"price_data.product": {
			Type:        "string",
			Description: "The ID of the [Product](https://docs.stripe.com/api/products) that this [Price](https://docs.stripe.com/api/prices) will belong to. One of `product` or `product_data` is required.",
		},
		"price_data.product_data.images": {
			Type:        "array",
			Description: "A list of up to 8 URLs of images for this product, meant to be displayable to the customer.",
		},
		"price_data.product_data.tax_code": {
			Type:        "string",
			Description: "A [tax code](https://docs.stripe.com/tax/tax-categories) ID.",
		},
		"price_data.unit_amount_decimal": {
			Type:        "string",
			Description: "Same as `unit_amount`, but accepts a decimal value in cents (or local equivalent) with at most 12 decimal places. Only one of `unit_amount` and `unit_amount_decimal` can be set.",
			Format:      "decimal",
		},
		"amount": {
			Type:        "integer",
			Description: "The integer amount in cents (or local equivalent) of the charge to be applied to the upcoming invoice. If you want to apply a credit to the customer's account, pass a negative amount.",
		},
		"quantity": {
			Type:        "integer",
			Description: "Non-negative integer. The quantity of units for the line item.",
		},
		"tax_rates": {
			Type:        "array",
			Description: "The tax rates which apply to the line item. When set, the `default_tax_rates` on the invoice do not apply to this line item. Pass an empty string to remove previously-defined tax rates.",
		},
		"description": {
			Type:        "string",
			Description: "An arbitrary string which you can attach to the invoice item. The description is displayed in the invoice for easy tracking.",
		},
		"price_data.product_data.unit_label": {
			Type:        "string",
			Description: "A label that represents units of this product. When set, this will be included in customers' receipts, invoices, Checkout, and the customer portal.",
		},
		"price_data.product_data.description": {
			Type:        "string",
			Description: "The product's description, meant to be displayable to the customer. Use this field to optionally store a long form explanation of the product being sold for your own rendering purposes.",
		},
		"pricing.price": {
			Type:        "string",
			Description: "The ID of the price object.",
		},
	},
}

var V1PaymentRecordsReportPaymentAttemptInformational = resource.OperationSpec{
	Name:    "report_payment_attempt_informational",
	Path:    "/v1/payment_records/{id}/report_payment_attempt_informational",
	Method:  "POST",
	Summary: "Report payment attempt informational",
	Params: map[string]*resource.ParamSpec{
		"description": {
			Type:        "string",
			Description: "An arbitrary string attached to the object. Often useful for displaying to users.",
		},
		"customer_details.customer": {
			Type:        "string",
			Description: "The customer who made the payment.",
		},
		"customer_details.email": {
			Type:        "string",
			Description: "The customer's phone number.",
		},
		"customer_details.name": {
			Type:        "string",
			Description: "The customer's name.",
		},
		"customer_details.phone": {
			Type:        "string",
			Description: "The customer's phone number.",
		},
	},
}

var V1PaymentRecordsReportRefund = resource.OperationSpec{
	Name:    "report_refund",
	Path:    "/v1/payment_records/{id}/report_refund",
	Method:  "POST",
	Summary: "Report a refund",
	Params: map[string]*resource.ParamSpec{
		"initiated_at": {
			Type:        "integer",
			Description: "When the reported refund was initiated. Measured in seconds since the Unix epoch.",
			Format:      "unix-time",
		},
		"outcome": {
			Type:        "string",
			Description: "The outcome of the reported refund.",
			Required:    true,
			Enum: []resource.EnumSpec{
				{Value: "refunded"},
			},
		},
		"processor_details.custom.refund_reference": {
			Type:        "string",
			Description: "A reference to the external refund. This field must be unique across all refunds.",
			Required:    true,
		},
		"processor_details.type": {
			Type:        "string",
			Description: "The type of the processor details. An additional hash is included on processor_details with a name matching this value. It contains additional information specific to the processor.",
			Required:    true,
			Enum: []resource.EnumSpec{
				{Value: "custom"},
			},
		},
		"refunded.refunded_at": {
			Type:        "integer",
			Description: "When the reported refund completed. Measured in seconds since the Unix epoch.",
			Required:    true,
			Format:      "unix-time",
		},
		"amount.currency": {
			Type:        "string",
			Description: "Three-letter [ISO currency code](https://www.iso.org/iso-4217-currency-codes.html), in lowercase. Must be a [supported currency](https://stripe.com/docs/currencies).",
			Required:    true,
			Format:      "currency",
		},
		"amount.value": {
			Type:        "integer",
			Description: "A positive integer representing the amount in the currency's [minor unit](https://docs.stripe.com/currencies#zero-decimal). For example, `100` can represent 1 USD or 100 JPY.",
			Required:    true,
		},
	},
}

var V1PaymentRecordsReportPayment = resource.OperationSpec{
	Name:    "report_payment",
	Path:    "/v1/payment_records/report_payment",
	Method:  "POST",
	Summary: "Report a payment",
	Params: map[string]*resource.ParamSpec{
		"shipping_details.address.state": {
			Type:        "string",
			Description: "State, county, province, or region ([ISO 3166-2](https://en.wikipedia.org/wiki/ISO_3166-2)).",
		},
		"shipping_details.address.line2": {
			Type:        "string",
			Description: "Address line 2, such as the apartment, suite, unit, or building.",
		},
		"payment_method_details.billing_details.email": {
			Type:        "string",
			Description: "The billing email associated with the method of payment.",
		},
		"payment_method_details.type": {
			Type:        "string",
			Description: "The type of the payment method details. An additional hash is included on the payment_method_details with a name matching this value. It contains additional information specific to the type.",
			Enum: []resource.EnumSpec{
				{Value: "custom"},
			},
		},
		"customer_details.customer": {
			Type:        "string",
			Description: "The customer who made the payment.",
		},
		"payment_method_details.billing_details.name": {
			Type:        "string",
			Description: "The billing name associated with the method of payment.",
		},
		"payment_method_details.billing_details.phone": {
			Type:        "string",
			Description: "The billing phone number associated with the method of payment.",
		},
		"processor_details.custom.payment_reference": {
			Type:        "string",
			Description: "An opaque string for manual reconciliation of this payment, for example a check number or a payment processor ID.",
			Required:    true,
		},
		"shipping_details.address.city": {
			Type:        "string",
			Description: "City, district, suburb, town, or village.",
		},
		"shipping_details.address.postal_code": {
			Type:        "string",
			Description: "ZIP or postal code.",
		},
		"amount_requested.currency": {
			Type:        "string",
			Description: "Three-letter [ISO currency code](https://www.iso.org/iso-4217-currency-codes.html), in lowercase. Must be a [supported currency](https://stripe.com/docs/currencies).",
			Required:    true,
			Format:      "currency",
		},
		"outcome": {
			Type:        "string",
			Description: "The outcome of the reported payment.",
			Enum: []resource.EnumSpec{
				{Value: "failed"},
				{Value: "guaranteed"},
			},
		},
		"payment_method_details.billing_details.address.state": {
			Type:        "string",
			Description: "State, county, province, or region ([ISO 3166-2](https://en.wikipedia.org/wiki/ISO_3166-2)).",
		},
		"description": {
			Type:        "string",
			Description: "An arbitrary string attached to the object. Often useful for displaying to users.",
		},
		"payment_method_details.billing_details.address.country": {
			Type:        "string",
			Description: "Two-letter country code ([ISO 3166-1 alpha-2](https://en.wikipedia.org/wiki/ISO_3166-1_alpha-2)).",
		},
		"payment_method_details.billing_details.address.line1": {
			Type:        "string",
			Description: "Address line 1, such as the street, PO Box, or company name.",
		},
		"payment_method_details.billing_details.address.line2": {
			Type:        "string",
			Description: "Address line 2, such as the apartment, suite, unit, or building.",
		},
		"guaranteed.guaranteed_at": {
			Type:        "integer",
			Description: "When the reported payment was guaranteed. Measured in seconds since the Unix epoch.",
			Required:    true,
			Format:      "unix-time",
		},
		"shipping_details.phone": {
			Type:        "string",
			Description: "The shipping recipient's phone number.",
		},
		"amount_requested.value": {
			Type:        "integer",
			Description: "A positive integer representing the amount in the currency's [minor unit](https://docs.stripe.com/currencies#zero-decimal). For example, `100` can represent 1 USD or 100 JPY.",
			Required:    true,
		},
		"customer_details.email": {
			Type:        "string",
			Description: "The customer's phone number.",
		},
		"customer_details.phone": {
			Type:        "string",
			Description: "The customer's phone number.",
		},
		"initiated_at": {
			Type:        "integer",
			Description: "When the reported payment was initiated. Measured in seconds since the Unix epoch.",
			Required:    true,
			Format:      "unix-time",
		},
		"processor_details.type": {
			Type:        "string",
			Description: "The type of the processor details. An additional hash is included on processor_details with a name matching this value. It contains additional information specific to the processor.",
			Required:    true,
			Enum: []resource.EnumSpec{
				{Value: "custom"},
			},
		},
		"shipping_details.address.line1": {
			Type:        "string",
			Description: "Address line 1, such as the street, PO Box, or company name.",
		},
		"shipping_details.name": {
			Type:        "string",
			Description: "The shipping recipient's name.",
		},
		"failed.failed_at": {
			Type:        "integer",
			Description: "When the reported payment failed. Measured in seconds since the Unix epoch.",
			Required:    true,
			Format:      "unix-time",
		},
		"payment_method_details.billing_details.address.city": {
			Type:        "string",
			Description: "City, district, suburb, town, or village.",
		},
		"payment_method_details.custom.display_name": {
			Type:        "string",
			Description: "Display name for the custom (user-defined) payment method type used to make this payment.",
		},
		"payment_method_details.custom.type": {
			Type:        "string",
			Description: "The custom payment method type associated with this payment.",
		},
		"shipping_details.address.country": {
			Type:        "string",
			Description: "Two-letter country code ([ISO 3166-1 alpha-2](https://en.wikipedia.org/wiki/ISO_3166-1_alpha-2)).",
		},
		"customer_details.name": {
			Type:        "string",
			Description: "The customer's name.",
		},
		"payment_method_details.billing_details.address.postal_code": {
			Type:        "string",
			Description: "ZIP or postal code.",
		},
		"payment_method_details.payment_method": {
			Type:        "string",
			Description: "ID of the Stripe Payment Method used to make this payment.",
		},
		"customer_presence": {
			Type:        "string",
			Description: "Indicates whether the customer was present in your checkout flow during this payment.",
			Enum: []resource.EnumSpec{
				{Value: "off_session"},
				{Value: "on_session"},
			},
		},
	},
}

var V1PaymentRecordsRetrieve = resource.OperationSpec{
	Name:    "retrieve",
	Path:    "/v1/payment_records/{id}",
	Method:  "GET",
	Summary: "Retrieve a Payment Record",
}

var V1PaymentRecordsReportPaymentAttempt = resource.OperationSpec{
	Name:    "report_payment_attempt",
	Path:    "/v1/payment_records/{id}/report_payment_attempt",
	Method:  "POST",
	Summary: "Report a payment attempt",
	Params: map[string]*resource.ParamSpec{
		"initiated_at": {
			Type:        "integer",
			Description: "When the reported payment was initiated. Measured in seconds since the Unix epoch.",
			Required:    true,
			Format:      "unix-time",
		},
		"payment_method_details.billing_details.address.postal_code": {
			Type:        "string",
			Description: "ZIP or postal code.",
		},
		"payment_method_details.billing_details.address.state": {
			Type:        "string",
			Description: "State, county, province, or region ([ISO 3166-2](https://en.wikipedia.org/wiki/ISO_3166-2)).",
		},
		"shipping_details.phone": {
			Type:        "string",
			Description: "The shipping recipient's phone number.",
		},
		"outcome": {
			Type:        "string",
			Description: "The outcome of the reported payment.",
			Enum: []resource.EnumSpec{
				{Value: "failed"},
				{Value: "guaranteed"},
			},
		},
		"failed.failed_at": {
			Type:        "integer",
			Description: "When the reported payment failed. Measured in seconds since the Unix epoch.",
			Required:    true,
			Format:      "unix-time",
		},
		"guaranteed.guaranteed_at": {
			Type:        "integer",
			Description: "When the reported payment was guaranteed. Measured in seconds since the Unix epoch.",
			Required:    true,
			Format:      "unix-time",
		},
		"payment_method_details.billing_details.address.country": {
			Type:        "string",
			Description: "Two-letter country code ([ISO 3166-1 alpha-2](https://en.wikipedia.org/wiki/ISO_3166-1_alpha-2)).",
		},
		"payment_method_details.billing_details.address.line1": {
			Type:        "string",
			Description: "Address line 1, such as the street, PO Box, or company name.",
		},
		"payment_method_details.custom.type": {
			Type:        "string",
			Description: "The custom payment method type associated with this payment.",
		},
		"shipping_details.address.line1": {
			Type:        "string",
			Description: "Address line 1, such as the street, PO Box, or company name.",
		},
		"shipping_details.address.line2": {
			Type:        "string",
			Description: "Address line 2, such as the apartment, suite, unit, or building.",
		},
		"shipping_details.address.postal_code": {
			Type:        "string",
			Description: "ZIP or postal code.",
		},
		"shipping_details.name": {
			Type:        "string",
			Description: "The shipping recipient's name.",
		},
		"description": {
			Type:        "string",
			Description: "An arbitrary string attached to the object. Often useful for displaying to users.",
		},
		"payment_method_details.billing_details.email": {
			Type:        "string",
			Description: "The billing email associated with the method of payment.",
		},
		"payment_method_details.billing_details.address.city": {
			Type:        "string",
			Description: "City, district, suburb, town, or village.",
		},
		"payment_method_details.billing_details.address.line2": {
			Type:        "string",
			Description: "Address line 2, such as the apartment, suite, unit, or building.",
		},
		"payment_method_details.custom.display_name": {
			Type:        "string",
			Description: "Display name for the custom (user-defined) payment method type used to make this payment.",
		},
		"payment_method_details.type": {
			Type:        "string",
			Description: "The type of the payment method details. An additional hash is included on the payment_method_details with a name matching this value. It contains additional information specific to the type.",
			Enum: []resource.EnumSpec{
				{Value: "custom"},
			},
		},
		"shipping_details.address.state": {
			Type:        "string",
			Description: "State, county, province, or region ([ISO 3166-2](https://en.wikipedia.org/wiki/ISO_3166-2)).",
		},
		"shipping_details.address.country": {
			Type:        "string",
			Description: "Two-letter country code ([ISO 3166-1 alpha-2](https://en.wikipedia.org/wiki/ISO_3166-1_alpha-2)).",
		},
		"payment_method_details.billing_details.name": {
			Type:        "string",
			Description: "The billing name associated with the method of payment.",
		},
		"payment_method_details.billing_details.phone": {
			Type:        "string",
			Description: "The billing phone number associated with the method of payment.",
		},
		"payment_method_details.payment_method": {
			Type:        "string",
			Description: "ID of the Stripe Payment Method used to make this payment.",
		},
		"shipping_details.address.city": {
			Type:        "string",
			Description: "City, district, suburb, town, or village.",
		},
	},
}

var V1PaymentRecordsReportPaymentAttemptCanceled = resource.OperationSpec{
	Name:    "report_payment_attempt_canceled",
	Path:    "/v1/payment_records/{id}/report_payment_attempt_canceled",
	Method:  "POST",
	Summary: "Report payment attempt canceled",
	Params: map[string]*resource.ParamSpec{
		"canceled_at": {
			Type:        "integer",
			Description: "When the reported payment was canceled. Measured in seconds since the Unix epoch.",
			Required:    true,
			Format:      "unix-time",
		},
	},
}

var V1PaymentRecordsReportPaymentAttemptFailed = resource.OperationSpec{
	Name:    "report_payment_attempt_failed",
	Path:    "/v1/payment_records/{id}/report_payment_attempt_failed",
	Method:  "POST",
	Summary: "Report payment attempt failed",
	Params: map[string]*resource.ParamSpec{
		"failed_at": {
			Type:        "integer",
			Description: "When the reported payment failed. Measured in seconds since the Unix epoch.",
			Required:    true,
			Format:      "unix-time",
		},
	},
}

var V1PaymentRecordsReportPaymentAttemptGuaranteed = resource.OperationSpec{
	Name:    "report_payment_attempt_guaranteed",
	Path:    "/v1/payment_records/{id}/report_payment_attempt_guaranteed",
	Method:  "POST",
	Summary: "Report payment attempt guaranteed",
	Params: map[string]*resource.ParamSpec{
		"guaranteed_at": {
			Type:        "integer",
			Description: "When the reported payment was guaranteed. Measured in seconds since the Unix epoch.",
			Required:    true,
			Format:      "unix-time",
		},
	},
}

var V1CouponsDelete = resource.OperationSpec{
	Name:    "delete",
	Path:    "/v1/coupons/{coupon}",
	Method:  "DELETE",
	Summary: "Delete a coupon",
}

var V1CouponsList = resource.OperationSpec{
	Name:    "list",
	Path:    "/v1/coupons",
	Method:  "GET",
	Summary: "List all coupons",
	Params: map[string]*resource.ParamSpec{
		"created": {
			Type:        "integer",
			Description: "A filter on the list, based on the object `created` field. The value can be a string with an integer Unix timestamp, or it can be a dictionary with a number of different query options.",
		},
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

var V1CouponsRetrieve = resource.OperationSpec{
	Name:    "retrieve",
	Path:    "/v1/coupons/{coupon}",
	Method:  "GET",
	Summary: "Retrieve a coupon",
}

var V1CouponsCreate = resource.OperationSpec{
	Name:    "create",
	Path:    "/v1/coupons",
	Method:  "POST",
	Summary: "Create a coupon",
	Params: map[string]*resource.ParamSpec{
		"applies_to.products": {
			Type:        "array",
			Description: "An array of Product IDs that this Coupon will apply to.",
		},
		"name": {
			Type:        "string",
			Description: "Name of the coupon displayed to customers on, for instance invoices, or receipts. By default the `id` is shown if `name` is not set.",
		},
		"duration": {
			Type:        "string",
			Description: "Specifies how long the discount will be in effect if used on a subscription. Defaults to `once`.",
			Enum: []resource.EnumSpec{
				{Value: "forever"},
				{Value: "once"},
				{Value: "repeating"},
			},
		},
		"amount_off": {
			Type:        "integer",
			Description: "A positive integer representing the amount to subtract from an invoice total (required if `percent_off` is not passed).",
		},
		"currency": {
			Type:        "string",
			Description: "Three-letter [ISO code for the currency](https://stripe.com/docs/currencies) of the `amount_off` parameter (required if `amount_off` is passed).",
			Format:      "currency",
		},
		"max_redemptions": {
			Type:        "integer",
			Description: "A positive integer specifying the number of times the coupon can be redeemed before it's no longer valid. For example, you might have a 50% off coupon that the first 20 readers of your blog can use.",
		},
		"redeem_by": {
			Type:        "integer",
			Description: "Unix timestamp specifying the last time at which the coupon can be redeemed (cannot be set to more than 5 years in the future). After the redeem_by date, the coupon can no longer be applied to new customers.",
			Format:      "unix-time",
		},
		"duration_in_months": {
			Type:        "integer",
			Description: "Required only if `duration` is `repeating`, in which case it must be a positive integer that specifies the number of months the discount will be in effect.",
		},
		"id": {
			Type:        "string",
			Description: "Unique string of your choice that will be used to identify this coupon when applying it to a customer. If you don't want to specify a particular code, you can leave the ID blank and we'll generate a random code for you.",
		},
		"percent_off": {
			Type:        "number",
			Description: "A positive float larger than 0, and smaller or equal to 100, that represents the discount the coupon will apply (required if `amount_off` is not passed).",
		},
	},
}

var V1CouponsUpdate = resource.OperationSpec{
	Name:    "update",
	Path:    "/v1/coupons/{coupon}",
	Method:  "POST",
	Summary: "Update a coupon",
	Params: map[string]*resource.ParamSpec{
		"name": {
			Type:        "string",
			Description: "Name of the coupon displayed to customers on, for instance invoices, or receipts. By default the `id` is shown if `name` is not set.",
		},
	},
}

var V1CardsDelete = resource.OperationSpec{
	Name:    "delete",
	Path:    "/v1/accounts/{account}/external_accounts/{id}",
	Method:  "DELETE",
	Summary: "Delete an external account",
}

var V1CardsUpdate = resource.OperationSpec{
	Name:   "update",
	Path:   "/v1/accounts/{account}/external_accounts/{id}",
	Method: "POST",
	Params: map[string]*resource.ParamSpec{
		"account_holder_name": {
			Type:        "string",
			Description: "The name of the person or business that owns the bank account.",
		},
		"address_country": {
			Type:        "string",
			Description: "Billing address country, if provided when creating card.",
		},
		"account_type": {
			Type:        "string",
			Description: "The bank account type. This can only be `checking` or `savings` in most countries. In Japan, this can only be `futsu` or `toza`.",
			Enum: []resource.EnumSpec{
				{Value: "checking"},
				{Value: "futsu"},
				{Value: "savings"},
				{Value: "toza"},
			},
		},
		"address_zip": {
			Type:        "string",
			Description: "ZIP or postal code.",
		},
		"default_for_currency": {
			Type:        "boolean",
			Description: "When set to true, this becomes the default external account for its currency.",
		},
		"name": {
			Type:        "string",
			Description: "Cardholder name.",
		},
		"account_holder_type": {
			Type:        "string",
			Description: "The type of entity that holds the account. This can be either `individual` or `company`.",
			Enum: []resource.EnumSpec{
				{Value: "company"},
				{Value: "individual"},
			},
		},
		"address_city": {
			Type:        "string",
			Description: "City/District/Suburb/Town/Village.",
		},
		"address_line1": {
			Type:        "string",
			Description: "Address line 1 (Street address/PO Box/Company name).",
		},
		"exp_month": {
			Type:        "string",
			Description: "Two digit number representing the card’s expiration month.",
		},
		"address_state": {
			Type:        "string",
			Description: "State/County/Province/Region.",
		},
		"documents.bank_account_ownership_verification.files": {
			Type:        "array",
			Description: "One or more document ids returned by a [file upload](https://api.stripe.com#create_file) with a `purpose` value of `account_requirement`.",
		},
		"exp_year": {
			Type:        "string",
			Description: "Four digit number representing the card’s expiration year.",
		},
		"address_line2": {
			Type:        "string",
			Description: "Address line 2 (Apartment/Suite/Unit/Building).",
		},
	},
}

var V1TokensRetrieve = resource.OperationSpec{
	Name:    "retrieve",
	Path:    "/v1/tokens/{token}",
	Method:  "GET",
	Summary: "Retrieve a token",
}

var V1TokensCreate = resource.OperationSpec{
	Name:    "create",
	Path:    "/v1/tokens",
	Method:  "POST",
	Summary: "Create a CVC update token",
	Params: map[string]*resource.ParamSpec{
		"account.company.address.state": {
			Type:        "string",
			Description: "State, county, province, or region ([ISO 3166-2](https://en.wikipedia.org/wiki/ISO_3166-2)).",
		},
		"person.registered_address.state": {
			Type:        "string",
			Description: "State, county, province, or region ([ISO 3166-2](https://en.wikipedia.org/wiki/ISO_3166-2)).",
		},
		"account.company.address_kana.city": {
			Type:        "string",
			Description: "City or ward.",
		},
		"account.company.address_kanji.postal_code": {
			Type:        "string",
			Description: "Postal code.",
		},
		"account.company.directors_provided": {
			Type:        "boolean",
			Description: "Whether the company's directors have been provided. Set this Boolean to `true` after creating all the company's directors with [the Persons API](/api/persons) for accounts with a `relationship.director` requirement. This value is not automatically set to `true` after creating directors, so it needs to be updated to indicate all directors have been provided.",
		},
		"person.us_cfpb_data.self_identified_gender": {
			Type:        "string",
			Description: "The persons self-identified gender",
		},
		"account.individual.address_kana.state": {
			Type:        "string",
			Description: "Prefecture.",
		},
		"account.individual.address.line2": {
			Type:        "string",
			Description: "Address line 2, such as the apartment, suite, unit, or building.",
		},
		"account.individual.email": {
			Type:        "string",
			Description: "The individual's email address.",
		},
		"bank_account.payment_method": {
			Type:        "string",
			Description: "The ID of a Payment Method with a `type` of `us_bank_account`. The Payment Method's bank account information will be copied and returned as a Bank Account Token. This parameter is exclusive with respect to all other parameters in the `bank_account` hash. You must include the top-level `customer` parameter if the Payment Method is attached to a `Customer` object. If the Payment Method is not attached to a `Customer` object, it will be consumed and cannot be used again. You may not use Payment Methods which were created by a Setup Intent with `attach_to_self=true`.",
		},
		"person.verification.additional_document.back": {
			Type:        "string",
			Description: "The back of an ID returned by a [file upload](https://api.stripe.com#create_file) with a `purpose` value of `identity_document`. The uploaded file needs to be a color image (smaller than 8,000px by 8,000px), in JPG, PNG, or PDF format, and less than 10 MB in size.",
		},
		"account.company.vat_id": {
			Type:        "string",
			Description: "The VAT number of the company.",
		},
		"account.company.directorship_declaration.date": {
			Type:        "integer",
			Description: "The Unix timestamp marking when the directorship declaration attestation was made.",
			Format:      "unix-time",
		},
		"account.company.address.country": {
			Type:        "string",
			Description: "Two-letter country code ([ISO 3166-1 alpha-2](https://en.wikipedia.org/wiki/ISO_3166-1_alpha-2)).",
		},
		"account.company.name_kanji": {
			Type:        "string",
			Description: "The Kanji variation of the company's legal name (Japan only).",
		},
		"account.individual.address_kana.town": {
			Type:        "string",
			Description: "Town or cho-me.",
		},
		"account.individual.verification.document.back": {
			Type:        "string",
			Description: "The back of an ID returned by a [file upload](https://api.stripe.com#create_file) with a `purpose` value of `identity_document`. The uploaded file needs to be a color image (smaller than 8,000px by 8,000px), in JPG, PNG, or PDF format, and less than 10 MB in size.",
		},
		"person.address_kanji.town": {
			Type:        "string",
			Description: "Town or cho-me.",
		},
		"person.relationship.authorizer": {
			Type:        "boolean",
			Description: "Whether the person is the authorizer of the account's representative.",
		},
		"person.id_number_secondary": {
			Type:        "string",
			Description: "The person's secondary ID number, as appropriate for their country, will be used for enhanced verification checks. In Thailand, this would be the laser code found on the back of an ID card. Instead of the number itself, you can also provide a [PII token provided by Stripe.js](https://docs.stripe.com/js/tokens/create_token?type=pii).",
		},
		"person.registered_address.postal_code": {
			Type:        "string",
			Description: "ZIP or postal code.",
		},
		"person.ssn_last_4": {
			Type:        "string",
			Description: "The last four digits of the person's Social Security number (U.S. only).",
		},
		"account.company.export_purpose_code": {
			Type:        "string",
			Description: "The purpose code to use for export transactions (India only).",
		},
		"account.company.address_kanji.state": {
			Type:        "string",
			Description: "Prefecture.",
		},
		"account.individual.gender": {
			Type:        "string",
			Description: "The individual's gender",
		},
		"account.company.address_kanji.town": {
			Type:        "string",
			Description: "Town or cho-me.",
		},
		"account.individual.address_kana.line2": {
			Type:        "string",
			Description: "Building details.",
		},
		"account.individual.address_kanji.state": {
			Type:        "string",
			Description: "Prefecture.",
		},
		"account.individual.ssn_last_4": {
			Type:        "string",
			Description: "The last four digits of the individual's Social Security Number (U.S. only).",
		},
		"account.individual.last_name_kanji": {
			Type:        "string",
			Description: "The Kanji variation of the individual's last name (Japan only).",
		},
		"cvc_update.cvc": {
			Type:        "string",
			Description: "The CVC value, in string form.",
			Required:    true,
		},
		"account.company.name_kana": {
			Type:        "string",
			Description: "The Kana variation of the company's legal name (Japan only).",
		},
		"account.company.tax_id_registrar": {
			Type:        "string",
			Description: "The jurisdiction in which the `tax_id` is registered (Germany-based companies only).",
		},
		"bank_account.routing_number": {
			Type:        "string",
			Description: "The routing number, sort code, or other country-appropriate institution number for the bank account. For US bank accounts, this is required and should be the ACH routing number, not the wire routing number. If you are providing an IBAN for `account_number`, this field is not required.",
		},
		"bank_account.account_holder_name": {
			Type:        "string",
			Description: "The name of the person or business that owns the bank account. This field is required when attaching the bank account to a `Customer` object.",
		},
		"person.address_kanji.postal_code": {
			Type:        "string",
			Description: "Postal code.",
		},
		"person.relationship.owner": {
			Type:        "boolean",
			Description: "Whether the person is an owner of the account’s legal entity.",
		},
		"account.company.representative_declaration.user_agent": {
			Type:        "string",
			Description: "The user agent of the browser from which the representative declaration attestation was made.",
		},
		"account.company.address_kanji.country": {
			Type:        "string",
			Description: "Two-letter country code ([ISO 3166-1 alpha-2](https://en.wikipedia.org/wiki/ISO_3166-1_alpha-2)).",
		},
		"account.company.address_kanji.line1": {
			Type:        "string",
			Description: "Block or building number.",
		},
		"bank_account.currency": {
			Type:        "string",
			Description: "The currency the bank account is in. This must be a country/currency pairing that [Stripe supports.](https://docs.stripe.com/payouts)",
			Format:      "currency",
		},
		"person.maiden_name": {
			Type:        "string",
			Description: "The person's maiden name.",
		},
		"person.first_name": {
			Type:        "string",
			Description: "The person's first name.",
		},
		"account.company.registration_number": {
			Type:        "string",
			Description: "The identification number given to a company when it is registered or incorporated, if distinct from the identification number used for filing taxes. (Examples are the CIN for companies and LLP IN for partnerships in India, and the Company Registration Number in Hong Kong).",
		},
		"account.individual.id_number_secondary": {
			Type:        "string",
			Description: "The government-issued secondary ID number of the individual, as appropriate for the representative's country, will be used for enhanced verification checks. In Thailand, this would be the laser code found on the back of an ID card. Instead of the number itself, you can also provide a [PII token created with Stripe.js](/js/tokens/create_token?type=pii).",
		},
		"account.individual.first_name_kanji": {
			Type:        "string",
			Description: "The Kanji variation of the individual's first name (Japan only).",
		},
		"customer": {
			Type:        "string",
			Description: "Create a token for the customer, which is owned by the application's account. You can only use this with an [OAuth access token](https://docs.stripe.com/connect/standard-accounts) or [Stripe-Account header](https://docs.stripe.com/connect/authentication). Learn more about [cloning saved payment methods](https://docs.stripe.com/connect/cloning-saved-payment-methods).",
		},
		"person.address.state": {
			Type:        "string",
			Description: "State, county, province, or region ([ISO 3166-2](https://en.wikipedia.org/wiki/ISO_3166-2)).",
		},
		"person.political_exposure": {
			Type:        "string",
			Description: "Indicates if the person or any of their representatives, family members, or other closely related persons, declares that they hold or have held an important public job or function, in any jurisdiction.",
			Enum: []resource.EnumSpec{
				{Value: "existing"},
				{Value: "none"},
			},
		},
		"person.verification.document.front": {
			Type:        "string",
			Description: "The front of an ID returned by a [file upload](https://api.stripe.com#create_file) with a `purpose` value of `identity_document`. The uploaded file needs to be a color image (smaller than 8,000px by 8,000px), in JPG, PNG, or PDF format, and less than 10 MB in size.",
		},
		"account.company.ownership_exemption_reason": {
			Type:        "string",
			Description: "This value is used to determine if a business is exempt from providing ultimate beneficial owners. See [this support article](https://support.stripe.com/questions/exemption-from-providing-ownership-details) and [changelog](https://docs.stripe.com/changelog/acacia/2025-01-27/ownership-exemption-reason-accounts-api) for more details.",
			Enum: []resource.EnumSpec{
				{Value: "qualified_entity_exceeds_ownership_threshold"},
				{Value: "qualifies_as_financial_institution"},
			},
		},
		"account.company.address_kana.line2": {
			Type:        "string",
			Description: "Building details.",
		},
		"account.company.address_kanji.city": {
			Type:        "string",
			Description: "City or ward.",
		},
		"account.tos_shown_and_accepted": {
			Type:        "boolean",
			Description: "Whether the user described by the data in the token has been shown [the Stripe Connected Account Agreement](/connect/account-tokens#stripe-connected-account-agreement). When creating an account token to create a new Connect account, this value must be `true`.",
		},
		"person.id_number": {
			Type:        "string",
			Description: "The person's ID number, as appropriate for their country. For example, a social security number in the U.S., social insurance number in Canada, etc. Instead of the number itself, you can also provide a [PII token provided by Stripe.js](https://docs.stripe.com/js/tokens/create_token?type=pii).",
		},
		"person.us_cfpb_data.race_details.race_other": {
			Type:        "string",
			Description: "Please specify your race, when other is selected.",
		},
		"account.company.ownership_declaration.user_agent": {
			Type:        "string",
			Description: "The user agent of the browser from which the beneficial owner attestation was made.",
		},
		"account.company.directorship_declaration.user_agent": {
			Type:        "string",
			Description: "The user agent of the browser from which the directorship declaration attestation was made.",
		},
		"account.company.representative_declaration.date": {
			Type:        "integer",
			Description: "The Unix timestamp marking when the representative declaration attestation was made.",
			Format:      "unix-time",
		},
		"account.individual.address_kanji.city": {
			Type:        "string",
			Description: "City or ward.",
		},
		"bank_account.account_holder_type": {
			Type:        "string",
			Description: "The type of entity that holds the account. It can be `company` or `individual`. This field is required when attaching the bank account to a `Customer` object.",
			Enum: []resource.EnumSpec{
				{Value: "company"},
				{Value: "individual"},
			},
		},
		"person.documents.passport.files": {
			Type:        "array",
			Description: "One or more document ids returned by a [file upload](https://api.stripe.com#create_file) with a `purpose` value of `account_requirement`.",
		},
		"person.address_kana.line2": {
			Type:        "string",
			Description: "Building details.",
		},
		"person.first_name_kana": {
			Type:        "string",
			Description: "The Kana variation of the person's first name (Japan only).",
		},
		"account.company.export_license_id": {
			Type:        "string",
			Description: "The export license ID number of the company, also referred as Import Export Code (India only).",
		},
		"account.individual.registered_address.line2": {
			Type:        "string",
			Description: "Address line 2, such as the apartment, suite, unit, or building.",
		},
		"account.individual.registered_address.state": {
			Type:        "string",
			Description: "State, county, province, or region ([ISO 3166-2](https://en.wikipedia.org/wiki/ISO_3166-2)).",
		},
		"person.address.line1": {
			Type:        "string",
			Description: "Address line 1, such as the street, PO Box, or company name.",
		},
		"account.company.address_kana.country": {
			Type:        "string",
			Description: "Two-letter country code ([ISO 3166-1 alpha-2](https://en.wikipedia.org/wiki/ISO_3166-1_alpha-2)).",
		},
		"account.individual.first_name": {
			Type:        "string",
			Description: "The individual's first name.",
		},
		"account.individual.relationship.owner": {
			Type:        "boolean",
			Description: "Whether the person is an owner of the account’s legal entity.",
		},
		"person.relationship.director": {
			Type:        "boolean",
			Description: "Whether the person is a director of the account's legal entity. Directors are typically members of the governing board of the company, or responsible for ensuring the company meets its regulatory obligations.",
		},
		"person.registered_address.line1": {
			Type:        "string",
			Description: "Address line 1, such as the street, PO Box, or company name.",
		},
		"account.company.directorship_declaration.ip": {
			Type:        "string",
			Description: "The IP address from which the directorship declaration attestation was made.",
		},
		"account.company.address.line2": {
			Type:        "string",
			Description: "Address line 2, such as the apartment, suite, unit, or building.",
		},
		"account.individual.relationship.title": {
			Type:        "string",
			Description: "The person's title (e.g., CEO, Support Engineer).",
		},
		"person.address_kana.country": {
			Type:        "string",
			Description: "Two-letter country code ([ISO 3166-1 alpha-2](https://en.wikipedia.org/wiki/ISO_3166-1_alpha-2)).",
		},
		"person.phone": {
			Type:        "string",
			Description: "The person's phone number.",
		},
		"account.company.ownership_declaration_shown_and_signed": {
			Type:        "boolean",
			Description: "Whether the user described by the data in the token has been shown the Ownership Declaration and indicated that it is correct.",
		},
		"account.individual.registered_address.line1": {
			Type:        "string",
			Description: "Address line 1, such as the street, PO Box, or company name.",
		},
		"person.additional_tos_acceptances.account.ip": {
			Type:        "string",
			Description: "The IP address from which the account representative accepted the service agreement.",
		},
		"person.address_kana.town": {
			Type:        "string",
			Description: "Town or cho-me.",
		},
		"account.business_type": {
			Type:        "string",
			Description: "The business type.",
			Enum: []resource.EnumSpec{
				{Value: "company"},
				{Value: "government_entity"},
				{Value: "individual"},
				{Value: "non_profit"},
			},
		},
		"account.individual.address.postal_code": {
			Type:        "string",
			Description: "ZIP or postal code.",
		},
		"account.individual.verification.additional_document.back": {
			Type:        "string",
			Description: "The back of an ID returned by a [file upload](https://api.stripe.com#create_file) with a `purpose` value of `identity_document`. The uploaded file needs to be a color image (smaller than 8,000px by 8,000px), in JPG, PNG, or PDF format, and less than 10 MB in size.",
		},
		"person.address.postal_code": {
			Type:        "string",
			Description: "ZIP or postal code.",
		},
		"person.address_kana.state": {
			Type:        "string",
			Description: "Prefecture.",
		},
		"person.registered_address.line2": {
			Type:        "string",
			Description: "Address line 2, such as the apartment, suite, unit, or building.",
		},
		"account.company.address.line1": {
			Type:        "string",
			Description: "Address line 1, such as the street, PO Box, or company name.",
		},
		"account.individual.address.country": {
			Type:        "string",
			Description: "Two-letter country code ([ISO 3166-1 alpha-2](https://en.wikipedia.org/wiki/ISO_3166-1_alpha-2)).",
		},
		"account.individual.address_kanji.town": {
			Type:        "string",
			Description: "Town or cho-me.",
		},
		"person.documents.company_authorization.files": {
			Type:        "array",
			Description: "One or more document ids returned by a [file upload](https://api.stripe.com#create_file) with a `purpose` value of `account_requirement`.",
		},
		"person.relationship.representative": {
			Type:        "boolean",
			Description: "Whether the person is authorized as the primary representative of the account. This is the person nominated by the business to provide information about themselves, and general information about the account. There can only be one representative at any given time. At the time the account is created, this person should be set to the person responsible for opening the account.",
		},
		"account.company.name": {
			Type:        "string",
			Description: "The company's legal name.",
		},
		"account.individual.address.line1": {
			Type:        "string",
			Description: "Address line 1, such as the street, PO Box, or company name.",
		},
		"account.individual.full_name_aliases": {
			Type:        "array",
			Description: "A list of alternate names or aliases that the individual is known by.",
		},
		"account.individual.maiden_name": {
			Type:        "string",
			Description: "The individual's maiden name.",
		},
		"account.individual.verification.additional_document.front": {
			Type:        "string",
			Description: "The front of an ID returned by a [file upload](https://api.stripe.com#create_file) with a `purpose` value of `identity_document`. The uploaded file needs to be a color image (smaller than 8,000px by 8,000px), in JPG, PNG, or PDF format, and less than 10 MB in size.",
		},
		"person.address.country": {
			Type:        "string",
			Description: "Two-letter country code ([ISO 3166-1 alpha-2](https://en.wikipedia.org/wiki/ISO_3166-1_alpha-2)).",
		},
		"person.relationship.percent_ownership": {
			Type:        "number",
			Description: "The percent owned by the person of the account's legal entity.",
		},
		"account.company.ownership_declaration.ip": {
			Type:        "string",
			Description: "The IP address from which the beneficial owner attestation was made.",
		},
		"account.company.address_kana.postal_code": {
			Type:        "string",
			Description: "Postal code.",
		},
		"account.company.address_kanji.line2": {
			Type:        "string",
			Description: "Building details.",
		},
		"account.individual.registered_address.country": {
			Type:        "string",
			Description: "Two-letter country code ([ISO 3166-1 alpha-2](https://en.wikipedia.org/wiki/ISO_3166-1_alpha-2)).",
		},
		"account.individual.address_kanji.postal_code": {
			Type:        "string",
			Description: "Postal code.",
		},
		"bank_account.account_number": {
			Type:        "string",
			Description: "The account number for the bank account, in string form. Must be a checking account.",
			Required:    true,
		},
		"person.first_name_kanji": {
			Type:        "string",
			Description: "The Kanji variation of the person's first name (Japan only).",
		},
		"bank_account.country": {
			Type:        "string",
			Description: "The country in which the bank account is located.",
			Required:    true,
		},
		"person.documents.visa.files": {
			Type:        "array",
			Description: "One or more document ids returned by a [file upload](https://api.stripe.com#create_file) with a `purpose` value of `account_requirement`.",
		},
		"account.company.ownership_declaration.date": {
			Type:        "integer",
			Description: "The Unix timestamp marking when the beneficial owner attestation was made.",
			Format:      "unix-time",
		},
		"account.company.address_kana.town": {
			Type:        "string",
			Description: "Town or cho-me.",
		},
		"account.company.executives_provided": {
			Type:        "boolean",
			Description: "Whether the company's executives have been provided. Set this Boolean to `true` after creating all the company's executives with [the Persons API](/api/persons) for accounts with a `relationship.executive` requirement.",
		},
		"account.individual.address_kana.country": {
			Type:        "string",
			Description: "Two-letter country code ([ISO 3166-1 alpha-2](https://en.wikipedia.org/wiki/ISO_3166-1_alpha-2)).",
		},
		"account.individual.relationship.executive": {
			Type:        "boolean",
			Description: "Whether the person has significant responsibility to control, manage, or direct the organization.",
		},
		"person.address.city": {
			Type:        "string",
			Description: "City, district, suburb, town, or village.",
		},
		"person.last_name_kana": {
			Type:        "string",
			Description: "The Kana variation of the person's last name (Japan only).",
		},
		"person.last_name_kanji": {
			Type:        "string",
			Description: "The Kanji variation of the person's last name (Japan only).",
		},
		"account.company.address.city": {
			Type:        "string",
			Description: "City, district, suburb, town, or village.",
		},
		"person.address_kanji.city": {
			Type:        "string",
			Description: "City or ward.",
		},
		"person.address_kanji.country": {
			Type:        "string",
			Description: "Two-letter country code ([ISO 3166-1 alpha-2](https://en.wikipedia.org/wiki/ISO_3166-1_alpha-2)).",
		},
		"person.us_cfpb_data.ethnicity_details.ethnicity_other": {
			Type:        "string",
			Description: "Please specify your origin, when other is selected.",
		},
		"account.company.owners_provided": {
			Type:        "boolean",
			Description: "Whether the company's owners have been provided. Set this Boolean to `true` after creating all the company's owners with [the Persons API](/api/persons) for accounts with a `relationship.owner` requirement.",
		},
		"account.individual.registered_address.postal_code": {
			Type:        "string",
			Description: "ZIP or postal code.",
		},
		"account.individual.address.city": {
			Type:        "string",
			Description: "City, district, suburb, town, or village.",
		},
		"account.individual.verification.document.front": {
			Type:        "string",
			Description: "The front of an ID returned by a [file upload](https://api.stripe.com#create_file) with a `purpose` value of `identity_document`. The uploaded file needs to be a color image (smaller than 8,000px by 8,000px), in JPG, PNG, or PDF format, and less than 10 MB in size.",
		},
		"bank_account.account_type": {
			Type:        "string",
			Description: "The bank account type. This can only be `checking` or `savings` in most countries. In Japan, this can only be `futsu` or `toza`.",
			Enum: []resource.EnumSpec{
				{Value: "checking"},
				{Value: "futsu"},
				{Value: "savings"},
				{Value: "toza"},
			},
		},
		"person.relationship.executive": {
			Type:        "boolean",
			Description: "Whether the person has significant responsibility to control, manage, or direct the organization.",
		},
		"person.last_name": {
			Type:        "string",
			Description: "The person's last name.",
		},
		"pii.id_number": {
			Type:        "string",
			Description: "The `id_number` for the PII, in string form.",
		},
		"account.individual.address_kana.city": {
			Type:        "string",
			Description: "City or ward.",
		},
		"account.individual.address_kanji.country": {
			Type:        "string",
			Description: "Two-letter country code ([ISO 3166-1 alpha-2](https://en.wikipedia.org/wiki/ISO_3166-1_alpha-2)).",
		},
		"account.individual.address_kanji.line1": {
			Type:        "string",
			Description: "Block or building number.",
		},
		"account.individual.last_name_kana": {
			Type:        "string",
			Description: "The Kana variation of the individual's last name (Japan only).",
		},
		"account.individual.phone": {
			Type:        "string",
			Description: "The individual's phone number.",
		},
		"person.address_kana.line1": {
			Type:        "string",
			Description: "Block or building number.",
		},
		"person.address_kana.postal_code": {
			Type:        "string",
			Description: "Postal code.",
		},
		"account.company.tax_id": {
			Type:        "string",
			Description: "The business ID number of the company, as appropriate for the company’s country. (Examples are an Employer ID Number in the U.S., a Business Number in Canada, or a Company Number in the UK.)",
		},
		"account.company.verification.document.back": {
			Type:        "string",
			Description: "The back of a document returned by a [file upload](https://api.stripe.com#create_file) with a `purpose` value of `additional_verification`. The uploaded file needs to be a color image (smaller than 8,000px by 8,000px), in JPG, PNG, or PDF format, and less than 10 MB in size.",
		},
		"account.individual.first_name_kana": {
			Type:        "string",
			Description: "The Kana variation of the individual's first name (Japan only).",
		},
		"account.individual.address_kanji.line2": {
			Type:        "string",
			Description: "Building details.",
		},
		"account.individual.relationship.percent_ownership": {
			Type:        "number",
			Description: "The percent owned by the person of the account's legal entity.",
		},
		"person.relationship.legal_guardian": {
			Type:        "boolean",
			Description: "Whether the person is the legal guardian of the account's representative.",
		},
		"person.gender": {
			Type:        "string",
			Description: "The person's gender (International regulations require either \"male\" or \"female\").",
		},
		"account.company.address_kana.line1": {
			Type:        "string",
			Description: "Block or building number.",
		},
		"account.company.verification.document.front": {
			Type:        "string",
			Description: "The front of a document returned by a [file upload](https://api.stripe.com#create_file) with a `purpose` value of `additional_verification`. The uploaded file needs to be a color image (smaller than 8,000px by 8,000px), in JPG, PNG, or PDF format, and less than 10 MB in size.",
		},
		"account.individual.address_kana.postal_code": {
			Type:        "string",
			Description: "Postal code.",
		},
		"account.individual.registered_address.city": {
			Type:        "string",
			Description: "City, district, suburb, town, or village.",
		},
		"person.address.line2": {
			Type:        "string",
			Description: "Address line 2, such as the apartment, suite, unit, or building.",
		},
		"person.us_cfpb_data.race_details.race": {
			Type:        "array",
			Description: "The persons race.",
		},
		"person.additional_tos_acceptances.account.date": {
			Type:        "integer",
			Description: "The Unix timestamp marking when the account representative accepted the service agreement.",
			Format:      "unix-time",
		},
		"person.registered_address.city": {
			Type:        "string",
			Description: "City, district, suburb, town, or village.",
		},
		"account.company.address.postal_code": {
			Type:        "string",
			Description: "ZIP or postal code.",
		},
		"account.individual.last_name": {
			Type:        "string",
			Description: "The individual's last name.",
		},
		"account.individual.id_number": {
			Type:        "string",
			Description: "The government-issued ID number of the individual, as appropriate for the representative's country. (Examples are a Social Security Number in the U.S., or a Social Insurance Number in Canada). Instead of the number itself, you can also provide a [PII token created with Stripe.js](/js/tokens/create_token?type=pii).",
		},
		"card": {
			Type:        "string",
			Description: "The card this token will represent. If you also pass in a customer, the card must be the ID of a card belonging to the customer. Otherwise, if you do not pass in a customer, this is a dictionary containing a user's credit card details, with the options described below.",
		},
		"person.address_kanji.line2": {
			Type:        "string",
			Description: "Building details.",
		},
		"person.nationality": {
			Type:        "string",
			Description: "The country where the person is a national. Two-letter country code ([ISO 3166-1 alpha-2](https://en.wikipedia.org/wiki/ISO_3166-1_alpha-2)), or \"XX\" if unavailable.",
		},
		"person.email": {
			Type:        "string",
			Description: "The person's email address.",
		},
		"account.individual.political_exposure": {
			Type:        "string",
			Description: "Indicates if the person or any of their representatives, family members, or other closely related persons, declares that they hold or have held an important public job or function, in any jurisdiction.",
			Enum: []resource.EnumSpec{
				{Value: "existing"},
				{Value: "none"},
			},
		},
		"person.relationship.title": {
			Type:        "string",
			Description: "The person's title (e.g., CEO, Support Engineer).",
		},
		"person.us_cfpb_data.ethnicity_details.ethnicity": {
			Type:        "array",
			Description: "The persons ethnicity",
		},
		"account.company.representative_declaration.ip": {
			Type:        "string",
			Description: "The IP address from which the representative declaration attestation was made.",
		},
		"account.company.address_kana.state": {
			Type:        "string",
			Description: "Prefecture.",
		},
		"account.individual.address_kana.line1": {
			Type:        "string",
			Description: "Block or building number.",
		},
		"account.individual.address.state": {
			Type:        "string",
			Description: "State, county, province, or region ([ISO 3166-2](https://en.wikipedia.org/wiki/ISO_3166-2)).",
		},
		"person.address_kanji.state": {
			Type:        "string",
			Description: "Prefecture.",
		},
		"person.additional_tos_acceptances.account.user_agent": {
			Type:        "string",
			Description: "The user agent of the browser from which the account representative accepted the service agreement.",
		},
		"person.address_kana.city": {
			Type:        "string",
			Description: "City or ward.",
		},
		"person.registered_address.country": {
			Type:        "string",
			Description: "Two-letter country code ([ISO 3166-1 alpha-2](https://en.wikipedia.org/wiki/ISO_3166-1_alpha-2)).",
		},
		"account.company.structure": {
			Type:        "string",
			Description: "The category identifying the legal structure of the company or legal entity. See [Business structure](/connect/identity-verification#business-structure) for more details. Pass an empty string to unset this value.",
			Enum: []resource.EnumSpec{
				{Value: "free_zone_establishment"},
				{Value: "free_zone_llc"},
				{Value: "government_instrumentality"},
				{Value: "governmental_unit"},
				{Value: "incorporated_non_profit"},
				{Value: "incorporated_partnership"},
				{Value: "limited_liability_partnership"},
				{Value: "llc"},
				{Value: "multi_member_llc"},
				{Value: "private_company"},
				{Value: "private_corporation"},
				{Value: "private_partnership"},
				{Value: "public_company"},
				{Value: "public_corporation"},
				{Value: "public_partnership"},
				{Value: "registered_charity"},
				{Value: "single_member_llc"},
				{Value: "sole_establishment"},
				{Value: "sole_proprietorship"},
				{Value: "tax_exempt_government_instrumentality"},
				{Value: "unincorporated_association"},
				{Value: "unincorporated_non_profit"},
				{Value: "unincorporated_partnership"},
			},
		},
		"account.individual.relationship.director": {
			Type:        "boolean",
			Description: "Whether the person is a director of the account's legal entity. Directors are typically members of the governing board of the company, or responsible for ensuring the company meets its regulatory obligations.",
		},
		"person.address_kanji.line1": {
			Type:        "string",
			Description: "Block or building number.",
		},
		"person.full_name_aliases": {
			Type:        "array",
			Description: "A list of alternate names or aliases that the person is known by.",
		},
		"person.verification.additional_document.front": {
			Type:        "string",
			Description: "The front of an ID returned by a [file upload](https://api.stripe.com#create_file) with a `purpose` value of `identity_document`. The uploaded file needs to be a color image (smaller than 8,000px by 8,000px), in JPG, PNG, or PDF format, and less than 10 MB in size.",
		},
		"person.verification.document.back": {
			Type:        "string",
			Description: "The back of an ID returned by a [file upload](https://api.stripe.com#create_file) with a `purpose` value of `identity_document`. The uploaded file needs to be a color image (smaller than 8,000px by 8,000px), in JPG, PNG, or PDF format, and less than 10 MB in size.",
		},
		"account.company.phone": {
			Type:        "string",
			Description: "The company's phone number (used for verification).",
		},
	},
}

var V1PaymentSourcesList = resource.OperationSpec{
	Name:   "list",
	Path:   "/v1/customers/{customer}/sources",
	Method: "GET",
	Params: map[string]*resource.ParamSpec{
		"ending_before": {
			Type:        "string",
			Description: "A cursor for use in pagination. `ending_before` is an object ID that defines your place in the list. For instance, if you make a list request and receive 100 objects, starting with `obj_bar`, your subsequent call can include `ending_before=obj_bar` in order to fetch the previous page of the list.",
		},
		"limit": {
			Type:        "integer",
			Description: "A limit on the number of objects to be returned. Limit can range between 1 and 100, and the default is 10.",
		},
		"object": {
			Type:        "string",
			Description: "Filter sources according to a particular object type.",
		},
		"starting_after": {
			Type:        "string",
			Description: "A cursor for use in pagination. `starting_after` is an object ID that defines your place in the list. For instance, if you make a list request and receive 100 objects, ending with `obj_foo`, your subsequent call can include `starting_after=obj_foo` in order to fetch the next page of the list.",
		},
	},
}

var V1PaymentSourcesRetrieve = resource.OperationSpec{
	Name:   "retrieve",
	Path:   "/v1/customers/{customer}/sources/{id}",
	Method: "GET",
}

var V1PaymentSourcesCreate = resource.OperationSpec{
	Name:    "create",
	Path:    "/v1/customers/{customer}/sources",
	Method:  "POST",
	Summary: "Create a card",
	Params: map[string]*resource.ParamSpec{
		"source": {
			Type:        "string",
			Description: "Please refer to full [documentation](https://api.stripe.com) instead.",
			Required:    true,
		},
		"validate": {
			Type: "boolean",
		},
	},
}

var V1TransferReversalsCreate = resource.OperationSpec{
	Name:    "create",
	Path:    "/v1/transfers/{id}/reversals",
	Method:  "POST",
	Summary: "Create a transfer reversal",
	Params: map[string]*resource.ParamSpec{
		"amount": {
			Type:        "integer",
			Description: "A positive integer in cents (or local equivalent) representing how much of this transfer to reverse. Can only reverse up to the unreversed amount remaining of the transfer. Partial transfer reversals are only allowed for transfers to Stripe Accounts. Defaults to the entire transfer amount.",
		},
		"description": {
			Type:        "string",
			Description: "An arbitrary string which you can attach to a reversal object. This will be unset if you POST an empty value.",
		},
		"refund_application_fee": {
			Type:        "boolean",
			Description: "Boolean indicating whether the application fee should be refunded when reversing this transfer. If a full transfer reversal is given, the full application fee will be refunded. Otherwise, the application fee will be refunded with an amount proportional to the amount of the transfer reversed.",
		},
	},
}

var V1TransferReversalsUpdate = resource.OperationSpec{
	Name:    "update",
	Path:    "/v1/transfers/{transfer}/reversals/{id}",
	Method:  "POST",
	Summary: "Update a reversal",
}

var V1TransferReversalsList = resource.OperationSpec{
	Name:    "list",
	Path:    "/v1/transfers/{id}/reversals",
	Method:  "GET",
	Summary: "List all reversals",
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

var V1TransferReversalsRetrieve = resource.OperationSpec{
	Name:    "retrieve",
	Path:    "/v1/transfers/{transfer}/reversals/{id}",
	Method:  "GET",
	Summary: "Retrieve a reversal",
}

var V1CreditNoteLineItemsList = resource.OperationSpec{
	Name:    "list",
	Path:    "/v1/credit_notes/{credit_note}/lines",
	Method:  "GET",
	Summary: "Retrieve a credit note's line items",
	Params: map[string]*resource.ParamSpec{
		"starting_after": {
			Type:        "string",
			Description: "A cursor for use in pagination. `starting_after` is an object ID that defines your place in the list. For instance, if you make a list request and receive 100 objects, ending with `obj_foo`, your subsequent call can include `starting_after=obj_foo` in order to fetch the next page of the list.",
		},
		"ending_before": {
			Type:        "string",
			Description: "A cursor for use in pagination. `ending_before` is an object ID that defines your place in the list. For instance, if you make a list request and receive 100 objects, starting with `obj_bar`, your subsequent call can include `ending_before=obj_bar` in order to fetch the previous page of the list.",
		},
		"limit": {
			Type:        "integer",
			Description: "A limit on the number of objects to be returned. Limit can range between 1 and 100, and the default is 10.",
		},
	},
}

var V1PaymentMethodConfigurationsList = resource.OperationSpec{
	Name:    "list",
	Path:    "/v1/payment_method_configurations",
	Method:  "GET",
	Summary: "List payment method configurations",
	Params: map[string]*resource.ParamSpec{
		"application": {
			Type:        "string",
			Description: "The Connect application to filter by.",
		},
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

var V1PaymentMethodConfigurationsRetrieve = resource.OperationSpec{
	Name:    "retrieve",
	Path:    "/v1/payment_method_configurations/{configuration}",
	Method:  "GET",
	Summary: "Retrieve payment method configuration",
}

var V1PaymentMethodConfigurationsCreate = resource.OperationSpec{
	Name:    "create",
	Path:    "/v1/payment_method_configurations",
	Method:  "POST",
	Summary: "Create a payment method configuration",
	Params: map[string]*resource.ParamSpec{
		"us_bank_account.display_preference.preference": {
			Type:        "string",
			Description: "The account's preference for whether or not to display this payment method.",
			Enum: []resource.EnumSpec{
				{Value: "none"},
				{Value: "off"},
				{Value: "on"},
			},
		},
		"customer_balance.display_preference.preference": {
			Type:        "string",
			Description: "The account's preference for whether or not to display this payment method.",
			Enum: []resource.EnumSpec{
				{Value: "none"},
				{Value: "off"},
				{Value: "on"},
			},
		},
		"apple_pay.display_preference.preference": {
			Type:        "string",
			Description: "The account's preference for whether or not to display this payment method.",
			Enum: []resource.EnumSpec{
				{Value: "none"},
				{Value: "off"},
				{Value: "on"},
			},
		},
		"affirm.display_preference.preference": {
			Type:        "string",
			Description: "The account's preference for whether or not to display this payment method.",
			Enum: []resource.EnumSpec{
				{Value: "none"},
				{Value: "off"},
				{Value: "on"},
			},
		},
		"eps.display_preference.preference": {
			Type:        "string",
			Description: "The account's preference for whether or not to display this payment method.",
			Enum: []resource.EnumSpec{
				{Value: "none"},
				{Value: "off"},
				{Value: "on"},
			},
		},
		"nz_bank_account.display_preference.preference": {
			Type:        "string",
			Description: "The account's preference for whether or not to display this payment method.",
			Enum: []resource.EnumSpec{
				{Value: "none"},
				{Value: "off"},
				{Value: "on"},
			},
		},
		"google_pay.display_preference.preference": {
			Type:        "string",
			Description: "The account's preference for whether or not to display this payment method.",
			Enum: []resource.EnumSpec{
				{Value: "none"},
				{Value: "off"},
				{Value: "on"},
			},
		},
		"amazon_pay.display_preference.preference": {
			Type:        "string",
			Description: "The account's preference for whether or not to display this payment method.",
			Enum: []resource.EnumSpec{
				{Value: "none"},
				{Value: "off"},
				{Value: "on"},
			},
		},
		"revolut_pay.display_preference.preference": {
			Type:        "string",
			Description: "The account's preference for whether or not to display this payment method.",
			Enum: []resource.EnumSpec{
				{Value: "none"},
				{Value: "off"},
				{Value: "on"},
			},
		},
		"fpx.display_preference.preference": {
			Type:        "string",
			Description: "The account's preference for whether or not to display this payment method.",
			Enum: []resource.EnumSpec{
				{Value: "none"},
				{Value: "off"},
				{Value: "on"},
			},
		},
		"pix.display_preference.preference": {
			Type:        "string",
			Description: "The account's preference for whether or not to display this payment method.",
			Enum: []resource.EnumSpec{
				{Value: "none"},
				{Value: "off"},
				{Value: "on"},
			},
		},
		"swish.display_preference.preference": {
			Type:        "string",
			Description: "The account's preference for whether or not to display this payment method.",
			Enum: []resource.EnumSpec{
				{Value: "none"},
				{Value: "off"},
				{Value: "on"},
			},
		},
		"parent": {
			Type:        "string",
			Description: "Configuration's parent configuration. Specify to create a child configuration.",
		},
		"billie.display_preference.preference": {
			Type:        "string",
			Description: "The account's preference for whether or not to display this payment method.",
			Enum: []resource.EnumSpec{
				{Value: "none"},
				{Value: "off"},
				{Value: "on"},
			},
		},
		"bancontact.display_preference.preference": {
			Type:        "string",
			Description: "The account's preference for whether or not to display this payment method.",
			Enum: []resource.EnumSpec{
				{Value: "none"},
				{Value: "off"},
				{Value: "on"},
			},
		},
		"klarna.display_preference.preference": {
			Type:        "string",
			Description: "The account's preference for whether or not to display this payment method.",
			Enum: []resource.EnumSpec{
				{Value: "none"},
				{Value: "off"},
				{Value: "on"},
			},
		},
		"payto.display_preference.preference": {
			Type:        "string",
			Description: "The account's preference for whether or not to display this payment method.",
			Enum: []resource.EnumSpec{
				{Value: "none"},
				{Value: "off"},
				{Value: "on"},
			},
		},
		"oxxo.display_preference.preference": {
			Type:        "string",
			Description: "The account's preference for whether or not to display this payment method.",
			Enum: []resource.EnumSpec{
				{Value: "none"},
				{Value: "off"},
				{Value: "on"},
			},
		},
		"afterpay_clearpay.display_preference.preference": {
			Type:        "string",
			Description: "The account's preference for whether or not to display this payment method.",
			Enum: []resource.EnumSpec{
				{Value: "none"},
				{Value: "off"},
				{Value: "on"},
			},
		},
		"fr_meal_voucher_conecs.display_preference.preference": {
			Type:        "string",
			Description: "The account's preference for whether or not to display this payment method.",
			Enum: []resource.EnumSpec{
				{Value: "none"},
				{Value: "off"},
				{Value: "on"},
			},
		},
		"alipay.display_preference.preference": {
			Type:        "string",
			Description: "The account's preference for whether or not to display this payment method.",
			Enum: []resource.EnumSpec{
				{Value: "none"},
				{Value: "off"},
				{Value: "on"},
			},
		},
		"boleto.display_preference.preference": {
			Type:        "string",
			Description: "The account's preference for whether or not to display this payment method.",
			Enum: []resource.EnumSpec{
				{Value: "none"},
				{Value: "off"},
				{Value: "on"},
			},
		},
		"p24.display_preference.preference": {
			Type:        "string",
			Description: "The account's preference for whether or not to display this payment method.",
			Enum: []resource.EnumSpec{
				{Value: "none"},
				{Value: "off"},
				{Value: "on"},
			},
		},
		"kakao_pay.display_preference.preference": {
			Type:        "string",
			Description: "The account's preference for whether or not to display this payment method.",
			Enum: []resource.EnumSpec{
				{Value: "none"},
				{Value: "off"},
				{Value: "on"},
			},
		},
		"au_becs_debit.display_preference.preference": {
			Type:        "string",
			Description: "The account's preference for whether or not to display this payment method.",
			Enum: []resource.EnumSpec{
				{Value: "none"},
				{Value: "off"},
				{Value: "on"},
			},
		},
		"paynow.display_preference.preference": {
			Type:        "string",
			Description: "The account's preference for whether or not to display this payment method.",
			Enum: []resource.EnumSpec{
				{Value: "none"},
				{Value: "off"},
				{Value: "on"},
			},
		},
		"paypal.display_preference.preference": {
			Type:        "string",
			Description: "The account's preference for whether or not to display this payment method.",
			Enum: []resource.EnumSpec{
				{Value: "none"},
				{Value: "off"},
				{Value: "on"},
			},
		},
		"wechat_pay.display_preference.preference": {
			Type:        "string",
			Description: "The account's preference for whether or not to display this payment method.",
			Enum: []resource.EnumSpec{
				{Value: "none"},
				{Value: "off"},
				{Value: "on"},
			},
		},
		"kr_card.display_preference.preference": {
			Type:        "string",
			Description: "The account's preference for whether or not to display this payment method.",
			Enum: []resource.EnumSpec{
				{Value: "none"},
				{Value: "off"},
				{Value: "on"},
			},
		},
		"multibanco.display_preference.preference": {
			Type:        "string",
			Description: "The account's preference for whether or not to display this payment method.",
			Enum: []resource.EnumSpec{
				{Value: "none"},
				{Value: "off"},
				{Value: "on"},
			},
		},
		"pay_by_bank.display_preference.preference": {
			Type:        "string",
			Description: "The account's preference for whether or not to display this payment method.",
			Enum: []resource.EnumSpec{
				{Value: "none"},
				{Value: "off"},
				{Value: "on"},
			},
		},
		"crypto.display_preference.preference": {
			Type:        "string",
			Description: "The account's preference for whether or not to display this payment method.",
			Enum: []resource.EnumSpec{
				{Value: "none"},
				{Value: "off"},
				{Value: "on"},
			},
		},
		"konbini.display_preference.preference": {
			Type:        "string",
			Description: "The account's preference for whether or not to display this payment method.",
			Enum: []resource.EnumSpec{
				{Value: "none"},
				{Value: "off"},
				{Value: "on"},
			},
		},
		"card.display_preference.preference": {
			Type:        "string",
			Description: "The account's preference for whether or not to display this payment method.",
			Enum: []resource.EnumSpec{
				{Value: "none"},
				{Value: "off"},
				{Value: "on"},
			},
		},
		"samsung_pay.display_preference.preference": {
			Type:        "string",
			Description: "The account's preference for whether or not to display this payment method.",
			Enum: []resource.EnumSpec{
				{Value: "none"},
				{Value: "off"},
				{Value: "on"},
			},
		},
		"sepa_debit.display_preference.preference": {
			Type:        "string",
			Description: "The account's preference for whether or not to display this payment method.",
			Enum: []resource.EnumSpec{
				{Value: "none"},
				{Value: "off"},
				{Value: "on"},
			},
		},
		"zip.display_preference.preference": {
			Type:        "string",
			Description: "The account's preference for whether or not to display this payment method.",
			Enum: []resource.EnumSpec{
				{Value: "none"},
				{Value: "off"},
				{Value: "on"},
			},
		},
		"blik.display_preference.preference": {
			Type:        "string",
			Description: "The account's preference for whether or not to display this payment method.",
			Enum: []resource.EnumSpec{
				{Value: "none"},
				{Value: "off"},
				{Value: "on"},
			},
		},
		"twint.display_preference.preference": {
			Type:        "string",
			Description: "The account's preference for whether or not to display this payment method.",
			Enum: []resource.EnumSpec{
				{Value: "none"},
				{Value: "off"},
				{Value: "on"},
			},
		},
		"payco.display_preference.preference": {
			Type:        "string",
			Description: "The account's preference for whether or not to display this payment method.",
			Enum: []resource.EnumSpec{
				{Value: "none"},
				{Value: "off"},
				{Value: "on"},
			},
		},
		"mobilepay.display_preference.preference": {
			Type:        "string",
			Description: "The account's preference for whether or not to display this payment method.",
			Enum: []resource.EnumSpec{
				{Value: "none"},
				{Value: "off"},
				{Value: "on"},
			},
		},
		"acss_debit.display_preference.preference": {
			Type:        "string",
			Description: "The account's preference for whether or not to display this payment method.",
			Enum: []resource.EnumSpec{
				{Value: "none"},
				{Value: "off"},
				{Value: "on"},
			},
		},
		"ideal.display_preference.preference": {
			Type:        "string",
			Description: "The account's preference for whether or not to display this payment method.",
			Enum: []resource.EnumSpec{
				{Value: "none"},
				{Value: "off"},
				{Value: "on"},
			},
		},
		"bacs_debit.display_preference.preference": {
			Type:        "string",
			Description: "The account's preference for whether or not to display this payment method.",
			Enum: []resource.EnumSpec{
				{Value: "none"},
				{Value: "off"},
				{Value: "on"},
			},
		},
		"jcb.display_preference.preference": {
			Type:        "string",
			Description: "The account's preference for whether or not to display this payment method.",
			Enum: []resource.EnumSpec{
				{Value: "none"},
				{Value: "off"},
				{Value: "on"},
			},
		},
		"name": {
			Type:        "string",
			Description: "Configuration name.",
		},
		"giropay.display_preference.preference": {
			Type:        "string",
			Description: "The account's preference for whether or not to display this payment method.",
			Enum: []resource.EnumSpec{
				{Value: "none"},
				{Value: "off"},
				{Value: "on"},
			},
		},
		"alma.display_preference.preference": {
			Type:        "string",
			Description: "The account's preference for whether or not to display this payment method.",
			Enum: []resource.EnumSpec{
				{Value: "none"},
				{Value: "off"},
				{Value: "on"},
			},
		},
		"satispay.display_preference.preference": {
			Type:        "string",
			Description: "The account's preference for whether or not to display this payment method.",
			Enum: []resource.EnumSpec{
				{Value: "none"},
				{Value: "off"},
				{Value: "on"},
			},
		},
		"sofort.display_preference.preference": {
			Type:        "string",
			Description: "The account's preference for whether or not to display this payment method.",
			Enum: []resource.EnumSpec{
				{Value: "none"},
				{Value: "off"},
				{Value: "on"},
			},
		},
		"cashapp.display_preference.preference": {
			Type:        "string",
			Description: "The account's preference for whether or not to display this payment method.",
			Enum: []resource.EnumSpec{
				{Value: "none"},
				{Value: "off"},
				{Value: "on"},
			},
		},
		"cartes_bancaires.display_preference.preference": {
			Type:        "string",
			Description: "The account's preference for whether or not to display this payment method.",
			Enum: []resource.EnumSpec{
				{Value: "none"},
				{Value: "off"},
				{Value: "on"},
			},
		},
		"naver_pay.display_preference.preference": {
			Type:        "string",
			Description: "The account's preference for whether or not to display this payment method.",
			Enum: []resource.EnumSpec{
				{Value: "none"},
				{Value: "off"},
				{Value: "on"},
			},
		},
		"mb_way.display_preference.preference": {
			Type:        "string",
			Description: "The account's preference for whether or not to display this payment method.",
			Enum: []resource.EnumSpec{
				{Value: "none"},
				{Value: "off"},
				{Value: "on"},
			},
		},
		"apple_pay_later.display_preference.preference": {
			Type:        "string",
			Description: "The account's preference for whether or not to display this payment method.",
			Enum: []resource.EnumSpec{
				{Value: "none"},
				{Value: "off"},
				{Value: "on"},
			},
		},
		"grabpay.display_preference.preference": {
			Type:        "string",
			Description: "The account's preference for whether or not to display this payment method.",
			Enum: []resource.EnumSpec{
				{Value: "none"},
				{Value: "off"},
				{Value: "on"},
			},
		},
		"promptpay.display_preference.preference": {
			Type:        "string",
			Description: "The account's preference for whether or not to display this payment method.",
			Enum: []resource.EnumSpec{
				{Value: "none"},
				{Value: "off"},
				{Value: "on"},
			},
		},
		"link.display_preference.preference": {
			Type:        "string",
			Description: "The account's preference for whether or not to display this payment method.",
			Enum: []resource.EnumSpec{
				{Value: "none"},
				{Value: "off"},
				{Value: "on"},
			},
		},
	},
}

var V1PaymentMethodConfigurationsUpdate = resource.OperationSpec{
	Name:    "update",
	Path:    "/v1/payment_method_configurations/{configuration}",
	Method:  "POST",
	Summary: "Update payment method configuration",
	Params: map[string]*resource.ParamSpec{
		"sofort.display_preference.preference": {
			Type:        "string",
			Description: "The account's preference for whether or not to display this payment method.",
			Enum: []resource.EnumSpec{
				{Value: "none"},
				{Value: "off"},
				{Value: "on"},
			},
		},
		"satispay.display_preference.preference": {
			Type:        "string",
			Description: "The account's preference for whether or not to display this payment method.",
			Enum: []resource.EnumSpec{
				{Value: "none"},
				{Value: "off"},
				{Value: "on"},
			},
		},
		"twint.display_preference.preference": {
			Type:        "string",
			Description: "The account's preference for whether or not to display this payment method.",
			Enum: []resource.EnumSpec{
				{Value: "none"},
				{Value: "off"},
				{Value: "on"},
			},
		},
		"cashapp.display_preference.preference": {
			Type:        "string",
			Description: "The account's preference for whether or not to display this payment method.",
			Enum: []resource.EnumSpec{
				{Value: "none"},
				{Value: "off"},
				{Value: "on"},
			},
		},
		"fpx.display_preference.preference": {
			Type:        "string",
			Description: "The account's preference for whether or not to display this payment method.",
			Enum: []resource.EnumSpec{
				{Value: "none"},
				{Value: "off"},
				{Value: "on"},
			},
		},
		"eps.display_preference.preference": {
			Type:        "string",
			Description: "The account's preference for whether or not to display this payment method.",
			Enum: []resource.EnumSpec{
				{Value: "none"},
				{Value: "off"},
				{Value: "on"},
			},
		},
		"swish.display_preference.preference": {
			Type:        "string",
			Description: "The account's preference for whether or not to display this payment method.",
			Enum: []resource.EnumSpec{
				{Value: "none"},
				{Value: "off"},
				{Value: "on"},
			},
		},
		"cartes_bancaires.display_preference.preference": {
			Type:        "string",
			Description: "The account's preference for whether or not to display this payment method.",
			Enum: []resource.EnumSpec{
				{Value: "none"},
				{Value: "off"},
				{Value: "on"},
			},
		},
		"pay_by_bank.display_preference.preference": {
			Type:        "string",
			Description: "The account's preference for whether or not to display this payment method.",
			Enum: []resource.EnumSpec{
				{Value: "none"},
				{Value: "off"},
				{Value: "on"},
			},
		},
		"wechat_pay.display_preference.preference": {
			Type:        "string",
			Description: "The account's preference for whether or not to display this payment method.",
			Enum: []resource.EnumSpec{
				{Value: "none"},
				{Value: "off"},
				{Value: "on"},
			},
		},
		"konbini.display_preference.preference": {
			Type:        "string",
			Description: "The account's preference for whether or not to display this payment method.",
			Enum: []resource.EnumSpec{
				{Value: "none"},
				{Value: "off"},
				{Value: "on"},
			},
		},
		"blik.display_preference.preference": {
			Type:        "string",
			Description: "The account's preference for whether or not to display this payment method.",
			Enum: []resource.EnumSpec{
				{Value: "none"},
				{Value: "off"},
				{Value: "on"},
			},
		},
		"name": {
			Type:        "string",
			Description: "Configuration name.",
		},
		"payto.display_preference.preference": {
			Type:        "string",
			Description: "The account's preference for whether or not to display this payment method.",
			Enum: []resource.EnumSpec{
				{Value: "none"},
				{Value: "off"},
				{Value: "on"},
			},
		},
		"zip.display_preference.preference": {
			Type:        "string",
			Description: "The account's preference for whether or not to display this payment method.",
			Enum: []resource.EnumSpec{
				{Value: "none"},
				{Value: "off"},
				{Value: "on"},
			},
		},
		"billie.display_preference.preference": {
			Type:        "string",
			Description: "The account's preference for whether or not to display this payment method.",
			Enum: []resource.EnumSpec{
				{Value: "none"},
				{Value: "off"},
				{Value: "on"},
			},
		},
		"multibanco.display_preference.preference": {
			Type:        "string",
			Description: "The account's preference for whether or not to display this payment method.",
			Enum: []resource.EnumSpec{
				{Value: "none"},
				{Value: "off"},
				{Value: "on"},
			},
		},
		"pix.display_preference.preference": {
			Type:        "string",
			Description: "The account's preference for whether or not to display this payment method.",
			Enum: []resource.EnumSpec{
				{Value: "none"},
				{Value: "off"},
				{Value: "on"},
			},
		},
		"bacs_debit.display_preference.preference": {
			Type:        "string",
			Description: "The account's preference for whether or not to display this payment method.",
			Enum: []resource.EnumSpec{
				{Value: "none"},
				{Value: "off"},
				{Value: "on"},
			},
		},
		"link.display_preference.preference": {
			Type:        "string",
			Description: "The account's preference for whether or not to display this payment method.",
			Enum: []resource.EnumSpec{
				{Value: "none"},
				{Value: "off"},
				{Value: "on"},
			},
		},
		"card.display_preference.preference": {
			Type:        "string",
			Description: "The account's preference for whether or not to display this payment method.",
			Enum: []resource.EnumSpec{
				{Value: "none"},
				{Value: "off"},
				{Value: "on"},
			},
		},
		"grabpay.display_preference.preference": {
			Type:        "string",
			Description: "The account's preference for whether or not to display this payment method.",
			Enum: []resource.EnumSpec{
				{Value: "none"},
				{Value: "off"},
				{Value: "on"},
			},
		},
		"paypal.display_preference.preference": {
			Type:        "string",
			Description: "The account's preference for whether or not to display this payment method.",
			Enum: []resource.EnumSpec{
				{Value: "none"},
				{Value: "off"},
				{Value: "on"},
			},
		},
		"jcb.display_preference.preference": {
			Type:        "string",
			Description: "The account's preference for whether or not to display this payment method.",
			Enum: []resource.EnumSpec{
				{Value: "none"},
				{Value: "off"},
				{Value: "on"},
			},
		},
		"bancontact.display_preference.preference": {
			Type:        "string",
			Description: "The account's preference for whether or not to display this payment method.",
			Enum: []resource.EnumSpec{
				{Value: "none"},
				{Value: "off"},
				{Value: "on"},
			},
		},
		"fr_meal_voucher_conecs.display_preference.preference": {
			Type:        "string",
			Description: "The account's preference for whether or not to display this payment method.",
			Enum: []resource.EnumSpec{
				{Value: "none"},
				{Value: "off"},
				{Value: "on"},
			},
		},
		"us_bank_account.display_preference.preference": {
			Type:        "string",
			Description: "The account's preference for whether or not to display this payment method.",
			Enum: []resource.EnumSpec{
				{Value: "none"},
				{Value: "off"},
				{Value: "on"},
			},
		},
		"google_pay.display_preference.preference": {
			Type:        "string",
			Description: "The account's preference for whether or not to display this payment method.",
			Enum: []resource.EnumSpec{
				{Value: "none"},
				{Value: "off"},
				{Value: "on"},
			},
		},
		"oxxo.display_preference.preference": {
			Type:        "string",
			Description: "The account's preference for whether or not to display this payment method.",
			Enum: []resource.EnumSpec{
				{Value: "none"},
				{Value: "off"},
				{Value: "on"},
			},
		},
		"mobilepay.display_preference.preference": {
			Type:        "string",
			Description: "The account's preference for whether or not to display this payment method.",
			Enum: []resource.EnumSpec{
				{Value: "none"},
				{Value: "off"},
				{Value: "on"},
			},
		},
		"crypto.display_preference.preference": {
			Type:        "string",
			Description: "The account's preference for whether or not to display this payment method.",
			Enum: []resource.EnumSpec{
				{Value: "none"},
				{Value: "off"},
				{Value: "on"},
			},
		},
		"alipay.display_preference.preference": {
			Type:        "string",
			Description: "The account's preference for whether or not to display this payment method.",
			Enum: []resource.EnumSpec{
				{Value: "none"},
				{Value: "off"},
				{Value: "on"},
			},
		},
		"amazon_pay.display_preference.preference": {
			Type:        "string",
			Description: "The account's preference for whether or not to display this payment method.",
			Enum: []resource.EnumSpec{
				{Value: "none"},
				{Value: "off"},
				{Value: "on"},
			},
		},
		"sepa_debit.display_preference.preference": {
			Type:        "string",
			Description: "The account's preference for whether or not to display this payment method.",
			Enum: []resource.EnumSpec{
				{Value: "none"},
				{Value: "off"},
				{Value: "on"},
			},
		},
		"apple_pay_later.display_preference.preference": {
			Type:        "string",
			Description: "The account's preference for whether or not to display this payment method.",
			Enum: []resource.EnumSpec{
				{Value: "none"},
				{Value: "off"},
				{Value: "on"},
			},
		},
		"revolut_pay.display_preference.preference": {
			Type:        "string",
			Description: "The account's preference for whether or not to display this payment method.",
			Enum: []resource.EnumSpec{
				{Value: "none"},
				{Value: "off"},
				{Value: "on"},
			},
		},
		"kakao_pay.display_preference.preference": {
			Type:        "string",
			Description: "The account's preference for whether or not to display this payment method.",
			Enum: []resource.EnumSpec{
				{Value: "none"},
				{Value: "off"},
				{Value: "on"},
			},
		},
		"naver_pay.display_preference.preference": {
			Type:        "string",
			Description: "The account's preference for whether or not to display this payment method.",
			Enum: []resource.EnumSpec{
				{Value: "none"},
				{Value: "off"},
				{Value: "on"},
			},
		},
		"apple_pay.display_preference.preference": {
			Type:        "string",
			Description: "The account's preference for whether or not to display this payment method.",
			Enum: []resource.EnumSpec{
				{Value: "none"},
				{Value: "off"},
				{Value: "on"},
			},
		},
		"affirm.display_preference.preference": {
			Type:        "string",
			Description: "The account's preference for whether or not to display this payment method.",
			Enum: []resource.EnumSpec{
				{Value: "none"},
				{Value: "off"},
				{Value: "on"},
			},
		},
		"p24.display_preference.preference": {
			Type:        "string",
			Description: "The account's preference for whether or not to display this payment method.",
			Enum: []resource.EnumSpec{
				{Value: "none"},
				{Value: "off"},
				{Value: "on"},
			},
		},
		"ideal.display_preference.preference": {
			Type:        "string",
			Description: "The account's preference for whether or not to display this payment method.",
			Enum: []resource.EnumSpec{
				{Value: "none"},
				{Value: "off"},
				{Value: "on"},
			},
		},
		"klarna.display_preference.preference": {
			Type:        "string",
			Description: "The account's preference for whether or not to display this payment method.",
			Enum: []resource.EnumSpec{
				{Value: "none"},
				{Value: "off"},
				{Value: "on"},
			},
		},
		"payco.display_preference.preference": {
			Type:        "string",
			Description: "The account's preference for whether or not to display this payment method.",
			Enum: []resource.EnumSpec{
				{Value: "none"},
				{Value: "off"},
				{Value: "on"},
			},
		},
		"active": {
			Type:        "boolean",
			Description: "Whether the configuration can be used for new payments.",
		},
		"alma.display_preference.preference": {
			Type:        "string",
			Description: "The account's preference for whether or not to display this payment method.",
			Enum: []resource.EnumSpec{
				{Value: "none"},
				{Value: "off"},
				{Value: "on"},
			},
		},
		"acss_debit.display_preference.preference": {
			Type:        "string",
			Description: "The account's preference for whether or not to display this payment method.",
			Enum: []resource.EnumSpec{
				{Value: "none"},
				{Value: "off"},
				{Value: "on"},
			},
		},
		"giropay.display_preference.preference": {
			Type:        "string",
			Description: "The account's preference for whether or not to display this payment method.",
			Enum: []resource.EnumSpec{
				{Value: "none"},
				{Value: "off"},
				{Value: "on"},
			},
		},
		"afterpay_clearpay.display_preference.preference": {
			Type:        "string",
			Description: "The account's preference for whether or not to display this payment method.",
			Enum: []resource.EnumSpec{
				{Value: "none"},
				{Value: "off"},
				{Value: "on"},
			},
		},
		"nz_bank_account.display_preference.preference": {
			Type:        "string",
			Description: "The account's preference for whether or not to display this payment method.",
			Enum: []resource.EnumSpec{
				{Value: "none"},
				{Value: "off"},
				{Value: "on"},
			},
		},
		"paynow.display_preference.preference": {
			Type:        "string",
			Description: "The account's preference for whether or not to display this payment method.",
			Enum: []resource.EnumSpec{
				{Value: "none"},
				{Value: "off"},
				{Value: "on"},
			},
		},
		"promptpay.display_preference.preference": {
			Type:        "string",
			Description: "The account's preference for whether or not to display this payment method.",
			Enum: []resource.EnumSpec{
				{Value: "none"},
				{Value: "off"},
				{Value: "on"},
			},
		},
		"mb_way.display_preference.preference": {
			Type:        "string",
			Description: "The account's preference for whether or not to display this payment method.",
			Enum: []resource.EnumSpec{
				{Value: "none"},
				{Value: "off"},
				{Value: "on"},
			},
		},
		"customer_balance.display_preference.preference": {
			Type:        "string",
			Description: "The account's preference for whether or not to display this payment method.",
			Enum: []resource.EnumSpec{
				{Value: "none"},
				{Value: "off"},
				{Value: "on"},
			},
		},
		"kr_card.display_preference.preference": {
			Type:        "string",
			Description: "The account's preference for whether or not to display this payment method.",
			Enum: []resource.EnumSpec{
				{Value: "none"},
				{Value: "off"},
				{Value: "on"},
			},
		},
		"au_becs_debit.display_preference.preference": {
			Type:        "string",
			Description: "The account's preference for whether or not to display this payment method.",
			Enum: []resource.EnumSpec{
				{Value: "none"},
				{Value: "off"},
				{Value: "on"},
			},
		},
		"boleto.display_preference.preference": {
			Type:        "string",
			Description: "The account's preference for whether or not to display this payment method.",
			Enum: []resource.EnumSpec{
				{Value: "none"},
				{Value: "off"},
				{Value: "on"},
			},
		},
		"samsung_pay.display_preference.preference": {
			Type:        "string",
			Description: "The account's preference for whether or not to display this payment method.",
			Enum: []resource.EnumSpec{
				{Value: "none"},
				{Value: "off"},
				{Value: "on"},
			},
		},
	},
}

var V1SetupIntentsVerifyMicrodeposits = resource.OperationSpec{
	Name:    "verify_microdeposits",
	Path:    "/v1/setup_intents/{intent}/verify_microdeposits",
	Method:  "POST",
	Summary: "Verify microdeposits on a SetupIntent",
	Params: map[string]*resource.ParamSpec{
		"amounts": {
			Type:        "array",
			Description: "Two positive integers, in *cents*, equal to the values of the microdeposits sent to the bank account.",
		},
		"descriptor_code": {
			Type:        "string",
			Description: "A six-character code starting with SM present in the microdeposit sent to the bank account.",
		},
	},
}

var V1SetupIntentsList = resource.OperationSpec{
	Name:    "list",
	Path:    "/v1/setup_intents",
	Method:  "GET",
	Summary: "List all SetupIntents",
	Params: map[string]*resource.ParamSpec{
		"limit": {
			Type:        "integer",
			Description: "A limit on the number of objects to be returned. Limit can range between 1 and 100, and the default is 10.",
		},
		"payment_method": {
			Type:        "string",
			Description: "Only return SetupIntents that associate with the specified payment method.",
		},
		"starting_after": {
			Type:        "string",
			Description: "A cursor for use in pagination. `starting_after` is an object ID that defines your place in the list. For instance, if you make a list request and receive 100 objects, ending with `obj_foo`, your subsequent call can include `starting_after=obj_foo` in order to fetch the next page of the list.",
		},
		"attach_to_self": {
			Type:        "boolean",
			Description: "If present, the SetupIntent's payment method will be attached to the in-context Stripe Account.\n\nIt can only be used for this Stripe Account’s own money movement flows like InboundTransfer and OutboundTransfers. It cannot be set to true when setting up a PaymentMethod for a Customer, and defaults to false when attaching a PaymentMethod to a Customer.",
		},
		"created": {
			Type:        "integer",
			Description: "A filter on the list, based on the object `created` field. The value can be a string with an integer Unix timestamp, or it can be a dictionary with a number of different query options.",
		},
		"customer": {
			Type:        "string",
			Description: "Only return SetupIntents for the customer specified by this customer ID.",
		},
		"customer_account": {
			Type:        "string",
			Description: "Only return SetupIntents for the account specified by this customer ID.",
		},
		"ending_before": {
			Type:        "string",
			Description: "A cursor for use in pagination. `ending_before` is an object ID that defines your place in the list. For instance, if you make a list request and receive 100 objects, starting with `obj_bar`, your subsequent call can include `ending_before=obj_bar` in order to fetch the previous page of the list.",
		},
	},
}

var V1SetupIntentsRetrieve = resource.OperationSpec{
	Name:    "retrieve",
	Path:    "/v1/setup_intents/{intent}",
	Method:  "GET",
	Summary: "Retrieve a SetupIntent",
	Params: map[string]*resource.ParamSpec{
		"client_secret": {
			Type:        "string",
			Description: "The client secret of the SetupIntent. We require this string if you use a publishable key to retrieve the SetupIntent.",
		},
	},
}

var V1SetupIntentsCreate = resource.OperationSpec{
	Name:    "create",
	Path:    "/v1/setup_intents",
	Method:  "POST",
	Summary: "Create a SetupIntent",
	Params: map[string]*resource.ParamSpec{
		"payment_method_options.klarna.on_demand.purchase_interval": {
			Type:        "string",
			Description: "Interval at which the customer is making purchases",
			Enum: []resource.EnumSpec{
				{Value: "day"},
				{Value: "month"},
				{Value: "week"},
				{Value: "year"},
			},
		},
		"payment_method_data.nz_bank_account.reference": {
			Type: "string",
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
		"payment_method_data.klarna.dob.day": {
			Type:        "integer",
			Description: "The day of birth, between 1 and 31.",
			Required:    true,
		},
		"payment_method_data.nz_bank_account.bank_code": {
			Type:        "string",
			Description: "The numeric code for the bank account's bank.",
			Required:    true,
		},
		"payment_method_options.card.mandate_options.description": {
			Type:        "string",
			Description: "A description of the mandate or subscription that is meant to be displayed to the customer.",
		},
		"payment_method_options.card.mandate_options.amount": {
			Type:        "integer",
			Description: "Amount to be charged for future payments.",
			Required:    true,
		},
		"payment_method_options.card.mandate_options.interval_count": {
			Type:        "integer",
			Description: "The number of intervals between payments. For example, `interval=month` and `interval_count=3` indicates one payment every three months. Maximum of one year interval allowed (1 year, 12 months, or 52 weeks). This parameter is optional when `interval=sporadic`.",
		},
		"payment_method_data.boleto.tax_id": {
			Type:        "string",
			Description: "The tax ID of the customer (CPF for individual consumers or CNPJ for businesses consumers)",
			Required:    true,
		},
		"customer": {
			Type:        "string",
			Description: "ID of the Customer this SetupIntent belongs to, if one exists.\n\nIf present, the SetupIntent's payment method will be attached to the Customer on successful setup. Payment methods attached to other Customers cannot be used with this SetupIntent.",
		},
		"payment_method_options.us_bank_account.networks.requested": {
			Type:        "array",
			Description: "Triggers validations to run across the selected networks",
		},
		"payment_method_options.card.three_d_secure.network_options.cartes_bancaires.cb_avalgo": {
			Type:        "string",
			Description: "The cryptogram calculation algorithm used by the card Issuer's ACS\nto calculate the Authentication cryptogram. Also known as `cavvAlgorithm`.\nmessageExtension: CB-AVALGO",
			Required:    true,
			Enum: []resource.EnumSpec{
				{Value: "0"},
				{Value: "1"},
				{Value: "2"},
				{Value: "3"},
				{Value: "4"},
				{Value: "A"},
			},
		},
		"usage": {
			Type:        "string",
			Description: "Indicates how the payment method is intended to be used in the future. If not provided, this value defaults to `off_session`.",
			Enum: []resource.EnumSpec{
				{Value: "off_session"},
				{Value: "on_session"},
			},
		},
		"confirmation_token": {
			Type:        "string",
			Description: "ID of the ConfirmationToken used to confirm this SetupIntent.\n\nIf the provided ConfirmationToken contains properties that are also being provided in this request, such as `payment_method`, then the values in this request will take precedence.",
		},
		"payment_method_options.klarna.on_demand.minimum_amount": {
			Type:        "integer",
			Description: "The lowest or minimum value you may charge a customer per purchase. You can use a value across your customer base, or segment based on customer type, country, etc.",
		},
		"payment_method_options.klarna.on_demand.purchase_interval_count": {
			Type:        "integer",
			Description: "The number of `purchase_interval` between charges",
		},
		"payment_method_options.sepa_debit.mandate_options.reference_prefix": {
			Type:        "string",
			Description: "Prefix used to generate the Mandate reference. Must be at most 12 characters long. Must consist of only uppercase letters, numbers, spaces, or the following special characters: '/', '_', '-', '&', '.'. Cannot begin with 'STRIPE'.",
		},
		"payment_method_options.us_bank_account.financial_connections.prefetch": {
			Type:        "array",
			Description: "List of data features that you would like to retrieve upon account creation.",
		},
		"payment_method_options.bacs_debit.mandate_options.reference_prefix": {
			Type:        "string",
			Description: "Prefix used to generate the Mandate reference. Must be at most 12 characters long. Must consist of only uppercase letters, numbers, spaces, or the following special characters: '/', '_', '-', '&', '.'. Cannot begin with 'DDIC' or 'STRIPE'.",
		},
		"payment_method_options.card.mandate_options.currency": {
			Type:        "string",
			Description: "Currency in which future payments will be charged. Three-letter [ISO currency code](https://www.iso.org/iso-4217-currency-codes.html), in lowercase. Must be a [supported currency](https://stripe.com/docs/currencies).",
			Required:    true,
			Format:      "currency",
		},
		"payment_method_data.bacs_debit.account_number": {
			Type:        "string",
			Description: "Account number of the bank account that the funds will be debited from.",
		},
		"payment_method_options.card.mandate_options.reference": {
			Type:        "string",
			Description: "Unique identifier for the mandate or subscription.",
			Required:    true,
		},
		"payment_method_options.card.three_d_secure.version": {
			Type:        "string",
			Description: "The version of 3D Secure that was performed.",
			Enum: []resource.EnumSpec{
				{Value: "1.0.2"},
				{Value: "2.1.0"},
				{Value: "2.2.0"},
				{Value: "2.3.0"},
				{Value: "2.3.1"},
			},
		},
		"payment_method_data.billing_details.tax_id": {
			Type:        "string",
			Description: "Taxpayer identification number. Used only for transactions between LATAM buyers and non-LATAM sellers.",
		},
		"payment_method_data.us_bank_account.account_number": {
			Type:        "string",
			Description: "Account number of the bank account.",
		},
		"payment_method_options.payto.mandate_options.start_date": {
			Type:        "string",
			Description: "Date, in YYYY-MM-DD format, from which payments will be collected. Defaults to confirmation time.",
		},
		"use_stripe_sdk": {
			Type:        "boolean",
			Description: "Set to `true` when confirming server-side and using Stripe.js, iOS, or Android client-side SDKs to handle the next actions.",
		},
		"payment_method_options.card.three_d_secure.cryptogram": {
			Type:        "string",
			Description: "The cryptogram, also known as the \"authentication value\" (AAV, CAVV or\nAEVV). This value is 20 bytes, base64-encoded into a 28-character string.\n(Most 3D Secure providers will return the base64-encoded version, which\nis what you should specify here.)",
		},
		"automatic_payment_methods.allow_redirects": {
			Type:        "string",
			Description: "Controls whether this SetupIntent will accept redirect-based payment methods.\n\nRedirect-based payment methods may require your customer to be redirected to a payment method's app or site for authentication or additional steps. To [confirm](https://docs.stripe.com/api/setup_intents/confirm) this SetupIntent, you may be required to provide a `return_url` to redirect customers back to your site after they authenticate or complete the setup.",
			Enum: []resource.EnumSpec{
				{Value: "always"},
				{Value: "never"},
			},
		},
		"payment_method_data.billing_details.email": {
			Type:        "string",
			Description: "Email address.",
		},
		"confirm": {
			Type:        "boolean",
			Description: "Set to `true` to attempt to confirm this SetupIntent immediately. This parameter defaults to `false`. If a card is the attached payment method, you can provide a `return_url` in case further authentication is necessary.",
		},
		"payment_method_data.ideal.bank": {
			Type:        "string",
			Description: "The customer's bank. Only use this parameter for existing customers. Don't use it for new customers.",
			Enum: []resource.EnumSpec{
				{Value: "abn_amro"},
				{Value: "adyen"},
				{Value: "asn_bank"},
				{Value: "bunq"},
				{Value: "buut"},
				{Value: "finom"},
				{Value: "handelsbanken"},
				{Value: "ing"},
				{Value: "knab"},
				{Value: "mollie"},
				{Value: "moneyou"},
				{Value: "n26"},
				{Value: "nn"},
				{Value: "rabobank"},
				{Value: "regiobank"},
				{Value: "revolut"},
				{Value: "sns_bank"},
				{Value: "triodos_bank"},
				{Value: "van_lanschot"},
				{Value: "yoursafe"},
			},
		},
		"payment_method_data.nz_bank_account.account_number": {
			Type:        "string",
			Description: "The account number for the bank account.",
			Required:    true,
		},
		"payment_method_data.nz_bank_account.suffix": {
			Type:        "string",
			Description: "The suffix of the bank account number.",
			Required:    true,
		},
		"payment_method_options.card.three_d_secure.requestor_challenge_indicator": {
			Type:        "string",
			Description: "The challenge indicator (`threeDSRequestorChallengeInd`) which was requested in the\nAReq sent to the card Issuer's ACS. A string containing 2 digits from 01-99.",
		},
		"payment_method_configuration": {
			Type:        "string",
			Description: "The ID of the [payment method configuration](https://docs.stripe.com/api/payment_method_configurations) to use with this SetupIntent.",
		},
		"payment_method_data.payto.bsb_number": {
			Type:        "string",
			Description: "Bank-State-Branch number of the bank account.",
		},
		"payment_method_options.link.persistent_token": {
			Type:        "string",
			Description: "[Deprecated] This is a legacy parameter that no longer has any function.",
		},
		"payment_method_options.card.mandate_options.start_date": {
			Type:        "integer",
			Description: "Start date of the mandate or subscription. Start date should not be lesser than yesterday.",
			Required:    true,
			Format:      "unix-time",
		},
		"payment_method_options.card.mandate_options.amount_type": {
			Type:        "string",
			Description: "One of `fixed` or `maximum`. If `fixed`, the `amount` param refers to the exact amount to be charged in future payments. If `maximum`, the amount charged can be up to the value passed for the `amount` param.",
			Required:    true,
			Enum: []resource.EnumSpec{
				{Value: "fixed"},
				{Value: "maximum"},
			},
		},
		"on_behalf_of": {
			Type:        "string",
			Description: "The Stripe account ID created for this SetupIntent.",
		},
		"payment_method_data.sepa_debit.iban": {
			Type:        "string",
			Description: "IBAN of the bank account.",
			Required:    true,
		},
		"payment_method_data.eps.bank": {
			Type:        "string",
			Description: "The customer's bank.",
			Enum: []resource.EnumSpec{
				{Value: "arzte_und_apotheker_bank"},
				{Value: "austrian_anadi_bank_ag"},
				{Value: "bank_austria"},
				{Value: "bankhaus_carl_spangler"},
				{Value: "bankhaus_schelhammer_und_schattera_ag"},
				{Value: "bawag_psk_ag"},
				{Value: "bks_bank_ag"},
				{Value: "brull_kallmus_bank_ag"},
				{Value: "btv_vier_lander_bank"},
				{Value: "capital_bank_grawe_gruppe_ag"},
				{Value: "deutsche_bank_ag"},
				{Value: "dolomitenbank"},
				{Value: "easybank_ag"},
				{Value: "erste_bank_und_sparkassen"},
				{Value: "hypo_alpeadriabank_international_ag"},
				{Value: "hypo_bank_burgenland_aktiengesellschaft"},
				{Value: "hypo_noe_lb_fur_niederosterreich_u_wien"},
				{Value: "hypo_oberosterreich_salzburg_steiermark"},
				{Value: "hypo_tirol_bank_ag"},
				{Value: "hypo_vorarlberg_bank_ag"},
				{Value: "marchfelder_bank"},
				{Value: "oberbank_ag"},
				{Value: "raiffeisen_bankengruppe_osterreich"},
				{Value: "schoellerbank_ag"},
				{Value: "sparda_bank_wien"},
				{Value: "volksbank_gruppe"},
				{Value: "volkskreditbank_ag"},
				{Value: "vr_bank_braunau"},
			},
		},
		"payment_method_data.acss_debit.institution_number": {
			Type:        "string",
			Description: "Institution number of the customer's bank.",
			Required:    true,
		},
		"payment_method_options.payto.mandate_options.amount": {
			Type:        "integer",
			Description: "Amount that will be collected. It is required when `amount_type` is `fixed`.",
		},
		"single_use.currency": {
			Type:        "string",
			Description: "Three-letter [ISO currency code](https://www.iso.org/iso-4217-currency-codes.html), in lowercase. Must be a [supported currency](https://stripe.com/docs/currencies).",
			Required:    true,
			Format:      "currency",
		},
		"payment_method_options.card.three_d_secure.transaction_id": {
			Type:        "string",
			Description: "For 3D Secure 1, the XID. For 3D Secure 2, the Directory Server\nTransaction ID (dsTransID).",
		},
		"excluded_payment_method_types": {
			Type:        "array",
			Description: "The list of payment method types to exclude from use with this SetupIntent.",
		},
		"payment_method_types": {
			Type:        "array",
			Description: "The list of payment method types (for example, card) that this SetupIntent can use. If you don't provide this, Stripe will dynamically show relevant payment methods from your [payment method settings](https://dashboard.stripe.com/settings/payment_methods). A list of valid payment method types can be found [here](https://docs.stripe.com/api/payment_methods/object#payment_method_object-type).",
		},
		"payment_method_data.type": {
			Type:        "string",
			Description: "The type of the PaymentMethod. An additional hash is included on the PaymentMethod with a name matching this value. It contains additional information specific to the PaymentMethod type.",
			Required:    true,
			Enum: []resource.EnumSpec{
				{Value: "acss_debit"},
				{Value: "affirm"},
				{Value: "afterpay_clearpay"},
				{Value: "alipay"},
				{Value: "alma"},
				{Value: "amazon_pay"},
				{Value: "au_becs_debit"},
				{Value: "bacs_debit"},
				{Value: "bancontact"},
				{Value: "billie"},
				{Value: "blik"},
				{Value: "boleto"},
				{Value: "cashapp"},
				{Value: "crypto"},
				{Value: "customer_balance"},
				{Value: "eps"},
				{Value: "fpx"},
				{Value: "giropay"},
				{Value: "grabpay"},
				{Value: "ideal"},
				{Value: "kakao_pay"},
				{Value: "klarna"},
				{Value: "konbini"},
				{Value: "kr_card"},
				{Value: "link"},
				{Value: "mb_way"},
				{Value: "mobilepay"},
				{Value: "multibanco"},
				{Value: "naver_pay"},
				{Value: "nz_bank_account"},
				{Value: "oxxo"},
				{Value: "p24"},
				{Value: "pay_by_bank"},
				{Value: "payco"},
				{Value: "paynow"},
				{Value: "paypal"},
				{Value: "payto"},
				{Value: "pix"},
				{Value: "promptpay"},
				{Value: "revolut_pay"},
				{Value: "samsung_pay"},
				{Value: "satispay"},
				{Value: "sepa_debit"},
				{Value: "sofort"},
				{Value: "swish"},
				{Value: "twint"},
				{Value: "us_bank_account"},
				{Value: "wechat_pay"},
				{Value: "zip"},
			},
		},
		"payment_method_data.acss_debit.account_number": {
			Type:        "string",
			Description: "Customer's bank account number.",
			Required:    true,
		},
		"single_use.amount": {
			Type:        "integer",
			Description: "Amount the customer is granting permission to collect later. A positive integer representing how much to charge in the [smallest currency unit](https://docs.stripe.com/currencies#zero-decimal) (e.g., 100 cents to charge $1.00 or 100 to charge ¥100, a zero-decimal currency). The minimum amount is $0.50 US or [equivalent in charge currency](https://docs.stripe.com/currencies#minimum-and-maximum-charge-amounts). The amount value supports up to eight digits (e.g., a value of 99999999 for a USD charge of $999,999.99).",
			Required:    true,
		},
		"description": {
			Type:        "string",
			Description: "An arbitrary string attached to the object. Often useful for displaying to users.",
		},
		"payment_method_options.us_bank_account.financial_connections.permissions": {
			Type:        "array",
			Description: "The list of permissions to request. If this parameter is passed, the `payment_method` permission must be included. Valid permissions include: `balances`, `ownership`, `payment_method`, and `transactions`.",
		},
		"payment_method": {
			Type:        "string",
			Description: "ID of the payment method (a PaymentMethod, Card, or saved Source object) to attach to this SetupIntent.",
		},
		"payment_method_data.allow_redisplay": {
			Type:        "string",
			Description: "This field indicates whether this payment method can be shown again to its customer in a checkout flow. Stripe products such as Checkout and Elements use this field to determine whether a payment method can be shown as a saved payment method in a checkout flow. The field defaults to `unspecified`.",
			Enum: []resource.EnumSpec{
				{Value: "always"},
				{Value: "limited"},
				{Value: "unspecified"},
			},
		},
		"payment_method_options.us_bank_account.financial_connections.filters.account_subcategories": {
			Type:        "array",
			Description: "The account subcategories to use to filter for selectable accounts. Valid subcategories are `checking` and `savings`.",
		},
		"payment_method_options.card.moto": {
			Type:        "boolean",
			Description: "When specified, this parameter signals that a card has been collected\nas MOTO (Mail Order Telephone Order) and thus out of scope for SCA. This\nparameter can only be provided during confirmation.",
		},
		"payment_method_options.card.three_d_secure.ares_trans_status": {
			Type:        "string",
			Description: "The `transStatus` returned from the card Issuer’s ACS in the ARes.",
			Enum: []resource.EnumSpec{
				{Value: "A"},
				{Value: "C"},
				{Value: "I"},
				{Value: "N"},
				{Value: "R"},
				{Value: "U"},
				{Value: "Y"},
			},
		},
		"payment_method_options.payto.mandate_options.amount_type": {
			Type:        "string",
			Description: "The type of amount that will be collected. The amount charged must be exact or up to the value of `amount` param for `fixed` or `maximum` type respectively. Defaults to `maximum`.",
			Enum: []resource.EnumSpec{
				{Value: "fixed"},
				{Value: "maximum"},
			},
		},
		"payment_method_options.acss_debit.mandate_options.interval_description": {
			Type:        "string",
			Description: "Description of the mandate interval. Only required if 'payment_schedule' parameter is 'interval' or 'combined'.",
		},
		"payment_method_options.acss_debit.mandate_options.transaction_type": {
			Type:        "string",
			Description: "Transaction type of the mandate.",
			Enum: []resource.EnumSpec{
				{Value: "business"},
				{Value: "personal"},
			},
		},
		"payment_method_options.acss_debit.verification_method": {
			Type:        "string",
			Description: "Bank account verification method.",
			Enum: []resource.EnumSpec{
				{Value: "automatic"},
				{Value: "instant"},
				{Value: "microdeposits"},
			},
		},
		"payment_method_options.card.three_d_secure.network_options.cartes_bancaires.cb_score": {
			Type:        "integer",
			Description: "The risk score returned from Cartes Bancaires in the ARes.\nmessage extension: CB-SCORE; numeric value 0-99",
		},
		"payment_method_data.us_bank_account.account_holder_type": {
			Type:        "string",
			Description: "Account holder type: individual or company.",
			Enum: []resource.EnumSpec{
				{Value: "company"},
				{Value: "individual"},
			},
		},
		"flow_directions": {
			Type:        "array",
			Description: "Indicates the directions of money movement for which this payment method is intended to be used.\n\nInclude `inbound` if you intend to use the payment method as the origin to pull funds from. Include `outbound` if you intend to use the payment method as the destination to send funds to. You can include both if you intend to use the payment method for both purposes.",
		},
		"payment_method_options.us_bank_account.mandate_options.collection_method": {
			Type:        "string",
			Description: "The method used to collect offline mandate customer acceptance.",
			Enum: []resource.EnumSpec{
				{Value: "paper"},
			},
		},
		"payment_method_options.us_bank_account.verification_method": {
			Type:        "string",
			Description: "Bank account verification method.",
			Enum: []resource.EnumSpec{
				{Value: "automatic"},
				{Value: "instant"},
				{Value: "microdeposits"},
			},
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
		"payment_method_data.au_becs_debit.bsb_number": {
			Type:        "string",
			Description: "Bank-State-Branch number of the bank account.",
			Required:    true,
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
		"payment_method_data.sofort.country": {
			Type:        "string",
			Description: "Two-letter ISO code representing the country the bank account is located in.",
			Required:    true,
			Enum: []resource.EnumSpec{
				{Value: "AT"},
				{Value: "BE"},
				{Value: "DE"},
				{Value: "ES"},
				{Value: "IT"},
				{Value: "NL"},
			},
		},
		"payment_method_data.p24.bank": {
			Type:        "string",
			Description: "The customer's bank.",
			Enum: []resource.EnumSpec{
				{Value: "alior_bank"},
				{Value: "bank_millennium"},
				{Value: "bank_nowy_bfg_sa"},
				{Value: "bank_pekao_sa"},
				{Value: "banki_spbdzielcze"},
				{Value: "blik"},
				{Value: "bnp_paribas"},
				{Value: "boz"},
				{Value: "citi_handlowy"},
				{Value: "credit_agricole"},
				{Value: "envelobank"},
				{Value: "etransfer_pocztowy24"},
				{Value: "getin_bank"},
				{Value: "ideabank"},
				{Value: "ing"},
				{Value: "inteligo"},
				{Value: "mbank_mtransfer"},
				{Value: "nest_przelew"},
				{Value: "noble_pay"},
				{Value: "pbac_z_ipko"},
				{Value: "plus_bank"},
				{Value: "santander_przelew24"},
				{Value: "tmobile_usbugi_bankowe"},
				{Value: "toyota_bank"},
				{Value: "velobank"},
				{Value: "volkswagen_bank"},
			},
		},
		"payment_method_options.acss_debit.mandate_options.custom_mandate_url": {
			Type:        "string",
			Description: "A URL for custom mandate text to render during confirmation step.\nThe URL will be rendered with additional GET parameters `payment_intent` and `payment_intent_client_secret` when confirming a Payment Intent,\nor `setup_intent` and `setup_intent_client_secret` when confirming a Setup Intent.",
		},
		"payment_method_data.billing_details.name": {
			Type:        "string",
			Description: "Full name.",
		},
		"attach_to_self": {
			Type:        "boolean",
			Description: "If present, the SetupIntent's payment method will be attached to the in-context Stripe Account.\n\nIt can only be used for this Stripe Account’s own money movement flows like InboundTransfer and OutboundTransfers. It cannot be set to true when setting up a PaymentMethod for a Customer, and defaults to false when attaching a PaymentMethod to a Customer.",
		},
		"payment_method_data.us_bank_account.financial_connections_account": {
			Type:        "string",
			Description: "The ID of a Financial Connections Account to use as a payment method.",
		},
		"payment_method_data.fpx.account_holder_type": {
			Type:        "string",
			Description: "Account holder type for FPX transaction",
			Enum: []resource.EnumSpec{
				{Value: "company"},
				{Value: "individual"},
			},
		},
		"customer_account": {
			Type:        "string",
			Description: "ID of the Account this SetupIntent belongs to, if one exists.\n\nIf present, the SetupIntent's payment method will be attached to the Account on successful setup. Payment methods attached to other Accounts cannot be used with this SetupIntent.",
		},
		"payment_method_options.klarna.on_demand.maximum_amount": {
			Type:        "integer",
			Description: "The maximum value you may charge a customer per purchase. You can use a value across your customer base, or segment based on customer type, country, etc.",
		},
		"return_url": {
			Type:        "string",
			Description: "The URL to redirect your customer back to after they authenticate or cancel their payment on the payment method's app or site. To redirect to a mobile application, you can alternatively supply an application URI scheme. This parameter can only be used with [`confirm=true`](https://docs.stripe.com/api/setup_intents/create#create_setup_intent-confirm).",
		},
		"payment_method_data.payto.pay_id": {
			Type:        "string",
			Description: "The PayID alias for the bank account.",
		},
		"payment_method_data.radar_options.session": {
			Type:        "string",
			Description: "A [Radar Session](https://docs.stripe.com/radar/radar-session) is a snapshot of the browser metadata and device details that help Radar make more accurate predictions on your payments.",
		},
		"payment_method_options.us_bank_account.financial_connections.return_url": {
			Type:        "string",
			Description: "For webview integrations only. Upon completing OAuth login in the native browser, the user will be redirected to this URL to return to your app.",
		},
		"payment_method_options.paypal.billing_agreement_id": {
			Type:        "string",
			Description: "The PayPal Billing Agreement ID (BAID). This is an ID generated by PayPal which represents the mandate between the merchant and the customer.",
		},
		"payment_method_options.card.mandate_options.end_date": {
			Type:        "integer",
			Description: "End date of the mandate or subscription. If not provided, the mandate will be active until canceled. If provided, end date should be after start date.",
			Format:      "unix-time",
		},
		"payment_method_options.klarna.on_demand.average_amount": {
			Type:        "integer",
			Description: "Your average amount value. You can use a value across your customer base, or segment based on customer type, country, etc.",
		},
		"payment_method_data.payto.account_number": {
			Type:        "string",
			Description: "The account number for the bank account.",
		},
		"payment_method_data.au_becs_debit.account_number": {
			Type:        "string",
			Description: "The account number for the bank account.",
			Required:    true,
		},
		"payment_method_options.payto.mandate_options.payments_per_period": {
			Type:        "integer",
			Description: "The number of payments that will be made during a payment period. Defaults to 1 except for when `payment_schedule` is `adhoc`. In that case, it defaults to no limit.",
		},
		"payment_method_options.card.three_d_secure.electronic_commerce_indicator": {
			Type:        "string",
			Description: "The Electronic Commerce Indicator (ECI) is returned by your 3D Secure\nprovider and indicates what degree of authentication was performed.",
			Enum: []resource.EnumSpec{
				{Value: "01"},
				{Value: "02"},
				{Value: "05"},
				{Value: "06"},
				{Value: "07"},
			},
		},
		"payment_method_data.us_bank_account.routing_number": {
			Type:        "string",
			Description: "Routing number of the bank account.",
		},
		"payment_method_options.klarna.currency": {
			Type:        "string",
			Description: "The currency of the SetupIntent. Three letter ISO currency code.",
			Format:      "currency",
		},
		"payment_method_data.nz_bank_account.branch_code": {
			Type:        "string",
			Description: "The numeric code for the bank account's bank branch.",
			Required:    true,
		},
		"payment_method_data.naver_pay.funding": {
			Type:        "string",
			Description: "Whether to use Naver Pay points or a card to fund this transaction. If not provided, this defaults to `card`.",
			Enum: []resource.EnumSpec{
				{Value: "card"},
				{Value: "points"},
			},
		},
		"payment_method_options.acss_debit.currency": {
			Type:        "string",
			Description: "Three-letter [ISO currency code](https://www.iso.org/iso-4217-currency-codes.html), in lowercase. Must be a [supported currency](https://stripe.com/docs/currencies).",
			Enum: []resource.EnumSpec{
				{Value: "cad"},
				{Value: "usd"},
			},
		},
		"payment_method_options.card.network": {
			Type:        "string",
			Description: "Selected network to process this SetupIntent on. Depends on the available networks of the card attached to the SetupIntent. Can be only set confirm-time.",
			Enum: []resource.EnumSpec{
				{Value: "amex"},
				{Value: "cartes_bancaires"},
				{Value: "diners"},
				{Value: "discover"},
				{Value: "eftpos_au"},
				{Value: "girocard"},
				{Value: "interac"},
				{Value: "jcb"},
				{Value: "link"},
				{Value: "mastercard"},
				{Value: "unionpay"},
				{Value: "unknown"},
				{Value: "visa"},
			},
		},
		"payment_method_data.klarna.dob.year": {
			Type:        "integer",
			Description: "The four-digit year of birth.",
			Required:    true,
		},
		"payment_method_data.bacs_debit.sort_code": {
			Type:        "string",
			Description: "Sort code of the bank account. (e.g., `10-20-30`)",
		},
		"payment_method_options.payto.mandate_options.end_date": {
			Type:        "string",
			Description: "Date, in YYYY-MM-DD format, after which payments will not be collected. Defaults to no end date.",
		},
		"payment_method_options.card.three_d_secure.network_options.cartes_bancaires.cb_exemption": {
			Type:        "string",
			Description: "The exemption indicator returned from Cartes Bancaires in the ARes.\nmessage extension: CB-EXEMPTION; string (4 characters)\nThis is a 3 byte bitmap (low significant byte first and most significant\nbit first) that has been Base64 encoded",
		},
		"payment_method_data.fpx.bank": {
			Type:        "string",
			Description: "The customer's bank.",
			Required:    true,
			Enum: []resource.EnumSpec{
				{Value: "affin_bank"},
				{Value: "agrobank"},
				{Value: "alliance_bank"},
				{Value: "ambank"},
				{Value: "bank_islam"},
				{Value: "bank_muamalat"},
				{Value: "bank_of_china"},
				{Value: "bank_rakyat"},
				{Value: "bsn"},
				{Value: "cimb"},
				{Value: "deutsche_bank"},
				{Value: "hong_leong_bank"},
				{Value: "hsbc"},
				{Value: "kfh"},
				{Value: "maybank2e"},
				{Value: "maybank2u"},
				{Value: "ocbc"},
				{Value: "pb_enterprise"},
				{Value: "public_bank"},
				{Value: "rhb"},
				{Value: "standard_chartered"},
				{Value: "uob"},
			},
		},
		"payment_method_options.acss_debit.mandate_options.default_for": {
			Type:        "array",
			Description: "List of Stripe products where this mandate can be selected automatically.",
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
		"automatic_payment_methods.enabled": {
			Type:        "boolean",
			Description: "Whether this feature is enabled.",
			Required:    true,
		},
		"payment_method_data.klarna.dob.month": {
			Type:        "integer",
			Description: "The month of birth, between 1 and 12.",
			Required:    true,
		},
		"payment_method_data.billing_details.phone": {
			Type:        "string",
			Description: "Billing phone number (including extension).",
		},
		"payment_method_data.acss_debit.transit_number": {
			Type:        "string",
			Description: "Transit number of the customer's bank.",
			Required:    true,
		},
		"payment_method_options.card.mandate_options.supported_types": {
			Type:        "array",
			Description: "Specifies the type of mandates supported. Possible values are `india`.",
		},
		"payment_method_options.card.mandate_options.interval": {
			Type:        "string",
			Description: "Specifies payment frequency. One of `day`, `week`, `month`, `year`, or `sporadic`.",
			Required:    true,
			Enum: []resource.EnumSpec{
				{Value: "day"},
				{Value: "month"},
				{Value: "sporadic"},
				{Value: "week"},
				{Value: "year"},
			},
		},
		"payment_method_data.us_bank_account.account_type": {
			Type:        "string",
			Description: "Account type: checkings or savings. Defaults to checking if omitted.",
			Enum: []resource.EnumSpec{
				{Value: "checking"},
				{Value: "savings"},
			},
		},
		"payment_method_data.nz_bank_account.account_holder_name": {
			Type:        "string",
			Description: "The name on the bank account. Only required if the account holder name is different from the name of the authorized signatory collected in the PaymentMethod’s billing details.",
		},
		"payment_method_options.klarna.preferred_locale": {
			Type:        "string",
			Description: "Preferred language of the Klarna authorization page that the customer is redirected to",
			Enum: []resource.EnumSpec{
				{Value: "cs-CZ"},
				{Value: "da-DK"},
				{Value: "de-AT"},
				{Value: "de-CH"},
				{Value: "de-DE"},
				{Value: "el-GR"},
				{Value: "en-AT"},
				{Value: "en-AU"},
				{Value: "en-BE"},
				{Value: "en-CA"},
				{Value: "en-CH"},
				{Value: "en-CZ"},
				{Value: "en-DE"},
				{Value: "en-DK"},
				{Value: "en-ES"},
				{Value: "en-FI"},
				{Value: "en-FR"},
				{Value: "en-GB"},
				{Value: "en-GR"},
				{Value: "en-IE"},
				{Value: "en-IT"},
				{Value: "en-NL"},
				{Value: "en-NO"},
				{Value: "en-NZ"},
				{Value: "en-PL"},
				{Value: "en-PT"},
				{Value: "en-RO"},
				{Value: "en-SE"},
				{Value: "en-US"},
				{Value: "es-ES"},
				{Value: "es-US"},
				{Value: "fi-FI"},
				{Value: "fr-BE"},
				{Value: "fr-CA"},
				{Value: "fr-CH"},
				{Value: "fr-FR"},
				{Value: "it-CH"},
				{Value: "it-IT"},
				{Value: "nb-NO"},
				{Value: "nl-BE"},
				{Value: "nl-NL"},
				{Value: "pl-PL"},
				{Value: "pt-PT"},
				{Value: "ro-RO"},
				{Value: "sv-FI"},
				{Value: "sv-SE"},
			},
		},
	},
}

var V1SetupIntentsUpdate = resource.OperationSpec{
	Name:    "update",
	Path:    "/v1/setup_intents/{intent}",
	Method:  "POST",
	Summary: "Update a SetupIntent",
	Params: map[string]*resource.ParamSpec{
		"payment_method_options.card.request_three_d_secure": {
			Type:        "string",
			Description: "We strongly recommend that you rely on our SCA Engine to automatically prompt your customers for authentication based on risk level and [other requirements](https://docs.stripe.com/strong-customer-authentication). However, if you wish to request 3D Secure based on logic from your own fraud engine, provide this option. If not provided, this value defaults to `automatic`. Read our guide on [manually requesting 3D Secure](https://docs.stripe.com/payments/3d-secure/authentication-flow#manual-three-ds) for more information on how this configuration interacts with Radar and our SCA Engine.",
			Enum: []resource.EnumSpec{
				{Value: "any"},
				{Value: "automatic"},
				{Value: "challenge"},
			},
		},
		"payment_method_options.acss_debit.mandate_options.custom_mandate_url": {
			Type:        "string",
			Description: "A URL for custom mandate text to render during confirmation step.\nThe URL will be rendered with additional GET parameters `payment_intent` and `payment_intent_client_secret` when confirming a Payment Intent,\nor `setup_intent` and `setup_intent_client_secret` when confirming a Setup Intent.",
		},
		"payment_method_options.payto.mandate_options.start_date": {
			Type:        "string",
			Description: "Date, in YYYY-MM-DD format, from which payments will be collected. Defaults to confirmation time.",
		},
		"payment_method_options.payto.mandate_options.payments_per_period": {
			Type:        "integer",
			Description: "The number of payments that will be made during a payment period. Defaults to 1 except for when `payment_schedule` is `adhoc`. In that case, it defaults to no limit.",
		},
		"payment_method_options.card.moto": {
			Type:        "boolean",
			Description: "When specified, this parameter signals that a card has been collected\nas MOTO (Mail Order Telephone Order) and thus out of scope for SCA. This\nparameter can only be provided during confirmation.",
		},
		"payment_method_data.sofort.country": {
			Type:        "string",
			Description: "Two-letter ISO code representing the country the bank account is located in.",
			Required:    true,
			Enum: []resource.EnumSpec{
				{Value: "AT"},
				{Value: "BE"},
				{Value: "DE"},
				{Value: "ES"},
				{Value: "IT"},
				{Value: "NL"},
			},
		},
		"payment_method_data.us_bank_account.financial_connections_account": {
			Type:        "string",
			Description: "The ID of a Financial Connections Account to use as a payment method.",
		},
		"payment_method_options.link.persistent_token": {
			Type:        "string",
			Description: "[Deprecated] This is a legacy parameter that no longer has any function.",
		},
		"payment_method_options.card.mandate_options.start_date": {
			Type:        "integer",
			Description: "Start date of the mandate or subscription. Start date should not be lesser than yesterday.",
			Required:    true,
			Format:      "unix-time",
		},
		"payment_method_options.card.mandate_options.amount": {
			Type:        "integer",
			Description: "Amount to be charged for future payments.",
			Required:    true,
		},
		"payment_method_configuration": {
			Type:        "string",
			Description: "The ID of the [payment method configuration](https://docs.stripe.com/api/payment_method_configurations) to use with this SetupIntent.",
		},
		"payment_method_data.ideal.bank": {
			Type:        "string",
			Description: "The customer's bank. Only use this parameter for existing customers. Don't use it for new customers.",
			Enum: []resource.EnumSpec{
				{Value: "abn_amro"},
				{Value: "adyen"},
				{Value: "asn_bank"},
				{Value: "bunq"},
				{Value: "buut"},
				{Value: "finom"},
				{Value: "handelsbanken"},
				{Value: "ing"},
				{Value: "knab"},
				{Value: "mollie"},
				{Value: "moneyou"},
				{Value: "n26"},
				{Value: "nn"},
				{Value: "rabobank"},
				{Value: "regiobank"},
				{Value: "revolut"},
				{Value: "sns_bank"},
				{Value: "triodos_bank"},
				{Value: "van_lanschot"},
				{Value: "yoursafe"},
			},
		},
		"payment_method_data.us_bank_account.account_type": {
			Type:        "string",
			Description: "Account type: checkings or savings. Defaults to checking if omitted.",
			Enum: []resource.EnumSpec{
				{Value: "checking"},
				{Value: "savings"},
			},
		},
		"payment_method_data.billing_details.name": {
			Type:        "string",
			Description: "Full name.",
		},
		"payment_method_data.acss_debit.transit_number": {
			Type:        "string",
			Description: "Transit number of the customer's bank.",
			Required:    true,
		},
		"payment_method_options.card.three_d_secure.ares_trans_status": {
			Type:        "string",
			Description: "The `transStatus` returned from the card Issuer’s ACS in the ARes.",
			Enum: []resource.EnumSpec{
				{Value: "A"},
				{Value: "C"},
				{Value: "I"},
				{Value: "N"},
				{Value: "R"},
				{Value: "U"},
				{Value: "Y"},
			},
		},
		"payment_method_options.card.mandate_options.supported_types": {
			Type:        "array",
			Description: "Specifies the type of mandates supported. Possible values are `india`.",
		},
		"payment_method_data.billing_details.email": {
			Type:        "string",
			Description: "Email address.",
		},
		"payment_method_data.nz_bank_account.branch_code": {
			Type:        "string",
			Description: "The numeric code for the bank account's bank branch.",
			Required:    true,
		},
		"payment_method_data.allow_redisplay": {
			Type:        "string",
			Description: "This field indicates whether this payment method can be shown again to its customer in a checkout flow. Stripe products such as Checkout and Elements use this field to determine whether a payment method can be shown as a saved payment method in a checkout flow. The field defaults to `unspecified`.",
			Enum: []resource.EnumSpec{
				{Value: "always"},
				{Value: "limited"},
				{Value: "unspecified"},
			},
		},
		"flow_directions": {
			Type:        "array",
			Description: "Indicates the directions of money movement for which this payment method is intended to be used.\n\nInclude `inbound` if you intend to use the payment method as the origin to pull funds from. Include `outbound` if you intend to use the payment method as the destination to send funds to. You can include both if you intend to use the payment method for both purposes.",
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
		"payment_method_options.card.three_d_secure.network_options.cartes_bancaires.cb_exemption": {
			Type:        "string",
			Description: "The exemption indicator returned from Cartes Bancaires in the ARes.\nmessage extension: CB-EXEMPTION; string (4 characters)\nThis is a 3 byte bitmap (low significant byte first and most significant\nbit first) that has been Base64 encoded",
		},
		"payment_method_options.card.three_d_secure.transaction_id": {
			Type:        "string",
			Description: "For 3D Secure 1, the XID. For 3D Secure 2, the Directory Server\nTransaction ID (dsTransID).",
		},
		"payment_method_options.card.mandate_options.description": {
			Type:        "string",
			Description: "A description of the mandate or subscription that is meant to be displayed to the customer.",
		},
		"payment_method_options.klarna.preferred_locale": {
			Type:        "string",
			Description: "Preferred language of the Klarna authorization page that the customer is redirected to",
			Enum: []resource.EnumSpec{
				{Value: "cs-CZ"},
				{Value: "da-DK"},
				{Value: "de-AT"},
				{Value: "de-CH"},
				{Value: "de-DE"},
				{Value: "el-GR"},
				{Value: "en-AT"},
				{Value: "en-AU"},
				{Value: "en-BE"},
				{Value: "en-CA"},
				{Value: "en-CH"},
				{Value: "en-CZ"},
				{Value: "en-DE"},
				{Value: "en-DK"},
				{Value: "en-ES"},
				{Value: "en-FI"},
				{Value: "en-FR"},
				{Value: "en-GB"},
				{Value: "en-GR"},
				{Value: "en-IE"},
				{Value: "en-IT"},
				{Value: "en-NL"},
				{Value: "en-NO"},
				{Value: "en-NZ"},
				{Value: "en-PL"},
				{Value: "en-PT"},
				{Value: "en-RO"},
				{Value: "en-SE"},
				{Value: "en-US"},
				{Value: "es-ES"},
				{Value: "es-US"},
				{Value: "fi-FI"},
				{Value: "fr-BE"},
				{Value: "fr-CA"},
				{Value: "fr-CH"},
				{Value: "fr-FR"},
				{Value: "it-CH"},
				{Value: "it-IT"},
				{Value: "nb-NO"},
				{Value: "nl-BE"},
				{Value: "nl-NL"},
				{Value: "pl-PL"},
				{Value: "pt-PT"},
				{Value: "ro-RO"},
				{Value: "sv-FI"},
				{Value: "sv-SE"},
			},
		},
		"payment_method_data.klarna.dob.month": {
			Type:        "integer",
			Description: "The month of birth, between 1 and 12.",
			Required:    true,
		},
		"payment_method_data.klarna.dob.day": {
			Type:        "integer",
			Description: "The day of birth, between 1 and 31.",
			Required:    true,
		},
		"payment_method_data.nz_bank_account.account_number": {
			Type:        "string",
			Description: "The account number for the bank account.",
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
		"payment_method_options.us_bank_account.financial_connections.permissions": {
			Type:        "array",
			Description: "The list of permissions to request. If this parameter is passed, the `payment_method` permission must be included. Valid permissions include: `balances`, `ownership`, `payment_method`, and `transactions`.",
		},
		"payment_method_options.card.mandate_options.currency": {
			Type:        "string",
			Description: "Currency in which future payments will be charged. Three-letter [ISO currency code](https://www.iso.org/iso-4217-currency-codes.html), in lowercase. Must be a [supported currency](https://stripe.com/docs/currencies).",
			Required:    true,
			Format:      "currency",
		},
		"payment_method_data.type": {
			Type:        "string",
			Description: "The type of the PaymentMethod. An additional hash is included on the PaymentMethod with a name matching this value. It contains additional information specific to the PaymentMethod type.",
			Required:    true,
			Enum: []resource.EnumSpec{
				{Value: "acss_debit"},
				{Value: "affirm"},
				{Value: "afterpay_clearpay"},
				{Value: "alipay"},
				{Value: "alma"},
				{Value: "amazon_pay"},
				{Value: "au_becs_debit"},
				{Value: "bacs_debit"},
				{Value: "bancontact"},
				{Value: "billie"},
				{Value: "blik"},
				{Value: "boleto"},
				{Value: "cashapp"},
				{Value: "crypto"},
				{Value: "customer_balance"},
				{Value: "eps"},
				{Value: "fpx"},
				{Value: "giropay"},
				{Value: "grabpay"},
				{Value: "ideal"},
				{Value: "kakao_pay"},
				{Value: "klarna"},
				{Value: "konbini"},
				{Value: "kr_card"},
				{Value: "link"},
				{Value: "mb_way"},
				{Value: "mobilepay"},
				{Value: "multibanco"},
				{Value: "naver_pay"},
				{Value: "nz_bank_account"},
				{Value: "oxxo"},
				{Value: "p24"},
				{Value: "pay_by_bank"},
				{Value: "payco"},
				{Value: "paynow"},
				{Value: "paypal"},
				{Value: "payto"},
				{Value: "pix"},
				{Value: "promptpay"},
				{Value: "revolut_pay"},
				{Value: "samsung_pay"},
				{Value: "satispay"},
				{Value: "sepa_debit"},
				{Value: "sofort"},
				{Value: "swish"},
				{Value: "twint"},
				{Value: "us_bank_account"},
				{Value: "wechat_pay"},
				{Value: "zip"},
			},
		},
		"payment_method_data.payto.pay_id": {
			Type:        "string",
			Description: "The PayID alias for the bank account.",
		},
		"payment_method_data.au_becs_debit.bsb_number": {
			Type:        "string",
			Description: "Bank-State-Branch number of the bank account.",
			Required:    true,
		},
		"payment_method_data.boleto.tax_id": {
			Type:        "string",
			Description: "The tax ID of the customer (CPF for individual consumers or CNPJ for businesses consumers)",
			Required:    true,
		},
		"payment_method_options.acss_debit.currency": {
			Type:        "string",
			Description: "Three-letter [ISO currency code](https://www.iso.org/iso-4217-currency-codes.html), in lowercase. Must be a [supported currency](https://stripe.com/docs/currencies).",
			Enum: []resource.EnumSpec{
				{Value: "cad"},
				{Value: "usd"},
			},
		},
		"payment_method_options.payto.mandate_options.end_date": {
			Type:        "string",
			Description: "Date, in YYYY-MM-DD format, after which payments will not be collected. Defaults to no end date.",
		},
		"payment_method_data.fpx.account_holder_type": {
			Type:        "string",
			Description: "Account holder type for FPX transaction",
			Enum: []resource.EnumSpec{
				{Value: "company"},
				{Value: "individual"},
			},
		},
		"payment_method_data.billing_details.phone": {
			Type:        "string",
			Description: "Billing phone number (including extension).",
		},
		"payment_method_data.acss_debit.account_number": {
			Type:        "string",
			Description: "Customer's bank account number.",
			Required:    true,
		},
		"payment_method_data.nz_bank_account.account_holder_name": {
			Type:        "string",
			Description: "The name on the bank account. Only required if the account holder name is different from the name of the authorized signatory collected in the PaymentMethod’s billing details.",
		},
		"description": {
			Type:        "string",
			Description: "An arbitrary string attached to the object. Often useful for displaying to users.",
		},
		"payment_method_options.card.network": {
			Type:        "string",
			Description: "Selected network to process this SetupIntent on. Depends on the available networks of the card attached to the SetupIntent. Can be only set confirm-time.",
			Enum: []resource.EnumSpec{
				{Value: "amex"},
				{Value: "cartes_bancaires"},
				{Value: "diners"},
				{Value: "discover"},
				{Value: "eftpos_au"},
				{Value: "girocard"},
				{Value: "interac"},
				{Value: "jcb"},
				{Value: "link"},
				{Value: "mastercard"},
				{Value: "unionpay"},
				{Value: "unknown"},
				{Value: "visa"},
			},
		},
		"payment_method_data.radar_options.session": {
			Type:        "string",
			Description: "A [Radar Session](https://docs.stripe.com/radar/radar-session) is a snapshot of the browser metadata and device details that help Radar make more accurate predictions on your payments.",
		},
		"payment_method_data.payto.account_number": {
			Type:        "string",
			Description: "The account number for the bank account.",
		},
		"payment_method_data.au_becs_debit.account_number": {
			Type:        "string",
			Description: "The account number for the bank account.",
			Required:    true,
		},
		"payment_method_options.acss_debit.mandate_options.default_for": {
			Type:        "array",
			Description: "List of Stripe products where this mandate can be selected automatically.",
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
		"payment_method_options.card.mandate_options.end_date": {
			Type:        "integer",
			Description: "End date of the mandate or subscription. If not provided, the mandate will be active until canceled. If provided, end date should be after start date.",
			Format:      "unix-time",
		},
		"payment_method_types": {
			Type:        "array",
			Description: "The list of payment method types (for example, card) that this SetupIntent can set up. If you don't provide this, Stripe will dynamically show relevant payment methods from your [payment method settings](https://dashboard.stripe.com/settings/payment_methods). A list of valid payment method types can be found [here](https://docs.stripe.com/api/payment_methods/object#payment_method_object-type).",
		},
		"payment_method_data.eps.bank": {
			Type:        "string",
			Description: "The customer's bank.",
			Enum: []resource.EnumSpec{
				{Value: "arzte_und_apotheker_bank"},
				{Value: "austrian_anadi_bank_ag"},
				{Value: "bank_austria"},
				{Value: "bankhaus_carl_spangler"},
				{Value: "bankhaus_schelhammer_und_schattera_ag"},
				{Value: "bawag_psk_ag"},
				{Value: "bks_bank_ag"},
				{Value: "brull_kallmus_bank_ag"},
				{Value: "btv_vier_lander_bank"},
				{Value: "capital_bank_grawe_gruppe_ag"},
				{Value: "deutsche_bank_ag"},
				{Value: "dolomitenbank"},
				{Value: "easybank_ag"},
				{Value: "erste_bank_und_sparkassen"},
				{Value: "hypo_alpeadriabank_international_ag"},
				{Value: "hypo_bank_burgenland_aktiengesellschaft"},
				{Value: "hypo_noe_lb_fur_niederosterreich_u_wien"},
				{Value: "hypo_oberosterreich_salzburg_steiermark"},
				{Value: "hypo_tirol_bank_ag"},
				{Value: "hypo_vorarlberg_bank_ag"},
				{Value: "marchfelder_bank"},
				{Value: "oberbank_ag"},
				{Value: "raiffeisen_bankengruppe_osterreich"},
				{Value: "schoellerbank_ag"},
				{Value: "sparda_bank_wien"},
				{Value: "volksbank_gruppe"},
				{Value: "volkskreditbank_ag"},
				{Value: "vr_bank_braunau"},
			},
		},
		"payment_method_data.naver_pay.funding": {
			Type:        "string",
			Description: "Whether to use Naver Pay points or a card to fund this transaction. If not provided, this defaults to `card`.",
			Enum: []resource.EnumSpec{
				{Value: "card"},
				{Value: "points"},
			},
		},
		"payment_method_options.card.three_d_secure.version": {
			Type:        "string",
			Description: "The version of 3D Secure that was performed.",
			Enum: []resource.EnumSpec{
				{Value: "1.0.2"},
				{Value: "2.1.0"},
				{Value: "2.2.0"},
				{Value: "2.3.0"},
				{Value: "2.3.1"},
			},
		},
		"payment_method_options.acss_debit.verification_method": {
			Type:        "string",
			Description: "Bank account verification method.",
			Enum: []resource.EnumSpec{
				{Value: "automatic"},
				{Value: "instant"},
				{Value: "microdeposits"},
			},
		},
		"payment_method_options.card.mandate_options.reference": {
			Type:        "string",
			Description: "Unique identifier for the mandate or subscription.",
			Required:    true,
		},
		"payment_method_data.fpx.bank": {
			Type:        "string",
			Description: "The customer's bank.",
			Required:    true,
			Enum: []resource.EnumSpec{
				{Value: "affin_bank"},
				{Value: "agrobank"},
				{Value: "alliance_bank"},
				{Value: "ambank"},
				{Value: "bank_islam"},
				{Value: "bank_muamalat"},
				{Value: "bank_of_china"},
				{Value: "bank_rakyat"},
				{Value: "bsn"},
				{Value: "cimb"},
				{Value: "deutsche_bank"},
				{Value: "hong_leong_bank"},
				{Value: "hsbc"},
				{Value: "kfh"},
				{Value: "maybank2e"},
				{Value: "maybank2u"},
				{Value: "ocbc"},
				{Value: "pb_enterprise"},
				{Value: "public_bank"},
				{Value: "rhb"},
				{Value: "standard_chartered"},
				{Value: "uob"},
			},
		},
		"payment_method_data.klarna.dob.year": {
			Type:        "integer",
			Description: "The four-digit year of birth.",
			Required:    true,
		},
		"payment_method_options.acss_debit.mandate_options.transaction_type": {
			Type:        "string",
			Description: "Transaction type of the mandate.",
			Enum: []resource.EnumSpec{
				{Value: "business"},
				{Value: "personal"},
			},
		},
		"payment_method_options.us_bank_account.networks.requested": {
			Type:        "array",
			Description: "Triggers validations to run across the selected networks",
		},
		"payment_method_options.card.three_d_secure.electronic_commerce_indicator": {
			Type:        "string",
			Description: "The Electronic Commerce Indicator (ECI) is returned by your 3D Secure\nprovider and indicates what degree of authentication was performed.",
			Enum: []resource.EnumSpec{
				{Value: "01"},
				{Value: "02"},
				{Value: "05"},
				{Value: "06"},
				{Value: "07"},
			},
		},
		"payment_method_options.card.three_d_secure.network_options.cartes_bancaires.cb_score": {
			Type:        "integer",
			Description: "The risk score returned from Cartes Bancaires in the ARes.\nmessage extension: CB-SCORE; numeric value 0-99",
		},
		"payment_method_options.klarna.currency": {
			Type:        "string",
			Description: "The currency of the SetupIntent. Three letter ISO currency code.",
			Format:      "currency",
		},
		"payment_method_options.klarna.on_demand.minimum_amount": {
			Type:        "integer",
			Description: "The lowest or minimum value you may charge a customer per purchase. You can use a value across your customer base, or segment based on customer type, country, etc.",
		},
		"payment_method_data.us_bank_account.account_number": {
			Type:        "string",
			Description: "Account number of the bank account.",
		},
		"payment_method_options.paypal.billing_agreement_id": {
			Type:        "string",
			Description: "The PayPal Billing Agreement ID (BAID). This is an ID generated by PayPal which represents the mandate between the merchant and the customer.",
		},
		"payment_method_options.bacs_debit.mandate_options.reference_prefix": {
			Type:        "string",
			Description: "Prefix used to generate the Mandate reference. Must be at most 12 characters long. Must consist of only uppercase letters, numbers, spaces, or the following special characters: '/', '_', '-', '&', '.'. Cannot begin with 'DDIC' or 'STRIPE'.",
		},
		"payment_method_options.payto.mandate_options.amount": {
			Type:        "integer",
			Description: "Amount that will be collected. It is required when `amount_type` is `fixed`.",
		},
		"payment_method_options.card.mandate_options.amount_type": {
			Type:        "string",
			Description: "One of `fixed` or `maximum`. If `fixed`, the `amount` param refers to the exact amount to be charged in future payments. If `maximum`, the amount charged can be up to the value passed for the `amount` param.",
			Required:    true,
			Enum: []resource.EnumSpec{
				{Value: "fixed"},
				{Value: "maximum"},
			},
		},
		"excluded_payment_method_types": {
			Type:        "array",
			Description: "The list of payment method types to exclude from use with this SetupIntent.",
		},
		"payment_method_data.nz_bank_account.reference": {
			Type: "string",
		},
		"payment_method_data.nz_bank_account.suffix": {
			Type:        "string",
			Description: "The suffix of the bank account number.",
			Required:    true,
		},
		"payment_method_options.us_bank_account.financial_connections.filters.account_subcategories": {
			Type:        "array",
			Description: "The account subcategories to use to filter for selectable accounts. Valid subcategories are `checking` and `savings`.",
		},
		"payment_method_options.us_bank_account.financial_connections.return_url": {
			Type:        "string",
			Description: "For webview integrations only. Upon completing OAuth login in the native browser, the user will be redirected to this URL to return to your app.",
		},
		"payment_method_options.card.three_d_secure.requestor_challenge_indicator": {
			Type:        "string",
			Description: "The challenge indicator (`threeDSRequestorChallengeInd`) which was requested in the\nAReq sent to the card Issuer's ACS. A string containing 2 digits from 01-99.",
		},
		"payment_method_options.klarna.on_demand.purchase_interval_count": {
			Type:        "integer",
			Description: "The number of `purchase_interval` between charges",
		},
		"payment_method_options.klarna.on_demand.average_amount": {
			Type:        "integer",
			Description: "Your average amount value. You can use a value across your customer base, or segment based on customer type, country, etc.",
		},
		"customer": {
			Type:        "string",
			Description: "ID of the Customer this SetupIntent belongs to, if one exists.\n\nIf present, the SetupIntent's payment method will be attached to the Customer on successful setup. Payment methods attached to other Customers cannot be used with this SetupIntent.",
		},
		"payment_method_data.payto.bsb_number": {
			Type:        "string",
			Description: "Bank-State-Branch number of the bank account.",
		},
		"payment_method_data.us_bank_account.routing_number": {
			Type:        "string",
			Description: "Routing number of the bank account.",
		},
		"payment_method_options.sepa_debit.mandate_options.reference_prefix": {
			Type:        "string",
			Description: "Prefix used to generate the Mandate reference. Must be at most 12 characters long. Must consist of only uppercase letters, numbers, spaces, or the following special characters: '/', '_', '-', '&', '.'. Cannot begin with 'STRIPE'.",
		},
		"payment_method_options.us_bank_account.financial_connections.prefetch": {
			Type:        "array",
			Description: "List of data features that you would like to retrieve upon account creation.",
		},
		"payment_method_options.card.mandate_options.interval": {
			Type:        "string",
			Description: "Specifies payment frequency. One of `day`, `week`, `month`, `year`, or `sporadic`.",
			Required:    true,
			Enum: []resource.EnumSpec{
				{Value: "day"},
				{Value: "month"},
				{Value: "sporadic"},
				{Value: "week"},
				{Value: "year"},
			},
		},
		"attach_to_self": {
			Type:        "boolean",
			Description: "If present, the SetupIntent's payment method will be attached to the in-context Stripe Account.\n\nIt can only be used for this Stripe Account’s own money movement flows like InboundTransfer and OutboundTransfers. It cannot be set to true when setting up a PaymentMethod for a Customer, and defaults to false when attaching a PaymentMethod to a Customer.",
		},
		"payment_method_data.billing_details.tax_id": {
			Type:        "string",
			Description: "Taxpayer identification number. Used only for transactions between LATAM buyers and non-LATAM sellers.",
		},
		"payment_method_data.acss_debit.institution_number": {
			Type:        "string",
			Description: "Institution number of the customer's bank.",
			Required:    true,
		},
		"customer_account": {
			Type:        "string",
			Description: "ID of the Account this SetupIntent belongs to, if one exists.\n\nIf present, the SetupIntent's payment method will be attached to the Account on successful setup. Payment methods attached to other Accounts cannot be used with this SetupIntent.",
		},
		"payment_method": {
			Type:        "string",
			Description: "ID of the payment method (a PaymentMethod, Card, or saved Source object) to attach to this SetupIntent. To unset this field to null, pass in an empty string.",
		},
		"payment_method_options.payto.mandate_options.amount_type": {
			Type:        "string",
			Description: "The type of amount that will be collected. The amount charged must be exact or up to the value of `amount` param for `fixed` or `maximum` type respectively. Defaults to `maximum`.",
			Enum: []resource.EnumSpec{
				{Value: "fixed"},
				{Value: "maximum"},
			},
		},
		"payment_method_options.card.mandate_options.interval_count": {
			Type:        "integer",
			Description: "The number of intervals between payments. For example, `interval=month` and `interval_count=3` indicates one payment every three months. Maximum of one year interval allowed (1 year, 12 months, or 52 weeks). This parameter is optional when `interval=sporadic`.",
		},
		"payment_method_data.nz_bank_account.bank_code": {
			Type:        "string",
			Description: "The numeric code for the bank account's bank.",
			Required:    true,
		},
		"payment_method_data.bacs_debit.account_number": {
			Type:        "string",
			Description: "Account number of the bank account that the funds will be debited from.",
		},
		"payment_method_data.bacs_debit.sort_code": {
			Type:        "string",
			Description: "Sort code of the bank account. (e.g., `10-20-30`)",
		},
		"payment_method_options.us_bank_account.mandate_options.collection_method": {
			Type:        "string",
			Description: "The method used to collect offline mandate customer acceptance.",
			Enum: []resource.EnumSpec{
				{Value: "paper"},
			},
		},
		"payment_method_options.card.three_d_secure.network_options.cartes_bancaires.cb_avalgo": {
			Type:        "string",
			Description: "The cryptogram calculation algorithm used by the card Issuer's ACS\nto calculate the Authentication cryptogram. Also known as `cavvAlgorithm`.\nmessageExtension: CB-AVALGO",
			Required:    true,
			Enum: []resource.EnumSpec{
				{Value: "0"},
				{Value: "1"},
				{Value: "2"},
				{Value: "3"},
				{Value: "4"},
				{Value: "A"},
			},
		},
		"payment_method_options.klarna.on_demand.maximum_amount": {
			Type:        "integer",
			Description: "The maximum value you may charge a customer per purchase. You can use a value across your customer base, or segment based on customer type, country, etc.",
		},
		"payment_method_data.p24.bank": {
			Type:        "string",
			Description: "The customer's bank.",
			Enum: []resource.EnumSpec{
				{Value: "alior_bank"},
				{Value: "bank_millennium"},
				{Value: "bank_nowy_bfg_sa"},
				{Value: "bank_pekao_sa"},
				{Value: "banki_spbdzielcze"},
				{Value: "blik"},
				{Value: "bnp_paribas"},
				{Value: "boz"},
				{Value: "citi_handlowy"},
				{Value: "credit_agricole"},
				{Value: "envelobank"},
				{Value: "etransfer_pocztowy24"},
				{Value: "getin_bank"},
				{Value: "ideabank"},
				{Value: "ing"},
				{Value: "inteligo"},
				{Value: "mbank_mtransfer"},
				{Value: "nest_przelew"},
				{Value: "noble_pay"},
				{Value: "pbac_z_ipko"},
				{Value: "plus_bank"},
				{Value: "santander_przelew24"},
				{Value: "tmobile_usbugi_bankowe"},
				{Value: "toyota_bank"},
				{Value: "velobank"},
				{Value: "volkswagen_bank"},
			},
		},
		"payment_method_data.sepa_debit.iban": {
			Type:        "string",
			Description: "IBAN of the bank account.",
			Required:    true,
		},
		"payment_method_data.us_bank_account.account_holder_type": {
			Type:        "string",
			Description: "Account holder type: individual or company.",
			Enum: []resource.EnumSpec{
				{Value: "company"},
				{Value: "individual"},
			},
		},
		"payment_method_options.acss_debit.mandate_options.interval_description": {
			Type:        "string",
			Description: "Description of the mandate interval. Only required if 'payment_schedule' parameter is 'interval' or 'combined'.",
		},
		"payment_method_options.us_bank_account.verification_method": {
			Type:        "string",
			Description: "Bank account verification method.",
			Enum: []resource.EnumSpec{
				{Value: "automatic"},
				{Value: "instant"},
				{Value: "microdeposits"},
			},
		},
		"payment_method_options.card.three_d_secure.cryptogram": {
			Type:        "string",
			Description: "The cryptogram, also known as the \"authentication value\" (AAV, CAVV or\nAEVV). This value is 20 bytes, base64-encoded into a 28-character string.\n(Most 3D Secure providers will return the base64-encoded version, which\nis what you should specify here.)",
		},
		"payment_method_options.klarna.on_demand.purchase_interval": {
			Type:        "string",
			Description: "Interval at which the customer is making purchases",
			Enum: []resource.EnumSpec{
				{Value: "day"},
				{Value: "month"},
				{Value: "week"},
				{Value: "year"},
			},
		},
	},
}

var V1SetupIntentsCancel = resource.OperationSpec{
	Name:    "cancel",
	Path:    "/v1/setup_intents/{intent}/cancel",
	Method:  "POST",
	Summary: "Cancel a SetupIntent",
	Params: map[string]*resource.ParamSpec{
		"cancellation_reason": {
			Type:        "string",
			Description: "Reason for canceling this SetupIntent. Possible values are: `abandoned`, `requested_by_customer`, or `duplicate`",
			Enum: []resource.EnumSpec{
				{Value: "abandoned"},
				{Value: "duplicate"},
				{Value: "requested_by_customer"},
			},
		},
	},
}

var V1SetupIntentsConfirm = resource.OperationSpec{
	Name:    "confirm",
	Path:    "/v1/setup_intents/{intent}/confirm",
	Method:  "POST",
	Summary: "Confirm a SetupIntent",
	Params: map[string]*resource.ParamSpec{
		"payment_method_options.card.three_d_secure.cryptogram": {
			Type:        "string",
			Description: "The cryptogram, also known as the \"authentication value\" (AAV, CAVV or\nAEVV). This value is 20 bytes, base64-encoded into a 28-character string.\n(Most 3D Secure providers will return the base64-encoded version, which\nis what you should specify here.)",
		},
		"confirmation_token": {
			Type:        "string",
			Description: "ID of the ConfirmationToken used to confirm this SetupIntent.\n\nIf the provided ConfirmationToken contains properties that are also being provided in this request, such as `payment_method`, then the values in this request will take precedence.",
		},
		"payment_method_data.billing_details.phone": {
			Type:        "string",
			Description: "Billing phone number (including extension).",
		},
		"payment_method_data.nz_bank_account.account_holder_name": {
			Type:        "string",
			Description: "The name on the bank account. Only required if the account holder name is different from the name of the authorized signatory collected in the PaymentMethod’s billing details.",
		},
		"payment_method_data.naver_pay.funding": {
			Type:        "string",
			Description: "Whether to use Naver Pay points or a card to fund this transaction. If not provided, this defaults to `card`.",
			Enum: []resource.EnumSpec{
				{Value: "card"},
				{Value: "points"},
			},
		},
		"payment_method_options.card.mandate_options.interval": {
			Type:        "string",
			Description: "Specifies payment frequency. One of `day`, `week`, `month`, `year`, or `sporadic`.",
			Required:    true,
			Enum: []resource.EnumSpec{
				{Value: "day"},
				{Value: "month"},
				{Value: "sporadic"},
				{Value: "week"},
				{Value: "year"},
			},
		},
		"payment_method_data.sepa_debit.iban": {
			Type:        "string",
			Description: "IBAN of the bank account.",
			Required:    true,
		},
		"payment_method_data.nz_bank_account.suffix": {
			Type:        "string",
			Description: "The suffix of the bank account number.",
			Required:    true,
		},
		"payment_method_options.klarna.preferred_locale": {
			Type:        "string",
			Description: "Preferred language of the Klarna authorization page that the customer is redirected to",
			Enum: []resource.EnumSpec{
				{Value: "cs-CZ"},
				{Value: "da-DK"},
				{Value: "de-AT"},
				{Value: "de-CH"},
				{Value: "de-DE"},
				{Value: "el-GR"},
				{Value: "en-AT"},
				{Value: "en-AU"},
				{Value: "en-BE"},
				{Value: "en-CA"},
				{Value: "en-CH"},
				{Value: "en-CZ"},
				{Value: "en-DE"},
				{Value: "en-DK"},
				{Value: "en-ES"},
				{Value: "en-FI"},
				{Value: "en-FR"},
				{Value: "en-GB"},
				{Value: "en-GR"},
				{Value: "en-IE"},
				{Value: "en-IT"},
				{Value: "en-NL"},
				{Value: "en-NO"},
				{Value: "en-NZ"},
				{Value: "en-PL"},
				{Value: "en-PT"},
				{Value: "en-RO"},
				{Value: "en-SE"},
				{Value: "en-US"},
				{Value: "es-ES"},
				{Value: "es-US"},
				{Value: "fi-FI"},
				{Value: "fr-BE"},
				{Value: "fr-CA"},
				{Value: "fr-CH"},
				{Value: "fr-FR"},
				{Value: "it-CH"},
				{Value: "it-IT"},
				{Value: "nb-NO"},
				{Value: "nl-BE"},
				{Value: "nl-NL"},
				{Value: "pl-PL"},
				{Value: "pt-PT"},
				{Value: "ro-RO"},
				{Value: "sv-FI"},
				{Value: "sv-SE"},
			},
		},
		"payment_method_options.us_bank_account.financial_connections.permissions": {
			Type:        "array",
			Description: "The list of permissions to request. If this parameter is passed, the `payment_method` permission must be included. Valid permissions include: `balances`, `ownership`, `payment_method`, and `transactions`.",
		},
		"payment_method_options.acss_debit.mandate_options.transaction_type": {
			Type:        "string",
			Description: "Transaction type of the mandate.",
			Enum: []resource.EnumSpec{
				{Value: "business"},
				{Value: "personal"},
			},
		},
		"payment_method_options.acss_debit.verification_method": {
			Type:        "string",
			Description: "Bank account verification method.",
			Enum: []resource.EnumSpec{
				{Value: "automatic"},
				{Value: "instant"},
				{Value: "microdeposits"},
			},
		},
		"payment_method_options.card.three_d_secure.network_options.cartes_bancaires.cb_avalgo": {
			Type:        "string",
			Description: "The cryptogram calculation algorithm used by the card Issuer's ACS\nto calculate the Authentication cryptogram. Also known as `cavvAlgorithm`.\nmessageExtension: CB-AVALGO",
			Required:    true,
			Enum: []resource.EnumSpec{
				{Value: "0"},
				{Value: "1"},
				{Value: "2"},
				{Value: "3"},
				{Value: "4"},
				{Value: "A"},
			},
		},
		"payment_method_options.card.three_d_secure.network_options.cartes_bancaires.cb_score": {
			Type:        "integer",
			Description: "The risk score returned from Cartes Bancaires in the ARes.\nmessage extension: CB-SCORE; numeric value 0-99",
		},
		"payment_method_data.billing_details.email": {
			Type:        "string",
			Description: "Email address.",
		},
		"payment_method_data.billing_details.tax_id": {
			Type:        "string",
			Description: "Taxpayer identification number. Used only for transactions between LATAM buyers and non-LATAM sellers.",
		},
		"payment_method_data.nz_bank_account.bank_code": {
			Type:        "string",
			Description: "The numeric code for the bank account's bank.",
			Required:    true,
		},
		"payment_method_data.nz_bank_account.reference": {
			Type: "string",
		},
		"payment_method_data.us_bank_account.routing_number": {
			Type:        "string",
			Description: "Routing number of the bank account.",
		},
		"payment_method_options.bacs_debit.mandate_options.reference_prefix": {
			Type:        "string",
			Description: "Prefix used to generate the Mandate reference. Must be at most 12 characters long. Must consist of only uppercase letters, numbers, spaces, or the following special characters: '/', '_', '-', '&', '.'. Cannot begin with 'DDIC' or 'STRIPE'.",
		},
		"payment_method_options.us_bank_account.financial_connections.prefetch": {
			Type:        "array",
			Description: "List of data features that you would like to retrieve upon account creation.",
		},
		"payment_method_options.us_bank_account.financial_connections.return_url": {
			Type:        "string",
			Description: "For webview integrations only. Upon completing OAuth login in the native browser, the user will be redirected to this URL to return to your app.",
		},
		"payment_method_data.boleto.tax_id": {
			Type:        "string",
			Description: "The tax ID of the customer (CPF for individual consumers or CNPJ for businesses consumers)",
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
		"payment_method_options.us_bank_account.verification_method": {
			Type:        "string",
			Description: "Bank account verification method.",
			Enum: []resource.EnumSpec{
				{Value: "automatic"},
				{Value: "instant"},
				{Value: "microdeposits"},
			},
		},
		"payment_method_options.card.mandate_options.start_date": {
			Type:        "integer",
			Description: "Start date of the mandate or subscription. Start date should not be lesser than yesterday.",
			Required:    true,
			Format:      "unix-time",
		},
		"payment_method_options.card.three_d_secure.ares_trans_status": {
			Type:        "string",
			Description: "The `transStatus` returned from the card Issuer’s ACS in the ARes.",
			Enum: []resource.EnumSpec{
				{Value: "A"},
				{Value: "C"},
				{Value: "I"},
				{Value: "N"},
				{Value: "R"},
				{Value: "U"},
				{Value: "Y"},
			},
		},
		"payment_method_options.card.three_d_secure.network_options.cartes_bancaires.cb_exemption": {
			Type:        "string",
			Description: "The exemption indicator returned from Cartes Bancaires in the ARes.\nmessage extension: CB-EXEMPTION; string (4 characters)\nThis is a 3 byte bitmap (low significant byte first and most significant\nbit first) that has been Base64 encoded",
		},
		"payment_method_options.card.three_d_secure.requestor_challenge_indicator": {
			Type:        "string",
			Description: "The challenge indicator (`threeDSRequestorChallengeInd`) which was requested in the\nAReq sent to the card Issuer's ACS. A string containing 2 digits from 01-99.",
		},
		"payment_method_data.sofort.country": {
			Type:        "string",
			Description: "Two-letter ISO code representing the country the bank account is located in.",
			Required:    true,
			Enum: []resource.EnumSpec{
				{Value: "AT"},
				{Value: "BE"},
				{Value: "DE"},
				{Value: "ES"},
				{Value: "IT"},
				{Value: "NL"},
			},
		},
		"payment_method_data.au_becs_debit.account_number": {
			Type:        "string",
			Description: "The account number for the bank account.",
			Required:    true,
		},
		"payment_method_data.klarna.dob.month": {
			Type:        "integer",
			Description: "The month of birth, between 1 and 12.",
			Required:    true,
		},
		"payment_method_options.klarna.on_demand.average_amount": {
			Type:        "integer",
			Description: "Your average amount value. You can use a value across your customer base, or segment based on customer type, country, etc.",
		},
		"payment_method_options.acss_debit.mandate_options.interval_description": {
			Type:        "string",
			Description: "Description of the mandate interval. Only required if 'payment_schedule' parameter is 'interval' or 'combined'.",
		},
		"payment_method_options.card.moto": {
			Type:        "boolean",
			Description: "When specified, this parameter signals that a card has been collected\nas MOTO (Mail Order Telephone Order) and thus out of scope for SCA. This\nparameter can only be provided during confirmation.",
		},
		"return_url": {
			Type:        "string",
			Description: "The URL to redirect your customer back to after they authenticate on the payment method's app or site.\nIf you'd prefer to redirect to a mobile application, you can alternatively supply an application URI scheme.\nThis parameter is only used for cards and other redirect-based payment methods.",
		},
		"use_stripe_sdk": {
			Type:        "boolean",
			Description: "Set to `true` when confirming server-side and using Stripe.js, iOS, or Android client-side SDKs to handle the next actions.",
		},
		"payment_method_data.acss_debit.account_number": {
			Type:        "string",
			Description: "Customer's bank account number.",
			Required:    true,
		},
		"payment_method_data.payto.bsb_number": {
			Type:        "string",
			Description: "Bank-State-Branch number of the bank account.",
		},
		"payment_method_options.acss_debit.mandate_options.default_for": {
			Type:        "array",
			Description: "List of Stripe products where this mandate can be selected automatically.",
		},
		"payment_method_options.card.mandate_options.amount_type": {
			Type:        "string",
			Description: "One of `fixed` or `maximum`. If `fixed`, the `amount` param refers to the exact amount to be charged in future payments. If `maximum`, the amount charged can be up to the value passed for the `amount` param.",
			Required:    true,
			Enum: []resource.EnumSpec{
				{Value: "fixed"},
				{Value: "maximum"},
			},
		},
		"payment_method_options.klarna.on_demand.purchase_interval": {
			Type:        "string",
			Description: "Interval at which the customer is making purchases",
			Enum: []resource.EnumSpec{
				{Value: "day"},
				{Value: "month"},
				{Value: "week"},
				{Value: "year"},
			},
		},
		"payment_method_options.us_bank_account.mandate_options.collection_method": {
			Type:        "string",
			Description: "The method used to collect offline mandate customer acceptance.",
			Enum: []resource.EnumSpec{
				{Value: "paper"},
			},
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
		"payment_method_options.card.three_d_secure.transaction_id": {
			Type:        "string",
			Description: "For 3D Secure 1, the XID. For 3D Secure 2, the Directory Server\nTransaction ID (dsTransID).",
		},
		"payment_method_data.bacs_debit.sort_code": {
			Type:        "string",
			Description: "Sort code of the bank account. (e.g., `10-20-30`)",
		},
		"payment_method_data.fpx.bank": {
			Type:        "string",
			Description: "The customer's bank.",
			Required:    true,
			Enum: []resource.EnumSpec{
				{Value: "affin_bank"},
				{Value: "agrobank"},
				{Value: "alliance_bank"},
				{Value: "ambank"},
				{Value: "bank_islam"},
				{Value: "bank_muamalat"},
				{Value: "bank_of_china"},
				{Value: "bank_rakyat"},
				{Value: "bsn"},
				{Value: "cimb"},
				{Value: "deutsche_bank"},
				{Value: "hong_leong_bank"},
				{Value: "hsbc"},
				{Value: "kfh"},
				{Value: "maybank2e"},
				{Value: "maybank2u"},
				{Value: "ocbc"},
				{Value: "pb_enterprise"},
				{Value: "public_bank"},
				{Value: "rhb"},
				{Value: "standard_chartered"},
				{Value: "uob"},
			},
		},
		"payment_method_options.klarna.on_demand.minimum_amount": {
			Type:        "integer",
			Description: "The lowest or minimum value you may charge a customer per purchase. You can use a value across your customer base, or segment based on customer type, country, etc.",
		},
		"payment_method_options.card.mandate_options.supported_types": {
			Type:        "array",
			Description: "Specifies the type of mandates supported. Possible values are `india`.",
		},
		"payment_method_data.bacs_debit.account_number": {
			Type:        "string",
			Description: "Account number of the bank account that the funds will be debited from.",
		},
		"payment_method_data.nz_bank_account.branch_code": {
			Type:        "string",
			Description: "The numeric code for the bank account's bank branch.",
			Required:    true,
		},
		"payment_method_options.payto.mandate_options.payments_per_period": {
			Type:        "integer",
			Description: "The number of payments that will be made during a payment period. Defaults to 1 except for when `payment_schedule` is `adhoc`. In that case, it defaults to no limit.",
		},
		"payment_method_options.acss_debit.currency": {
			Type:        "string",
			Description: "Three-letter [ISO currency code](https://www.iso.org/iso-4217-currency-codes.html), in lowercase. Must be a [supported currency](https://stripe.com/docs/currencies).",
			Enum: []resource.EnumSpec{
				{Value: "cad"},
				{Value: "usd"},
			},
		},
		"payment_method_options.card.mandate_options.currency": {
			Type:        "string",
			Description: "Currency in which future payments will be charged. Three-letter [ISO currency code](https://www.iso.org/iso-4217-currency-codes.html), in lowercase. Must be a [supported currency](https://stripe.com/docs/currencies).",
			Required:    true,
			Format:      "currency",
		},
		"payment_method": {
			Type:        "string",
			Description: "ID of the payment method (a PaymentMethod, Card, or saved Source object) to attach to this SetupIntent.",
		},
		"payment_method_data.au_becs_debit.bsb_number": {
			Type:        "string",
			Description: "Bank-State-Branch number of the bank account.",
			Required:    true,
		},
		"payment_method_data.nz_bank_account.account_number": {
			Type:        "string",
			Description: "The account number for the bank account.",
			Required:    true,
		},
		"payment_method_data.acss_debit.transit_number": {
			Type:        "string",
			Description: "Transit number of the customer's bank.",
			Required:    true,
		},
		"payment_method_data.us_bank_account.account_holder_type": {
			Type:        "string",
			Description: "Account holder type: individual or company.",
			Enum: []resource.EnumSpec{
				{Value: "company"},
				{Value: "individual"},
			},
		},
		"payment_method_data.us_bank_account.account_number": {
			Type:        "string",
			Description: "Account number of the bank account.",
		},
		"payment_method_options.link.persistent_token": {
			Type:        "string",
			Description: "[Deprecated] This is a legacy parameter that no longer has any function.",
		},
		"payment_method_options.us_bank_account.networks.requested": {
			Type:        "array",
			Description: "Triggers validations to run across the selected networks",
		},
		"payment_method_data.allow_redisplay": {
			Type:        "string",
			Description: "This field indicates whether this payment method can be shown again to its customer in a checkout flow. Stripe products such as Checkout and Elements use this field to determine whether a payment method can be shown as a saved payment method in a checkout flow. The field defaults to `unspecified`.",
			Enum: []resource.EnumSpec{
				{Value: "always"},
				{Value: "limited"},
				{Value: "unspecified"},
			},
		},
		"payment_method_data.payto.pay_id": {
			Type:        "string",
			Description: "The PayID alias for the bank account.",
		},
		"payment_method_data.klarna.dob.year": {
			Type:        "integer",
			Description: "The four-digit year of birth.",
			Required:    true,
		},
		"payment_method_data.radar_options.session": {
			Type:        "string",
			Description: "A [Radar Session](https://docs.stripe.com/radar/radar-session) is a snapshot of the browser metadata and device details that help Radar make more accurate predictions on your payments.",
		},
		"payment_method_options.klarna.currency": {
			Type:        "string",
			Description: "The currency of the SetupIntent. Three letter ISO currency code.",
			Format:      "currency",
		},
		"payment_method_options.card.mandate_options.end_date": {
			Type:        "integer",
			Description: "End date of the mandate or subscription. If not provided, the mandate will be active until canceled. If provided, end date should be after start date.",
			Format:      "unix-time",
		},
		"payment_method_options.card.mandate_options.interval_count": {
			Type:        "integer",
			Description: "The number of intervals between payments. For example, `interval=month` and `interval_count=3` indicates one payment every three months. Maximum of one year interval allowed (1 year, 12 months, or 52 weeks). This parameter is optional when `interval=sporadic`.",
		},
		"payment_method_options.card.mandate_options.reference": {
			Type:        "string",
			Description: "Unique identifier for the mandate or subscription.",
			Required:    true,
		},
		"payment_method_data.us_bank_account.financial_connections_account": {
			Type:        "string",
			Description: "The ID of a Financial Connections Account to use as a payment method.",
		},
		"payment_method_options.klarna.on_demand.purchase_interval_count": {
			Type:        "integer",
			Description: "The number of `purchase_interval` between charges",
		},
		"payment_method_options.card.mandate_options.description": {
			Type:        "string",
			Description: "A description of the mandate or subscription that is meant to be displayed to the customer.",
		},
		"payment_method_options.card.network": {
			Type:        "string",
			Description: "Selected network to process this SetupIntent on. Depends on the available networks of the card attached to the SetupIntent. Can be only set confirm-time.",
			Enum: []resource.EnumSpec{
				{Value: "amex"},
				{Value: "cartes_bancaires"},
				{Value: "diners"},
				{Value: "discover"},
				{Value: "eftpos_au"},
				{Value: "girocard"},
				{Value: "interac"},
				{Value: "jcb"},
				{Value: "link"},
				{Value: "mastercard"},
				{Value: "unionpay"},
				{Value: "unknown"},
				{Value: "visa"},
			},
		},
		"payment_method_options.card.three_d_secure.electronic_commerce_indicator": {
			Type:        "string",
			Description: "The Electronic Commerce Indicator (ECI) is returned by your 3D Secure\nprovider and indicates what degree of authentication was performed.",
			Enum: []resource.EnumSpec{
				{Value: "01"},
				{Value: "02"},
				{Value: "05"},
				{Value: "06"},
				{Value: "07"},
			},
		},
		"payment_method_data.ideal.bank": {
			Type:        "string",
			Description: "The customer's bank. Only use this parameter for existing customers. Don't use it for new customers.",
			Enum: []resource.EnumSpec{
				{Value: "abn_amro"},
				{Value: "adyen"},
				{Value: "asn_bank"},
				{Value: "bunq"},
				{Value: "buut"},
				{Value: "finom"},
				{Value: "handelsbanken"},
				{Value: "ing"},
				{Value: "knab"},
				{Value: "mollie"},
				{Value: "moneyou"},
				{Value: "n26"},
				{Value: "nn"},
				{Value: "rabobank"},
				{Value: "regiobank"},
				{Value: "revolut"},
				{Value: "sns_bank"},
				{Value: "triodos_bank"},
				{Value: "van_lanschot"},
				{Value: "yoursafe"},
			},
		},
		"payment_method_data.type": {
			Type:        "string",
			Description: "The type of the PaymentMethod. An additional hash is included on the PaymentMethod with a name matching this value. It contains additional information specific to the PaymentMethod type.",
			Required:    true,
			Enum: []resource.EnumSpec{
				{Value: "acss_debit"},
				{Value: "affirm"},
				{Value: "afterpay_clearpay"},
				{Value: "alipay"},
				{Value: "alma"},
				{Value: "amazon_pay"},
				{Value: "au_becs_debit"},
				{Value: "bacs_debit"},
				{Value: "bancontact"},
				{Value: "billie"},
				{Value: "blik"},
				{Value: "boleto"},
				{Value: "cashapp"},
				{Value: "crypto"},
				{Value: "customer_balance"},
				{Value: "eps"},
				{Value: "fpx"},
				{Value: "giropay"},
				{Value: "grabpay"},
				{Value: "ideal"},
				{Value: "kakao_pay"},
				{Value: "klarna"},
				{Value: "konbini"},
				{Value: "kr_card"},
				{Value: "link"},
				{Value: "mb_way"},
				{Value: "mobilepay"},
				{Value: "multibanco"},
				{Value: "naver_pay"},
				{Value: "nz_bank_account"},
				{Value: "oxxo"},
				{Value: "p24"},
				{Value: "pay_by_bank"},
				{Value: "payco"},
				{Value: "paynow"},
				{Value: "paypal"},
				{Value: "payto"},
				{Value: "pix"},
				{Value: "promptpay"},
				{Value: "revolut_pay"},
				{Value: "samsung_pay"},
				{Value: "satispay"},
				{Value: "sepa_debit"},
				{Value: "sofort"},
				{Value: "swish"},
				{Value: "twint"},
				{Value: "us_bank_account"},
				{Value: "wechat_pay"},
				{Value: "zip"},
			},
		},
		"payment_method_data.eps.bank": {
			Type:        "string",
			Description: "The customer's bank.",
			Enum: []resource.EnumSpec{
				{Value: "arzte_und_apotheker_bank"},
				{Value: "austrian_anadi_bank_ag"},
				{Value: "bank_austria"},
				{Value: "bankhaus_carl_spangler"},
				{Value: "bankhaus_schelhammer_und_schattera_ag"},
				{Value: "bawag_psk_ag"},
				{Value: "bks_bank_ag"},
				{Value: "brull_kallmus_bank_ag"},
				{Value: "btv_vier_lander_bank"},
				{Value: "capital_bank_grawe_gruppe_ag"},
				{Value: "deutsche_bank_ag"},
				{Value: "dolomitenbank"},
				{Value: "easybank_ag"},
				{Value: "erste_bank_und_sparkassen"},
				{Value: "hypo_alpeadriabank_international_ag"},
				{Value: "hypo_bank_burgenland_aktiengesellschaft"},
				{Value: "hypo_noe_lb_fur_niederosterreich_u_wien"},
				{Value: "hypo_oberosterreich_salzburg_steiermark"},
				{Value: "hypo_tirol_bank_ag"},
				{Value: "hypo_vorarlberg_bank_ag"},
				{Value: "marchfelder_bank"},
				{Value: "oberbank_ag"},
				{Value: "raiffeisen_bankengruppe_osterreich"},
				{Value: "schoellerbank_ag"},
				{Value: "sparda_bank_wien"},
				{Value: "volksbank_gruppe"},
				{Value: "volkskreditbank_ag"},
				{Value: "vr_bank_braunau"},
			},
		},
		"payment_method_data.us_bank_account.account_type": {
			Type:        "string",
			Description: "Account type: checkings or savings. Defaults to checking if omitted.",
			Enum: []resource.EnumSpec{
				{Value: "checking"},
				{Value: "savings"},
			},
		},
		"payment_method_options.klarna.on_demand.maximum_amount": {
			Type:        "integer",
			Description: "The maximum value you may charge a customer per purchase. You can use a value across your customer base, or segment based on customer type, country, etc.",
		},
		"payment_method_options.paypal.billing_agreement_id": {
			Type:        "string",
			Description: "The PayPal Billing Agreement ID (BAID). This is an ID generated by PayPal which represents the mandate between the merchant and the customer.",
		},
		"payment_method_options.payto.mandate_options.end_date": {
			Type:        "string",
			Description: "Date, in YYYY-MM-DD format, after which payments will not be collected. Defaults to no end date.",
		},
		"payment_method_options.payto.mandate_options.start_date": {
			Type:        "string",
			Description: "Date, in YYYY-MM-DD format, from which payments will be collected. Defaults to confirmation time.",
		},
		"payment_method_data.p24.bank": {
			Type:        "string",
			Description: "The customer's bank.",
			Enum: []resource.EnumSpec{
				{Value: "alior_bank"},
				{Value: "bank_millennium"},
				{Value: "bank_nowy_bfg_sa"},
				{Value: "bank_pekao_sa"},
				{Value: "banki_spbdzielcze"},
				{Value: "blik"},
				{Value: "bnp_paribas"},
				{Value: "boz"},
				{Value: "citi_handlowy"},
				{Value: "credit_agricole"},
				{Value: "envelobank"},
				{Value: "etransfer_pocztowy24"},
				{Value: "getin_bank"},
				{Value: "ideabank"},
				{Value: "ing"},
				{Value: "inteligo"},
				{Value: "mbank_mtransfer"},
				{Value: "nest_przelew"},
				{Value: "noble_pay"},
				{Value: "pbac_z_ipko"},
				{Value: "plus_bank"},
				{Value: "santander_przelew24"},
				{Value: "tmobile_usbugi_bankowe"},
				{Value: "toyota_bank"},
				{Value: "velobank"},
				{Value: "volkswagen_bank"},
			},
		},
		"payment_method_data.acss_debit.institution_number": {
			Type:        "string",
			Description: "Institution number of the customer's bank.",
			Required:    true,
		},
		"payment_method_data.klarna.dob.day": {
			Type:        "integer",
			Description: "The day of birth, between 1 and 31.",
			Required:    true,
		},
		"payment_method_options.payto.mandate_options.amount": {
			Type:        "integer",
			Description: "Amount that will be collected. It is required when `amount_type` is `fixed`.",
		},
		"payment_method_options.us_bank_account.financial_connections.filters.account_subcategories": {
			Type:        "array",
			Description: "The account subcategories to use to filter for selectable accounts. Valid subcategories are `checking` and `savings`.",
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
		"payment_method_options.card.three_d_secure.version": {
			Type:        "string",
			Description: "The version of 3D Secure that was performed.",
			Enum: []resource.EnumSpec{
				{Value: "1.0.2"},
				{Value: "2.1.0"},
				{Value: "2.2.0"},
				{Value: "2.3.0"},
				{Value: "2.3.1"},
			},
		},
		"payment_method_data.billing_details.name": {
			Type:        "string",
			Description: "Full name.",
		},
		"payment_method_data.payto.account_number": {
			Type:        "string",
			Description: "The account number for the bank account.",
		},
		"payment_method_options.sepa_debit.mandate_options.reference_prefix": {
			Type:        "string",
			Description: "Prefix used to generate the Mandate reference. Must be at most 12 characters long. Must consist of only uppercase letters, numbers, spaces, or the following special characters: '/', '_', '-', '&', '.'. Cannot begin with 'STRIPE'.",
		},
		"payment_method_data.fpx.account_holder_type": {
			Type:        "string",
			Description: "Account holder type for FPX transaction",
			Enum: []resource.EnumSpec{
				{Value: "company"},
				{Value: "individual"},
			},
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
		"payment_method_options.payto.mandate_options.amount_type": {
			Type:        "string",
			Description: "The type of amount that will be collected. The amount charged must be exact or up to the value of `amount` param for `fixed` or `maximum` type respectively. Defaults to `maximum`.",
			Enum: []resource.EnumSpec{
				{Value: "fixed"},
				{Value: "maximum"},
			},
		},
		"payment_method_options.acss_debit.mandate_options.custom_mandate_url": {
			Type:        "string",
			Description: "A URL for custom mandate text to render during confirmation step.\nThe URL will be rendered with additional GET parameters `payment_intent` and `payment_intent_client_secret` when confirming a Payment Intent,\nor `setup_intent` and `setup_intent_client_secret` when confirming a Setup Intent.",
		},
		"payment_method_options.card.mandate_options.amount": {
			Type:        "integer",
			Description: "Amount to be charged for future payments.",
			Required:    true,
		},
	},
}

var V1CustomerBalanceTransactionsRetrieve = resource.OperationSpec{
	Name:    "retrieve",
	Path:    "/v1/customers/{customer}/balance_transactions/{transaction}",
	Method:  "GET",
	Summary: "Retrieve a customer balance transaction",
}

var V1CustomerBalanceTransactionsCreate = resource.OperationSpec{
	Name:    "create",
	Path:    "/v1/customers/{customer}/balance_transactions",
	Method:  "POST",
	Summary: "Create a customer balance transaction",
	Params: map[string]*resource.ParamSpec{
		"description": {
			Type:        "string",
			Description: "An arbitrary string attached to the object. Often useful for displaying to users.",
		},
		"amount": {
			Type:        "integer",
			Description: "The integer amount in **cents (or local equivalent)** to apply to the customer's credit balance.",
			Required:    true,
		},
		"currency": {
			Type:        "string",
			Description: "Three-letter [ISO currency code](https://www.iso.org/iso-4217-currency-codes.html), in lowercase. Must be a [supported currency](https://stripe.com/docs/currencies). Specifies the [`invoice_credit_balance`](https://docs.stripe.com/api/customers/object#customer_object-invoice_credit_balance) that this transaction will apply to. If the customer's `currency` is not set, it will be updated to this value.",
			Required:    true,
			Format:      "currency",
		},
	},
}

var V1CustomerBalanceTransactionsUpdate = resource.OperationSpec{
	Name:    "update",
	Path:    "/v1/customers/{customer}/balance_transactions/{transaction}",
	Method:  "POST",
	Summary: "Update a customer credit balance transaction",
	Params: map[string]*resource.ParamSpec{
		"description": {
			Type:        "string",
			Description: "An arbitrary string attached to the object. Often useful for displaying to users.",
		},
	},
}

var V1CustomerBalanceTransactionsList = resource.OperationSpec{
	Name:    "list",
	Path:    "/v1/customers/{customer}/balance_transactions",
	Method:  "GET",
	Summary: "List customer balance transactions",
	Params: map[string]*resource.ParamSpec{
		"invoice": {
			Type:        "string",
			Description: "Only return transactions that are related to the specified invoice.",
		},
		"limit": {
			Type:        "integer",
			Description: "A limit on the number of objects to be returned. Limit can range between 1 and 100, and the default is 10.",
		},
		"starting_after": {
			Type:        "string",
			Description: "A cursor for use in pagination. `starting_after` is an object ID that defines your place in the list. For instance, if you make a list request and receive 100 objects, ending with `obj_foo`, your subsequent call can include `starting_after=obj_foo` in order to fetch the next page of the list.",
		},
		"created": {
			Type:        "integer",
			Description: "Only return customer balance transactions that were created during the given date interval.",
		},
		"ending_before": {
			Type:        "string",
			Description: "A cursor for use in pagination. `ending_before` is an object ID that defines your place in the list. For instance, if you make a list request and receive 100 objects, starting with `obj_bar`, your subsequent call can include `ending_before=obj_bar` in order to fetch the previous page of the list.",
		},
	},
}

var V1CustomerSessionsCreate = resource.OperationSpec{
	Name:    "create",
	Path:    "/v1/customer_sessions",
	Method:  "POST",
	Summary: "Create a Customer Session",
	Params: map[string]*resource.ParamSpec{
		"components.buy_button.enabled": {
			Type:        "boolean",
			Description: "Whether the buy button is enabled.",
			Required:    true,
		},
		"components.customer_sheet.features.payment_method_remove": {
			Type:        "string",
			Description: "Controls whether the customer sheet displays the option to remove a saved payment method.\"\n\nAllowing buyers to remove their saved payment methods impacts subscriptions that depend on that payment method. Removing the payment method detaches the [`customer` object](https://docs.stripe.com/api/payment_methods/object#payment_method_object-customer) from that [PaymentMethod](https://docs.stripe.com/api/payment_methods).",
			Enum: []resource.EnumSpec{
				{Value: "disabled"},
				{Value: "enabled"},
			},
		},
		"components.mobile_payment_element.features.payment_method_save": {
			Type:        "string",
			Description: "Controls whether the mobile payment element displays a checkbox offering to save a new payment method.\n\nIf a customer checks the box, the [`allow_redisplay`](https://docs.stripe.com/api/payment_methods/object#payment_method_object-allow_redisplay) value on the PaymentMethod is set to `'always'` at confirmation time. For PaymentIntents, the [`setup_future_usage`](https://docs.stripe.com/api/payment_intents/object#payment_intent_object-setup_future_usage) value is also set to the value defined in `payment_method_save_usage`.",
			Enum: []resource.EnumSpec{
				{Value: "disabled"},
				{Value: "enabled"},
			},
		},
		"components.mobile_payment_element.features.payment_method_save_allow_redisplay_override": {
			Type:        "string",
			Description: "Allows overriding the value of allow_override when saving a new payment method when payment_method_save is set to disabled. Use values: \"always\", \"limited\", or \"unspecified\".\n\nIf not specified, defaults to `nil` (no override value).",
			Enum: []resource.EnumSpec{
				{Value: "always"},
				{Value: "limited"},
				{Value: "unspecified"},
			},
		},
		"components.mobile_payment_element.features.payment_method_remove": {
			Type:        "string",
			Description: "Controls whether the mobile payment element displays the option to remove a saved payment method.\"\n\nAllowing buyers to remove their saved payment methods impacts subscriptions that depend on that payment method. Removing the payment method detaches the [`customer` object](https://docs.stripe.com/api/payment_methods/object#payment_method_object-customer) from that [PaymentMethod](https://docs.stripe.com/api/payment_methods).",
			Enum: []resource.EnumSpec{
				{Value: "disabled"},
				{Value: "enabled"},
			},
		},
		"components.payment_element.features.payment_method_remove": {
			Type:        "string",
			Description: "Controls whether the Payment Element displays the option to remove a saved payment method. This parameter defaults to `disabled`.\n\nAllowing buyers to remove their saved payment methods impacts subscriptions that depend on that payment method. Removing the payment method detaches the [`customer` object](https://docs.stripe.com/api/payment_methods/object#payment_method_object-customer) from that [PaymentMethod](https://docs.stripe.com/api/payment_methods).",
			Enum: []resource.EnumSpec{
				{Value: "disabled"},
				{Value: "enabled"},
			},
		},
		"components.payment_element.features.payment_method_redisplay_limit": {
			Type:        "integer",
			Description: "Determines the max number of saved payment methods for the Payment Element to display. This parameter defaults to `3`. The maximum redisplay limit is `10`.",
		},
		"components.payment_element.features.payment_method_save": {
			Type:        "string",
			Description: "Controls whether the Payment Element displays a checkbox offering to save a new payment method. This parameter defaults to `disabled`.\n\nIf a customer checks the box, the [`allow_redisplay`](https://docs.stripe.com/api/payment_methods/object#payment_method_object-allow_redisplay) value on the PaymentMethod is set to `'always'` at confirmation time. For PaymentIntents, the [`setup_future_usage`](https://docs.stripe.com/api/payment_intents/object#payment_intent_object-setup_future_usage) value is also set to the value defined in `payment_method_save_usage`.",
			Enum: []resource.EnumSpec{
				{Value: "disabled"},
				{Value: "enabled"},
			},
		},
		"customer": {
			Type:        "string",
			Description: "The ID of an existing customer for which to create the Customer Session.",
		},
		"customer_account": {
			Type:        "string",
			Description: "The ID of an existing Account for which to create the Customer Session.",
		},
		"components.customer_sheet.enabled": {
			Type:        "boolean",
			Description: "Whether the customer sheet is enabled.",
			Required:    true,
		},
		"components.customer_sheet.features.payment_method_allow_redisplay_filters": {
			Type:        "array",
			Description: "A list of [`allow_redisplay`](https://docs.stripe.com/api/payment_methods/object#payment_method_object-allow_redisplay) values that controls which saved payment methods the customer sheet displays by filtering to only show payment methods with an `allow_redisplay` value that is present in this list.\n\nIf not specified, defaults to [\"always\"]. In order to display all saved payment methods, specify [\"always\", \"limited\", \"unspecified\"].",
		},
		"components.mobile_payment_element.enabled": {
			Type:        "boolean",
			Description: "Whether the mobile payment element is enabled.",
			Required:    true,
		},
		"components.mobile_payment_element.features.payment_method_allow_redisplay_filters": {
			Type:        "array",
			Description: "A list of [`allow_redisplay`](https://docs.stripe.com/api/payment_methods/object#payment_method_object-allow_redisplay) values that controls which saved payment methods the mobile payment element displays by filtering to only show payment methods with an `allow_redisplay` value that is present in this list.\n\nIf not specified, defaults to [\"always\"]. In order to display all saved payment methods, specify [\"always\", \"limited\", \"unspecified\"].",
		},
		"components.mobile_payment_element.features.payment_method_redisplay": {
			Type:        "string",
			Description: "Controls whether or not the mobile payment element shows saved payment methods.",
			Enum: []resource.EnumSpec{
				{Value: "disabled"},
				{Value: "enabled"},
			},
		},
		"components.payment_element.features.payment_method_redisplay": {
			Type:        "string",
			Description: "Controls whether or not the Payment Element shows saved payment methods. This parameter defaults to `disabled`.",
			Enum: []resource.EnumSpec{
				{Value: "disabled"},
				{Value: "enabled"},
			},
		},
		"components.pricing_table.enabled": {
			Type:        "boolean",
			Description: "Whether the pricing table is enabled.",
			Required:    true,
		},
		"components.payment_element.features.payment_method_allow_redisplay_filters": {
			Type:        "array",
			Description: "A list of [`allow_redisplay`](https://docs.stripe.com/api/payment_methods/object#payment_method_object-allow_redisplay) values that controls which saved payment methods the Payment Element displays by filtering to only show payment methods with an `allow_redisplay` value that is present in this list.\n\nIf not specified, defaults to [\"always\"]. In order to display all saved payment methods, specify [\"always\", \"limited\", \"unspecified\"].",
		},
		"components.payment_element.features.payment_method_save_usage": {
			Type:        "string",
			Description: "When using PaymentIntents and the customer checks the save checkbox, this field determines the [`setup_future_usage`](https://docs.stripe.com/api/payment_intents/object#payment_intent_object-setup_future_usage) value used to confirm the PaymentIntent.\n\nWhen using SetupIntents, directly configure the [`usage`](https://docs.stripe.com/api/setup_intents/object#setup_intent_object-usage) value on SetupIntent creation.",
			Enum: []resource.EnumSpec{
				{Value: "off_session"},
				{Value: "on_session"},
			},
		},
		"components.payment_element.enabled": {
			Type:        "boolean",
			Description: "Whether the Payment Element is enabled.",
			Required:    true,
		},
	},
}

var V1RefundsList = resource.OperationSpec{
	Name:    "list",
	Path:    "/v1/refunds",
	Method:  "GET",
	Summary: "List all refunds",
	Params: map[string]*resource.ParamSpec{
		"starting_after": {
			Type:        "string",
			Description: "A cursor for use in pagination. `starting_after` is an object ID that defines your place in the list. For instance, if you make a list request and receive 100 objects, ending with `obj_foo`, your subsequent call can include `starting_after=obj_foo` in order to fetch the next page of the list.",
		},
		"charge": {
			Type:        "string",
			Description: "Only return refunds for the charge specified by this charge ID.",
		},
		"created": {
			Type:        "integer",
			Description: "Only return refunds that were created during the given date interval.",
		},
		"ending_before": {
			Type:        "string",
			Description: "A cursor for use in pagination. `ending_before` is an object ID that defines your place in the list. For instance, if you make a list request and receive 100 objects, starting with `obj_bar`, your subsequent call can include `ending_before=obj_bar` in order to fetch the previous page of the list.",
		},
		"limit": {
			Type:        "integer",
			Description: "A limit on the number of objects to be returned. Limit can range between 1 and 100, and the default is 10.",
		},
		"payment_intent": {
			Type:        "string",
			Description: "Only return refunds for the PaymentIntent specified by this ID.",
		},
	},
}

var V1RefundsRetrieve = resource.OperationSpec{
	Name:    "retrieve",
	Path:    "/v1/refunds/{refund}",
	Method:  "GET",
	Summary: "Retrieve a refund",
}

var V1RefundsCreate = resource.OperationSpec{
	Name:    "create",
	Path:    "/v1/refunds",
	Method:  "POST",
	Summary: "Create customer balance refund",
	Params: map[string]*resource.ParamSpec{
		"refund_application_fee": {
			Type:        "boolean",
			Description: "Boolean indicating whether the application fee should be refunded when refunding this charge. If a full charge refund is given, the full application fee will be refunded. Otherwise, the application fee will be refunded in an amount proportional to the amount of the charge refunded. An application fee can be refunded only by the application that created the charge.",
		},
		"origin": {
			Type:        "string",
			Description: "Origin of the refund",
			Enum: []resource.EnumSpec{
				{Value: "customer_balance"},
			},
		},
		"reverse_transfer": {
			Type:        "boolean",
			Description: "Boolean indicating whether the transfer should be reversed when refunding this charge. The transfer will be reversed proportionally to the amount being refunded (either the entire or partial amount).<br><br>A transfer can be reversed only by the application that created the charge.",
		},
		"currency": {
			Type:        "string",
			Description: "Three-letter [ISO currency code](https://www.iso.org/iso-4217-currency-codes.html), in lowercase. Must be a [supported currency](https://stripe.com/docs/currencies).",
			Format:      "currency",
		},
		"charge": {
			Type:        "string",
			Description: "The identifier of the charge to refund.",
		},
		"customer": {
			Type:        "string",
			Description: "Customer whose customer balance to refund from.",
		},
		"payment_intent": {
			Type:        "string",
			Description: "The identifier of the PaymentIntent to refund.",
		},
		"amount": {
			Type: "integer",
		},
		"instructions_email": {
			Type:        "string",
			Description: "For payment methods without native refund support (e.g., Konbini, PromptPay), use this email from the customer to receive refund instructions.",
		},
		"reason": {
			Type:        "string",
			Description: "String indicating the reason for the refund. If set, possible values are `duplicate`, `fraudulent`, and `requested_by_customer`. If you believe the charge to be fraudulent, specifying `fraudulent` as the reason will add the associated card and email to your [block lists](https://docs.stripe.com/radar/lists), and will also help us improve our fraud detection algorithms.",
			Enum: []resource.EnumSpec{
				{Value: "duplicate"},
				{Value: "fraudulent"},
				{Value: "requested_by_customer"},
			},
		},
	},
}

var V1RefundsUpdate = resource.OperationSpec{
	Name:    "update",
	Path:    "/v1/refunds/{refund}",
	Method:  "POST",
	Summary: "Update a refund",
}

var V1RefundsCancel = resource.OperationSpec{
	Name:    "cancel",
	Path:    "/v1/refunds/{refund}/cancel",
	Method:  "POST",
	Summary: "Cancel a refund",
}

var V1RefundsTestHelpersExpire = resource.OperationSpec{
	Name:    "expire",
	Path:    "/v1/test_helpers/refunds/{refund}/expire",
	Method:  "POST",
	Summary: "Expire a pending refund.",
}

var V1ConfirmationTokensRetrieve = resource.OperationSpec{
	Name:    "retrieve",
	Path:    "/v1/confirmation_tokens/{confirmation_token}",
	Method:  "GET",
	Summary: "Retrieve a ConfirmationToken",
}

var V1ConfirmationTokensTestHelpersCreate = resource.OperationSpec{
	Name:    "create",
	Path:    "/v1/test_helpers/confirmation_tokens",
	Method:  "POST",
	Summary: "Create a test Confirmation Token",
	Params: map[string]*resource.ParamSpec{
		"shipping.address.state": {
			Type:        "string",
			Description: "State, county, province, or region ([ISO 3166-2](https://en.wikipedia.org/wiki/ISO_3166-2)).",
		},
		"payment_method_data.acss_debit.account_number": {
			Type:        "string",
			Description: "Customer's bank account number.",
			Required:    true,
		},
		"payment_method_data.allow_redisplay": {
			Type:        "string",
			Description: "This field indicates whether this payment method can be shown again to its customer in a checkout flow. Stripe products such as Checkout and Elements use this field to determine whether a payment method can be shown as a saved payment method in a checkout flow. The field defaults to `unspecified`.",
			Enum: []resource.EnumSpec{
				{Value: "always"},
				{Value: "limited"},
				{Value: "unspecified"},
			},
		},
		"payment_method_data.nz_bank_account.bank_code": {
			Type:        "string",
			Description: "The numeric code for the bank account's bank.",
			Required:    true,
		},
		"payment_method_data.nz_bank_account.branch_code": {
			Type:        "string",
			Description: "The numeric code for the bank account's bank branch.",
			Required:    true,
		},
		"payment_method_data.radar_options.session": {
			Type:        "string",
			Description: "A [Radar Session](https://docs.stripe.com/radar/radar-session) is a snapshot of the browser metadata and device details that help Radar make more accurate predictions on your payments.",
		},
		"payment_method_data.us_bank_account.account_number": {
			Type:        "string",
			Description: "Account number of the bank account.",
		},
		"payment_method_data.us_bank_account.financial_connections_account": {
			Type:        "string",
			Description: "The ID of a Financial Connections Account to use as a payment method.",
		},
		"payment_method_data.klarna.dob.day": {
			Type:        "integer",
			Description: "The day of birth, between 1 and 31.",
			Required:    true,
		},
		"payment_method_data.bacs_debit.account_number": {
			Type:        "string",
			Description: "Account number of the bank account that the funds will be debited from.",
		},
		"payment_method_data.bacs_debit.sort_code": {
			Type:        "string",
			Description: "Sort code of the bank account. (e.g., `10-20-30`)",
		},
		"payment_method_data.eps.bank": {
			Type:        "string",
			Description: "The customer's bank.",
			Enum: []resource.EnumSpec{
				{Value: "arzte_und_apotheker_bank"},
				{Value: "austrian_anadi_bank_ag"},
				{Value: "bank_austria"},
				{Value: "bankhaus_carl_spangler"},
				{Value: "bankhaus_schelhammer_und_schattera_ag"},
				{Value: "bawag_psk_ag"},
				{Value: "bks_bank_ag"},
				{Value: "brull_kallmus_bank_ag"},
				{Value: "btv_vier_lander_bank"},
				{Value: "capital_bank_grawe_gruppe_ag"},
				{Value: "deutsche_bank_ag"},
				{Value: "dolomitenbank"},
				{Value: "easybank_ag"},
				{Value: "erste_bank_und_sparkassen"},
				{Value: "hypo_alpeadriabank_international_ag"},
				{Value: "hypo_bank_burgenland_aktiengesellschaft"},
				{Value: "hypo_noe_lb_fur_niederosterreich_u_wien"},
				{Value: "hypo_oberosterreich_salzburg_steiermark"},
				{Value: "hypo_tirol_bank_ag"},
				{Value: "hypo_vorarlberg_bank_ag"},
				{Value: "marchfelder_bank"},
				{Value: "oberbank_ag"},
				{Value: "raiffeisen_bankengruppe_osterreich"},
				{Value: "schoellerbank_ag"},
				{Value: "sparda_bank_wien"},
				{Value: "volksbank_gruppe"},
				{Value: "volkskreditbank_ag"},
				{Value: "vr_bank_braunau"},
			},
		},
		"payment_method_data.klarna.dob.year": {
			Type:        "integer",
			Description: "The four-digit year of birth.",
			Required:    true,
		},
		"payment_method_options.card.installments.plan.count": {
			Type:        "integer",
			Description: "For `fixed_count` installment plans, this is required. It represents the number of installment payments your customer will make to their credit card.",
		},
		"setup_future_usage": {
			Type:        "string",
			Description: "Indicates that you intend to make future payments with this ConfirmationToken's payment method.\n\nThe presence of this property will [attach the payment method](https://docs.stripe.com/payments/save-during-payment) to the PaymentIntent's Customer, if present, after the PaymentIntent is confirmed and any required actions from the user are complete.",
			Enum: []resource.EnumSpec{
				{Value: "off_session"},
				{Value: "on_session"},
			},
		},
		"shipping.address.postal_code": {
			Type:        "string",
			Description: "ZIP or postal code.",
		},
		"payment_method_data.boleto.tax_id": {
			Type:        "string",
			Description: "The tax ID of the customer (CPF for individual consumers or CNPJ for businesses consumers)",
			Required:    true,
		},
		"payment_method_data.nz_bank_account.suffix": {
			Type:        "string",
			Description: "The suffix of the bank account number.",
			Required:    true,
		},
		"payment_method_data.p24.bank": {
			Type:        "string",
			Description: "The customer's bank.",
			Enum: []resource.EnumSpec{
				{Value: "alior_bank"},
				{Value: "bank_millennium"},
				{Value: "bank_nowy_bfg_sa"},
				{Value: "bank_pekao_sa"},
				{Value: "banki_spbdzielcze"},
				{Value: "blik"},
				{Value: "bnp_paribas"},
				{Value: "boz"},
				{Value: "citi_handlowy"},
				{Value: "credit_agricole"},
				{Value: "envelobank"},
				{Value: "etransfer_pocztowy24"},
				{Value: "getin_bank"},
				{Value: "ideabank"},
				{Value: "ing"},
				{Value: "inteligo"},
				{Value: "mbank_mtransfer"},
				{Value: "nest_przelew"},
				{Value: "noble_pay"},
				{Value: "pbac_z_ipko"},
				{Value: "plus_bank"},
				{Value: "santander_przelew24"},
				{Value: "tmobile_usbugi_bankowe"},
				{Value: "toyota_bank"},
				{Value: "velobank"},
				{Value: "volkswagen_bank"},
			},
		},
		"payment_method_data.payto.pay_id": {
			Type:        "string",
			Description: "The PayID alias for the bank account.",
		},
		"payment_method_data.au_becs_debit.bsb_number": {
			Type:        "string",
			Description: "Bank-State-Branch number of the bank account.",
			Required:    true,
		},
		"return_url": {
			Type:        "string",
			Description: "Return URL used to confirm the Intent.",
		},
		"shipping.address.city": {
			Type:        "string",
			Description: "City, district, suburb, town, or village.",
		},
		"shipping.address.country": {
			Type:        "string",
			Description: "Two-letter country code ([ISO 3166-1 alpha-2](https://en.wikipedia.org/wiki/ISO_3166-1_alpha-2)).",
		},
		"payment_method_data.sepa_debit.iban": {
			Type:        "string",
			Description: "IBAN of the bank account.",
			Required:    true,
		},
		"payment_method_data.sofort.country": {
			Type:        "string",
			Description: "Two-letter ISO code representing the country the bank account is located in.",
			Required:    true,
			Enum: []resource.EnumSpec{
				{Value: "AT"},
				{Value: "BE"},
				{Value: "DE"},
				{Value: "ES"},
				{Value: "IT"},
				{Value: "NL"},
			},
		},
		"payment_method_data.klarna.dob.month": {
			Type:        "integer",
			Description: "The month of birth, between 1 and 12.",
			Required:    true,
		},
		"payment_method_data.au_becs_debit.account_number": {
			Type:        "string",
			Description: "The account number for the bank account.",
			Required:    true,
		},
		"payment_method_options.card.installments.plan.type": {
			Type:        "string",
			Description: "Type of installment plan, one of `fixed_count`, `bonus`, or `revolving`.",
			Required:    true,
			Enum: []resource.EnumSpec{
				{Value: "bonus"},
				{Value: "fixed_count"},
				{Value: "revolving"},
			},
		},
		"shipping.address.line1": {
			Type:        "string",
			Description: "Address line 1, such as the street, PO Box, or company name.",
		},
		"shipping.address.line2": {
			Type:        "string",
			Description: "Address line 2, such as the apartment, suite, unit, or building.",
		},
		"shipping.phone": {
			Type:        "string",
			Description: "Recipient phone (including extension)",
		},
		"payment_method_data.acss_debit.transit_number": {
			Type:        "string",
			Description: "Transit number of the customer's bank.",
			Required:    true,
		},
		"payment_method_data.billing_details.phone": {
			Type:        "string",
			Description: "Billing phone number (including extension).",
		},
		"shipping.name": {
			Type:        "string",
			Description: "Recipient name.",
			Required:    true,
		},
		"payment_method": {
			Type:        "string",
			Description: "ID of an existing PaymentMethod.",
		},
		"payment_method_data.billing_details.name": {
			Type:        "string",
			Description: "Full name.",
		},
		"payment_method_data.billing_details.tax_id": {
			Type:        "string",
			Description: "Taxpayer identification number. Used only for transactions between LATAM buyers and non-LATAM sellers.",
		},
		"payment_method_data.ideal.bank": {
			Type:        "string",
			Description: "The customer's bank. Only use this parameter for existing customers. Don't use it for new customers.",
			Enum: []resource.EnumSpec{
				{Value: "abn_amro"},
				{Value: "adyen"},
				{Value: "asn_bank"},
				{Value: "bunq"},
				{Value: "buut"},
				{Value: "finom"},
				{Value: "handelsbanken"},
				{Value: "ing"},
				{Value: "knab"},
				{Value: "mollie"},
				{Value: "moneyou"},
				{Value: "n26"},
				{Value: "nn"},
				{Value: "rabobank"},
				{Value: "regiobank"},
				{Value: "revolut"},
				{Value: "sns_bank"},
				{Value: "triodos_bank"},
				{Value: "van_lanschot"},
				{Value: "yoursafe"},
			},
		},
		"payment_method_data.fpx.bank": {
			Type:        "string",
			Description: "The customer's bank.",
			Required:    true,
			Enum: []resource.EnumSpec{
				{Value: "affin_bank"},
				{Value: "agrobank"},
				{Value: "alliance_bank"},
				{Value: "ambank"},
				{Value: "bank_islam"},
				{Value: "bank_muamalat"},
				{Value: "bank_of_china"},
				{Value: "bank_rakyat"},
				{Value: "bsn"},
				{Value: "cimb"},
				{Value: "deutsche_bank"},
				{Value: "hong_leong_bank"},
				{Value: "hsbc"},
				{Value: "kfh"},
				{Value: "maybank2e"},
				{Value: "maybank2u"},
				{Value: "ocbc"},
				{Value: "pb_enterprise"},
				{Value: "public_bank"},
				{Value: "rhb"},
				{Value: "standard_chartered"},
				{Value: "uob"},
			},
		},
		"payment_method_data.nz_bank_account.account_holder_name": {
			Type:        "string",
			Description: "The name on the bank account. Only required if the account holder name is different from the name of the authorized signatory collected in the PaymentMethod’s billing details.",
		},
		"payment_method_data.nz_bank_account.account_number": {
			Type:        "string",
			Description: "The account number for the bank account.",
			Required:    true,
		},
		"payment_method_data.payto.bsb_number": {
			Type:        "string",
			Description: "Bank-State-Branch number of the bank account.",
		},
		"payment_method_data.us_bank_account.routing_number": {
			Type:        "string",
			Description: "Routing number of the bank account.",
		},
		"payment_method_data.naver_pay.funding": {
			Type:        "string",
			Description: "Whether to use Naver Pay points or a card to fund this transaction. If not provided, this defaults to `card`.",
			Enum: []resource.EnumSpec{
				{Value: "card"},
				{Value: "points"},
			},
		},
		"payment_method_data.payto.account_number": {
			Type:        "string",
			Description: "The account number for the bank account.",
		},
		"payment_method_data.us_bank_account.account_holder_type": {
			Type:        "string",
			Description: "Account holder type: individual or company.",
			Enum: []resource.EnumSpec{
				{Value: "company"},
				{Value: "individual"},
			},
		},
		"payment_method_data.us_bank_account.account_type": {
			Type:        "string",
			Description: "Account type: checkings or savings. Defaults to checking if omitted.",
			Enum: []resource.EnumSpec{
				{Value: "checking"},
				{Value: "savings"},
			},
		},
		"payment_method_data.acss_debit.institution_number": {
			Type:        "string",
			Description: "Institution number of the customer's bank.",
			Required:    true,
		},
		"payment_method_data.billing_details.email": {
			Type:        "string",
			Description: "Email address.",
		},
		"payment_method_data.fpx.account_holder_type": {
			Type:        "string",
			Description: "Account holder type for FPX transaction",
			Enum: []resource.EnumSpec{
				{Value: "company"},
				{Value: "individual"},
			},
		},
		"payment_method_data.nz_bank_account.reference": {
			Type: "string",
		},
		"payment_method_data.type": {
			Type:        "string",
			Description: "The type of the PaymentMethod. An additional hash is included on the PaymentMethod with a name matching this value. It contains additional information specific to the PaymentMethod type.",
			Required:    true,
			Enum: []resource.EnumSpec{
				{Value: "acss_debit"},
				{Value: "affirm"},
				{Value: "afterpay_clearpay"},
				{Value: "alipay"},
				{Value: "alma"},
				{Value: "amazon_pay"},
				{Value: "au_becs_debit"},
				{Value: "bacs_debit"},
				{Value: "bancontact"},
				{Value: "billie"},
				{Value: "blik"},
				{Value: "boleto"},
				{Value: "cashapp"},
				{Value: "crypto"},
				{Value: "customer_balance"},
				{Value: "eps"},
				{Value: "fpx"},
				{Value: "giropay"},
				{Value: "grabpay"},
				{Value: "ideal"},
				{Value: "kakao_pay"},
				{Value: "klarna"},
				{Value: "konbini"},
				{Value: "kr_card"},
				{Value: "link"},
				{Value: "mb_way"},
				{Value: "mobilepay"},
				{Value: "multibanco"},
				{Value: "naver_pay"},
				{Value: "nz_bank_account"},
				{Value: "oxxo"},
				{Value: "p24"},
				{Value: "pay_by_bank"},
				{Value: "payco"},
				{Value: "paynow"},
				{Value: "paypal"},
				{Value: "payto"},
				{Value: "pix"},
				{Value: "promptpay"},
				{Value: "revolut_pay"},
				{Value: "samsung_pay"},
				{Value: "satispay"},
				{Value: "sepa_debit"},
				{Value: "sofort"},
				{Value: "swish"},
				{Value: "twint"},
				{Value: "us_bank_account"},
				{Value: "wechat_pay"},
				{Value: "zip"},
			},
		},
		"payment_method_options.card.installments.plan.interval": {
			Type:        "string",
			Description: "For `fixed_count` installment plans, this is required. It represents the interval between installment payments your customer will make to their credit card.\nOne of `month`.",
			Enum: []resource.EnumSpec{
				{Value: "month"},
			},
		},
	},
}

var V1InvoicePaymentsList = resource.OperationSpec{
	Name:    "list",
	Path:    "/v1/invoice_payments",
	Method:  "GET",
	Summary: "List all payments for an invoice",
	Params: map[string]*resource.ParamSpec{
		"invoice": {
			Type:        "string",
			Description: "The identifier of the invoice whose payments to return.",
		},
		"limit": {
			Type:        "integer",
			Description: "A limit on the number of objects to be returned. Limit can range between 1 and 100, and the default is 10.",
		},
		"starting_after": {
			Type:        "string",
			Description: "A cursor for use in pagination. `starting_after` is an object ID that defines your place in the list. For instance, if you make a list request and receive 100 objects, ending with `obj_foo`, your subsequent call can include `starting_after=obj_foo` in order to fetch the next page of the list.",
		},
		"status": {
			Type:        "string",
			Description: "The status of the invoice payments to return.",
			Enum: []resource.EnumSpec{
				{Value: "canceled"},
				{Value: "open"},
				{Value: "paid"},
			},
		},
		"created": {
			Type:        "integer",
			Description: "Only return invoice payments that were created during the given date interval.",
		},
		"ending_before": {
			Type:        "string",
			Description: "A cursor for use in pagination. `ending_before` is an object ID that defines your place in the list. For instance, if you make a list request and receive 100 objects, starting with `obj_bar`, your subsequent call can include `ending_before=obj_bar` in order to fetch the previous page of the list.",
		},
	},
}

var V1InvoicePaymentsRetrieve = resource.OperationSpec{
	Name:    "retrieve",
	Path:    "/v1/invoice_payments/{invoice_payment}",
	Method:  "GET",
	Summary: "Retrieve an InvoicePayment",
}

var V1ScheduledQueryRunsList = resource.OperationSpec{
	Name:    "list",
	Path:    "/v1/sigma/scheduled_query_runs",
	Method:  "GET",
	Summary: "List all scheduled query runs",
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

var V1ScheduledQueryRunsRetrieve = resource.OperationSpec{
	Name:    "retrieve",
	Path:    "/v1/sigma/scheduled_query_runs/{scheduled_query_run}",
	Method:  "GET",
	Summary: "Retrieve a scheduled query run",
}

var V1PaymentAttemptRecordsList = resource.OperationSpec{
	Name:    "list",
	Path:    "/v1/payment_attempt_records",
	Method:  "GET",
	Summary: "List Payment Attempt Records",
	Params: map[string]*resource.ParamSpec{
		"limit": {
			Type:        "integer",
			Description: "A limit on the number of objects to be returned. Limit can range between 1 and 100, and the default is 10.",
		},
		"payment_record": {
			Type:        "string",
			Description: "The ID of the Payment Record.",
			Required:    true,
		},
		"starting_after": {
			Type:        "string",
			Description: "A cursor for use in pagination. `starting_after` is an object ID that defines your place in the list. For instance, if you make a list request and receive 100 objects, ending with `obj_foo`, your subsequent call can include `starting_after=obj_foo` in order to fetch the next page of the list.",
		},
	},
}

var V1PaymentAttemptRecordsRetrieve = resource.OperationSpec{
	Name:    "retrieve",
	Path:    "/v1/payment_attempt_records/{id}",
	Method:  "GET",
	Summary: "Retrieve a Payment Attempt Record",
}

var V1EphemeralKeysDelete = resource.OperationSpec{
	Name:    "delete",
	Path:    "/v1/ephemeral_keys/{key}",
	Method:  "DELETE",
	Summary: "Immediately invalidate an ephemeral key",
}

var V1EphemeralKeysCreate = resource.OperationSpec{
	Name:    "create",
	Path:    "/v1/ephemeral_keys",
	Method:  "POST",
	Summary: "Create an ephemeral key",
	Params: map[string]*resource.ParamSpec{
		"verification_session": {
			Type:        "string",
			Description: "The ID of the Identity VerificationSession you'd like to access using the resulting ephemeral key",
		},
		"customer": {
			Type:        "string",
			Description: "The ID of the Customer you'd like to modify using the resulting ephemeral key.",
		},
		"issuing_card": {
			Type:        "string",
			Description: "The ID of the Issuing Card you'd like to access using the resulting ephemeral key.",
		},
		"nonce": {
			Type:        "string",
			Description: "A single-use token, created by Stripe.js, used for creating ephemeral keys for Issuing Cards without exchanging sensitive information.",
		},
	},
}

var V1ChargesCreate = resource.OperationSpec{
	Name:   "create",
	Path:   "/v1/charges",
	Method: "POST",
	Params: map[string]*resource.ParamSpec{
		"description": {
			Type:        "string",
			Description: "An arbitrary string which you can attach to a `Charge` object. It is displayed when in the web interface alongside the charge. Note that if you use Stripe to send automatic email receipts to your customers, your receipt emails will include the `description` of the charge(s) that they are describing.",
		},
		"shipping.address.line1": {
			Type:        "string",
			Description: "Address line 1, such as the street, PO Box, or company name.",
		},
		"shipping.name": {
			Type:        "string",
			Description: "Recipient name.",
			Required:    true,
		},
		"statement_descriptor": {
			Type:        "string",
			Description: "For a non-card charge, text that appears on the customer's statement as the statement descriptor. This value overrides the account's default statement descriptor. For information about requirements, including the 22-character limit, see [the Statement Descriptor docs](https://docs.stripe.com/get-started/account/statement-descriptors).\n\nFor a card charge, this value is ignored unless you don't specify a `statement_descriptor_suffix`, in which case this value is used as the suffix.",
		},
		"source": {
			Type:        "string",
			Description: "A payment source to be charged. This can be the ID of a [card](https://docs.stripe.com/api#cards) (i.e., credit or debit card), a [bank account](https://docs.stripe.com/api#bank_accounts), a [source](https://docs.stripe.com/api#sources), a [token](https://docs.stripe.com/api#tokens), or a [connected account](https://docs.stripe.com/connect/account-debits#charging-a-connected-account). For certain sources---namely, [cards](https://docs.stripe.com/api#cards), [bank accounts](https://docs.stripe.com/api#bank_accounts), and attached [sources](https://docs.stripe.com/api#sources)---you must also pass the ID of the associated customer.",
		},
		"destination.account": {
			Type:        "string",
			Description: "ID of an existing, connected Stripe account.",
			Required:    true,
		},
		"destination.amount": {
			Type:        "integer",
			Description: "The amount to transfer to the destination account without creating an `Application Fee` object. Cannot be combined with the `application_fee` parameter. Must be less than or equal to the charge amount.",
		},
		"shipping.address.country": {
			Type:        "string",
			Description: "Two-letter country code ([ISO 3166-1 alpha-2](https://en.wikipedia.org/wiki/ISO_3166-1_alpha-2)).",
		},
		"shipping.address.city": {
			Type:        "string",
			Description: "City, district, suburb, town, or village.",
		},
		"shipping.carrier": {
			Type:        "string",
			Description: "The delivery service that shipped a physical product, such as Fedex, UPS, USPS, etc.",
		},
		"shipping.phone": {
			Type:        "string",
			Description: "Recipient phone (including extension).",
		},
		"statement_descriptor_suffix": {
			Type:        "string",
			Description: "Provides information about a card charge. Concatenated to the account's [statement descriptor prefix](https://docs.stripe.com/get-started/account/statement-descriptors#static) to form the complete statement descriptor that appears on the customer's statement. If the account has no prefix value, the suffix is concatenated to the account's statement descriptor.",
		},
		"amount": {
			Type:        "integer",
			Description: "Amount intended to be collected by this payment. A positive integer representing how much to charge in the [smallest currency unit](https://docs.stripe.com/currencies#zero-decimal) (e.g., 100 cents to charge $1.00 or 100 to charge ¥100, a zero-decimal currency). The minimum amount is $0.50 US or [equivalent in charge currency](https://docs.stripe.com/currencies#minimum-and-maximum-charge-amounts). The amount value supports up to eight digits (e.g., a value of 99999999 for a USD charge of $999,999.99).",
		},
		"customer": {
			Type:        "string",
			Description: "The ID of an existing customer that will be charged in this request.",
		},
		"transfer_data.amount": {
			Type:        "integer",
			Description: "The amount transferred to the destination account, if specified. By default, the entire charge amount is transferred to the destination account.",
		},
		"transfer_data.destination": {
			Type:        "string",
			Description: "ID of an existing, connected Stripe account.",
			Required:    true,
		},
		"currency": {
			Type:        "string",
			Description: "Three-letter [ISO currency code](https://www.iso.org/iso-4217-currency-codes.html), in lowercase. Must be a [supported currency](https://stripe.com/docs/currencies).",
			Format:      "currency",
		},
		"on_behalf_of": {
			Type:        "string",
			Description: "The Stripe account ID for which these funds are intended. You can specify the business of record as the connected account using the `on_behalf_of` attribute on the charge. For details, see [Creating Separate Charges and Transfers](https://docs.stripe.com/connect/separate-charges-and-transfers#settlement-merchant).",
		},
		"transfer_group": {
			Type:        "string",
			Description: "A string that identifies this transaction as part of a group. For details, see [Grouping transactions](https://docs.stripe.com/connect/separate-charges-and-transfers#transfer-options).",
		},
		"radar_options.session": {
			Type:        "string",
			Description: "A [Radar Session](https://docs.stripe.com/radar/radar-session) is a snapshot of the browser metadata and device details that help Radar make more accurate predictions on your payments.",
		},
		"shipping.address.line2": {
			Type:        "string",
			Description: "Address line 2, such as the apartment, suite, unit, or building.",
		},
		"shipping.address.postal_code": {
			Type:        "string",
			Description: "ZIP or postal code.",
		},
		"shipping.address.state": {
			Type:        "string",
			Description: "State, county, province, or region ([ISO 3166-2](https://en.wikipedia.org/wiki/ISO_3166-2)).",
		},
		"shipping.tracking_number": {
			Type:        "string",
			Description: "The tracking number for a physical product, obtained from the delivery service. If multiple tracking numbers were generated for this purchase, please separate them with commas.",
		},
		"application_fee": {
			Type: "integer",
		},
		"application_fee_amount": {
			Type:        "integer",
			Description: "A fee in cents (or local equivalent) that will be applied to the charge and transferred to the application owner's Stripe account. The request must be made with an OAuth key or the `Stripe-Account` header in order to take an application fee. For more information, see the application fees [documentation](https://docs.stripe.com/connect/direct-charges#collect-fees).",
		},
		"capture": {
			Type:        "boolean",
			Description: "Whether to immediately capture the charge. Defaults to `true`. When `false`, the charge issues an authorization (or pre-authorization), and will need to be [captured](https://api.stripe.com#capture_charge) later. Uncaptured charges expire after a set number of days (7 by default). For more information, see the [authorizing charges and settling later](https://docs.stripe.com/charges/placing-a-hold) documentation.",
		},
		"receipt_email": {
			Type:        "string",
			Description: "The email address to which this charge's [receipt](https://docs.stripe.com/dashboard/receipts) will be sent. The receipt will not be sent until the charge is paid, and no receipts will be sent for test mode charges. If this charge is for a [Customer](https://docs.stripe.com/api/customers/object), the email address specified here will override the customer's email address. If `receipt_email` is specified for a charge in live mode, a receipt will be sent regardless of your [email settings](https://dashboard.stripe.com/account/emails).",
		},
	},
}

var V1ChargesUpdate = resource.OperationSpec{
	Name:    "update",
	Path:    "/v1/charges/{charge}",
	Method:  "POST",
	Summary: "Update a charge",
	Params: map[string]*resource.ParamSpec{
		"shipping.tracking_number": {
			Type:        "string",
			Description: "The tracking number for a physical product, obtained from the delivery service. If multiple tracking numbers were generated for this purchase, please separate them with commas.",
		},
		"fraud_details.user_report": {
			Type:        "string",
			Description: "Either `safe` or `fraudulent`.",
			Required:    true,
			Enum: []resource.EnumSpec{
				{Value: "fraudulent"},
				{Value: "safe"},
			},
		},
		"shipping.address.country": {
			Type:        "string",
			Description: "Two-letter country code ([ISO 3166-1 alpha-2](https://en.wikipedia.org/wiki/ISO_3166-1_alpha-2)).",
		},
		"shipping.phone": {
			Type:        "string",
			Description: "Recipient phone (including extension).",
		},
		"receipt_email": {
			Type:        "string",
			Description: "This is the email address that the receipt for this charge will be sent to. If this field is updated, then a new email receipt will be sent to the updated address.",
		},
		"shipping.address.line1": {
			Type:        "string",
			Description: "Address line 1, such as the street, PO Box, or company name.",
		},
		"shipping.address.postal_code": {
			Type:        "string",
			Description: "ZIP or postal code.",
		},
		"shipping.carrier": {
			Type:        "string",
			Description: "The delivery service that shipped a physical product, such as Fedex, UPS, USPS, etc.",
		},
		"description": {
			Type:        "string",
			Description: "An arbitrary string which you can attach to a charge object. It is displayed when in the web interface alongside the charge. Note that if you use Stripe to send automatic email receipts to your customers, your receipt emails will include the `description` of the charge(s) that they are describing.",
		},
		"shipping.address.state": {
			Type:        "string",
			Description: "State, county, province, or region ([ISO 3166-2](https://en.wikipedia.org/wiki/ISO_3166-2)).",
		},
		"shipping.address.city": {
			Type:        "string",
			Description: "City, district, suburb, town, or village.",
		},
		"shipping.address.line2": {
			Type:        "string",
			Description: "Address line 2, such as the apartment, suite, unit, or building.",
		},
		"shipping.name": {
			Type:        "string",
			Description: "Recipient name.",
			Required:    true,
		},
		"transfer_group": {
			Type:        "string",
			Description: "A string that identifies this transaction as part of a group. `transfer_group` may only be provided if it has not been set. See the [Connect documentation](https://docs.stripe.com/connect/separate-charges-and-transfers#transfer-options) for details.",
		},
		"customer": {
			Type:        "string",
			Description: "The ID of an existing customer that will be associated with this request. This field may only be updated if there is no existing associated customer with this charge.",
		},
	},
}

var V1ChargesCapture = resource.OperationSpec{
	Name:    "capture",
	Path:    "/v1/charges/{charge}/capture",
	Method:  "POST",
	Summary: "Capture a payment",
	Params: map[string]*resource.ParamSpec{
		"application_fee": {
			Type:        "integer",
			Description: "An application fee to add on to this charge.",
		},
		"application_fee_amount": {
			Type:        "integer",
			Description: "An application fee amount to add on to this charge, which must be less than or equal to the original amount.",
		},
		"statement_descriptor": {
			Type:        "string",
			Description: "For a non-card charge, text that appears on the customer's statement as the statement descriptor. This value overrides the account's default statement descriptor. For information about requirements, including the 22-character limit, see [the Statement Descriptor docs](https://docs.stripe.com/get-started/account/statement-descriptors).\n\nFor a card charge, this value is ignored unless you don't specify a `statement_descriptor_suffix`, in which case this value is used as the suffix.",
		},
		"statement_descriptor_suffix": {
			Type:        "string",
			Description: "Provides information about a card charge. Concatenated to the account's [statement descriptor prefix](https://docs.stripe.com/get-started/account/statement-descriptors#static) to form the complete statement descriptor that appears on the customer's statement. If the account has no prefix value, the suffix is concatenated to the account's statement descriptor.",
		},
		"transfer_data.amount": {
			Type:        "integer",
			Description: "The amount transferred to the destination account, if specified. By default, the entire charge amount is transferred to the destination account.",
		},
		"transfer_group": {
			Type:        "string",
			Description: "A string that identifies this transaction as part of a group. `transfer_group` may only be provided if it has not been set. See the [Connect documentation](https://docs.stripe.com/connect/separate-charges-and-transfers#transfer-options) for details.",
		},
		"amount": {
			Type:        "integer",
			Description: "The amount to capture, which must be less than or equal to the original amount.",
		},
		"receipt_email": {
			Type:        "string",
			Description: "The email address to send this charge's receipt to. This will override the previously-specified email address for this charge, if one was set. Receipts will not be sent in test mode.",
		},
	},
}

var V1ChargesList = resource.OperationSpec{
	Name:    "list",
	Path:    "/v1/charges",
	Method:  "GET",
	Summary: "List all charges",
	Params: map[string]*resource.ParamSpec{
		"created": {
			Type:        "integer",
			Description: "Only return charges that were created during the given date interval.",
		},
		"customer": {
			Type:        "string",
			Description: "Only return charges for the customer specified by this customer ID.",
		},
		"ending_before": {
			Type:        "string",
			Description: "A cursor for use in pagination. `ending_before` is an object ID that defines your place in the list. For instance, if you make a list request and receive 100 objects, starting with `obj_bar`, your subsequent call can include `ending_before=obj_bar` in order to fetch the previous page of the list.",
		},
		"limit": {
			Type:        "integer",
			Description: "A limit on the number of objects to be returned. Limit can range between 1 and 100, and the default is 10.",
		},
		"payment_intent": {
			Type:        "string",
			Description: "Only return charges that were created by the PaymentIntent specified by this PaymentIntent ID.",
		},
		"starting_after": {
			Type:        "string",
			Description: "A cursor for use in pagination. `starting_after` is an object ID that defines your place in the list. For instance, if you make a list request and receive 100 objects, ending with `obj_foo`, your subsequent call can include `starting_after=obj_foo` in order to fetch the next page of the list.",
		},
		"transfer_group": {
			Type:        "string",
			Description: "Only return charges for this transfer group, limited to 100.",
		},
	},
}

var V1ChargesRetrieve = resource.OperationSpec{
	Name:    "retrieve",
	Path:    "/v1/charges/{charge}",
	Method:  "GET",
	Summary: "Retrieve a charge",
}

var V1ChargesSearch = resource.OperationSpec{
	Name:    "search",
	Path:    "/v1/charges/search",
	Method:  "GET",
	Summary: "Search charges",
	Params: map[string]*resource.ParamSpec{
		"limit": {
			Type:        "integer",
			Description: "A limit on the number of objects to be returned. Limit can range between 1 and 100, and the default is 10.",
		},
		"page": {
			Type:        "string",
			Description: "A cursor for pagination across multiple pages of results. Don't include this parameter on the first call. Use the next_page value returned in a previous response to request subsequent results.",
		},
		"query": {
			Type:        "string",
			Description: "The search query string. See [search query language](https://docs.stripe.com/search#search-query-language) and the list of supported [query fields for charges](https://docs.stripe.com/search#query-fields-for-charges).",
			Required:    true,
		},
	},
}

var V1TaxIdsDelete = resource.OperationSpec{
	Name:    "delete",
	Path:    "/v1/customers/{customer}/tax_ids/{id}",
	Method:  "DELETE",
	Summary: "Delete a Customer tax ID",
}

var V1TaxIdsList = resource.OperationSpec{
	Name:    "list",
	Path:    "/v1/customers/{customer}/tax_ids",
	Method:  "GET",
	Summary: "List all Customer tax IDs",
	Params: map[string]*resource.ParamSpec{
		"starting_after": {
			Type:        "string",
			Description: "A cursor for use in pagination. `starting_after` is an object ID that defines your place in the list. For instance, if you make a list request and receive 100 objects, ending with `obj_foo`, your subsequent call can include `starting_after=obj_foo` in order to fetch the next page of the list.",
		},
		"ending_before": {
			Type:        "string",
			Description: "A cursor for use in pagination. `ending_before` is an object ID that defines your place in the list. For instance, if you make a list request and receive 100 objects, starting with `obj_bar`, your subsequent call can include `ending_before=obj_bar` in order to fetch the previous page of the list.",
		},
		"limit": {
			Type:        "integer",
			Description: "A limit on the number of objects to be returned. Limit can range between 1 and 100, and the default is 10.",
		},
	},
}

var V1TaxIdsRetrieve = resource.OperationSpec{
	Name:    "retrieve",
	Path:    "/v1/customers/{customer}/tax_ids/{id}",
	Method:  "GET",
	Summary: "Retrieve a Customer tax ID",
}

var V1TaxIdsCreate = resource.OperationSpec{
	Name:    "create",
	Path:    "/v1/customers/{customer}/tax_ids",
	Method:  "POST",
	Summary: "Create a Customer tax ID",
	Params: map[string]*resource.ParamSpec{
		"type": {
			Type:        "string",
			Description: "Type of the tax ID, one of `ad_nrt`, `ae_trn`, `al_tin`, `am_tin`, `ao_tin`, `ar_cuit`, `au_abn`, `au_arn`, `aw_tin`, `az_tin`, `ba_tin`, `bb_tin`, `bd_bin`, `bf_ifu`, `bg_uic`, `bh_vat`, `bj_ifu`, `bo_tin`, `br_cnpj`, `br_cpf`, `bs_tin`, `by_tin`, `ca_bn`, `ca_gst_hst`, `ca_pst_bc`, `ca_pst_mb`, `ca_pst_sk`, `ca_qst`, `cd_nif`, `ch_uid`, `ch_vat`, `cl_tin`, `cm_niu`, `cn_tin`, `co_nit`, `cr_tin`, `cv_nif`, `de_stn`, `do_rcn`, `ec_ruc`, `eg_tin`, `es_cif`, `et_tin`, `eu_oss_vat`, `eu_vat`, `gb_vat`, `ge_vat`, `gn_nif`, `hk_br`, `hr_oib`, `hu_tin`, `id_npwp`, `il_vat`, `in_gst`, `is_vat`, `jp_cn`, `jp_rn`, `jp_trn`, `ke_pin`, `kg_tin`, `kh_tin`, `kr_brn`, `kz_bin`, `la_tin`, `li_uid`, `li_vat`, `lk_vat`, `ma_vat`, `md_vat`, `me_pib`, `mk_vat`, `mr_nif`, `mx_rfc`, `my_frp`, `my_itn`, `my_sst`, `ng_tin`, `no_vat`, `no_voec`, `np_pan`, `nz_gst`, `om_vat`, `pe_ruc`, `ph_tin`, `pl_nip`, `ro_tin`, `rs_pib`, `ru_inn`, `ru_kpp`, `sa_vat`, `sg_gst`, `sg_uen`, `si_tin`, `sn_ninea`, `sr_fin`, `sv_nit`, `th_vat`, `tj_tin`, `tr_tin`, `tw_vat`, `tz_vat`, `ua_vat`, `ug_tin`, `us_ein`, `uy_ruc`, `uz_tin`, `uz_vat`, `ve_rif`, `vn_tin`, `za_vat`, `zm_tin`, or `zw_tin`",
			Required:    true,
			Enum: []resource.EnumSpec{
				{Value: "ad_nrt"},
				{Value: "ae_trn"},
				{Value: "al_tin"},
				{Value: "am_tin"},
				{Value: "ao_tin"},
				{Value: "ar_cuit"},
				{Value: "au_abn"},
				{Value: "au_arn"},
				{Value: "aw_tin"},
				{Value: "az_tin"},
				{Value: "ba_tin"},
				{Value: "bb_tin"},
				{Value: "bd_bin"},
				{Value: "bf_ifu"},
				{Value: "bg_uic"},
				{Value: "bh_vat"},
				{Value: "bj_ifu"},
				{Value: "bo_tin"},
				{Value: "br_cnpj"},
				{Value: "br_cpf"},
				{Value: "bs_tin"},
				{Value: "by_tin"},
				{Value: "ca_bn"},
				{Value: "ca_gst_hst"},
				{Value: "ca_pst_bc"},
				{Value: "ca_pst_mb"},
				{Value: "ca_pst_sk"},
				{Value: "ca_qst"},
				{Value: "cd_nif"},
				{Value: "ch_uid"},
				{Value: "ch_vat"},
				{Value: "cl_tin"},
				{Value: "cm_niu"},
				{Value: "cn_tin"},
				{Value: "co_nit"},
				{Value: "cr_tin"},
				{Value: "cv_nif"},
				{Value: "de_stn"},
				{Value: "do_rcn"},
				{Value: "ec_ruc"},
				{Value: "eg_tin"},
				{Value: "es_cif"},
				{Value: "et_tin"},
				{Value: "eu_oss_vat"},
				{Value: "eu_vat"},
				{Value: "gb_vat"},
				{Value: "ge_vat"},
				{Value: "gn_nif"},
				{Value: "hk_br"},
				{Value: "hr_oib"},
				{Value: "hu_tin"},
				{Value: "id_npwp"},
				{Value: "il_vat"},
				{Value: "in_gst"},
				{Value: "is_vat"},
				{Value: "jp_cn"},
				{Value: "jp_rn"},
				{Value: "jp_trn"},
				{Value: "ke_pin"},
				{Value: "kg_tin"},
				{Value: "kh_tin"},
				{Value: "kr_brn"},
				{Value: "kz_bin"},
				{Value: "la_tin"},
				{Value: "li_uid"},
				{Value: "li_vat"},
				{Value: "lk_vat"},
				{Value: "ma_vat"},
				{Value: "md_vat"},
				{Value: "me_pib"},
				{Value: "mk_vat"},
				{Value: "mr_nif"},
				{Value: "mx_rfc"},
				{Value: "my_frp"},
				{Value: "my_itn"},
				{Value: "my_sst"},
				{Value: "ng_tin"},
				{Value: "no_vat"},
				{Value: "no_voec"},
				{Value: "np_pan"},
				{Value: "nz_gst"},
				{Value: "om_vat"},
				{Value: "pe_ruc"},
				{Value: "ph_tin"},
				{Value: "pl_nip"},
				{Value: "ro_tin"},
				{Value: "rs_pib"},
				{Value: "ru_inn"},
				{Value: "ru_kpp"},
				{Value: "sa_vat"},
				{Value: "sg_gst"},
				{Value: "sg_uen"},
				{Value: "si_tin"},
				{Value: "sn_ninea"},
				{Value: "sr_fin"},
				{Value: "sv_nit"},
				{Value: "th_vat"},
				{Value: "tj_tin"},
				{Value: "tr_tin"},
				{Value: "tw_vat"},
				{Value: "tz_vat"},
				{Value: "ua_vat"},
				{Value: "ug_tin"},
				{Value: "us_ein"},
				{Value: "uy_ruc"},
				{Value: "uz_tin"},
				{Value: "uz_vat"},
				{Value: "ve_rif"},
				{Value: "vn_tin"},
				{Value: "za_vat"},
				{Value: "zm_tin"},
				{Value: "zw_tin"},
			},
		},
		"value": {
			Type:        "string",
			Description: "Value of the tax ID.",
			Required:    true,
		},
	},
}

var V1PaymentMethodDomainsList = resource.OperationSpec{
	Name:    "list",
	Path:    "/v1/payment_method_domains",
	Method:  "GET",
	Summary: "List payment method domains",
	Params: map[string]*resource.ParamSpec{
		"domain_name": {
			Type:        "string",
			Description: "The domain name that this payment method domain object represents.",
		},
		"enabled": {
			Type:        "boolean",
			Description: "Whether this payment method domain is enabled. If the domain is not enabled, payment methods will not appear in Elements or Embedded Checkout",
		},
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

var V1PaymentMethodDomainsRetrieve = resource.OperationSpec{
	Name:    "retrieve",
	Path:    "/v1/payment_method_domains/{payment_method_domain}",
	Method:  "GET",
	Summary: "Retrieve a payment method domain",
}

var V1PaymentMethodDomainsCreate = resource.OperationSpec{
	Name:    "create",
	Path:    "/v1/payment_method_domains",
	Method:  "POST",
	Summary: "Create a payment method domain",
	Params: map[string]*resource.ParamSpec{
		"domain_name": {
			Type:        "string",
			Description: "The domain name that this payment method domain object represents.",
			Required:    true,
		},
		"enabled": {
			Type:        "boolean",
			Description: "Whether this payment method domain is enabled. If the domain is not enabled, payment methods that require a payment method domain will not appear in Elements or Embedded Checkout.",
		},
	},
}

var V1PaymentMethodDomainsUpdate = resource.OperationSpec{
	Name:    "update",
	Path:    "/v1/payment_method_domains/{payment_method_domain}",
	Method:  "POST",
	Summary: "Update a payment method domain",
	Params: map[string]*resource.ParamSpec{
		"enabled": {
			Type:        "boolean",
			Description: "Whether this payment method domain is enabled. If the domain is not enabled, payment methods that require a payment method domain will not appear in Elements or Embedded Checkout.",
		},
	},
}

var V1PaymentMethodDomainsValidate = resource.OperationSpec{
	Name:    "validate",
	Path:    "/v1/payment_method_domains/{payment_method_domain}/validate",
	Method:  "POST",
	Summary: "Validate an existing payment method domain",
}

var V1DisputesClose = resource.OperationSpec{
	Name:    "close",
	Path:    "/v1/disputes/{dispute}/close",
	Method:  "POST",
	Summary: "Close a dispute",
}

var V1DisputesList = resource.OperationSpec{
	Name:    "list",
	Path:    "/v1/disputes",
	Method:  "GET",
	Summary: "List all disputes",
	Params: map[string]*resource.ParamSpec{
		"starting_after": {
			Type:        "string",
			Description: "A cursor for use in pagination. `starting_after` is an object ID that defines your place in the list. For instance, if you make a list request and receive 100 objects, ending with `obj_foo`, your subsequent call can include `starting_after=obj_foo` in order to fetch the next page of the list.",
		},
		"charge": {
			Type:        "string",
			Description: "Only return disputes associated to the charge specified by this charge ID.",
		},
		"created": {
			Type:        "integer",
			Description: "Only return disputes that were created during the given date interval.",
		},
		"ending_before": {
			Type:        "string",
			Description: "A cursor for use in pagination. `ending_before` is an object ID that defines your place in the list. For instance, if you make a list request and receive 100 objects, starting with `obj_bar`, your subsequent call can include `ending_before=obj_bar` in order to fetch the previous page of the list.",
		},
		"limit": {
			Type:        "integer",
			Description: "A limit on the number of objects to be returned. Limit can range between 1 and 100, and the default is 10.",
		},
		"payment_intent": {
			Type:        "string",
			Description: "Only return disputes associated to the PaymentIntent specified by this PaymentIntent ID.",
		},
	},
}

var V1DisputesRetrieve = resource.OperationSpec{
	Name:    "retrieve",
	Path:    "/v1/disputes/{dispute}",
	Method:  "GET",
	Summary: "Retrieve a dispute",
}

var V1DisputesUpdate = resource.OperationSpec{
	Name:    "update",
	Path:    "/v1/disputes/{dispute}",
	Method:  "POST",
	Summary: "Update a dispute",
	Params: map[string]*resource.ParamSpec{
		"evidence.customer_signature": {
			Type:        "string",
			Description: "(ID of a [file upload](https://stripe.com/docs/guides/file-upload)) A relevant document or contract showing the customer's signature.",
		},
		"evidence.refund_policy": {
			Type:        "string",
			Description: "(ID of a [file upload](https://stripe.com/docs/guides/file-upload)) Your refund policy, as shown to the customer.",
		},
		"evidence.product_description": {
			Type:        "string",
			Description: "A description of the product or service that was sold. Has a maximum character count of 20,000.",
		},
		"evidence.refund_policy_disclosure": {
			Type:        "string",
			Description: "Documentation demonstrating that the customer was shown your refund policy prior to purchase. Has a maximum character count of 20,000.",
		},
		"evidence.shipping_address": {
			Type:        "string",
			Description: "The address to which a physical product was shipped. You should try to include as complete address information as possible.",
		},
		"evidence.shipping_carrier": {
			Type:        "string",
			Description: "The delivery service that shipped a physical product, such as Fedex, UPS, USPS, etc. If multiple carriers were used for this purchase, please separate them with commas.",
		},
		"evidence.duplicate_charge_id": {
			Type:        "string",
			Description: "The Stripe ID for the prior charge which appears to be a duplicate of the disputed charge.",
		},
		"evidence.cancellation_rebuttal": {
			Type:        "string",
			Description: "A justification for why the customer's subscription was not canceled. Has a maximum character count of 20,000.",
		},
		"evidence.customer_communication": {
			Type:        "string",
			Description: "(ID of a [file upload](https://stripe.com/docs/guides/file-upload)) Any communication with the customer that you feel is relevant to your case. Examples include emails proving that the customer received the product or service, or demonstrating their use of or satisfaction with the product or service.",
		},
		"evidence.customer_purchase_ip": {
			Type:        "string",
			Description: "The IP address that the customer used when making the purchase.",
		},
		"evidence.customer_name": {
			Type:        "string",
			Description: "The name of the customer.",
		},
		"submit": {
			Type:        "boolean",
			Description: "Whether to immediately submit evidence to the bank. If `false`, evidence is staged on the dispute. Staged evidence is visible in the API and Dashboard, and can be submitted to the bank by making another request with this attribute set to `true` (the default).",
		},
		"evidence.uncategorized_text": {
			Type:        "string",
			Description: "Any additional evidence or statements. Has a maximum character count of 20,000.",
		},
		"evidence.customer_email_address": {
			Type:        "string",
			Description: "The email address of the customer.",
		},
		"evidence.service_documentation": {
			Type:        "string",
			Description: "(ID of a [file upload](https://stripe.com/docs/guides/file-upload)) Documentation showing proof that a service was provided to the customer. This could include a copy of a signed contract, work order, or other form of written agreement.",
		},
		"evidence.access_activity_log": {
			Type:        "string",
			Description: "Any server or activity logs showing proof that the customer accessed or downloaded the purchased digital product. This information should include IP addresses, corresponding timestamps, and any detailed recorded activity. Has a maximum character count of 20,000.",
		},
		"evidence.duplicate_charge_explanation": {
			Type:        "string",
			Description: "An explanation of the difference between the disputed charge versus the prior charge that appears to be a duplicate. Has a maximum character count of 20,000.",
		},
		"evidence.cancellation_policy_disclosure": {
			Type:        "string",
			Description: "An explanation of how and when the customer was shown your refund policy prior to purchase. Has a maximum character count of 20,000.",
		},
		"evidence.receipt": {
			Type:        "string",
			Description: "(ID of a [file upload](https://stripe.com/docs/guides/file-upload)) Any receipt or message sent to the customer notifying them of the charge.",
		},
		"evidence.shipping_date": {
			Type:        "string",
			Description: "The date on which a physical product began its route to the shipping address, in a clear human-readable format.",
		},
		"evidence.shipping_tracking_number": {
			Type:        "string",
			Description: "The tracking number for a physical product, obtained from the delivery service. If multiple tracking numbers were generated for this purchase, please separate them with commas.",
		},
		"evidence.shipping_documentation": {
			Type:        "string",
			Description: "(ID of a [file upload](https://stripe.com/docs/guides/file-upload)) Documentation showing proof that a product was shipped to the customer at the same address the customer provided to you. This could include a copy of the shipment receipt, shipping label, etc. It should show the customer's full shipping address, if possible.",
		},
		"evidence.uncategorized_file": {
			Type:        "string",
			Description: "(ID of a [file upload](https://stripe.com/docs/guides/file-upload)) Any additional evidence or statements.",
		},
		"evidence.duplicate_charge_documentation": {
			Type:        "string",
			Description: "(ID of a [file upload](https://stripe.com/docs/guides/file-upload)) Documentation for the prior charge that can uniquely identify the charge, such as a receipt, shipping label, work order, etc. This document should be paired with a similar document from the disputed payment that proves the two payments are separate.",
		},
		"evidence.refund_refusal_explanation": {
			Type:        "string",
			Description: "A justification for why the customer is not entitled to a refund. Has a maximum character count of 20,000.",
		},
		"evidence.service_date": {
			Type:        "string",
			Description: "The date on which the customer received or began receiving the purchased service, in a clear human-readable format.",
		},
		"evidence.billing_address": {
			Type:        "string",
			Description: "The billing address provided by the customer.",
		},
		"evidence.cancellation_policy": {
			Type:        "string",
			Description: "(ID of a [file upload](https://stripe.com/docs/guides/file-upload)) Your subscription cancellation policy, as shown to the customer.",
		},
	},
}

var V1WebhookEndpointsDelete = resource.OperationSpec{
	Name:    "delete",
	Path:    "/v1/webhook_endpoints/{webhook_endpoint}",
	Method:  "DELETE",
	Summary: "Delete a webhook endpoint",
}

var V1WebhookEndpointsList = resource.OperationSpec{
	Name:    "list",
	Path:    "/v1/webhook_endpoints",
	Method:  "GET",
	Summary: "List all webhook endpoints",
	Params: map[string]*resource.ParamSpec{
		"starting_after": {
			Type:        "string",
			Description: "A cursor for use in pagination. `starting_after` is an object ID that defines your place in the list. For instance, if you make a list request and receive 100 objects, ending with `obj_foo`, your subsequent call can include `starting_after=obj_foo` in order to fetch the next page of the list.",
		},
		"ending_before": {
			Type:        "string",
			Description: "A cursor for use in pagination. `ending_before` is an object ID that defines your place in the list. For instance, if you make a list request and receive 100 objects, starting with `obj_bar`, your subsequent call can include `ending_before=obj_bar` in order to fetch the previous page of the list.",
		},
		"limit": {
			Type:        "integer",
			Description: "A limit on the number of objects to be returned. Limit can range between 1 and 100, and the default is 10.",
		},
	},
}

var V1WebhookEndpointsRetrieve = resource.OperationSpec{
	Name:    "retrieve",
	Path:    "/v1/webhook_endpoints/{webhook_endpoint}",
	Method:  "GET",
	Summary: "Retrieve a webhook endpoint",
}

var V1WebhookEndpointsCreate = resource.OperationSpec{
	Name:    "create",
	Path:    "/v1/webhook_endpoints",
	Method:  "POST",
	Summary: "Create a webhook endpoint",
	Params: map[string]*resource.ParamSpec{
		"url": {
			Type:        "string",
			Description: "The URL of the webhook endpoint.",
			Required:    true,
		},
		"api_version": {
			Type:        "string",
			Description: "Events sent to this endpoint will be generated with this Stripe Version instead of your account's default Stripe Version.",
			Enum: []resource.EnumSpec{
				{Value: "2011-01-01"},
				{Value: "2011-06-21"},
				{Value: "2011-06-28"},
				{Value: "2011-08-01"},
				{Value: "2011-09-15"},
				{Value: "2011-11-17"},
				{Value: "2012-02-23"},
				{Value: "2012-03-25"},
				{Value: "2012-06-18"},
				{Value: "2012-06-28"},
				{Value: "2012-07-09"},
				{Value: "2012-09-24"},
				{Value: "2012-10-26"},
				{Value: "2012-11-07"},
				{Value: "2013-02-11"},
				{Value: "2013-02-13"},
				{Value: "2013-07-05"},
				{Value: "2013-08-12"},
				{Value: "2013-08-13"},
				{Value: "2013-10-29"},
				{Value: "2013-12-03"},
				{Value: "2014-01-31"},
				{Value: "2014-03-13"},
				{Value: "2014-03-28"},
				{Value: "2014-05-19"},
				{Value: "2014-06-13"},
				{Value: "2014-06-17"},
				{Value: "2014-07-22"},
				{Value: "2014-07-26"},
				{Value: "2014-08-04"},
				{Value: "2014-08-20"},
				{Value: "2014-09-08"},
				{Value: "2014-10-07"},
				{Value: "2014-11-05"},
				{Value: "2014-11-20"},
				{Value: "2014-12-08"},
				{Value: "2014-12-17"},
				{Value: "2014-12-22"},
				{Value: "2015-01-11"},
				{Value: "2015-01-26"},
				{Value: "2015-02-10"},
				{Value: "2015-02-16"},
				{Value: "2015-02-18"},
				{Value: "2015-03-24"},
				{Value: "2015-04-07"},
				{Value: "2015-06-15"},
				{Value: "2015-07-07"},
				{Value: "2015-07-13"},
				{Value: "2015-07-28"},
				{Value: "2015-08-07"},
				{Value: "2015-08-19"},
				{Value: "2015-09-03"},
				{Value: "2015-09-08"},
				{Value: "2015-09-23"},
				{Value: "2015-10-01"},
				{Value: "2015-10-12"},
				{Value: "2015-10-16"},
				{Value: "2016-02-03"},
				{Value: "2016-02-19"},
				{Value: "2016-02-22"},
				{Value: "2016-02-23"},
				{Value: "2016-02-29"},
				{Value: "2016-03-07"},
				{Value: "2016-06-15"},
				{Value: "2016-07-06"},
				{Value: "2016-10-19"},
				{Value: "2017-01-27"},
				{Value: "2017-02-14"},
				{Value: "2017-04-06"},
				{Value: "2017-05-25"},
				{Value: "2017-06-05"},
				{Value: "2017-08-15"},
				{Value: "2017-12-14"},
				{Value: "2018-01-23"},
				{Value: "2018-02-05"},
				{Value: "2018-02-06"},
				{Value: "2018-02-28"},
				{Value: "2018-05-21"},
				{Value: "2018-07-27"},
				{Value: "2018-08-23"},
				{Value: "2018-09-06"},
				{Value: "2018-09-24"},
				{Value: "2018-10-31"},
				{Value: "2018-11-08"},
				{Value: "2019-02-11"},
				{Value: "2019-02-19"},
				{Value: "2019-03-14"},
				{Value: "2019-05-16"},
				{Value: "2019-08-14"},
				{Value: "2019-09-09"},
				{Value: "2019-10-08"},
				{Value: "2019-10-17"},
				{Value: "2019-11-05"},
				{Value: "2019-12-03"},
				{Value: "2020-03-02"},
				{Value: "2020-08-27"},
				{Value: "2022-08-01"},
				{Value: "2022-11-15"},
				{Value: "2023-08-16"},
				{Value: "2023-10-16"},
				{Value: "2024-04-10"},
				{Value: "2024-06-20"},
				{Value: "2024-09-30.acacia"},
				{Value: "2024-10-28.acacia"},
				{Value: "2024-11-20.acacia"},
				{Value: "2024-12-18.acacia"},
				{Value: "2025-01-27.acacia"},
				{Value: "2025-02-24.acacia"},
				{Value: "2025-03-01.dashboard"},
				{Value: "2025-03-31.basil"},
				{Value: "2025-04-30.basil"},
				{Value: "2025-05-28.basil"},
				{Value: "2025-06-30.basil"},
				{Value: "2025-07-30.basil"},
				{Value: "2025-08-27.basil"},
				{Value: "2025-09-30.clover"},
				{Value: "2025-10-29.clover"},
				{Value: "2025-11-17.clover"},
				{Value: "2025-12-15.clover"},
				{Value: "2026-01-28.clover"},
				{Value: "2026-02-25.clover"},
			},
		},
		"connect": {
			Type:        "boolean",
			Description: "Whether this endpoint should receive events from connected accounts (`true`), or from your account (`false`). Defaults to `false`.",
		},
		"description": {
			Type:        "string",
			Description: "An optional description of what the webhook is used for.",
		},
		"enabled_events": {
			Type:        "array",
			Description: "The list of events to enable for this endpoint. You may specify `['*']` to enable all events, except those that require explicit selection.",
			Required:    true,
		},
	},
}

var V1WebhookEndpointsUpdate = resource.OperationSpec{
	Name:    "update",
	Path:    "/v1/webhook_endpoints/{webhook_endpoint}",
	Method:  "POST",
	Summary: "Update a webhook endpoint",
	Params: map[string]*resource.ParamSpec{
		"disabled": {
			Type:        "boolean",
			Description: "Disable the webhook endpoint if set to true.",
		},
		"enabled_events": {
			Type:        "array",
			Description: "The list of events to enable for this endpoint. You may specify `['*']` to enable all events, except those that require explicit selection.",
		},
		"url": {
			Type:        "string",
			Description: "The URL of the webhook endpoint.",
		},
		"description": {
			Type:        "string",
			Description: "An optional description of what the webhook is used for.",
		},
	},
}

var V1FeeRefundsRetrieve = resource.OperationSpec{
	Name:    "retrieve",
	Path:    "/v1/application_fees/{fee}/refunds/{id}",
	Method:  "GET",
	Summary: "Retrieve an application fee refund",
}

var V1FeeRefundsList = resource.OperationSpec{
	Name:    "list",
	Path:    "/v1/application_fees/{id}/refunds",
	Method:  "GET",
	Summary: "List all application fee refunds",
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

var V1FeeRefundsUpdate = resource.OperationSpec{
	Name:    "update",
	Path:    "/v1/application_fees/{fee}/refunds/{id}",
	Method:  "POST",
	Summary: "Update an application fee refund",
}

var V1FeeRefundsCreate = resource.OperationSpec{
	Name:    "create",
	Path:    "/v1/application_fees/{id}/refunds",
	Method:  "POST",
	Summary: "Create an application fee refund",
	Params: map[string]*resource.ParamSpec{
		"amount": {
			Type:        "integer",
			Description: "A positive integer, in _cents (or local equivalent)_, representing how much of this fee to refund. Can refund only up to the remaining unrefunded amount of the fee.",
		},
	},
}

var V1PricesCreate = resource.OperationSpec{
	Name:    "create",
	Path:    "/v1/prices",
	Method:  "POST",
	Summary: "Create a price",
	Params: map[string]*resource.ParamSpec{
		"product_data.tax_code": {
			Type:        "string",
			Description: "A [tax code](https://docs.stripe.com/tax/tax-categories) ID.",
		},
		"product_data.unit_label": {
			Type:        "string",
			Description: "A label that represents units of this product. When set, this will be included in customers' receipts, invoices, Checkout, and the customer portal.",
		},
		"recurring.meter": {
			Type:        "string",
			Description: "The meter tracking the usage of a metered price",
		},
		"product": {
			Type:        "string",
			Description: "The ID of the [Product](https://docs.stripe.com/api/products) that this [Price](https://docs.stripe.com/api/prices) will belong to.",
		},
		"billing_scheme": {
			Type:        "string",
			Description: "Describes how to compute the price per period. Either `per_unit` or `tiered`. `per_unit` indicates that the fixed amount (specified in `unit_amount` or `unit_amount_decimal`) will be charged per unit in `quantity` (for prices with `usage_type=licensed`), or per unit of total usage (for prices with `usage_type=metered`). `tiered` indicates that the unit pricing will be computed using a tiering strategy as defined using the `tiers` and `tiers_mode` attributes.",
			Enum: []resource.EnumSpec{
				{Value: "per_unit"},
				{Value: "tiered"},
			},
		},
		"currency": {
			Type:        "string",
			Description: "Three-letter [ISO currency code](https://www.iso.org/iso-4217-currency-codes.html), in lowercase. Must be a [supported currency](https://stripe.com/docs/currencies).",
			Required:    true,
			Format:      "currency",
		},
		"product_data.active": {
			Type:        "boolean",
			Description: "Whether the product is currently available for purchase. Defaults to `true`.",
		},
		"custom_unit_amount.preset": {
			Type:        "integer",
			Description: "The starting unit amount which can be updated by the customer.",
		},
		"nickname": {
			Type:        "string",
			Description: "A brief description of the price, hidden from customers.",
		},
		"tiers_mode": {
			Type:        "string",
			Description: "Defines if the tiering price should be `graduated` or `volume` based. In `volume`-based tiering, the maximum quantity within a period determines the per unit price, in `graduated` tiering pricing can successively change as the quantity grows.",
			Enum: []resource.EnumSpec{
				{Value: "graduated"},
				{Value: "volume"},
			},
		},
		"transform_quantity.round": {
			Type:        "string",
			Description: "After division, either round the result `up` or `down`.",
			Required:    true,
			Enum: []resource.EnumSpec{
				{Value: "down"},
				{Value: "up"},
			},
		},
		"product_data.name": {
			Type:        "string",
			Description: "The product's name, meant to be displayable to the customer.",
			Required:    true,
		},
		"active": {
			Type:        "boolean",
			Description: "Whether the price can be used for new purchases. Defaults to `true`.",
		},
		"lookup_key": {
			Type:        "string",
			Description: "A lookup key used to retrieve prices dynamically from a static string. This may be up to 200 characters.",
		},
		"recurring.usage_type": {
			Type:        "string",
			Description: "Configures how the quantity per period should be determined. Can be either `metered` or `licensed`. `licensed` automatically bills the `quantity` set when adding it to a subscription. `metered` aggregates the total usage based on usage records. Defaults to `licensed`.",
			Enum: []resource.EnumSpec{
				{Value: "licensed"},
				{Value: "metered"},
			},
		},
		"recurring.interval": {
			Type:        "string",
			Description: "Specifies billing frequency. Either `day`, `week`, `month` or `year`.",
			Required:    true,
			Enum: []resource.EnumSpec{
				{Value: "day"},
				{Value: "month"},
				{Value: "week"},
				{Value: "year"},
			},
		},
		"transfer_lookup_key": {
			Type:        "boolean",
			Description: "If set to true, will atomically remove the lookup key from the existing price, and assign it to this price.",
		},
		"unit_amount": {
			Type:        "integer",
			Description: "A positive integer in cents (or local equivalent) (or 0 for a free price) representing how much to charge. One of `unit_amount`, `unit_amount_decimal`, or `custom_unit_amount` is required, unless `billing_scheme=tiered`.",
		},
		"product_data.id": {
			Type:        "string",
			Description: "The identifier for the product. Must be unique. If not provided, an identifier will be randomly generated.",
		},
		"custom_unit_amount.maximum": {
			Type:        "integer",
			Description: "The maximum unit amount the customer can specify for this item.",
		},
		"tax_behavior": {
			Type:        "string",
			Description: "Only required if a [default tax behavior](https://docs.stripe.com/tax/products-prices-tax-categories-tax-behavior#setting-a-default-tax-behavior-(recommended)) was not provided in the Stripe Tax settings. Specifies whether the price is considered inclusive of taxes or exclusive of taxes. One of `inclusive`, `exclusive`, or `unspecified`. Once specified as either `inclusive` or `exclusive`, it cannot be changed.",
			Enum: []resource.EnumSpec{
				{Value: "exclusive"},
				{Value: "inclusive"},
				{Value: "unspecified"},
			},
		},
		"transform_quantity.divide_by": {
			Type:        "integer",
			Description: "Divide usage by this number.",
			Required:    true,
		},
		"unit_amount_decimal": {
			Type:        "string",
			Description: "Same as `unit_amount`, but accepts a decimal value in cents (or local equivalent) with at most 12 decimal places. Only one of `unit_amount` and `unit_amount_decimal` can be set.",
			Format:      "decimal",
		},
		"product_data.statement_descriptor": {
			Type:        "string",
			Description: "An arbitrary string to be displayed on your customer's credit card or bank statement. While most banks display this information consistently, some may display it incorrectly or not at all.\n\nThis may be up to 22 characters. The statement description may not include `<`, `>`, `\\`, `\"`, `'` characters, and will appear on your customer's statement in capital letters. Non-ASCII characters are automatically stripped.",
		},
		"recurring.trial_period_days": {
			Type:        "integer",
			Description: "Default number of trial days when subscribing a customer to this price using [`trial_from_plan=true`](https://docs.stripe.com/api#create_subscription-trial_from_plan).",
		},
		"recurring.interval_count": {
			Type:        "integer",
			Description: "The number of intervals between subscription billings. For example, `interval=month` and `interval_count=3` bills every 3 months. Maximum of three years interval allowed (3 years, 36 months, or 156 weeks).",
		},
		"custom_unit_amount.enabled": {
			Type:        "boolean",
			Description: "Pass in `true` to enable `custom_unit_amount`, otherwise omit `custom_unit_amount`.",
			Required:    true,
		},
		"custom_unit_amount.minimum": {
			Type:        "integer",
			Description: "The minimum unit amount the customer can specify for this item. Must be at least the minimum charge amount.",
		},
	},
}

var V1PricesUpdate = resource.OperationSpec{
	Name:    "update",
	Path:    "/v1/prices/{price}",
	Method:  "POST",
	Summary: "Update a price",
	Params: map[string]*resource.ParamSpec{
		"active": {
			Type:        "boolean",
			Description: "Whether the price can be used for new purchases. Defaults to `true`.",
		},
		"lookup_key": {
			Type:        "string",
			Description: "A lookup key used to retrieve prices dynamically from a static string. This may be up to 200 characters.",
		},
		"nickname": {
			Type:        "string",
			Description: "A brief description of the price, hidden from customers.",
		},
		"tax_behavior": {
			Type:        "string",
			Description: "Only required if a [default tax behavior](https://docs.stripe.com/tax/products-prices-tax-categories-tax-behavior#setting-a-default-tax-behavior-(recommended)) was not provided in the Stripe Tax settings. Specifies whether the price is considered inclusive of taxes or exclusive of taxes. One of `inclusive`, `exclusive`, or `unspecified`. Once specified as either `inclusive` or `exclusive`, it cannot be changed.",
			Enum: []resource.EnumSpec{
				{Value: "exclusive"},
				{Value: "inclusive"},
				{Value: "unspecified"},
			},
		},
		"transfer_lookup_key": {
			Type:        "boolean",
			Description: "If set to true, will atomically remove the lookup key from the existing price, and assign it to this price.",
		},
	},
}

var V1PricesList = resource.OperationSpec{
	Name:    "list",
	Path:    "/v1/prices",
	Method:  "GET",
	Summary: "List all prices",
	Params: map[string]*resource.ParamSpec{
		"currency": {
			Type:        "string",
			Description: "Only return prices for the given currency.",
			Format:      "currency",
		},
		"limit": {
			Type:        "integer",
			Description: "A limit on the number of objects to be returned. Limit can range between 1 and 100, and the default is 10.",
		},
		"active": {
			Type:        "boolean",
			Description: "Only return prices that are active or inactive (e.g., pass `false` to list all inactive prices).",
		},
		"created": {
			Type:        "integer",
			Description: "A filter on the list, based on the object `created` field. The value can be a string with an integer Unix timestamp, or it can be a dictionary with a number of different query options.",
		},
		"ending_before": {
			Type:        "string",
			Description: "A cursor for use in pagination. `ending_before` is an object ID that defines your place in the list. For instance, if you make a list request and receive 100 objects, starting with `obj_bar`, your subsequent call can include `ending_before=obj_bar` in order to fetch the previous page of the list.",
		},
		"lookup_keys": {
			Type:        "array",
			Description: "Only return the price with these lookup_keys, if any exist. You can specify up to 10 lookup_keys.",
		},
		"product": {
			Type:        "string",
			Description: "Only return prices for the given product.",
		},
		"starting_after": {
			Type:        "string",
			Description: "A cursor for use in pagination. `starting_after` is an object ID that defines your place in the list. For instance, if you make a list request and receive 100 objects, ending with `obj_foo`, your subsequent call can include `starting_after=obj_foo` in order to fetch the next page of the list.",
		},
		"type": {
			Type:        "string",
			Description: "Only return prices of type `recurring` or `one_time`.",
			Enum: []resource.EnumSpec{
				{Value: "one_time"},
				{Value: "recurring"},
			},
		},
	},
}

var V1PricesRetrieve = resource.OperationSpec{
	Name:    "retrieve",
	Path:    "/v1/prices/{price}",
	Method:  "GET",
	Summary: "Retrieve a price",
}

var V1PricesSearch = resource.OperationSpec{
	Name:    "search",
	Path:    "/v1/prices/search",
	Method:  "GET",
	Summary: "Search prices",
	Params: map[string]*resource.ParamSpec{
		"page": {
			Type:        "string",
			Description: "A cursor for pagination across multiple pages of results. Don't include this parameter on the first call. Use the next_page value returned in a previous response to request subsequent results.",
		},
		"query": {
			Type:        "string",
			Description: "The search query string. See [search query language](https://docs.stripe.com/search#search-query-language) and the list of supported [query fields for prices](https://docs.stripe.com/search#query-fields-for-prices).",
			Required:    true,
		},
		"limit": {
			Type:        "integer",
			Description: "A limit on the number of objects to be returned. Limit can range between 1 and 100, and the default is 10.",
		},
	},
}

var V1BalanceSettingssRetrieve = resource.OperationSpec{
	Name:    "retrieve",
	Path:    "/v1/balance_settings",
	Method:  "GET",
	Summary: "Retrieve balance settings",
}

var V1BalanceSettingssUpdate = resource.OperationSpec{
	Name:    "update",
	Path:    "/v1/balance_settings",
	Method:  "POST",
	Summary: "Update balance settings",
	Params: map[string]*resource.ParamSpec{
		"payments.payouts.schedule.monthly_payout_days": {
			Type:        "array",
			Description: "The days of the month when available funds are paid out, specified as an array of numbers between 1--31. Payouts nominally scheduled between the 29th and 31st of the month are instead sent on the last day of a shorter month. Required and applicable only if `interval` is `monthly`.",
		},
		"payments.payouts.schedule.weekly_payout_days": {
			Type:        "array",
			Description: "The days of the week when available funds are paid out, specified as an array, e.g., [`monday`, `tuesday`]. Required and applicable only if `interval` is `weekly`.",
		},
		"payments.payouts.statement_descriptor": {
			Type:        "string",
			Description: "The text that appears on the bank account statement for payouts. If not set, this defaults to the platform's bank descriptor as set in the Dashboard.",
		},
		"payments.settlement_timing.delay_days_override": {
			Type:        "integer",
			Description: "Change `delay_days` for this account, which determines the number of days charge funds are held before becoming available. The maximum value is 31. Passing an empty string to `delay_days_override` will return `delay_days` to the default, which is the lowest available value for the account. [Learn more about controlling delay days](/connect/manage-payout-schedule).",
		},
		"payments.debit_negative_balances": {
			Type:        "boolean",
			Description: "A Boolean indicating whether Stripe should try to reclaim negative balances from an attached bank account. For details, see [Understanding Connect Account Balances](/connect/account-balances).",
		},
		"payments.payouts.schedule.interval": {
			Type:        "string",
			Description: "How frequently available funds are paid out. One of: `daily`, `manual`, `weekly`, or `monthly`. Default is `daily`.",
			Enum: []resource.EnumSpec{
				{Value: "daily"},
				{Value: "manual"},
				{Value: "monthly"},
				{Value: "weekly"},
			},
		},
	},
}

var V1AccountLinksCreate = resource.OperationSpec{
	Name:    "create",
	Path:    "/v1/account_links",
	Method:  "POST",
	Summary: "Create an account link",
	Params: map[string]*resource.ParamSpec{
		"refresh_url": {
			Type:        "string",
			Description: "The URL the user will be redirected to if the account link is expired, has been previously-visited, or is otherwise invalid. The URL you specify should attempt to generate a new account link with the same parameters used to create the original account link, then redirect the user to the new account link's URL so they can continue with Connect Onboarding. If a new account link cannot be generated or the redirect fails you should display a useful error to the user.",
		},
		"return_url": {
			Type:        "string",
			Description: "The URL that the user will be redirected to upon leaving or completing the linked flow.",
		},
		"type": {
			Type:        "string",
			Description: "The type of account link the user is requesting.\n\nYou can create Account Links of type `account_update` only for connected accounts where your platform is responsible for collecting requirements, including Custom accounts. You can't create them for accounts that have access to a Stripe-hosted Dashboard. If you use [Connect embedded components](/connect/get-started-connect-embedded-components), you can include components that allow your connected accounts to update their own information. For an account without Stripe-hosted Dashboard access where Stripe is liable for negative balances, you must use embedded components.",
			Required:    true,
			Enum: []resource.EnumSpec{
				{Value: "account_onboarding"},
				{Value: "account_update"},
			},
		},
		"account": {
			Type:        "string",
			Description: "The identifier of the account to create an account link for.",
			Required:    true,
		},
		"collect": {
			Type:        "string",
			Description: "The collect parameter is deprecated. Use `collection_options` instead.",
			Enum: []resource.EnumSpec{
				{Value: "currently_due"},
				{Value: "eventually_due"},
			},
		},
		"collection_options.fields": {
			Type:        "string",
			Description: "Specifies whether the platform collects only currently_due requirements (`currently_due`) or both currently_due and eventually_due requirements (`eventually_due`). If you don't specify `collection_options`, the default value is `currently_due`.",
			Enum: []resource.EnumSpec{
				{Value: "currently_due"},
				{Value: "eventually_due"},
			},
		},
		"collection_options.future_requirements": {
			Type:        "string",
			Description: "Specifies whether the platform collects future_requirements in addition to requirements in Connect Onboarding. The default value is `omit`.",
			Enum: []resource.EnumSpec{
				{Value: "include"},
				{Value: "omit"},
			},
		},
	},
}

var V1LoginLinksCreate = resource.OperationSpec{
	Name:    "create",
	Path:    "/v1/accounts/{account}/login_links",
	Method:  "POST",
	Summary: "Create a login link",
}

var V1SubscriptionsUpdate = resource.OperationSpec{
	Name:    "update",
	Path:    "/v1/subscriptions/{subscription_exposed_id}",
	Method:  "POST",
	Summary: "Update a subscription",
	Params: map[string]*resource.ParamSpec{
		"days_until_due": {
			Type:        "integer",
			Description: "Number of days a customer has to pay invoices generated by this subscription. Valid only for subscriptions where `collection_method` is set to `send_invoice`.",
		},
		"cancel_at_period_end": {
			Type:        "boolean",
			Description: "Indicate whether this subscription should cancel at the end of the current period (`current_period_end`). Defaults to `false`.",
		},
		"cancel_at": {
			Type:        "integer",
			Description: "A timestamp at which the subscription should cancel. If set to a date before the current period ends, this will cause a proration if prorations have been enabled using `proration_behavior`. If set during a future period, this will always cause a proration for that period.",
		},
		"on_behalf_of": {
			Type:        "string",
			Description: "The account on behalf of which to charge, for each of the subscription's invoices.",
		},
		"trial_settings.end_behavior.missing_payment_method": {
			Type:        "string",
			Description: "Indicates how the subscription should change when the trial ends if the user did not provide a payment method.",
			Required:    true,
			Enum: []resource.EnumSpec{
				{Value: "cancel"},
				{Value: "create_invoice"},
				{Value: "pause"},
			},
		},
		"automatic_tax.liability.account": {
			Type:        "string",
			Description: "The connected account being referenced when `type` is `account`.",
		},
		"collection_method": {
			Type:        "string",
			Description: "Either `charge_automatically`, or `send_invoice`. When charging automatically, Stripe will attempt to pay this subscription at the end of the cycle using the default source attached to the customer. When sending an invoice, Stripe will email your customer an invoice with payment instructions and mark the subscription as `active`. Defaults to `charge_automatically`.",
			Enum: []resource.EnumSpec{
				{Value: "charge_automatically"},
				{Value: "send_invoice"},
			},
		},
		"default_tax_rates": {
			Type:        "array",
			Description: "The tax rates that will apply to any subscription item that does not have `tax_rates` set. Invoices created will have their `default_tax_rates` populated from the subscription. Pass an empty string to remove previously-defined tax rates.",
		},
		"invoice_settings.issuer.account": {
			Type:        "string",
			Description: "The connected account being referenced when `type` is `account`.",
		},
		"off_session": {
			Type:        "boolean",
			Description: "Indicates if a customer is on or off-session while an invoice payment is attempted. Defaults to `false` (on-session).",
		},
		"cancellation_details.feedback": {
			Type:        "string",
			Description: "The customer submitted reason for why they canceled, if the subscription was canceled explicitly by the user.",
			Enum: []resource.EnumSpec{
				{Value: "customer_service"},
				{Value: "low_quality"},
				{Value: "missing_features"},
				{Value: "other"},
				{Value: "switched_service"},
				{Value: "too_complex"},
				{Value: "too_expensive"},
				{Value: "unused"},
			},
		},
		"default_payment_method": {
			Type:        "string",
			Description: "ID of the default payment method for the subscription. It must belong to the customer associated with the subscription. This takes precedence over `default_source`. If neither are set, invoices will use the customer's [invoice_settings.default_payment_method](https://docs.stripe.com/api/customers/object#customer_object-invoice_settings-default_payment_method) or [default_source](https://docs.stripe.com/api/customers/object#customer_object-default_source).",
		},
		"payment_behavior": {
			Type:        "string",
			Description: "Use `allow_incomplete` to transition the subscription to `status=past_due` if a payment is required but cannot be paid. This allows you to manage scenarios where additional user actions are needed to pay a subscription's invoice. For example, SCA regulation may require 3DS authentication to complete payment. See the [SCA Migration Guide](https://docs.stripe.com/billing/migration/strong-customer-authentication) for Billing to learn more. This is the default behavior.\n\nUse `default_incomplete` to transition the subscription to `status=past_due` when payment is required and await explicit confirmation of the invoice's payment intent. This allows simpler management of scenarios where additional user actions are needed to pay a subscription’s invoice. Such as failed payments, [SCA regulation](https://docs.stripe.com/billing/migration/strong-customer-authentication), or collecting a mandate for a bank debit payment method.\n\nUse `pending_if_incomplete` to update the subscription using [pending updates](https://docs.stripe.com/billing/subscriptions/pending-updates). When you use `pending_if_incomplete` you can only pass the parameters [supported by pending updates](https://docs.stripe.com/billing/pending-updates-reference#supported-attributes).\n\nUse `error_if_incomplete` if you want Stripe to return an HTTP 402 status code if a subscription's invoice cannot be paid. For example, if a payment method requires 3DS authentication due to SCA regulation and further user action is needed, this parameter does not update the subscription and returns an error instead. This was the default behavior for API versions prior to 2019-03-14. See the [changelog](https://docs.stripe.com/changelog/2019-03-14) to learn more.",
			Enum: []resource.EnumSpec{
				{Value: "allow_incomplete"},
				{Value: "default_incomplete"},
				{Value: "error_if_incomplete"},
				{Value: "pending_if_incomplete"},
			},
		},
		"proration_behavior": {
			Type:        "string",
			Description: "Determines how to handle [prorations](https://docs.stripe.com/billing/subscriptions/prorations) when the billing cycle changes (e.g., when switching plans, resetting `billing_cycle_anchor=now`, or starting a trial), or if an item's `quantity` changes. The default value is `create_prorations`.",
			Enum: []resource.EnumSpec{
				{Value: "always_invoice"},
				{Value: "create_prorations"},
				{Value: "none"},
			},
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
		"proration_date": {
			Type:        "integer",
			Description: "If set, prorations will be calculated as though the subscription was updated at the given time. This can be used to apply exactly the same prorations that were previewed with the [create preview](https://stripe.com/docs/api/invoices/create_preview) endpoint. `proration_date` can also be used to implement custom proration logic, such as prorating by day instead of by second, by providing the time that you wish to use for proration calculations.",
			Format:      "unix-time",
		},
		"invoice_settings.issuer.type": {
			Type:        "string",
			Description: "Type of the account referenced in the request.",
			Required:    true,
			Enum: []resource.EnumSpec{
				{Value: "account"},
				{Value: "self"},
			},
		},
		"application_fee_percent": {
			Type:        "number",
			Description: "A non-negative decimal between 0 and 100, with at most two decimal places. This represents the percentage of the subscription invoice total that will be transferred to the application owner's Stripe account. The request must be made by a platform account on a connected account in order to set an application fee percentage. For more information, see the application fees [documentation](https://stripe.com/docs/connect/subscriptions#collecting-fees-on-subscriptions).",
		},
		"cancellation_details.comment": {
			Type:        "string",
			Description: "Additional comments about why the user canceled the subscription, if the subscription was canceled explicitly by the user.",
		},
		"description": {
			Type:        "string",
			Description: "The subscription's description, meant to be displayable to the customer. Use this field to optionally store an explanation of the subscription for rendering in Stripe surfaces and certain local payment methods UIs.",
		},
		"automatic_tax.enabled": {
			Type:        "boolean",
			Description: "Enabled automatic tax calculation which will automatically compute tax rates on all invoices generated by the subscription.",
			Required:    true,
		},
		"payment_settings.payment_method_types": {
			Type:        "array",
			Description: "The list of payment method types (e.g. card) to provide to the invoice’s PaymentIntent. If not set, Stripe attempts to automatically determine the types to use by looking at the invoice’s default payment method, the subscription’s default payment method, the customer’s default payment method, and your [invoice template settings](https://dashboard.stripe.com/settings/billing/invoice). Should not be specified with payment_method_configuration",
		},
		"invoice_settings.account_tax_ids": {
			Type:        "array",
			Description: "The account tax IDs associated with the subscription. Will be set on invoices generated by the subscription.",
		},
		"billing_cycle_anchor": {
			Type:        "string",
			Description: "Either `now` or `unchanged`. Setting the value to `now` resets the subscription's billing cycle anchor to the current time (in UTC). For more information, see the billing cycle [documentation](https://docs.stripe.com/billing/subscriptions/billing-cycle).",
			Enum: []resource.EnumSpec{
				{Value: "now"},
				{Value: "unchanged"},
			},
		},
		"trial_from_plan": {
			Type:        "boolean",
			Description: "Indicates if a plan's `trial_period_days` should be applied to the subscription. Setting `trial_end` per subscription is preferred, and this defaults to `false`. Setting this flag to `true` together with `trial_end` is not allowed. See [Using trial periods on subscriptions](https://docs.stripe.com/billing/subscriptions/trials) to learn more.",
		},
		"trial_end": {
			Type:        "string",
			Description: "Unix timestamp representing the end of the trial period the customer will get before being charged for the first time. This will always overwrite any trials that might apply via a subscribed plan. If set, `trial_end` will override the default trial period of the plan the customer is being subscribed to. The `billing_cycle_anchor` will be updated to the `trial_end` value. The special value `now` can be provided to end the customer's trial immediately. Can be at most two years from `billing_cycle_anchor`.",
		},
		"default_source": {
			Type:        "string",
			Description: "ID of the default payment source for the subscription. It must belong to the customer associated with the subscription and be in a chargeable state. If `default_payment_method` is also set, `default_payment_method` will take precedence. If neither are set, invoices will use the customer's [invoice_settings.default_payment_method](https://docs.stripe.com/api/customers/object#customer_object-invoice_settings-default_payment_method) or [default_source](https://docs.stripe.com/api/customers/object#customer_object-default_source).",
		},
		"payment_settings.save_default_payment_method": {
			Type:        "string",
			Description: "Configure whether Stripe updates `subscription.default_payment_method` when payment succeeds. Defaults to `off` if unspecified.",
			Enum: []resource.EnumSpec{
				{Value: "off"},
				{Value: "on_subscription"},
			},
		},
	},
}

var V1SubscriptionsCancel = resource.OperationSpec{
	Name:    "cancel",
	Path:    "/v1/subscriptions/{subscription_exposed_id}",
	Method:  "DELETE",
	Summary: "Cancel a subscription",
}

var V1SubscriptionsList = resource.OperationSpec{
	Name:    "list",
	Path:    "/v1/subscriptions",
	Method:  "GET",
	Summary: "List subscriptions",
	Params: map[string]*resource.ParamSpec{
		"current_period_start": {
			Type:        "integer",
			Description: "Only return subscriptions whose maximum item current_period_start falls within the given date interval.",
		},
		"customer": {
			Type:        "string",
			Description: "The ID of the customer whose subscriptions you're retrieving.",
		},
		"customer_account": {
			Type:        "string",
			Description: "The ID of the account representing the customer whose subscriptions you're retrieving.",
		},
		"limit": {
			Type:        "integer",
			Description: "A limit on the number of objects to be returned. Limit can range between 1 and 100, and the default is 10.",
		},
		"price": {
			Type:        "string",
			Description: "Filter for subscriptions that contain this recurring price ID.",
		},
		"starting_after": {
			Type:        "string",
			Description: "A cursor for use in pagination. `starting_after` is an object ID that defines your place in the list. For instance, if you make a list request and receive 100 objects, ending with `obj_foo`, your subsequent call can include `starting_after=obj_foo` in order to fetch the next page of the list.",
		},
		"test_clock": {
			Type:        "string",
			Description: "Filter for subscriptions that are associated with the specified test clock. The response will not include subscriptions with test clocks if this and the customer parameter is not set.",
		},
		"collection_method": {
			Type:        "string",
			Description: "The collection method of the subscriptions to retrieve. Either `charge_automatically` or `send_invoice`.",
			Enum: []resource.EnumSpec{
				{Value: "charge_automatically"},
				{Value: "send_invoice"},
			},
		},
		"created": {
			Type:        "integer",
			Description: "Only return subscriptions that were created during the given date interval.",
		},
		"current_period_end": {
			Type:        "integer",
			Description: "Only return subscriptions whose minimum item current_period_end falls within the given date interval.",
		},
		"ending_before": {
			Type:        "string",
			Description: "A cursor for use in pagination. `ending_before` is an object ID that defines your place in the list. For instance, if you make a list request and receive 100 objects, starting with `obj_bar`, your subsequent call can include `ending_before=obj_bar` in order to fetch the previous page of the list.",
		},
		"plan": {
			Type:        "string",
			Description: "The ID of the plan whose subscriptions will be retrieved.",
		},
		"status": {
			Type:        "string",
			Description: "The status of the subscriptions to retrieve. Passing in a value of `canceled` will return all canceled subscriptions, including those belonging to deleted customers. Pass `ended` to find subscriptions that are canceled and subscriptions that are expired due to [incomplete payment](https://docs.stripe.com/billing/subscriptions/overview#subscription-statuses). Passing in a value of `all` will return subscriptions of all statuses. If no value is supplied, all subscriptions that have not been canceled are returned.",
			Enum: []resource.EnumSpec{
				{Value: "active"},
				{Value: "all"},
				{Value: "canceled"},
				{Value: "ended"},
				{Value: "incomplete"},
				{Value: "incomplete_expired"},
				{Value: "past_due"},
				{Value: "paused"},
				{Value: "trialing"},
				{Value: "unpaid"},
			},
		},
	},
}

var V1SubscriptionsResume = resource.OperationSpec{
	Name:    "resume",
	Path:    "/v1/subscriptions/{subscription}/resume",
	Method:  "POST",
	Summary: "Resume a subscription",
	Params: map[string]*resource.ParamSpec{
		"proration_behavior": {
			Type:        "string",
			Description: "Determines how to handle [prorations](https://docs.stripe.com/billing/subscriptions/prorations) resulting from the `billing_cycle_anchor` being `unchanged`. When the `billing_cycle_anchor` is set to `now` (default value), no prorations are generated. If no value is passed, the default is `create_prorations`.",
			Enum: []resource.EnumSpec{
				{Value: "always_invoice"},
				{Value: "create_prorations"},
				{Value: "none"},
			},
		},
		"proration_date": {
			Type:        "integer",
			Description: "If set, prorations will be calculated as though the subscription was resumed at the given time. This can be used to apply exactly the same prorations that were previewed with the [create preview](https://stripe.com/docs/api/invoices/create_preview) endpoint.",
			Format:      "unix-time",
		},
		"billing_cycle_anchor": {
			Type:        "string",
			Description: "The billing cycle anchor that applies when the subscription is resumed. Either `now` or `unchanged`. The default is `now`. For more information, see the billing cycle [documentation](https://docs.stripe.com/billing/subscriptions/billing-cycle).",
			Enum: []resource.EnumSpec{
				{Value: "now"},
				{Value: "unchanged"},
			},
		},
	},
}

var V1SubscriptionsDeleteDiscount = resource.OperationSpec{
	Name:    "delete_discount",
	Path:    "/v1/subscriptions/{subscription_exposed_id}/discount",
	Method:  "DELETE",
	Summary: "Delete a subscription discount",
}

var V1SubscriptionsRetrieve = resource.OperationSpec{
	Name:    "retrieve",
	Path:    "/v1/subscriptions/{subscription_exposed_id}",
	Method:  "GET",
	Summary: "Retrieve a subscription",
}

var V1SubscriptionsSearch = resource.OperationSpec{
	Name:    "search",
	Path:    "/v1/subscriptions/search",
	Method:  "GET",
	Summary: "Search subscriptions",
	Params: map[string]*resource.ParamSpec{
		"limit": {
			Type:        "integer",
			Description: "A limit on the number of objects to be returned. Limit can range between 1 and 100, and the default is 10.",
		},
		"page": {
			Type:        "string",
			Description: "A cursor for pagination across multiple pages of results. Don't include this parameter on the first call. Use the next_page value returned in a previous response to request subsequent results.",
		},
		"query": {
			Type:        "string",
			Description: "The search query string. See [search query language](https://docs.stripe.com/search#search-query-language) and the list of supported [query fields for subscriptions](https://docs.stripe.com/search#query-fields-for-subscriptions).",
			Required:    true,
		},
	},
}

var V1SubscriptionsCreate = resource.OperationSpec{
	Name:    "create",
	Path:    "/v1/subscriptions",
	Method:  "POST",
	Summary: "Create a subscription",
	Params: map[string]*resource.ParamSpec{
		"customer_account": {
			Type:        "string",
			Description: "The identifier of the account representing the customer to subscribe.",
		},
		"trial_period_days": {
			Type:        "integer",
			Description: "Integer representing the number of trial period days before the customer is charged for the first time. This will always overwrite any trials that might apply via a subscribed plan. See [Using trial periods on subscriptions](https://docs.stripe.com/billing/subscriptions/trials) to learn more.",
		},
		"proration_behavior": {
			Type:        "string",
			Description: "Determines how to handle [prorations](https://docs.stripe.com/billing/subscriptions/prorations) resulting from the `billing_cycle_anchor`. If no value is passed, the default is `create_prorations`.",
			Enum: []resource.EnumSpec{
				{Value: "always_invoice"},
				{Value: "create_prorations"},
				{Value: "none"},
			},
		},
		"billing_cycle_anchor_config.hour": {
			Type:        "integer",
			Description: "The hour of the day the anchor should be. Ranges from 0 to 23.",
		},
		"on_behalf_of": {
			Type:        "string",
			Description: "The account on behalf of which to charge, for each of the subscription's invoices.",
		},
		"billing_cycle_anchor": {
			Type:        "integer",
			Description: "A future timestamp in UTC format to anchor the subscription's [billing cycle](https://docs.stripe.com/subscriptions/billing-cycle). The anchor is the reference point that aligns future billing cycle dates. It sets the day of week for `week` intervals, the day of month for `month` and `year` intervals, and the month of year for `year` intervals.",
			Format:      "unix-time",
		},
		"cancel_at_period_end": {
			Type:        "boolean",
			Description: "Indicate whether this subscription should cancel at the end of the current period (`current_period_end`). Defaults to `false`.",
		},
		"invoice_settings.issuer.account": {
			Type:        "string",
			Description: "The connected account being referenced when `type` is `account`.",
		},
		"description": {
			Type:        "string",
			Description: "The subscription's description, meant to be displayable to the customer. Use this field to optionally store an explanation of the subscription for rendering in Stripe surfaces and certain local payment methods UIs.",
		},
		"billing_mode.type": {
			Type:        "string",
			Description: "Controls the calculation and orchestration of prorations and invoices for subscriptions. If no value is passed, the default is `flexible`.",
			Required:    true,
			Enum: []resource.EnumSpec{
				{Value: "classic"},
				{Value: "flexible"},
			},
		},
		"default_payment_method": {
			Type:        "string",
			Description: "ID of the default payment method for the subscription. It must belong to the customer associated with the subscription. This takes precedence over `default_source`. If neither are set, invoices will use the customer's [invoice_settings.default_payment_method](https://docs.stripe.com/api/customers/object#customer_object-invoice_settings-default_payment_method) or [default_source](https://docs.stripe.com/api/customers/object#customer_object-default_source).",
		},
		"billing_cycle_anchor_config.minute": {
			Type:        "integer",
			Description: "The minute of the hour the anchor should be. Ranges from 0 to 59.",
		},
		"trial_end": {
			Type:        "string",
			Description: "Unix timestamp representing the end of the trial period the customer will get before being charged for the first time. If set, trial_end will override the default trial period of the plan the customer is being subscribed to. The special value `now` can be provided to end the customer's trial immediately. Can be at most two years from `billing_cycle_anchor`. See [Using trial periods on subscriptions](https://docs.stripe.com/billing/subscriptions/trials) to learn more.",
		},
		"days_until_due": {
			Type:        "integer",
			Description: "Number of days a customer has to pay invoices generated by this subscription. Valid only for subscriptions where `collection_method` is set to `send_invoice`.",
		},
		"cancel_at": {
			Type:        "integer",
			Description: "A timestamp at which the subscription should cancel. If set to a date before the current period ends, this will cause a proration if prorations have been enabled using `proration_behavior`. If set during a future period, this will always cause a proration for that period.",
		},
		"backdate_start_date": {
			Type:        "integer",
			Description: "A past timestamp to backdate the subscription's start date to. If set, the first invoice will contain line items for the timespan between the start date and the current time. Can be combined with trials and the billing cycle anchor.",
			Format:      "unix-time",
		},
		"billing_cycle_anchor_config.day_of_month": {
			Type:        "integer",
			Description: "The day of the month the anchor should be. Ranges from 1 to 31.",
			Required:    true,
		},
		"off_session": {
			Type:        "boolean",
			Description: "Indicates if a customer is on or off-session while an invoice payment is attempted. Defaults to `false` (on-session).",
		},
		"payment_behavior": {
			Type:        "string",
			Description: "Only applies to subscriptions with `collection_method=charge_automatically`.\n\nUse `allow_incomplete` to create Subscriptions with `status=incomplete` if the first invoice can't be paid. Creating Subscriptions with this status allows you to manage scenarios where additional customer actions are needed to pay a subscription's invoice. For example, SCA regulation may require 3DS authentication to complete payment. See the [SCA Migration Guide](https://docs.stripe.com/billing/migration/strong-customer-authentication) for Billing to learn more. This is the default behavior.\n\nUse `default_incomplete` to create Subscriptions with `status=incomplete` when the first invoice requires payment, otherwise start as active. Subscriptions transition to `status=active` when successfully confirming the PaymentIntent on the first invoice. This allows simpler management of scenarios where additional customer actions are needed to pay a subscription’s invoice, such as failed payments, [SCA regulation](https://docs.stripe.com/billing/migration/strong-customer-authentication), or collecting a mandate for a bank debit payment method. If the PaymentIntent is not confirmed within 23 hours Subscriptions transition to `status=incomplete_expired`, which is a terminal state.\n\nUse `error_if_incomplete` if you want Stripe to return an HTTP 402 status code if a subscription's first invoice can't be paid. For example, if a payment method requires 3DS authentication due to SCA regulation and further customer action is needed, this parameter doesn't create a Subscription and returns an error instead. This was the default behavior for API versions prior to 2019-03-14. See the [changelog](https://docs.stripe.com/upgrades#2019-03-14) to learn more.\n\n`pending_if_incomplete` is only used with updates and cannot be passed when creating a Subscription.\n\nSubscriptions with `collection_method=send_invoice` are automatically activated regardless of the first Invoice status.",
			Enum: []resource.EnumSpec{
				{Value: "allow_incomplete"},
				{Value: "default_incomplete"},
				{Value: "error_if_incomplete"},
				{Value: "pending_if_incomplete"},
			},
		},
		"billing_mode.flexible.proration_discounts": {
			Type:        "string",
			Description: "Controls how invoices and invoice items display proration amounts and discount amounts.",
			Enum: []resource.EnumSpec{
				{Value: "included"},
				{Value: "itemized"},
			},
		},
		"default_source": {
			Type:        "string",
			Description: "ID of the default payment source for the subscription. It must belong to the customer associated with the subscription and be in a chargeable state. If `default_payment_method` is also set, `default_payment_method` will take precedence. If neither are set, invoices will use the customer's [invoice_settings.default_payment_method](https://docs.stripe.com/api/customers/object#customer_object-invoice_settings-default_payment_method) or [default_source](https://docs.stripe.com/api/customers/object#customer_object-default_source).",
		},
		"trial_settings.end_behavior.missing_payment_method": {
			Type:        "string",
			Description: "Indicates how the subscription should change when the trial ends if the user did not provide a payment method.",
			Required:    true,
			Enum: []resource.EnumSpec{
				{Value: "cancel"},
				{Value: "create_invoice"},
				{Value: "pause"},
			},
		},
		"application_fee_percent": {
			Type:        "number",
			Description: "A non-negative decimal between 0 and 100, with at most two decimal places. This represents the percentage of the subscription invoice total that will be transferred to the application owner's Stripe account. The request must be made by a platform account on a connected account in order to set an application fee percentage. For more information, see the application fees [documentation](https://stripe.com/docs/connect/subscriptions#collecting-fees-on-subscriptions).",
		},
		"payment_settings.payment_method_types": {
			Type:        "array",
			Description: "The list of payment method types (e.g. card) to provide to the invoice’s PaymentIntent. If not set, Stripe attempts to automatically determine the types to use by looking at the invoice’s default payment method, the subscription’s default payment method, the customer’s default payment method, and your [invoice template settings](https://dashboard.stripe.com/settings/billing/invoice). Should not be specified with payment_method_configuration",
		},
		"billing_cycle_anchor_config.second": {
			Type:        "integer",
			Description: "The second of the minute the anchor should be. Ranges from 0 to 59.",
		},
		"trial_from_plan": {
			Type:        "boolean",
			Description: "Indicates if a plan's `trial_period_days` should be applied to the subscription. Setting `trial_end` per subscription is preferred, and this defaults to `false`. Setting this flag to `true` together with `trial_end` is not allowed. See [Using trial periods on subscriptions](https://docs.stripe.com/billing/subscriptions/trials) to learn more.",
		},
		"currency": {
			Type:        "string",
			Description: "Three-letter [ISO currency code](https://www.iso.org/iso-4217-currency-codes.html), in lowercase. Must be a [supported currency](https://stripe.com/docs/currencies).",
			Format:      "currency",
		},
		"invoice_settings.account_tax_ids": {
			Type:        "array",
			Description: "The account tax IDs associated with the subscription. Will be set on invoices generated by the subscription.",
		},
		"transfer_data.amount_percent": {
			Type:        "number",
			Description: "A non-negative decimal between 0 and 100, with at most two decimal places. This represents the percentage of the subscription invoice total that will be transferred to the destination account. By default, the entire amount is transferred to the destination.",
		},
		"customer": {
			Type:        "string",
			Description: "The identifier of the customer to subscribe.",
		},
		"payment_settings.save_default_payment_method": {
			Type:        "string",
			Description: "Configure whether Stripe updates `subscription.default_payment_method` when payment succeeds. Defaults to `off` if unspecified.",
			Enum: []resource.EnumSpec{
				{Value: "off"},
				{Value: "on_subscription"},
			},
		},
		"billing_cycle_anchor_config.month": {
			Type:        "integer",
			Description: "The month to start full cycle periods. Ranges from 1 to 12.",
		},
		"automatic_tax.liability.account": {
			Type:        "string",
			Description: "The connected account being referenced when `type` is `account`.",
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
		"invoice_settings.issuer.type": {
			Type:        "string",
			Description: "Type of the account referenced in the request.",
			Required:    true,
			Enum: []resource.EnumSpec{
				{Value: "account"},
				{Value: "self"},
			},
		},
		"transfer_data.destination": {
			Type:        "string",
			Description: "ID of an existing, connected Stripe account.",
			Required:    true,
		},
		"default_tax_rates": {
			Type:        "array",
			Description: "The tax rates that will apply to any subscription item that does not have `tax_rates` set. Invoices created will have their `default_tax_rates` populated from the subscription.",
		},
		"collection_method": {
			Type:        "string",
			Description: "Either `charge_automatically`, or `send_invoice`. When charging automatically, Stripe will attempt to pay this subscription at the end of the cycle using the default source attached to the customer. When sending an invoice, Stripe will email your customer an invoice with payment instructions and mark the subscription as `active`. Defaults to `charge_automatically`.",
			Enum: []resource.EnumSpec{
				{Value: "charge_automatically"},
				{Value: "send_invoice"},
			},
		},
		"automatic_tax.enabled": {
			Type:        "boolean",
			Description: "Enabled automatic tax calculation which will automatically compute tax rates on all invoices generated by the subscription.",
			Required:    true,
		},
	},
}

var V1SubscriptionsMigrate = resource.OperationSpec{
	Name:    "migrate",
	Path:    "/v1/subscriptions/{subscription}/migrate",
	Method:  "POST",
	Summary: "Migrate a subscription",
	Params: map[string]*resource.ParamSpec{
		"billing_mode.flexible.proration_discounts": {
			Type:        "string",
			Description: "Controls how invoices and invoice items display proration amounts and discount amounts.",
			Enum: []resource.EnumSpec{
				{Value: "included"},
				{Value: "itemized"},
			},
		},
		"billing_mode.type": {
			Type:        "string",
			Description: "Controls the calculation and orchestration of prorations and invoices for subscriptions.",
			Required:    true,
			Enum: []resource.EnumSpec{
				{Value: "flexible"},
			},
		},
	},
}

var V1InvoiceitemsCreate = resource.OperationSpec{
	Name:    "create",
	Path:    "/v1/invoiceitems",
	Method:  "POST",
	Summary: "Create an invoice item",
	Params: map[string]*resource.ParamSpec{
		"pricing.price": {
			Type:        "string",
			Description: "The ID of the price object.",
		},
		"price_data.product": {
			Type:        "string",
			Description: "The ID of the [Product](https://docs.stripe.com/api/products) that this [Price](https://docs.stripe.com/api/prices) will belong to.",
			Required:    true,
		},
		"tax_behavior": {
			Type:        "string",
			Description: "Only required if a [default tax behavior](https://docs.stripe.com/tax/products-prices-tax-categories-tax-behavior#setting-a-default-tax-behavior-(recommended)) was not provided in the Stripe Tax settings. Specifies whether the price is considered inclusive of taxes or exclusive of taxes. One of `inclusive`, `exclusive`, or `unspecified`. Once specified as either `inclusive` or `exclusive`, it cannot be changed.",
			Enum: []resource.EnumSpec{
				{Value: "exclusive"},
				{Value: "inclusive"},
				{Value: "unspecified"},
			},
		},
		"period.end": {
			Type:        "integer",
			Description: "The end of the period, which must be greater than or equal to the start. This value is inclusive.",
			Required:    true,
			Format:      "unix-time",
		},
		"amount": {
			Type:        "integer",
			Description: "The integer amount in cents (or local equivalent) of the charge to be applied to the upcoming invoice. Passing in a negative `amount` will reduce the `amount_due` on the invoice.",
		},
		"price_data.unit_amount": {
			Type:        "integer",
			Description: "A positive integer in cents (or local equivalent) (or 0 for a free price) representing how much to charge.",
		},
		"price_data.currency": {
			Type:        "string",
			Description: "Three-letter [ISO currency code](https://www.iso.org/iso-4217-currency-codes.html), in lowercase. Must be a [supported currency](https://stripe.com/docs/currencies).",
			Required:    true,
			Format:      "currency",
		},
		"tax_code": {
			Type:        "string",
			Description: "A [tax code](https://docs.stripe.com/tax/tax-categories) ID.",
		},
		"tax_rates": {
			Type:        "array",
			Description: "The tax rates which apply to the invoice item. When set, the `default_tax_rates` on the invoice do not apply to this invoice item.",
		},
		"unit_amount_decimal": {
			Type:        "string",
			Description: "The decimal unit amount in cents (or local equivalent) of the charge to be applied to the upcoming invoice. This `unit_amount_decimal` will be multiplied by the quantity to get the full amount. Passing in a negative `unit_amount_decimal` will reduce the `amount_due` on the invoice. Accepts at most 12 decimal places.",
			Format:      "decimal",
		},
		"description": {
			Type:        "string",
			Description: "An arbitrary string which you can attach to the invoice item. The description is displayed in the invoice for easy tracking.",
		},
		"invoice": {
			Type:        "string",
			Description: "The ID of an existing invoice to add this invoice item to. For subscription invoices, when left blank, the invoice item will be added to the next upcoming scheduled invoice. For standalone invoices, the invoice item won't be automatically added unless you pass `pending_invoice_item_behavior: 'include'` when creating the invoice. This is useful when adding invoice items in response to an invoice.created webhook. You can only add invoice items to draft invoices and there is a maximum of 250 items per invoice.",
		},
		"customer_account": {
			Type:        "string",
			Description: "The ID of the account representing the customer to bill for this invoice item.",
		},
		"price_data.unit_amount_decimal": {
			Type:        "string",
			Description: "Same as `unit_amount`, but accepts a decimal value in cents (or local equivalent) with at most 12 decimal places. Only one of `unit_amount` and `unit_amount_decimal` can be set.",
			Format:      "decimal",
		},
		"price_data.tax_behavior": {
			Type:        "string",
			Description: "Only required if a [default tax behavior](https://docs.stripe.com/tax/products-prices-tax-categories-tax-behavior#setting-a-default-tax-behavior-(recommended)) was not provided in the Stripe Tax settings. Specifies whether the price is considered inclusive of taxes or exclusive of taxes. One of `inclusive`, `exclusive`, or `unspecified`. Once specified as either `inclusive` or `exclusive`, it cannot be changed.",
			Enum: []resource.EnumSpec{
				{Value: "exclusive"},
				{Value: "inclusive"},
				{Value: "unspecified"},
			},
		},
		"discountable": {
			Type:        "boolean",
			Description: "Controls whether discounts apply to this invoice item. Defaults to false for prorations or negative invoice items, and true for all other invoice items.",
		},
		"customer": {
			Type:        "string",
			Description: "The ID of the customer to bill for this invoice item.",
		},
		"quantity": {
			Type:        "integer",
			Description: "Non-negative integer. The quantity of units for the invoice item.",
		},
		"subscription": {
			Type:        "string",
			Description: "The ID of a subscription to add this invoice item to. When left blank, the invoice item is added to the next upcoming scheduled invoice. When set, scheduled invoices for subscriptions other than the specified subscription will ignore the invoice item. Use this when you want to express that an invoice item has been accrued within the context of a particular subscription.",
		},
		"currency": {
			Type:        "string",
			Description: "Three-letter [ISO currency code](https://www.iso.org/iso-4217-currency-codes.html), in lowercase. Must be a [supported currency](https://stripe.com/docs/currencies).",
			Format:      "currency",
		},
		"period.start": {
			Type:        "integer",
			Description: "The start of the period. This value is inclusive.",
			Required:    true,
			Format:      "unix-time",
		},
	},
}

var V1InvoiceitemsUpdate = resource.OperationSpec{
	Name:    "update",
	Path:    "/v1/invoiceitems/{invoiceitem}",
	Method:  "POST",
	Summary: "Update an invoice item",
	Params: map[string]*resource.ParamSpec{
		"tax_behavior": {
			Type:        "string",
			Description: "Only required if a [default tax behavior](https://docs.stripe.com/tax/products-prices-tax-categories-tax-behavior#setting-a-default-tax-behavior-(recommended)) was not provided in the Stripe Tax settings. Specifies whether the price is considered inclusive of taxes or exclusive of taxes. One of `inclusive`, `exclusive`, or `unspecified`. Once specified as either `inclusive` or `exclusive`, it cannot be changed.",
			Enum: []resource.EnumSpec{
				{Value: "exclusive"},
				{Value: "inclusive"},
				{Value: "unspecified"},
			},
		},
		"amount": {
			Type:        "integer",
			Description: "The integer amount in cents (or local equivalent) of the charge to be applied to the upcoming invoice. If you want to apply a credit to the customer's account, pass a negative amount.",
		},
		"period.start": {
			Type:        "integer",
			Description: "The start of the period. This value is inclusive.",
			Required:    true,
			Format:      "unix-time",
		},
		"pricing.price": {
			Type:        "string",
			Description: "The ID of the price object.",
		},
		"tax_rates": {
			Type:        "array",
			Description: "The tax rates which apply to the invoice item. When set, the `default_tax_rates` on the invoice do not apply to this invoice item. Pass an empty string to remove previously-defined tax rates.",
		},
		"price_data.currency": {
			Type:        "string",
			Description: "Three-letter [ISO currency code](https://www.iso.org/iso-4217-currency-codes.html), in lowercase. Must be a [supported currency](https://stripe.com/docs/currencies).",
			Required:    true,
			Format:      "currency",
		},
		"tax_code": {
			Type:        "string",
			Description: "A [tax code](https://docs.stripe.com/tax/tax-categories) ID.",
		},
		"price_data.tax_behavior": {
			Type:        "string",
			Description: "Only required if a [default tax behavior](https://docs.stripe.com/tax/products-prices-tax-categories-tax-behavior#setting-a-default-tax-behavior-(recommended)) was not provided in the Stripe Tax settings. Specifies whether the price is considered inclusive of taxes or exclusive of taxes. One of `inclusive`, `exclusive`, or `unspecified`. Once specified as either `inclusive` or `exclusive`, it cannot be changed.",
			Enum: []resource.EnumSpec{
				{Value: "exclusive"},
				{Value: "inclusive"},
				{Value: "unspecified"},
			},
		},
		"price_data.unit_amount_decimal": {
			Type:        "string",
			Description: "Same as `unit_amount`, but accepts a decimal value in cents (or local equivalent) with at most 12 decimal places. Only one of `unit_amount` and `unit_amount_decimal` can be set.",
			Format:      "decimal",
		},
		"description": {
			Type:        "string",
			Description: "An arbitrary string which you can attach to the invoice item. The description is displayed in the invoice for easy tracking.",
		},
		"unit_amount_decimal": {
			Type:        "string",
			Description: "The decimal unit amount in cents (or local equivalent) of the charge to be applied to the upcoming invoice. This `unit_amount_decimal` will be multiplied by the quantity to get the full amount. Passing in a negative `unit_amount_decimal` will reduce the `amount_due` on the invoice. Accepts at most 12 decimal places.",
			Format:      "decimal",
		},
		"discountable": {
			Type:        "boolean",
			Description: "Controls whether discounts apply to this invoice item. Defaults to false for prorations or negative invoice items, and true for all other invoice items. Cannot be set to true for prorations.",
		},
		"period.end": {
			Type:        "integer",
			Description: "The end of the period, which must be greater than or equal to the start. This value is inclusive.",
			Required:    true,
			Format:      "unix-time",
		},
		"quantity": {
			Type:        "integer",
			Description: "Non-negative integer. The quantity of units for the invoice item.",
		},
		"price_data.product": {
			Type:        "string",
			Description: "The ID of the [Product](https://docs.stripe.com/api/products) that this [Price](https://docs.stripe.com/api/prices) will belong to.",
			Required:    true,
		},
		"price_data.unit_amount": {
			Type:        "integer",
			Description: "A positive integer in cents (or local equivalent) (or 0 for a free price) representing how much to charge.",
		},
	},
}

var V1InvoiceitemsDelete = resource.OperationSpec{
	Name:    "delete",
	Path:    "/v1/invoiceitems/{invoiceitem}",
	Method:  "DELETE",
	Summary: "Delete an invoice item",
}

var V1InvoiceitemsList = resource.OperationSpec{
	Name:    "list",
	Path:    "/v1/invoiceitems",
	Method:  "GET",
	Summary: "List all invoice items",
	Params: map[string]*resource.ParamSpec{
		"pending": {
			Type:        "boolean",
			Description: "Set to `true` to only show pending invoice items, which are not yet attached to any invoices. Set to `false` to only show invoice items already attached to invoices. If unspecified, no filter is applied.",
		},
		"starting_after": {
			Type:        "string",
			Description: "A cursor for use in pagination. `starting_after` is an object ID that defines your place in the list. For instance, if you make a list request and receive 100 objects, ending with `obj_foo`, your subsequent call can include `starting_after=obj_foo` in order to fetch the next page of the list.",
		},
		"created": {
			Type:        "integer",
			Description: "Only return invoice items that were created during the given date interval.",
		},
		"customer": {
			Type:        "string",
			Description: "The identifier of the customer whose invoice items to return. If none is provided, returns all invoice items.",
		},
		"customer_account": {
			Type:        "string",
			Description: "The identifier of the account representing the customer whose invoice items to return. If none is provided, returns all invoice items.",
		},
		"ending_before": {
			Type:        "string",
			Description: "A cursor for use in pagination. `ending_before` is an object ID that defines your place in the list. For instance, if you make a list request and receive 100 objects, starting with `obj_bar`, your subsequent call can include `ending_before=obj_bar` in order to fetch the previous page of the list.",
		},
		"invoice": {
			Type:        "string",
			Description: "Only return invoice items belonging to this invoice. If none is provided, all invoice items will be returned. If specifying an invoice, no customer identifier is needed.",
		},
		"limit": {
			Type:        "integer",
			Description: "A limit on the number of objects to be returned. Limit can range between 1 and 100, and the default is 10.",
		},
	},
}

var V1InvoiceitemsRetrieve = resource.OperationSpec{
	Name:    "retrieve",
	Path:    "/v1/invoiceitems/{invoiceitem}",
	Method:  "GET",
	Summary: "Retrieve an invoice item",
}

var V1CashBalancesRetrieve = resource.OperationSpec{
	Name:    "retrieve",
	Path:    "/v1/customers/{customer}/cash_balance",
	Method:  "GET",
	Summary: "Retrieve a cash balance",
}

var V1CashBalancesUpdate = resource.OperationSpec{
	Name:    "update",
	Path:    "/v1/customers/{customer}/cash_balance",
	Method:  "POST",
	Summary: "Update a cash balance's settings",
	Params: map[string]*resource.ParamSpec{
		"settings.reconciliation_mode": {
			Type:        "string",
			Description: "Controls how funds transferred by the customer are applied to payment intents and invoices. Valid options are `automatic`, `manual`, or `merchant_default`. For more information about these reconciliation modes, see [Reconciliation](https://docs.stripe.com/payments/customer-balance/reconciliation).",
			Enum: []resource.EnumSpec{
				{Value: "automatic"},
				{Value: "manual"},
				{Value: "merchant_default"},
			},
		},
	},
}

var V1AccountsUpdate = resource.OperationSpec{
	Name:    "update",
	Path:    "/v1/accounts/{account}",
	Method:  "POST",
	Summary: "Update an account",
	Params: map[string]*resource.ParamSpec{
		"capabilities.swish_payments.requested": {
			Type:        "boolean",
			Description: "Passing true requests the capability for the account, if it is not already requested. A requested capability may not immediately become active. Any requirements to activate the capability are returned in the `requirements` arrays.",
		},
		"settings.payments.statement_descriptor": {
			Type:        "string",
			Description: "The default text that appears on statements for non-card charges outside of Japan. For card charges, if you don't set a `statement_descriptor_prefix`, this text is also used as the statement descriptor prefix. In that case, if concatenating the statement descriptor suffix causes the combined statement descriptor to exceed 22 characters, we truncate the `statement_descriptor` text to limit the full descriptor to 22 characters. For more information about statement descriptors and their requirements, see the [account settings documentation](https://docs.stripe.com/get-started/account/statement-descriptors).",
		},
		"individual.id_number_secondary": {
			Type:        "string",
			Description: "The government-issued secondary ID number of the individual, as appropriate for the representative's country, will be used for enhanced verification checks. In Thailand, this would be the laser code found on the back of an ID card. Instead of the number itself, you can also provide a [PII token created with Stripe.js](/js/tokens/create_token?type=pii).",
		},
		"business_profile.annual_revenue.currency": {
			Type:        "string",
			Description: "Three-letter [ISO currency code](https://www.iso.org/iso-4217-currency-codes.html), in lowercase. Must be a [supported currency](https://stripe.com/docs/currencies).",
			Required:    true,
			Format:      "currency",
		},
		"individual.registered_address.line1": {
			Type:        "string",
			Description: "Address line 1, such as the street, PO Box, or company name.",
		},
		"individual.registered_address.postal_code": {
			Type:        "string",
			Description: "ZIP or postal code.",
		},
		"documents.company_license.files": {
			Type:        "array",
			Description: "One or more document ids returned by a [file upload](https://api.stripe.com#create_file) with a `purpose` value of `account_requirement`.",
		},
		"capabilities.grabpay_payments.requested": {
			Type:        "boolean",
			Description: "Passing true requests the capability for the account, if it is not already requested. A requested capability may not immediately become active. Any requirements to activate the capability are returned in the `requirements` arrays.",
		},
		"capabilities.payco_payments.requested": {
			Type:        "boolean",
			Description: "Passing true requests the capability for the account, if it is not already requested. A requested capability may not immediately become active. Any requirements to activate the capability are returned in the `requirements` arrays.",
		},
		"company.tax_id": {
			Type:        "string",
			Description: "The business ID number of the company, as appropriate for the company’s country. (Examples are an Employer ID Number in the U.S., a Business Number in Canada, or a Company Number in the UK.)",
		},
		"capabilities.samsung_pay_payments.requested": {
			Type:        "boolean",
			Description: "Passing true requests the capability for the account, if it is not already requested. A requested capability may not immediately become active. Any requirements to activate the capability are returned in the `requirements` arrays.",
		},
		"capabilities.transfers.requested": {
			Type:        "boolean",
			Description: "Passing true requests the capability for the account, if it is not already requested. A requested capability may not immediately become active. Any requirements to activate the capability are returned in the `requirements` arrays.",
		},
		"settings.treasury.tos_acceptance.date": {
			Type:        "integer",
			Description: "The Unix timestamp marking when the account representative accepted the service agreement.",
			Format:      "unix-time",
		},
		"individual.email": {
			Type:        "string",
			Description: "The individual's email address.",
		},
		"individual.first_name": {
			Type:        "string",
			Description: "The individual's first name.",
		},
		"company.name": {
			Type:        "string",
			Description: "The company's legal name.",
		},
		"settings.card_payments.statement_descriptor_prefix_kana": {
			Type:        "string",
			Description: "The Kana variation of the default text that appears on credit card statements when a charge is made (Japan only). This field prefixes any dynamic `statement_descriptor_suffix_kana` specified on the charge. `statement_descriptor_prefix_kana` is useful for maximizing descriptor space for the dynamic portion.",
		},
		"individual.first_name_kanji": {
			Type:        "string",
			Description: "The Kanji variation of the individual's first name (Japan only).",
		},
		"business_profile.support_url": {
			Type:        "string",
			Description: "A publicly available website for handling support issues.",
		},
		"capabilities.card_payments.requested": {
			Type:        "boolean",
			Description: "Passing true requests the capability for the account, if it is not already requested. A requested capability may not immediately become active. Any requirements to activate the capability are returned in the `requirements` arrays.",
		},
		"capabilities.tax_reporting_us_1099_misc.requested": {
			Type:        "boolean",
			Description: "Passing true requests the capability for the account, if it is not already requested. A requested capability may not immediately become active. Any requirements to activate the capability are returned in the `requirements` arrays.",
		},
		"individual.registered_address.line2": {
			Type:        "string",
			Description: "Address line 2, such as the apartment, suite, unit, or building.",
		},
		"documents.proof_of_registration.signer.person": {
			Type:        "string",
			Description: "The token of the person signing the document, if applicable.",
		},
		"business_profile.support_email": {
			Type:        "string",
			Description: "A publicly available email address for sending support issues to.",
		},
		"individual.political_exposure": {
			Type:        "string",
			Description: "Indicates if the person or any of their representatives, family members, or other closely related persons, declares that they hold or have held an important public job or function, in any jurisdiction.",
			Enum: []resource.EnumSpec{
				{Value: "existing"},
				{Value: "none"},
			},
		},
		"company.vat_id": {
			Type:        "string",
			Description: "The VAT number of the company.",
		},
		"company.ownership_declaration.user_agent": {
			Type:        "string",
			Description: "The user agent of the browser from which the beneficial owner attestation was made.",
		},
		"business_profile.mcc": {
			Type:        "string",
			Description: "[The merchant category code for the account](/connect/setting-mcc). MCCs are used to classify businesses based on the goods or services they provide.",
		},
		"individual.relationship.executive": {
			Type:        "boolean",
			Description: "Whether the person has significant responsibility to control, manage, or direct the organization.",
		},
		"documents.proof_of_ultimate_beneficial_ownership.files": {
			Type:        "array",
			Description: "One or more document ids returned by a [file upload](https://api.stripe.com#create_file) with a `purpose` value of `account_requirement`.",
		},
		"capabilities.mb_way_payments.requested": {
			Type:        "boolean",
			Description: "Passing true requests the capability for the account, if it is not already requested. A requested capability may not immediately become active. Any requirements to activate the capability are returned in the `requirements` arrays.",
		},
		"capabilities.cartes_bancaires_payments.requested": {
			Type:        "boolean",
			Description: "Passing true requests the capability for the account, if it is not already requested. A requested capability may not immediately become active. Any requirements to activate the capability are returned in the `requirements` arrays.",
		},
		"settings.payments.statement_descriptor_kanji": {
			Type:        "string",
			Description: "The Kanji variation of `statement_descriptor` used for charges in Japan. Japanese statement descriptors have [special requirements](https://docs.stripe.com/get-started/account/statement-descriptors#set-japanese-statement-descriptors).",
		},
		"individual.address_kana.postal_code": {
			Type:        "string",
			Description: "Postal code.",
		},
		"documents.company_ministerial_decree.files": {
			Type:        "array",
			Description: "One or more document ids returned by a [file upload](https://api.stripe.com#create_file) with a `purpose` value of `account_requirement`.",
		},
		"business_type": {
			Type:        "string",
			Description: "The business type. Once you create an [Account Link](/api/account_links) or [Account Session](/api/account_sessions), this property can only be updated for accounts where [controller.requirement_collection](/api/accounts/object#account_object-controller-requirement_collection) is `application`, which includes Custom accounts.",
			Enum: []resource.EnumSpec{
				{Value: "company"},
				{Value: "government_entity"},
				{Value: "individual"},
				{Value: "non_profit"},
			},
		},
		"individual.registered_address.country": {
			Type:        "string",
			Description: "Two-letter country code ([ISO 3166-1 alpha-2](https://en.wikipedia.org/wiki/ISO_3166-1_alpha-2)).",
		},
		"individual.address_kana.line1": {
			Type:        "string",
			Description: "Block or building number.",
		},
		"company.export_license_id": {
			Type:        "string",
			Description: "The export license ID number of the company, also referred as Import Export Code (India only).",
		},
		"documents.company_registration_verification.files": {
			Type:        "array",
			Description: "One or more document ids returned by a [file upload](https://api.stripe.com#create_file) with a `purpose` value of `account_requirement`.",
		},
		"email": {
			Type:        "string",
			Description: "The email address of the account holder. This is only to make the account easier to identify to you. If [controller.requirement_collection](/api/accounts/object#account_object-controller-requirement_collection) is `application`, which includes Custom accounts, Stripe doesn't email the account without your consent.",
		},
		"business_profile.monthly_estimated_revenue.currency": {
			Type:        "string",
			Description: "Three-letter [ISO currency code](https://www.iso.org/iso-4217-currency-codes.html), in lowercase. Must be a [supported currency](https://stripe.com/docs/currencies).",
			Required:    true,
			Format:      "currency",
		},
		"capabilities.zip_payments.requested": {
			Type:        "boolean",
			Description: "Passing true requests the capability for the account, if it is not already requested. A requested capability may not immediately become active. Any requirements to activate the capability are returned in the `requirements` arrays.",
		},
		"capabilities.us_bank_transfer_payments.requested": {
			Type:        "boolean",
			Description: "Passing true requests the capability for the account, if it is not already requested. A requested capability may not immediately become active. Any requirements to activate the capability are returned in the `requirements` arrays.",
		},
		"capabilities.gb_bank_transfer_payments.requested": {
			Type:        "boolean",
			Description: "Passing true requests the capability for the account, if it is not already requested. A requested capability may not immediately become active. Any requirements to activate the capability are returned in the `requirements` arrays.",
		},
		"tos_acceptance.service_agreement": {
			Type:        "string",
			Description: "The user's service agreement type.",
		},
		"company.address.state": {
			Type:        "string",
			Description: "State, county, province, or region ([ISO 3166-2](https://en.wikipedia.org/wiki/ISO_3166-2)).",
		},
		"company.representative_declaration.date": {
			Type:        "integer",
			Description: "The Unix timestamp marking when the representative declaration attestation was made.",
			Format:      "unix-time",
		},
		"business_profile.annual_revenue.amount": {
			Type:        "integer",
			Description: "A non-negative integer representing the amount in the [smallest currency unit](/currencies#zero-decimal).",
			Required:    true,
		},
		"capabilities.afterpay_clearpay_payments.requested": {
			Type:        "boolean",
			Description: "Passing true requests the capability for the account, if it is not already requested. A requested capability may not immediately become active. Any requirements to activate the capability are returned in the `requirements` arrays.",
		},
		"settings.payouts.debit_negative_balances": {
			Type:        "boolean",
			Description: "A Boolean indicating whether Stripe should try to reclaim negative balances from an attached bank account. For details, see [Understanding Connect Account Balances](/connect/account-balances).",
		},
		"settings.payouts.schedule.delay_days": {
			Type:        "string",
			Description: "The number of days charge funds are held before being paid out. May also be set to `minimum`, representing the lowest available value for the account country. Default is `minimum`. The `delay_days` parameter remains at the last configured value if `interval` is `manual`. [Learn more about controlling payout delay days](/connect/manage-payout-schedule).",
		},
		"company.address.postal_code": {
			Type:        "string",
			Description: "ZIP or postal code.",
		},
		"company.directors_provided": {
			Type:        "boolean",
			Description: "Whether the company's directors have been provided. Set this Boolean to `true` after creating all the company's directors with [the Persons API](/api/persons) for accounts with a `relationship.director` requirement. This value is not automatically set to `true` after creating directors, so it needs to be updated to indicate all directors have been provided.",
		},
		"company.address_kanji.town": {
			Type:        "string",
			Description: "Town or cho-me.",
		},
		"company.address_kanji.country": {
			Type:        "string",
			Description: "Two-letter country code ([ISO 3166-1 alpha-2](https://en.wikipedia.org/wiki/ISO_3166-1_alpha-2)).",
		},
		"business_profile.support_phone": {
			Type:        "string",
			Description: "A publicly available phone number to call with support issues.",
		},
		"capabilities.bancontact_payments.requested": {
			Type:        "boolean",
			Description: "Passing true requests the capability for the account, if it is not already requested. A requested capability may not immediately become active. Any requirements to activate the capability are returned in the `requirements` arrays.",
		},
		"capabilities.naver_pay_payments.requested": {
			Type:        "boolean",
			Description: "Passing true requests the capability for the account, if it is not already requested. A requested capability may not immediately become active. Any requirements to activate the capability are returned in the `requirements` arrays.",
		},
		"capabilities.jcb_payments.requested": {
			Type:        "boolean",
			Description: "Passing true requests the capability for the account, if it is not already requested. A requested capability may not immediately become active. Any requirements to activate the capability are returned in the `requirements` arrays.",
		},
		"individual.id_number": {
			Type:        "string",
			Description: "The government-issued ID number of the individual, as appropriate for the representative's country. (Examples are a Social Security Number in the U.S., or a Social Insurance Number in Canada). Instead of the number itself, you can also provide a [PII token created with Stripe.js](/js/tokens/create_token?type=pii).",
		},
		"individual.verification.document.front": {
			Type:        "string",
			Description: "The front of an ID returned by a [file upload](https://api.stripe.com#create_file) with a `purpose` value of `identity_document`. The uploaded file needs to be a color image (smaller than 8,000px by 8,000px), in JPG, PNG, or PDF format, and less than 10 MB in size.",
		},
		"capabilities.pix_payments.requested": {
			Type:        "boolean",
			Description: "Passing true requests the capability for the account, if it is not already requested. A requested capability may not immediately become active. Any requirements to activate the capability are returned in the `requirements` arrays.",
		},
		"capabilities.klarna_payments.requested": {
			Type:        "boolean",
			Description: "Passing true requests the capability for the account, if it is not already requested. A requested capability may not immediately become active. Any requirements to activate the capability are returned in the `requirements` arrays.",
		},
		"settings.branding.icon": {
			Type:        "string",
			Description: "(ID of a [file upload](https://stripe.com/docs/guides/file-upload)) An icon for the account. Must be square and at least 128px x 128px.",
		},
		"settings.payouts.schedule.weekly_payout_days": {
			Type:        "array",
			Description: "The days of the week when available funds are paid out, specified as an array, e.g., [`monday`, `tuesday`]. Required and applicable only if `interval` is `weekly`.",
		},
		"individual.relationship.owner": {
			Type:        "boolean",
			Description: "Whether the person is an owner of the account’s legal entity.",
		},
		"individual.verification.additional_document.front": {
			Type:        "string",
			Description: "The front of an ID returned by a [file upload](https://api.stripe.com#create_file) with a `purpose` value of `identity_document`. The uploaded file needs to be a color image (smaller than 8,000px by 8,000px), in JPG, PNG, or PDF format, and less than 10 MB in size.",
		},
		"business_profile.support_address.state": {
			Type:        "string",
			Description: "State, county, province, or region ([ISO 3166-2](https://en.wikipedia.org/wiki/ISO_3166-2)).",
		},
		"settings.payouts.statement_descriptor": {
			Type:        "string",
			Description: "The text that appears on the bank account statement for payouts. If not set, this defaults to the platform's bank descriptor as set in the Dashboard.",
		},
		"settings.card_payments.decline_on.avs_failure": {
			Type:        "boolean",
			Description: "Whether Stripe automatically declines charges with an incorrect ZIP or postal code. This setting only applies when a ZIP or postal code is provided and they fail bank verification.",
		},
		"individual.last_name_kana": {
			Type:        "string",
			Description: "The Kana variation of the individual's last name (Japan only).",
		},
		"individual.address_kanji.town": {
			Type:        "string",
			Description: "Town or cho-me.",
		},
		"company.address_kana.line2": {
			Type:        "string",
			Description: "Building details.",
		},
		"company.structure": {
			Type:        "string",
			Description: "The category identifying the legal structure of the company or legal entity. See [Business structure](/connect/identity-verification#business-structure) for more details. Pass an empty string to unset this value.",
			Enum: []resource.EnumSpec{
				{Value: "free_zone_establishment"},
				{Value: "free_zone_llc"},
				{Value: "government_instrumentality"},
				{Value: "governmental_unit"},
				{Value: "incorporated_non_profit"},
				{Value: "incorporated_partnership"},
				{Value: "limited_liability_partnership"},
				{Value: "llc"},
				{Value: "multi_member_llc"},
				{Value: "private_company"},
				{Value: "private_corporation"},
				{Value: "private_partnership"},
				{Value: "public_company"},
				{Value: "public_corporation"},
				{Value: "public_partnership"},
				{Value: "registered_charity"},
				{Value: "single_member_llc"},
				{Value: "sole_establishment"},
				{Value: "sole_proprietorship"},
				{Value: "tax_exempt_government_instrumentality"},
				{Value: "unincorporated_association"},
				{Value: "unincorporated_non_profit"},
				{Value: "unincorporated_partnership"},
			},
		},
		"capabilities.oxxo_payments.requested": {
			Type:        "boolean",
			Description: "Passing true requests the capability for the account, if it is not already requested. A requested capability may not immediately become active. Any requirements to activate the capability are returned in the `requirements` arrays.",
		},
		"capabilities.promptpay_payments.requested": {
			Type:        "boolean",
			Description: "Passing true requests the capability for the account, if it is not already requested. A requested capability may not immediately become active. Any requirements to activate the capability are returned in the `requirements` arrays.",
		},
		"individual.full_name_aliases": {
			Type:        "array",
			Description: "A list of alternate names or aliases that the individual is known by.",
		},
		"default_currency": {
			Type:        "string",
			Description: "Three-letter ISO currency code representing the default currency for the account. This must be a currency that [Stripe supports in the account's country](https://docs.stripe.com/payouts).",
			Format:      "currency",
		},
		"business_profile.monthly_estimated_revenue.amount": {
			Type:        "integer",
			Description: "A non-negative integer representing how much to charge in the [smallest currency unit](/currencies#zero-decimal).",
			Required:    true,
		},
		"company.directorship_declaration.user_agent": {
			Type:        "string",
			Description: "The user agent of the browser from which the directorship declaration attestation was made.",
		},
		"capabilities.sepa_debit_payments.requested": {
			Type:        "boolean",
			Description: "Passing true requests the capability for the account, if it is not already requested. A requested capability may not immediately become active. Any requirements to activate the capability are returned in the `requirements` arrays.",
		},
		"capabilities.boleto_payments.requested": {
			Type:        "boolean",
			Description: "Passing true requests the capability for the account, if it is not already requested. A requested capability may not immediately become active. Any requirements to activate the capability are returned in the `requirements` arrays.",
		},
		"individual.address.city": {
			Type:        "string",
			Description: "City, district, suburb, town, or village.",
		},
		"documents.proof_of_ultimate_beneficial_ownership.signer.person": {
			Type:        "string",
			Description: "The token of the person signing the document, if applicable.",
		},
		"individual.address.state": {
			Type:        "string",
			Description: "State, county, province, or region ([ISO 3166-2](https://en.wikipedia.org/wiki/ISO_3166-2)).",
		},
		"documents.proof_of_registration.files": {
			Type:        "array",
			Description: "One or more document ids returned by a [file upload](https://api.stripe.com#create_file) with a `purpose` value of `account_requirement`.",
		},
		"settings.payouts.schedule.monthly_payout_days": {
			Type:        "array",
			Description: "The days of the month when available funds are paid out, specified as an array of numbers between 1--31. Payouts nominally scheduled between the 29th and 31st of the month are instead sent on the last day of a shorter month. Required and applicable only if `interval` is `monthly` and `monthly_anchor` is not set.",
		},
		"individual.address_kanji.postal_code": {
			Type:        "string",
			Description: "Postal code.",
		},
		"individual.ssn_last_4": {
			Type:        "string",
			Description: "The last four digits of the individual's Social Security Number (U.S. only).",
		},
		"company.verification.document.back": {
			Type:        "string",
			Description: "The back of a document returned by a [file upload](https://api.stripe.com#create_file) with a `purpose` value of `additional_verification`. The uploaded file needs to be a color image (smaller than 8,000px by 8,000px), in JPG, PNG, or PDF format, and less than 10 MB in size.",
		},
		"business_profile.annual_revenue.fiscal_year_end": {
			Type:        "string",
			Description: "The close-out date of the preceding fiscal year in ISO 8601 format. E.g. 2023-12-31 for the 31st of December, 2023.",
			Required:    true,
		},
		"capabilities.nz_bank_account_becs_debit_payments.requested": {
			Type:        "boolean",
			Description: "Passing true requests the capability for the account, if it is not already requested. A requested capability may not immediately become active. Any requirements to activate the capability are returned in the `requirements` arrays.",
		},
		"capabilities.kakao_pay_payments.requested": {
			Type:        "boolean",
			Description: "Passing true requests the capability for the account, if it is not already requested. A requested capability may not immediately become active. Any requirements to activate the capability are returned in the `requirements` arrays.",
		},
		"individual.first_name_kana": {
			Type:        "string",
			Description: "The Kana variation of the individual's first name (Japan only).",
		},
		"individual.address.line1": {
			Type:        "string",
			Description: "Address line 1, such as the street, PO Box, or company name.",
		},
		"capabilities.blik_payments.requested": {
			Type:        "boolean",
			Description: "Passing true requests the capability for the account, if it is not already requested. A requested capability may not immediately become active. Any requirements to activate the capability are returned in the `requirements` arrays.",
		},
		"groups.payments_pricing": {
			Type:        "string",
			Description: "The group the account is in to determine their payments pricing, and null if the account is on customized pricing. [See the Platform pricing tool documentation](https://docs.stripe.com/connect/platform-pricing-tools) for details.",
		},
		"individual.registered_address.state": {
			Type:        "string",
			Description: "State, county, province, or region ([ISO 3166-2](https://en.wikipedia.org/wiki/ISO_3166-2)).",
		},
		"individual.address.country": {
			Type:        "string",
			Description: "Two-letter country code ([ISO 3166-1 alpha-2](https://en.wikipedia.org/wiki/ISO_3166-1_alpha-2)).",
		},
		"individual.last_name": {
			Type:        "string",
			Description: "The individual's last name.",
		},
		"company.address_kanji.line1": {
			Type:        "string",
			Description: "Block or building number.",
		},
		"settings.card_issuing.tos_acceptance.date": {
			Type:        "integer",
			Description: "The Unix timestamp marking when the account representative accepted the service agreement.",
			Format:      "unix-time",
		},
		"capabilities.satispay_payments.requested": {
			Type:        "boolean",
			Description: "Passing true requests the capability for the account, if it is not already requested. A requested capability may not immediately become active. Any requirements to activate the capability are returned in the `requirements` arrays.",
		},
		"capabilities.tax_reporting_us_1099_k.requested": {
			Type:        "boolean",
			Description: "Passing true requests the capability for the account, if it is not already requested. A requested capability may not immediately become active. Any requirements to activate the capability are returned in the `requirements` arrays.",
		},
		"company.representative_declaration.user_agent": {
			Type:        "string",
			Description: "The user agent of the browser from which the representative declaration attestation was made.",
		},
		"company.name_kana": {
			Type:        "string",
			Description: "The Kana variation of the company's legal name (Japan only).",
		},
		"capabilities.ideal_payments.requested": {
			Type:        "boolean",
			Description: "Passing true requests the capability for the account, if it is not already requested. A requested capability may not immediately become active. Any requirements to activate the capability are returned in the `requirements` arrays.",
		},
		"capabilities.kr_card_payments.requested": {
			Type:        "boolean",
			Description: "Passing true requests the capability for the account, if it is not already requested. A requested capability may not immediately become active. Any requirements to activate the capability are returned in the `requirements` arrays.",
		},
		"capabilities.billie_payments.requested": {
			Type:        "boolean",
			Description: "Passing true requests the capability for the account, if it is not already requested. A requested capability may not immediately become active. Any requirements to activate the capability are returned in the `requirements` arrays.",
		},
		"individual.address_kanji.line2": {
			Type:        "string",
			Description: "Building details.",
		},
		"individual.verification.document.back": {
			Type:        "string",
			Description: "The back of an ID returned by a [file upload](https://api.stripe.com#create_file) with a `purpose` value of `identity_document`. The uploaded file needs to be a color image (smaller than 8,000px by 8,000px), in JPG, PNG, or PDF format, and less than 10 MB in size.",
		},
		"business_profile.support_address.line1": {
			Type:        "string",
			Description: "Address line 1, such as the street, PO Box, or company name.",
		},
		"business_profile.minority_owned_business_designation": {
			Type:        "array",
			Description: "Whether the business is a minority-owned, women-owned, and/or LGBTQI+ -owned business.",
		},
		"capabilities.fpx_payments.requested": {
			Type:        "boolean",
			Description: "Passing true requests the capability for the account, if it is not already requested. A requested capability may not immediately become active. Any requirements to activate the capability are returned in the `requirements` arrays.",
		},
		"capabilities.crypto_payments.requested": {
			Type:        "boolean",
			Description: "Passing true requests the capability for the account, if it is not already requested. A requested capability may not immediately become active. Any requirements to activate the capability are returned in the `requirements` arrays.",
		},
		"individual.address_kana.town": {
			Type:        "string",
			Description: "Town or cho-me.",
		},
		"company.address_kana.city": {
			Type:        "string",
			Description: "City or ward.",
		},
		"company.ownership_declaration.date": {
			Type:        "integer",
			Description: "The Unix timestamp marking when the beneficial owner attestation was made.",
			Format:      "unix-time",
		},
		"company.registration_number": {
			Type:        "string",
			Description: "The identification number given to a company when it is registered or incorporated, if distinct from the identification number used for filing taxes. (Examples are the CIN for companies and LLP IN for partnerships in India, and the Company Registration Number in Hong Kong).",
		},
		"documents.proof_of_address.files": {
			Type:        "array",
			Description: "One or more document ids returned by a [file upload](https://api.stripe.com#create_file) with a `purpose` value of `account_requirement`.",
		},
		"capabilities.india_international_payments.requested": {
			Type:        "boolean",
			Description: "Passing true requests the capability for the account, if it is not already requested. A requested capability may not immediately become active. Any requirements to activate the capability are returned in the `requirements` arrays.",
		},
		"capabilities.card_issuing.requested": {
			Type:        "boolean",
			Description: "Passing true requests the capability for the account, if it is not already requested. A requested capability may not immediately become active. Any requirements to activate the capability are returned in the `requirements` arrays.",
		},
		"company.address_kana.postal_code": {
			Type:        "string",
			Description: "Postal code.",
		},
		"capabilities.sepa_bank_transfer_payments.requested": {
			Type:        "boolean",
			Description: "Passing true requests the capability for the account, if it is not already requested. A requested capability may not immediately become active. Any requirements to activate the capability are returned in the `requirements` arrays.",
		},
		"settings.card_issuing.tos_acceptance.user_agent": {
			Type:        "string",
			Description: "The user agent of the browser from which the account representative accepted the service agreement.",
		},
		"settings.card_payments.statement_descriptor_prefix": {
			Type:        "string",
			Description: "The default text that appears on credit card statements when a charge is made. This field prefixes any dynamic `statement_descriptor` specified on the charge. `statement_descriptor_prefix` is useful for maximizing descriptor space for the dynamic portion.",
		},
		"individual.address_kana.line2": {
			Type:        "string",
			Description: "Building details.",
		},
		"capabilities.bacs_debit_payments.requested": {
			Type:        "boolean",
			Description: "Passing true requests the capability for the account, if it is not already requested. A requested capability may not immediately become active. Any requirements to activate the capability are returned in the `requirements` arrays.",
		},
		"capabilities.paynow_payments.requested": {
			Type:        "boolean",
			Description: "Passing true requests the capability for the account, if it is not already requested. A requested capability may not immediately become active. Any requirements to activate the capability are returned in the `requirements` arrays.",
		},
		"individual.last_name_kanji": {
			Type:        "string",
			Description: "The Kanji variation of the individual's last name (Japan only).",
		},
		"individual.maiden_name": {
			Type:        "string",
			Description: "The individual's maiden name.",
		},
		"company.directorship_declaration.ip": {
			Type:        "string",
			Description: "The IP address from which the directorship declaration attestation was made.",
		},
		"settings.card_issuing.tos_acceptance.ip": {
			Type:        "string",
			Description: "The IP address from which the account representative accepted the service agreement.",
		},
		"individual.address.line2": {
			Type:        "string",
			Description: "Address line 2, such as the apartment, suite, unit, or building.",
		},
		"company.owners_provided": {
			Type:        "boolean",
			Description: "Whether the company's owners have been provided. Set this Boolean to `true` after creating all the company's owners with [the Persons API](/api/persons) for accounts with a `relationship.owner` requirement.",
		},
		"company.ownership_exemption_reason": {
			Type:        "string",
			Description: "This value is used to determine if a business is exempt from providing ultimate beneficial owners. See [this support article](https://support.stripe.com/questions/exemption-from-providing-ownership-details) and [changelog](https://docs.stripe.com/changelog/acacia/2025-01-27/ownership-exemption-reason-accounts-api) for more details.",
			Enum: []resource.EnumSpec{
				{Value: "qualified_entity_exceeds_ownership_threshold"},
				{Value: "qualifies_as_financial_institution"},
			},
		},
		"settings.branding.primary_color": {
			Type:        "string",
			Description: "A CSS hex color value representing the primary branding color for this account.",
		},
		"external_account": {
			Type:        "string",
			Description: "A card or bank account to attach to the account for receiving [payouts](/connect/bank-debit-card-payouts) (you won’t be able to use it for top-ups). You can provide either a token, like the ones returned by [Stripe.js](/js), or a dictionary, as documented in the `external_account` parameter for [bank account](/api#account_create_bank_account) creation. <br><br>By default, providing an external account sets it as the new default external account for its currency, and deletes the old default if one exists. To add additional external accounts without replacing the existing default for the currency, use the [bank account](/api#account_create_bank_account) or [card creation](/api#account_create_card) APIs. After you create an [Account Link](/api/account_links) or [Account Session](/api/account_sessions), this property can only be updated for accounts where [controller.requirement_collection](/api/accounts/object#account_object-controller-requirement_collection) is `application`, which includes Custom accounts.",
		},
		"business_profile.url": {
			Type:        "string",
			Description: "The business's publicly available website.",
		},
		"capabilities.affirm_payments.requested": {
			Type:        "boolean",
			Description: "Passing true requests the capability for the account, if it is not already requested. A requested capability may not immediately become active. Any requirements to activate the capability are returned in the `requirements` arrays.",
		},
		"individual.relationship.percent_ownership": {
			Type:        "number",
			Description: "The percent owned by the person of the account's legal entity.",
		},
		"company.tax_id_registrar": {
			Type:        "string",
			Description: "The jurisdiction in which the `tax_id` is registered (Germany-based companies only).",
		},
		"company.phone": {
			Type:        "string",
			Description: "The company's phone number (used for verification).",
		},
		"documents.company_tax_id_verification.files": {
			Type:        "array",
			Description: "One or more document ids returned by a [file upload](https://api.stripe.com#create_file) with a `purpose` value of `account_requirement`.",
		},
		"capabilities.multibanco_payments.requested": {
			Type:        "boolean",
			Description: "Passing true requests the capability for the account, if it is not already requested. A requested capability may not immediately become active. Any requirements to activate the capability are returned in the `requirements` arrays.",
		},
		"individual.address.postal_code": {
			Type:        "string",
			Description: "ZIP or postal code.",
		},
		"individual.address_kanji.country": {
			Type:        "string",
			Description: "Two-letter country code ([ISO 3166-1 alpha-2](https://en.wikipedia.org/wiki/ISO_3166-1_alpha-2)).",
		},
		"capabilities.p24_payments.requested": {
			Type:        "boolean",
			Description: "Passing true requests the capability for the account, if it is not already requested. A requested capability may not immediately become active. Any requirements to activate the capability are returned in the `requirements` arrays.",
		},
		"settings.card_payments.statement_descriptor_prefix_kanji": {
			Type:        "string",
			Description: "The Kanji variation of the default text that appears on credit card statements when a charge is made (Japan only). This field prefixes any dynamic `statement_descriptor_suffix_kanji` specified on the charge. `statement_descriptor_prefix_kanji` is useful for maximizing descriptor space for the dynamic portion.",
		},
		"business_profile.product_description": {
			Type:        "string",
			Description: "Internal-only description of the product sold by, or service provided by, the business. Used by Stripe for risk and underwriting purposes.",
		},
		"settings.branding.logo": {
			Type:        "string",
			Description: "(ID of a [file upload](https://stripe.com/docs/guides/file-upload)) A logo for the account that will be used in Checkout instead of the icon and without the account's name next to it if provided. Must be at least 128px x 128px.",
		},
		"company.address.line2": {
			Type:        "string",
			Description: "Address line 2, such as the apartment, suite, unit, or building.",
		},
		"company.ownership_declaration.ip": {
			Type:        "string",
			Description: "The IP address from which the beneficial owner attestation was made.",
		},
		"settings.card_payments.decline_on.cvc_failure": {
			Type:        "boolean",
			Description: "Whether Stripe automatically declines charges with an incorrect CVC. This setting only applies when a CVC is provided and it fails bank verification.",
		},
		"documents.company_memorandum_of_association.files": {
			Type:        "array",
			Description: "One or more document ids returned by a [file upload](https://api.stripe.com#create_file) with a `purpose` value of `account_requirement`.",
		},
		"capabilities.amazon_pay_payments.requested": {
			Type:        "boolean",
			Description: "Passing true requests the capability for the account, if it is not already requested. A requested capability may not immediately become active. Any requirements to activate the capability are returned in the `requirements` arrays.",
		},
		"capabilities.jp_bank_transfer_payments.requested": {
			Type:        "boolean",
			Description: "Passing true requests the capability for the account, if it is not already requested. A requested capability may not immediately become active. Any requirements to activate the capability are returned in the `requirements` arrays.",
		},
		"individual.gender": {
			Type:        "string",
			Description: "The individual's gender",
		},
		"company.address.country": {
			Type:        "string",
			Description: "Two-letter country code ([ISO 3166-1 alpha-2](https://en.wikipedia.org/wiki/ISO_3166-1_alpha-2)).",
		},
		"company.name_kanji": {
			Type:        "string",
			Description: "The Kanji variation of the company's legal name (Japan only).",
		},
		"business_profile.support_address.country": {
			Type:        "string",
			Description: "Two-letter country code ([ISO 3166-1 alpha-2](https://en.wikipedia.org/wiki/ISO_3166-1_alpha-2)).",
		},
		"settings.branding.secondary_color": {
			Type:        "string",
			Description: "A CSS hex color value representing the secondary branding color for this account.",
		},
		"settings.treasury.tos_acceptance.user_agent": {
			Type:        "string",
			Description: "The user agent of the browser from which the account representative accepted the service agreement.",
		},
		"settings.bacs_debit_payments.display_name": {
			Type:        "string",
			Description: "The Bacs Direct Debit Display Name for this account. For payments made with Bacs Direct Debit, this name appears on the mandate as the statement descriptor. Mobile banking apps display it as the name of the business. To use custom branding, set the Bacs Direct Debit Display Name during or right after creation. Custom branding incurs an additional monthly fee for the platform. If you don't set the display name before requesting Bacs capability, it's automatically set as \"Stripe\" and the account is onboarded to Stripe branding, which is free.",
		},
		"company.verification.document.front": {
			Type:        "string",
			Description: "The front of a document returned by a [file upload](https://api.stripe.com#create_file) with a `purpose` value of `additional_verification`. The uploaded file needs to be a color image (smaller than 8,000px by 8,000px), in JPG, PNG, or PDF format, and less than 10 MB in size.",
		},
		"individual.address_kanji.city": {
			Type:        "string",
			Description: "City or ward.",
		},
		"company.address_kana.country": {
			Type:        "string",
			Description: "Two-letter country code ([ISO 3166-1 alpha-2](https://en.wikipedia.org/wiki/ISO_3166-1_alpha-2)).",
		},
		"business_profile.support_address.city": {
			Type:        "string",
			Description: "City, district, suburb, town, or village.",
		},
		"capabilities.mx_bank_transfer_payments.requested": {
			Type:        "boolean",
			Description: "Passing true requests the capability for the account, if it is not already requested. A requested capability may not immediately become active. Any requirements to activate the capability are returned in the `requirements` arrays.",
		},
		"capabilities.au_becs_debit_payments.requested": {
			Type:        "boolean",
			Description: "Passing true requests the capability for the account, if it is not already requested. A requested capability may not immediately become active. Any requirements to activate the capability are returned in the `requirements` arrays.",
		},
		"capabilities.acss_debit_payments.requested": {
			Type:        "boolean",
			Description: "Passing true requests the capability for the account, if it is not already requested. A requested capability may not immediately become active. Any requirements to activate the capability are returned in the `requirements` arrays.",
		},
		"individual.address_kana.country": {
			Type:        "string",
			Description: "Two-letter country code ([ISO 3166-1 alpha-2](https://en.wikipedia.org/wiki/ISO_3166-1_alpha-2)).",
		},
		"tos_acceptance.date": {
			Type:        "integer",
			Description: "The Unix timestamp marking when the account representative accepted their service agreement.",
			Format:      "unix-time",
		},
		"company.representative_declaration.ip": {
			Type:        "string",
			Description: "The IP address from which the representative declaration attestation was made.",
		},
		"capabilities.twint_payments.requested": {
			Type:        "boolean",
			Description: "Passing true requests the capability for the account, if it is not already requested. A requested capability may not immediately become active. Any requirements to activate the capability are returned in the `requirements` arrays.",
		},
		"settings.payouts.schedule.weekly_anchor": {
			Type:        "string",
			Description: "The day of the week when available funds are paid out, specified as `monday`, `tuesday`, etc. Required and applicable only if `interval` is `weekly`.",
			Enum: []resource.EnumSpec{
				{Value: "friday"},
				{Value: "monday"},
				{Value: "saturday"},
				{Value: "sunday"},
				{Value: "thursday"},
				{Value: "tuesday"},
				{Value: "wednesday"},
			},
		},
		"individual.relationship.director": {
			Type:        "boolean",
			Description: "Whether the person is a director of the account's legal entity. Directors are typically members of the governing board of the company, or responsible for ensuring the company meets its regulatory obligations.",
		},
		"company.address.line1": {
			Type:        "string",
			Description: "Address line 1, such as the street, PO Box, or company name.",
		},
		"business_profile.name": {
			Type:        "string",
			Description: "The customer-facing business name.",
		},
		"capabilities.pay_by_bank_payments.requested": {
			Type:        "boolean",
			Description: "Passing true requests the capability for the account, if it is not already requested. A requested capability may not immediately become active. Any requirements to activate the capability are returned in the `requirements` arrays.",
		},
		"capabilities.legacy_payments.requested": {
			Type:        "boolean",
			Description: "Passing true requests the capability for the account, if it is not already requested. A requested capability may not immediately become active. Any requirements to activate the capability are returned in the `requirements` arrays.",
		},
		"settings.payouts.schedule.monthly_anchor": {
			Type:        "integer",
			Description: "The day of the month when available funds are paid out, specified as a number between 1--31. Payouts nominally scheduled between the 29th and 31st of the month are instead sent on the last day of a shorter month. Required and applicable only if `interval` is `monthly`.",
		},
		"individual.address_kanji.line1": {
			Type:        "string",
			Description: "Block or building number.",
		},
		"company.address_kana.line1": {
			Type:        "string",
			Description: "Block or building number.",
		},
		"capabilities.treasury.requested": {
			Type:        "boolean",
			Description: "Passing true requests the capability for the account, if it is not already requested. A requested capability may not immediately become active. Any requirements to activate the capability are returned in the `requirements` arrays.",
		},
		"capabilities.payto_payments.requested": {
			Type:        "boolean",
			Description: "Passing true requests the capability for the account, if it is not already requested. A requested capability may not immediately become active. Any requirements to activate the capability are returned in the `requirements` arrays.",
		},
		"tos_acceptance.user_agent": {
			Type:        "string",
			Description: "The user agent of the browser from which the account representative accepted their service agreement.",
		},
		"tos_acceptance.ip": {
			Type:        "string",
			Description: "The IP address from which the account representative accepted their service agreement.",
		},
		"capabilities.alma_payments.requested": {
			Type:        "boolean",
			Description: "Passing true requests the capability for the account, if it is not already requested. A requested capability may not immediately become active. Any requirements to activate the capability are returned in the `requirements` arrays.",
		},
		"capabilities.konbini_payments.requested": {
			Type:        "boolean",
			Description: "Passing true requests the capability for the account, if it is not already requested. A requested capability may not immediately become active. Any requirements to activate the capability are returned in the `requirements` arrays.",
		},
		"capabilities.cashapp_payments.requested": {
			Type:        "boolean",
			Description: "Passing true requests the capability for the account, if it is not already requested. A requested capability may not immediately become active. Any requirements to activate the capability are returned in the `requirements` arrays.",
		},
		"account_token": {
			Type:        "string",
			Description: "An [account token](https://api.stripe.com#create_account_token), used to securely provide details to the account.",
		},
		"company.address_kana.state": {
			Type:        "string",
			Description: "Prefecture.",
		},
		"capabilities.sofort_payments.requested": {
			Type:        "boolean",
			Description: "Passing true requests the capability for the account, if it is not already requested. A requested capability may not immediately become active. Any requirements to activate the capability are returned in the `requirements` arrays.",
		},
		"capabilities.mobilepay_payments.requested": {
			Type:        "boolean",
			Description: "Passing true requests the capability for the account, if it is not already requested. A requested capability may not immediately become active. Any requirements to activate the capability are returned in the `requirements` arrays.",
		},
		"settings.invoices.default_account_tax_ids": {
			Type:        "array",
			Description: "The list of default Account Tax IDs to automatically include on invoices. Account Tax IDs get added when an invoice is finalized.",
		},
		"individual.relationship.title": {
			Type:        "string",
			Description: "The person's title (e.g., CEO, Support Engineer).",
		},
		"company.directorship_declaration.date": {
			Type:        "integer",
			Description: "The Unix timestamp marking when the directorship declaration attestation was made.",
			Format:      "unix-time",
		},
		"company.address_kanji.city": {
			Type:        "string",
			Description: "City or ward.",
		},
		"capabilities.revolut_pay_payments.requested": {
			Type:        "boolean",
			Description: "Passing true requests the capability for the account, if it is not already requested. A requested capability may not immediately become active. Any requirements to activate the capability are returned in the `requirements` arrays.",
		},
		"capabilities.us_bank_account_ach_payments.requested": {
			Type:        "boolean",
			Description: "Passing true requests the capability for the account, if it is not already requested. A requested capability may not immediately become active. Any requirements to activate the capability are returned in the `requirements` arrays.",
		},
		"company.executives_provided": {
			Type:        "boolean",
			Description: "Whether the company's executives have been provided. Set this Boolean to `true` after creating all the company's executives with [the Persons API](/api/persons) for accounts with a `relationship.executive` requirement.",
		},
		"company.address_kanji.postal_code": {
			Type:        "string",
			Description: "Postal code.",
		},
		"business_profile.support_address.postal_code": {
			Type:        "string",
			Description: "ZIP or postal code.",
		},
		"capabilities.link_payments.requested": {
			Type:        "boolean",
			Description: "Passing true requests the capability for the account, if it is not already requested. A requested capability may not immediately become active. Any requirements to activate the capability are returned in the `requirements` arrays.",
		},
		"individual.address_kana.state": {
			Type:        "string",
			Description: "Prefecture.",
		},
		"settings.invoices.hosted_payment_method_save": {
			Type:        "string",
			Description: "Whether to save the payment method after a payment is completed for a one-time invoice or a subscription invoice when the customer already has a default payment method on the hosted invoice page.",
			Enum: []resource.EnumSpec{
				{Value: "always"},
				{Value: "never"},
				{Value: "offer"},
			},
		},
		"settings.payouts.schedule.interval": {
			Type:        "string",
			Description: "How frequently available funds are paid out. One of: `daily`, `manual`, `weekly`, or `monthly`. Default is `daily`.",
			Enum: []resource.EnumSpec{
				{Value: "daily"},
				{Value: "manual"},
				{Value: "monthly"},
				{Value: "weekly"},
			},
		},
		"individual.verification.additional_document.back": {
			Type:        "string",
			Description: "The back of an ID returned by a [file upload](https://api.stripe.com#create_file) with a `purpose` value of `identity_document`. The uploaded file needs to be a color image (smaller than 8,000px by 8,000px), in JPG, PNG, or PDF format, and less than 10 MB in size.",
		},
		"individual.address_kana.city": {
			Type:        "string",
			Description: "City or ward.",
		},
		"company.export_purpose_code": {
			Type:        "string",
			Description: "The purpose code to use for export transactions (India only).",
		},
		"individual.address_kanji.state": {
			Type:        "string",
			Description: "Prefecture.",
		},
		"company.address_kanji.line2": {
			Type:        "string",
			Description: "Building details.",
		},
		"company.address_kanji.state": {
			Type:        "string",
			Description: "Prefecture.",
		},
		"settings.payments.statement_descriptor_kana": {
			Type:        "string",
			Description: "The Kana variation of `statement_descriptor` used for charges in Japan. Japanese statement descriptors have [special requirements](https://docs.stripe.com/get-started/account/statement-descriptors#set-japanese-statement-descriptors).",
		},
		"capabilities.bank_transfer_payments.requested": {
			Type:        "boolean",
			Description: "Passing true requests the capability for the account, if it is not already requested. A requested capability may not immediately become active. Any requirements to activate the capability are returned in the `requirements` arrays.",
		},
		"capabilities.giropay_payments.requested": {
			Type:        "boolean",
			Description: "Passing true requests the capability for the account, if it is not already requested. A requested capability may not immediately become active. Any requirements to activate the capability are returned in the `requirements` arrays.",
		},
		"company.address.city": {
			Type:        "string",
			Description: "City, district, suburb, town, or village.",
		},
		"capabilities.eps_payments.requested": {
			Type:        "boolean",
			Description: "Passing true requests the capability for the account, if it is not already requested. A requested capability may not immediately become active. Any requirements to activate the capability are returned in the `requirements` arrays.",
		},
		"individual.phone": {
			Type:        "string",
			Description: "The individual's phone number.",
		},
		"business_profile.estimated_worker_count": {
			Type:        "integer",
			Description: "An estimated upper bound of employees, contractors, vendors, etc. currently working for the business.",
		},
		"business_profile.support_address.line2": {
			Type:        "string",
			Description: "Address line 2, such as the apartment, suite, unit, or building.",
		},
		"settings.treasury.tos_acceptance.ip": {
			Type:        "string",
			Description: "The IP address from which the account representative accepted the service agreement.",
		},
		"documents.bank_account_ownership_verification.files": {
			Type:        "array",
			Description: "One or more document ids returned by a [file upload](https://api.stripe.com#create_file) with a `purpose` value of `account_requirement`.",
		},
		"individual.registered_address.city": {
			Type:        "string",
			Description: "City, district, suburb, town, or village.",
		},
		"company.address_kana.town": {
			Type:        "string",
			Description: "Town or cho-me.",
		},
	},
}

var V1AccountsReject = resource.OperationSpec{
	Name:    "reject",
	Path:    "/v1/accounts/{account}/reject",
	Method:  "POST",
	Summary: "Reject an account",
	Params: map[string]*resource.ParamSpec{
		"reason": {
			Type:        "string",
			Description: "The reason for rejecting the account. Can be `fraud`, `terms_of_service`, or `other`.",
			Required:    true,
		},
	},
}

var V1AccountsDelete = resource.OperationSpec{
	Name:    "delete",
	Path:    "/v1/accounts/{account}",
	Method:  "DELETE",
	Summary: "Delete an account",
}

var V1AccountsRetrieve = resource.OperationSpec{
	Name:    "retrieve",
	Path:    "/v1/account",
	Method:  "GET",
	Summary: "Retrieve account",
}

var V1AccountsList = resource.OperationSpec{
	Name:    "list",
	Path:    "/v1/accounts",
	Method:  "GET",
	Summary: "List all connected accounts",
	Params: map[string]*resource.ParamSpec{
		"limit": {
			Type:        "integer",
			Description: "A limit on the number of objects to be returned. Limit can range between 1 and 100, and the default is 10.",
		},
		"starting_after": {
			Type:        "string",
			Description: "A cursor for use in pagination. `starting_after` is an object ID that defines your place in the list. For instance, if you make a list request and receive 100 objects, ending with `obj_foo`, your subsequent call can include `starting_after=obj_foo` in order to fetch the next page of the list.",
		},
		"created": {
			Type:        "integer",
			Description: "Only return connected accounts that were created during the given date interval.",
		},
		"ending_before": {
			Type:        "string",
			Description: "A cursor for use in pagination. `ending_before` is an object ID that defines your place in the list. For instance, if you make a list request and receive 100 objects, starting with `obj_bar`, your subsequent call can include `ending_before=obj_bar` in order to fetch the previous page of the list.",
		},
	},
}

var V1AccountsCapabilities = resource.OperationSpec{
	Name:    "capabilities",
	Path:    "/v1/accounts/{account}/capabilities",
	Method:  "GET",
	Summary: "List all account capabilities",
}

var V1AccountsPersons = resource.OperationSpec{
	Name:    "persons",
	Path:    "/v1/accounts/{account}/persons",
	Method:  "GET",
	Summary: "List all persons",
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

var V1AccountsCreate = resource.OperationSpec{
	Name:   "create",
	Path:   "/v1/accounts",
	Method: "POST",
	Params: map[string]*resource.ParamSpec{
		"company.address_kana.line2": {
			Type:        "string",
			Description: "Building details.",
		},
		"company.ownership_exemption_reason": {
			Type:        "string",
			Description: "This value is used to determine if a business is exempt from providing ultimate beneficial owners. See [this support article](https://support.stripe.com/questions/exemption-from-providing-ownership-details) and [changelog](https://docs.stripe.com/changelog/acacia/2025-01-27/ownership-exemption-reason-accounts-api) for more details.",
			Enum: []resource.EnumSpec{
				{Value: "qualified_entity_exceeds_ownership_threshold"},
				{Value: "qualifies_as_financial_institution"},
			},
		},
		"individual.address_kanji.line2": {
			Type:        "string",
			Description: "Building details.",
		},
		"company.registration_number": {
			Type:        "string",
			Description: "The identification number given to a company when it is registered or incorporated, if distinct from the identification number used for filing taxes. (Examples are the CIN for companies and LLP IN for partnerships in India, and the Company Registration Number in Hong Kong).",
		},
		"individual.first_name_kanji": {
			Type:        "string",
			Description: "The Kanji variation of the individual's first name (Japan only).",
		},
		"individual.phone": {
			Type:        "string",
			Description: "The individual's phone number.",
		},
		"settings.payments.statement_descriptor_kana": {
			Type:        "string",
			Description: "The Kana variation of `statement_descriptor` used for charges in Japan. Japanese statement descriptors have [special requirements](https://docs.stripe.com/get-started/account/statement-descriptors#set-japanese-statement-descriptors).",
		},
		"company.address_kanji.city": {
			Type:        "string",
			Description: "City or ward.",
		},
		"settings.branding.secondary_color": {
			Type:        "string",
			Description: "A CSS hex color value representing the secondary branding color for this account.",
		},
		"company.ownership_declaration.date": {
			Type:        "integer",
			Description: "The Unix timestamp marking when the beneficial owner attestation was made.",
			Format:      "unix-time",
		},
		"capabilities.boleto_payments.requested": {
			Type:        "boolean",
			Description: "Passing true requests the capability for the account, if it is not already requested. A requested capability may not immediately become active. Any requirements to activate the capability are returned in the `requirements` arrays.",
		},
		"capabilities.blik_payments.requested": {
			Type:        "boolean",
			Description: "Passing true requests the capability for the account, if it is not already requested. A requested capability may not immediately become active. Any requirements to activate the capability are returned in the `requirements` arrays.",
		},
		"documents.bank_account_ownership_verification.files": {
			Type:        "array",
			Description: "One or more document ids returned by a [file upload](https://api.stripe.com#create_file) with a `purpose` value of `account_requirement`.",
		},
		"individual.address_kana.state": {
			Type:        "string",
			Description: "Prefecture.",
		},
		"capabilities.mx_bank_transfer_payments.requested": {
			Type:        "boolean",
			Description: "Passing true requests the capability for the account, if it is not already requested. A requested capability may not immediately become active. Any requirements to activate the capability are returned in the `requirements` arrays.",
		},
		"individual.political_exposure": {
			Type:        "string",
			Description: "Indicates if the person or any of their representatives, family members, or other closely related persons, declares that they hold or have held an important public job or function, in any jurisdiction.",
			Enum: []resource.EnumSpec{
				{Value: "existing"},
				{Value: "none"},
			},
		},
		"company.directorship_declaration.user_agent": {
			Type:        "string",
			Description: "The user agent of the browser from which the directorship declaration attestation was made.",
		},
		"settings.payouts.statement_descriptor": {
			Type:        "string",
			Description: "The text that appears on the bank account statement for payouts. If not set, this defaults to the platform's bank descriptor as set in the Dashboard.",
		},
		"company.name": {
			Type:        "string",
			Description: "The company's legal name.",
		},
		"company.address_kana.city": {
			Type:        "string",
			Description: "City or ward.",
		},
		"business_profile.annual_revenue.currency": {
			Type:        "string",
			Description: "Three-letter [ISO currency code](https://www.iso.org/iso-4217-currency-codes.html), in lowercase. Must be a [supported currency](https://stripe.com/docs/currencies).",
			Required:    true,
			Format:      "currency",
		},
		"documents.company_memorandum_of_association.files": {
			Type:        "array",
			Description: "One or more document ids returned by a [file upload](https://api.stripe.com#create_file) with a `purpose` value of `account_requirement`.",
		},
		"settings.payouts.debit_negative_balances": {
			Type:        "boolean",
			Description: "A Boolean indicating whether Stripe should try to reclaim negative balances from an attached bank account. For details, see [Understanding Connect Account Balances](/connect/account-balances).",
		},
		"company.export_license_id": {
			Type:        "string",
			Description: "The export license ID number of the company, also referred as Import Export Code (India only).",
		},
		"capabilities.payto_payments.requested": {
			Type:        "boolean",
			Description: "Passing true requests the capability for the account, if it is not already requested. A requested capability may not immediately become active. Any requirements to activate the capability are returned in the `requirements` arrays.",
		},
		"individual.address.city": {
			Type:        "string",
			Description: "City, district, suburb, town, or village.",
		},
		"settings.bacs_debit_payments.display_name": {
			Type:        "string",
			Description: "The Bacs Direct Debit Display Name for this account. For payments made with Bacs Direct Debit, this name appears on the mandate as the statement descriptor. Mobile banking apps display it as the name of the business. To use custom branding, set the Bacs Direct Debit Display Name during or right after creation. Custom branding incurs an additional monthly fee for the platform. If you don't set the display name before requesting Bacs capability, it's automatically set as \"Stripe\" and the account is onboarded to Stripe branding, which is free.",
		},
		"capabilities.us_bank_transfer_payments.requested": {
			Type:        "boolean",
			Description: "Passing true requests the capability for the account, if it is not already requested. A requested capability may not immediately become active. Any requirements to activate the capability are returned in the `requirements` arrays.",
		},
		"documents.proof_of_address.files": {
			Type:        "array",
			Description: "One or more document ids returned by a [file upload](https://api.stripe.com#create_file) with a `purpose` value of `account_requirement`.",
		},
		"capabilities.bank_transfer_payments.requested": {
			Type:        "boolean",
			Description: "Passing true requests the capability for the account, if it is not already requested. A requested capability may not immediately become active. Any requirements to activate the capability are returned in the `requirements` arrays.",
		},
		"capabilities.cashapp_payments.requested": {
			Type:        "boolean",
			Description: "Passing true requests the capability for the account, if it is not already requested. A requested capability may not immediately become active. Any requirements to activate the capability are returned in the `requirements` arrays.",
		},
		"capabilities.acss_debit_payments.requested": {
			Type:        "boolean",
			Description: "Passing true requests the capability for the account, if it is not already requested. A requested capability may not immediately become active. Any requirements to activate the capability are returned in the `requirements` arrays.",
		},
		"documents.proof_of_ultimate_beneficial_ownership.signer.person": {
			Type:        "string",
			Description: "The token of the person signing the document, if applicable.",
		},
		"individual.verification.additional_document.back": {
			Type:        "string",
			Description: "The back of an ID returned by a [file upload](https://api.stripe.com#create_file) with a `purpose` value of `identity_document`. The uploaded file needs to be a color image (smaller than 8,000px by 8,000px), in JPG, PNG, or PDF format, and less than 10 MB in size.",
		},
		"settings.branding.icon": {
			Type:        "string",
			Description: "(ID of a [file upload](https://stripe.com/docs/guides/file-upload)) An icon for the account. Must be square and at least 128px x 128px.",
		},
		"business_profile.annual_revenue.amount": {
			Type:        "integer",
			Description: "A non-negative integer representing the amount in the [smallest currency unit](/currencies#zero-decimal).",
			Required:    true,
		},
		"individual.address_kanji.line1": {
			Type:        "string",
			Description: "Block or building number.",
		},
		"settings.treasury.tos_acceptance.date": {
			Type:        "integer",
			Description: "The Unix timestamp marking when the account representative accepted the service agreement.",
			Format:      "unix-time",
		},
		"type": {
			Type:        "string",
			Description: "The type of Stripe account to create. May be one of `custom`, `express` or `standard`.",
			Enum: []resource.EnumSpec{
				{Value: "custom"},
				{Value: "express"},
				{Value: "standard"},
			},
		},
		"email": {
			Type:        "string",
			Description: "The email address of the account holder. This is only to make the account easier to identify to you. If [controller.requirement_collection](/api/accounts/object#account_object-controller-requirement_collection) is `application`, which includes Custom accounts, Stripe doesn't email the account without your consent.",
		},
		"individual.address_kana.town": {
			Type:        "string",
			Description: "Town or cho-me.",
		},
		"settings.card_payments.decline_on.avs_failure": {
			Type:        "boolean",
			Description: "Whether Stripe automatically declines charges with an incorrect ZIP or postal code. This setting only applies when a ZIP or postal code is provided and they fail bank verification.",
		},
		"controller.requirement_collection": {
			Type:        "string",
			Description: "A value indicating responsibility for collecting updated information when requirements on the account are due or change. Defaults to `stripe`.",
			Enum: []resource.EnumSpec{
				{Value: "application"},
				{Value: "stripe"},
			},
		},
		"capabilities.card_issuing.requested": {
			Type:        "boolean",
			Description: "Passing true requests the capability for the account, if it is not already requested. A requested capability may not immediately become active. Any requirements to activate the capability are returned in the `requirements` arrays.",
		},
		"capabilities.kr_card_payments.requested": {
			Type:        "boolean",
			Description: "Passing true requests the capability for the account, if it is not already requested. A requested capability may not immediately become active. Any requirements to activate the capability are returned in the `requirements` arrays.",
		},
		"capabilities.transfers.requested": {
			Type:        "boolean",
			Description: "Passing true requests the capability for the account, if it is not already requested. A requested capability may not immediately become active. Any requirements to activate the capability are returned in the `requirements` arrays.",
		},
		"tos_acceptance.ip": {
			Type:        "string",
			Description: "The IP address from which the account representative accepted their service agreement.",
		},
		"company.directorship_declaration.ip": {
			Type:        "string",
			Description: "The IP address from which the directorship declaration attestation was made.",
		},
		"company.address_kanji.state": {
			Type:        "string",
			Description: "Prefecture.",
		},
		"settings.card_payments.statement_descriptor_prefix_kanji": {
			Type:        "string",
			Description: "The Kanji variation of the default text that appears on credit card statements when a charge is made (Japan only). This field prefixes any dynamic `statement_descriptor_suffix_kanji` specified on the charge. `statement_descriptor_prefix_kanji` is useful for maximizing descriptor space for the dynamic portion.",
		},
		"capabilities.mobilepay_payments.requested": {
			Type:        "boolean",
			Description: "Passing true requests the capability for the account, if it is not already requested. A requested capability may not immediately become active. Any requirements to activate the capability are returned in the `requirements` arrays.",
		},
		"capabilities.satispay_payments.requested": {
			Type:        "boolean",
			Description: "Passing true requests the capability for the account, if it is not already requested. A requested capability may not immediately become active. Any requirements to activate the capability are returned in the `requirements` arrays.",
		},
		"capabilities.crypto_payments.requested": {
			Type:        "boolean",
			Description: "Passing true requests the capability for the account, if it is not already requested. A requested capability may not immediately become active. Any requirements to activate the capability are returned in the `requirements` arrays.",
		},
		"individual.gender": {
			Type:        "string",
			Description: "The individual's gender",
		},
		"company.name_kana": {
			Type:        "string",
			Description: "The Kana variation of the company's legal name (Japan only).",
		},
		"business_profile.monthly_estimated_revenue.currency": {
			Type:        "string",
			Description: "Three-letter [ISO currency code](https://www.iso.org/iso-4217-currency-codes.html), in lowercase. Must be a [supported currency](https://stripe.com/docs/currencies).",
			Required:    true,
			Format:      "currency",
		},
		"individual.address_kanji.state": {
			Type:        "string",
			Description: "Prefecture.",
		},
		"controller.losses.payments": {
			Type:        "string",
			Description: "A value indicating who is liable when this account can't pay back negative balances resulting from payments. Defaults to `stripe`.",
			Enum: []resource.EnumSpec{
				{Value: "application"},
				{Value: "stripe"},
			},
		},
		"external_account": {
			Type:        "string",
			Description: "A card or bank account to attach to the account for receiving [payouts](/connect/bank-debit-card-payouts) (you won’t be able to use it for top-ups). You can provide either a token, like the ones returned by [Stripe.js](/js), or a dictionary, as documented in the `external_account` parameter for [bank account](/api#account_create_bank_account) creation. <br><br>By default, providing an external account sets it as the new default external account for its currency, and deletes the old default if one exists. To add additional external accounts without replacing the existing default for the currency, use the [bank account](/api#account_create_bank_account) or [card creation](/api#account_create_card) APIs. After you create an [Account Link](/api/account_links) or [Account Session](/api/account_sessions), this property can only be updated for accounts where [controller.requirement_collection](/api/accounts/object#account_object-controller-requirement_collection) is `application`, which includes Custom accounts.",
		},
		"individual.address.line1": {
			Type:        "string",
			Description: "Address line 1, such as the street, PO Box, or company name.",
		},
		"settings.card_payments.statement_descriptor_prefix": {
			Type:        "string",
			Description: "The default text that appears on credit card statements when a charge is made. This field prefixes any dynamic `statement_descriptor` specified on the charge. `statement_descriptor_prefix` is useful for maximizing descriptor space for the dynamic portion.",
		},
		"business_profile.support_address.city": {
			Type:        "string",
			Description: "City, district, suburb, town, or village.",
		},
		"individual.address_kanji.town": {
			Type:        "string",
			Description: "Town or cho-me.",
		},
		"settings.payments.statement_descriptor": {
			Type:        "string",
			Description: "The default text that appears on statements for non-card charges outside of Japan. For card charges, if you don't set a `statement_descriptor_prefix`, this text is also used as the statement descriptor prefix. In that case, if concatenating the statement descriptor suffix causes the combined statement descriptor to exceed 22 characters, we truncate the `statement_descriptor` text to limit the full descriptor to 22 characters. For more information about statement descriptors and their requirements, see the [account settings documentation](https://docs.stripe.com/get-started/account/statement-descriptors).",
		},
		"company.ownership_declaration.ip": {
			Type:        "string",
			Description: "The IP address from which the beneficial owner attestation was made.",
		},
		"capabilities.p24_payments.requested": {
			Type:        "boolean",
			Description: "Passing true requests the capability for the account, if it is not already requested. A requested capability may not immediately become active. Any requirements to activate the capability are returned in the `requirements` arrays.",
		},
		"documents.company_license.files": {
			Type:        "array",
			Description: "One or more document ids returned by a [file upload](https://api.stripe.com#create_file) with a `purpose` value of `account_requirement`.",
		},
		"individual.address.line2": {
			Type:        "string",
			Description: "Address line 2, such as the apartment, suite, unit, or building.",
		},
		"settings.payouts.schedule.weekly_anchor": {
			Type:        "string",
			Description: "The day of the week when available funds are paid out, specified as `monday`, `tuesday`, etc. Required and applicable only if `interval` is `weekly`.",
			Enum: []resource.EnumSpec{
				{Value: "friday"},
				{Value: "monday"},
				{Value: "saturday"},
				{Value: "sunday"},
				{Value: "thursday"},
				{Value: "tuesday"},
				{Value: "wednesday"},
			},
		},
		"company.address.country": {
			Type:        "string",
			Description: "Two-letter country code ([ISO 3166-1 alpha-2](https://en.wikipedia.org/wiki/ISO_3166-1_alpha-2)).",
		},
		"controller.fees.payer": {
			Type:        "string",
			Description: "A value indicating the responsible payer of Stripe fees on this account. Defaults to `account`. Learn more about [fee behavior on connected accounts](https://docs.stripe.com/connect/direct-charges-fee-payer-behavior).",
			Enum: []resource.EnumSpec{
				{Value: "account"},
				{Value: "application"},
			},
		},
		"settings.payouts.schedule.monthly_anchor": {
			Type:        "integer",
			Description: "The day of the month when available funds are paid out, specified as a number between 1--31. Payouts nominally scheduled between the 29th and 31st of the month are instead sent on the last day of a shorter month. Required and applicable only if `interval` is `monthly`.",
		},
		"settings.treasury.tos_acceptance.ip": {
			Type:        "string",
			Description: "The IP address from which the account representative accepted the service agreement.",
		},
		"company.verification.document.front": {
			Type:        "string",
			Description: "The front of a document returned by a [file upload](https://api.stripe.com#create_file) with a `purpose` value of `additional_verification`. The uploaded file needs to be a color image (smaller than 8,000px by 8,000px), in JPG, PNG, or PDF format, and less than 10 MB in size.",
		},
		"company.address_kana.town": {
			Type:        "string",
			Description: "Town or cho-me.",
		},
		"company.name_kanji": {
			Type:        "string",
			Description: "The Kanji variation of the company's legal name (Japan only).",
		},
		"business_profile.product_description": {
			Type:        "string",
			Description: "Internal-only description of the product sold by, or service provided by, the business. Used by Stripe for risk and underwriting purposes.",
		},
		"business_profile.support_address.line2": {
			Type:        "string",
			Description: "Address line 2, such as the apartment, suite, unit, or building.",
		},
		"business_profile.support_url": {
			Type:        "string",
			Description: "A publicly available website for handling support issues.",
		},
		"business_profile.annual_revenue.fiscal_year_end": {
			Type:        "string",
			Description: "The close-out date of the preceding fiscal year in ISO 8601 format. E.g. 2023-12-31 for the 31st of December, 2023.",
			Required:    true,
		},
		"business_profile.monthly_estimated_revenue.amount": {
			Type:        "integer",
			Description: "A non-negative integer representing how much to charge in the [smallest currency unit](/currencies#zero-decimal).",
			Required:    true,
		},
		"business_profile.support_address.postal_code": {
			Type:        "string",
			Description: "ZIP or postal code.",
		},
		"business_profile.support_email": {
			Type:        "string",
			Description: "A publicly available email address for sending support issues to.",
		},
		"individual.registered_address.city": {
			Type:        "string",
			Description: "City, district, suburb, town, or village.",
		},
		"company.address_kana.country": {
			Type:        "string",
			Description: "Two-letter country code ([ISO 3166-1 alpha-2](https://en.wikipedia.org/wiki/ISO_3166-1_alpha-2)).",
		},
		"capabilities.grabpay_payments.requested": {
			Type:        "boolean",
			Description: "Passing true requests the capability for the account, if it is not already requested. A requested capability may not immediately become active. Any requirements to activate the capability are returned in the `requirements` arrays.",
		},
		"individual.verification.additional_document.front": {
			Type:        "string",
			Description: "The front of an ID returned by a [file upload](https://api.stripe.com#create_file) with a `purpose` value of `identity_document`. The uploaded file needs to be a color image (smaller than 8,000px by 8,000px), in JPG, PNG, or PDF format, and less than 10 MB in size.",
		},
		"individual.first_name_kana": {
			Type:        "string",
			Description: "The Kana variation of the individual's first name (Japan only).",
		},
		"capabilities.payco_payments.requested": {
			Type:        "boolean",
			Description: "Passing true requests the capability for the account, if it is not already requested. A requested capability may not immediately become active. Any requirements to activate the capability are returned in the `requirements` arrays.",
		},
		"documents.company_tax_id_verification.files": {
			Type:        "array",
			Description: "One or more document ids returned by a [file upload](https://api.stripe.com#create_file) with a `purpose` value of `account_requirement`.",
		},
		"company.address_kanji.line1": {
			Type:        "string",
			Description: "Block or building number.",
		},
		"business_profile.mcc": {
			Type:        "string",
			Description: "[The merchant category code for the account](/connect/setting-mcc). MCCs are used to classify businesses based on the goods or services they provide.",
		},
		"capabilities.naver_pay_payments.requested": {
			Type:        "boolean",
			Description: "Passing true requests the capability for the account, if it is not already requested. A requested capability may not immediately become active. Any requirements to activate the capability are returned in the `requirements` arrays.",
		},
		"capabilities.sofort_payments.requested": {
			Type:        "boolean",
			Description: "Passing true requests the capability for the account, if it is not already requested. A requested capability may not immediately become active. Any requirements to activate the capability are returned in the `requirements` arrays.",
		},
		"individual.address.postal_code": {
			Type:        "string",
			Description: "ZIP or postal code.",
		},
		"company.address.state": {
			Type:        "string",
			Description: "State, county, province, or region ([ISO 3166-2](https://en.wikipedia.org/wiki/ISO_3166-2)).",
		},
		"business_profile.name": {
			Type:        "string",
			Description: "The customer-facing business name.",
		},
		"documents.company_ministerial_decree.files": {
			Type:        "array",
			Description: "One or more document ids returned by a [file upload](https://api.stripe.com#create_file) with a `purpose` value of `account_requirement`.",
		},
		"individual.address_kanji.postal_code": {
			Type:        "string",
			Description: "Postal code.",
		},
		"settings.invoices.hosted_payment_method_save": {
			Type:        "string",
			Description: "Whether to save the payment method after a payment is completed for a one-time invoice or a subscription invoice when the customer already has a default payment method on the hosted invoice page.",
			Enum: []resource.EnumSpec{
				{Value: "always"},
				{Value: "never"},
				{Value: "offer"},
			},
		},
		"business_profile.support_address.line1": {
			Type:        "string",
			Description: "Address line 1, such as the street, PO Box, or company name.",
		},
		"individual.maiden_name": {
			Type:        "string",
			Description: "The individual's maiden name.",
		},
		"company.address.postal_code": {
			Type:        "string",
			Description: "ZIP or postal code.",
		},
		"capabilities.amazon_pay_payments.requested": {
			Type:        "boolean",
			Description: "Passing true requests the capability for the account, if it is not already requested. A requested capability may not immediately become active. Any requirements to activate the capability are returned in the `requirements` arrays.",
		},
		"capabilities.ideal_payments.requested": {
			Type:        "boolean",
			Description: "Passing true requests the capability for the account, if it is not already requested. A requested capability may not immediately become active. Any requirements to activate the capability are returned in the `requirements` arrays.",
		},
		"individual.registered_address.line1": {
			Type:        "string",
			Description: "Address line 1, such as the street, PO Box, or company name.",
		},
		"individual.relationship.executive": {
			Type:        "boolean",
			Description: "Whether the person has significant responsibility to control, manage, or direct the organization.",
		},
		"settings.treasury.tos_acceptance.user_agent": {
			Type:        "string",
			Description: "The user agent of the browser from which the account representative accepted the service agreement.",
		},
		"capabilities.bacs_debit_payments.requested": {
			Type:        "boolean",
			Description: "Passing true requests the capability for the account, if it is not already requested. A requested capability may not immediately become active. Any requirements to activate the capability are returned in the `requirements` arrays.",
		},
		"capabilities.india_international_payments.requested": {
			Type:        "boolean",
			Description: "Passing true requests the capability for the account, if it is not already requested. A requested capability may not immediately become active. Any requirements to activate the capability are returned in the `requirements` arrays.",
		},
		"capabilities.mb_way_payments.requested": {
			Type:        "boolean",
			Description: "Passing true requests the capability for the account, if it is not already requested. A requested capability may not immediately become active. Any requirements to activate the capability are returned in the `requirements` arrays.",
		},
		"individual.address.state": {
			Type:        "string",
			Description: "State, county, province, or region ([ISO 3166-2](https://en.wikipedia.org/wiki/ISO_3166-2)).",
		},
		"controller.stripe_dashboard.type": {
			Type:        "string",
			Description: "Whether this account should have access to the full Stripe Dashboard (`full`), to the Express Dashboard (`express`), or to no Stripe-hosted dashboard (`none`). Defaults to `full`.",
			Enum: []resource.EnumSpec{
				{Value: "express"},
				{Value: "full"},
				{Value: "none"},
			},
		},
		"capabilities.link_payments.requested": {
			Type:        "boolean",
			Description: "Passing true requests the capability for the account, if it is not already requested. A requested capability may not immediately become active. Any requirements to activate the capability are returned in the `requirements` arrays.",
		},
		"settings.branding.primary_color": {
			Type:        "string",
			Description: "A CSS hex color value representing the primary branding color for this account.",
		},
		"tos_acceptance.service_agreement": {
			Type:        "string",
			Description: "The user's service agreement type.",
		},
		"capabilities.revolut_pay_payments.requested": {
			Type:        "boolean",
			Description: "Passing true requests the capability for the account, if it is not already requested. A requested capability may not immediately become active. Any requirements to activate the capability are returned in the `requirements` arrays.",
		},
		"capabilities.alma_payments.requested": {
			Type:        "boolean",
			Description: "Passing true requests the capability for the account, if it is not already requested. A requested capability may not immediately become active. Any requirements to activate the capability are returned in the `requirements` arrays.",
		},
		"documents.proof_of_registration.files": {
			Type:        "array",
			Description: "One or more document ids returned by a [file upload](https://api.stripe.com#create_file) with a `purpose` value of `account_requirement`.",
		},
		"documents.proof_of_registration.signer.person": {
			Type:        "string",
			Description: "The token of the person signing the document, if applicable.",
		},
		"settings.card_payments.decline_on.cvc_failure": {
			Type:        "boolean",
			Description: "Whether Stripe automatically declines charges with an incorrect CVC. This setting only applies when a CVC is provided and it fails bank verification.",
		},
		"company.directors_provided": {
			Type:        "boolean",
			Description: "Whether the company's directors have been provided. Set this Boolean to `true` after creating all the company's directors with [the Persons API](/api/persons) for accounts with a `relationship.director` requirement. This value is not automatically set to `true` after creating directors, so it needs to be updated to indicate all directors have been provided.",
		},
		"company.tax_id_registrar": {
			Type:        "string",
			Description: "The jurisdiction in which the `tax_id` is registered (Germany-based companies only).",
		},
		"business_profile.minority_owned_business_designation": {
			Type:        "array",
			Description: "Whether the business is a minority-owned, women-owned, and/or LGBTQI+ -owned business.",
		},
		"capabilities.billie_payments.requested": {
			Type:        "boolean",
			Description: "Passing true requests the capability for the account, if it is not already requested. A requested capability may not immediately become active. Any requirements to activate the capability are returned in the `requirements` arrays.",
		},
		"company.address_kana.line1": {
			Type:        "string",
			Description: "Block or building number.",
		},
		"company.address_kanji.town": {
			Type:        "string",
			Description: "Town or cho-me.",
		},
		"individual.address_kanji.city": {
			Type:        "string",
			Description: "City or ward.",
		},
		"capabilities.swish_payments.requested": {
			Type:        "boolean",
			Description: "Passing true requests the capability for the account, if it is not already requested. A requested capability may not immediately become active. Any requirements to activate the capability are returned in the `requirements` arrays.",
		},
		"country": {
			Type:        "string",
			Description: "The country in which the account holder resides, or in which the business is legally established. This should be an ISO 3166-1 alpha-2 country code. For example, if you are in the United States and the business for which you're creating an account is legally represented in Canada, you would use `CA` as the country for the account being created. Available countries include [Stripe's global markets](https://stripe.com/global) as well as countries where [cross-border payouts](https://stripe.com/docs/connect/cross-border-payouts) are supported.",
		},
		"individual.registered_address.postal_code": {
			Type:        "string",
			Description: "ZIP or postal code.",
		},
		"capabilities.pay_by_bank_payments.requested": {
			Type:        "boolean",
			Description: "Passing true requests the capability for the account, if it is not already requested. A requested capability may not immediately become active. Any requirements to activate the capability are returned in the `requirements` arrays.",
		},
		"individual.full_name_aliases": {
			Type:        "array",
			Description: "A list of alternate names or aliases that the individual is known by.",
		},
		"capabilities.konbini_payments.requested": {
			Type:        "boolean",
			Description: "Passing true requests the capability for the account, if it is not already requested. A requested capability may not immediately become active. Any requirements to activate the capability are returned in the `requirements` arrays.",
		},
		"business_profile.estimated_worker_count": {
			Type:        "integer",
			Description: "An estimated upper bound of employees, contractors, vendors, etc. currently working for the business.",
		},
		"documents.proof_of_ultimate_beneficial_ownership.files": {
			Type:        "array",
			Description: "One or more document ids returned by a [file upload](https://api.stripe.com#create_file) with a `purpose` value of `account_requirement`.",
		},
		"individual.registered_address.line2": {
			Type:        "string",
			Description: "Address line 2, such as the apartment, suite, unit, or building.",
		},
		"settings.card_issuing.tos_acceptance.user_agent": {
			Type:        "string",
			Description: "The user agent of the browser from which the account representative accepted the service agreement.",
		},
		"settings.branding.logo": {
			Type:        "string",
			Description: "(ID of a [file upload](https://stripe.com/docs/guides/file-upload)) A logo for the account that will be used in Checkout instead of the icon and without the account's name next to it if provided. Must be at least 128px x 128px.",
		},
		"business_profile.support_address.country": {
			Type:        "string",
			Description: "Two-letter country code ([ISO 3166-1 alpha-2](https://en.wikipedia.org/wiki/ISO_3166-1_alpha-2)).",
		},
		"capabilities.treasury.requested": {
			Type:        "boolean",
			Description: "Passing true requests the capability for the account, if it is not already requested. A requested capability may not immediately become active. Any requirements to activate the capability are returned in the `requirements` arrays.",
		},
		"individual.relationship.percent_ownership": {
			Type:        "number",
			Description: "The percent owned by the person of the account's legal entity.",
		},
		"company.representative_declaration.ip": {
			Type:        "string",
			Description: "The IP address from which the representative declaration attestation was made.",
		},
		"business_type": {
			Type:        "string",
			Description: "The business type. Once you create an [Account Link](/api/account_links) or [Account Session](/api/account_sessions), this property can only be updated for accounts where [controller.requirement_collection](/api/accounts/object#account_object-controller-requirement_collection) is `application`, which includes Custom accounts.",
			Enum: []resource.EnumSpec{
				{Value: "company"},
				{Value: "government_entity"},
				{Value: "individual"},
				{Value: "non_profit"},
			},
		},
		"capabilities.eps_payments.requested": {
			Type:        "boolean",
			Description: "Passing true requests the capability for the account, if it is not already requested. A requested capability may not immediately become active. Any requirements to activate the capability are returned in the `requirements` arrays.",
		},
		"individual.address_kanji.country": {
			Type:        "string",
			Description: "Two-letter country code ([ISO 3166-1 alpha-2](https://en.wikipedia.org/wiki/ISO_3166-1_alpha-2)).",
		},
		"company.representative_declaration.user_agent": {
			Type:        "string",
			Description: "The user agent of the browser from which the representative declaration attestation was made.",
		},
		"capabilities.jp_bank_transfer_payments.requested": {
			Type:        "boolean",
			Description: "Passing true requests the capability for the account, if it is not already requested. A requested capability may not immediately become active. Any requirements to activate the capability are returned in the `requirements` arrays.",
		},
		"capabilities.us_bank_account_ach_payments.requested": {
			Type:        "boolean",
			Description: "Passing true requests the capability for the account, if it is not already requested. A requested capability may not immediately become active. Any requirements to activate the capability are returned in the `requirements` arrays.",
		},
		"individual.address_kana.postal_code": {
			Type:        "string",
			Description: "Postal code.",
		},
		"settings.payments.statement_descriptor_kanji": {
			Type:        "string",
			Description: "The Kanji variation of `statement_descriptor` used for charges in Japan. Japanese statement descriptors have [special requirements](https://docs.stripe.com/get-started/account/statement-descriptors#set-japanese-statement-descriptors).",
		},
		"documents.company_registration_verification.files": {
			Type:        "array",
			Description: "One or more document ids returned by a [file upload](https://api.stripe.com#create_file) with a `purpose` value of `account_requirement`.",
		},
		"capabilities.cartes_bancaires_payments.requested": {
			Type:        "boolean",
			Description: "Passing true requests the capability for the account, if it is not already requested. A requested capability may not immediately become active. Any requirements to activate the capability are returned in the `requirements` arrays.",
		},
		"capabilities.tax_reporting_us_1099_k.requested": {
			Type:        "boolean",
			Description: "Passing true requests the capability for the account, if it is not already requested. A requested capability may not immediately become active. Any requirements to activate the capability are returned in the `requirements` arrays.",
		},
		"individual.address_kana.country": {
			Type:        "string",
			Description: "Two-letter country code ([ISO 3166-1 alpha-2](https://en.wikipedia.org/wiki/ISO_3166-1_alpha-2)).",
		},
		"individual.address_kana.line2": {
			Type:        "string",
			Description: "Building details.",
		},
		"company.vat_id": {
			Type:        "string",
			Description: "The VAT number of the company.",
		},
		"capabilities.jcb_payments.requested": {
			Type:        "boolean",
			Description: "Passing true requests the capability for the account, if it is not already requested. A requested capability may not immediately become active. Any requirements to activate the capability are returned in the `requirements` arrays.",
		},
		"individual.last_name_kana": {
			Type:        "string",
			Description: "The Kana variation of the individual's last name (Japan only).",
		},
		"company.structure": {
			Type:        "string",
			Description: "The category identifying the legal structure of the company or legal entity. See [Business structure](/connect/identity-verification#business-structure) for more details. Pass an empty string to unset this value.",
			Enum: []resource.EnumSpec{
				{Value: "free_zone_establishment"},
				{Value: "free_zone_llc"},
				{Value: "government_instrumentality"},
				{Value: "governmental_unit"},
				{Value: "incorporated_non_profit"},
				{Value: "incorporated_partnership"},
				{Value: "limited_liability_partnership"},
				{Value: "llc"},
				{Value: "multi_member_llc"},
				{Value: "private_company"},
				{Value: "private_corporation"},
				{Value: "private_partnership"},
				{Value: "public_company"},
				{Value: "public_corporation"},
				{Value: "public_partnership"},
				{Value: "registered_charity"},
				{Value: "single_member_llc"},
				{Value: "sole_establishment"},
				{Value: "sole_proprietorship"},
				{Value: "tax_exempt_government_instrumentality"},
				{Value: "unincorporated_association"},
				{Value: "unincorporated_non_profit"},
				{Value: "unincorporated_partnership"},
			},
		},
		"individual.relationship.director": {
			Type:        "boolean",
			Description: "Whether the person is a director of the account's legal entity. Directors are typically members of the governing board of the company, or responsible for ensuring the company meets its regulatory obligations.",
		},
		"individual.relationship.title": {
			Type:        "string",
			Description: "The person's title (e.g., CEO, Support Engineer).",
		},
		"company.tax_id": {
			Type:        "string",
			Description: "The business ID number of the company, as appropriate for the company’s country. (Examples are an Employer ID Number in the U.S., a Business Number in Canada, or a Company Number in the UK.)",
		},
		"capabilities.bancontact_payments.requested": {
			Type:        "boolean",
			Description: "Passing true requests the capability for the account, if it is not already requested. A requested capability may not immediately become active. Any requirements to activate the capability are returned in the `requirements` arrays.",
		},
		"company.phone": {
			Type:        "string",
			Description: "The company's phone number (used for verification).",
		},
		"tos_acceptance.date": {
			Type:        "integer",
			Description: "The Unix timestamp marking when the account representative accepted their service agreement.",
			Format:      "unix-time",
		},
		"capabilities.legacy_payments.requested": {
			Type:        "boolean",
			Description: "Passing true requests the capability for the account, if it is not already requested. A requested capability may not immediately become active. Any requirements to activate the capability are returned in the `requirements` arrays.",
		},
		"individual.relationship.owner": {
			Type:        "boolean",
			Description: "Whether the person is an owner of the account’s legal entity.",
		},
		"settings.payouts.schedule.monthly_payout_days": {
			Type:        "array",
			Description: "The days of the month when available funds are paid out, specified as an array of numbers between 1--31. Payouts nominally scheduled between the 29th and 31st of the month are instead sent on the last day of a shorter month. Required and applicable only if `interval` is `monthly` and `monthly_anchor` is not set.",
		},
		"settings.payouts.schedule.weekly_payout_days": {
			Type:        "array",
			Description: "The days of the week when available funds are paid out, specified as an array, e.g., [`monday`, `tuesday`]. Required and applicable only if `interval` is `weekly`.",
		},
		"default_currency": {
			Type:        "string",
			Description: "Three-letter ISO currency code representing the default currency for the account. This must be a currency that [Stripe supports in the account's country](https://docs.stripe.com/payouts).",
			Format:      "currency",
		},
		"company.address.line1": {
			Type:        "string",
			Description: "Address line 1, such as the street, PO Box, or company name.",
		},
		"capabilities.klarna_payments.requested": {
			Type:        "boolean",
			Description: "Passing true requests the capability for the account, if it is not already requested. A requested capability may not immediately become active. Any requirements to activate the capability are returned in the `requirements` arrays.",
		},
		"individual.first_name": {
			Type:        "string",
			Description: "The individual's first name.",
		},
		"individual.id_number": {
			Type:        "string",
			Description: "The government-issued ID number of the individual, as appropriate for the representative's country. (Examples are a Social Security Number in the U.S., or a Social Insurance Number in Canada). Instead of the number itself, you can also provide a [PII token created with Stripe.js](/js/tokens/create_token?type=pii).",
		},
		"settings.card_payments.statement_descriptor_prefix_kana": {
			Type:        "string",
			Description: "The Kana variation of the default text that appears on credit card statements when a charge is made (Japan only). This field prefixes any dynamic `statement_descriptor_suffix_kana` specified on the charge. `statement_descriptor_prefix_kana` is useful for maximizing descriptor space for the dynamic portion.",
		},
		"company.address_kanji.postal_code": {
			Type:        "string",
			Description: "Postal code.",
		},
		"company.representative_declaration.date": {
			Type:        "integer",
			Description: "The Unix timestamp marking when the representative declaration attestation was made.",
			Format:      "unix-time",
		},
		"capabilities.promptpay_payments.requested": {
			Type:        "boolean",
			Description: "Passing true requests the capability for the account, if it is not already requested. A requested capability may not immediately become active. Any requirements to activate the capability are returned in the `requirements` arrays.",
		},
		"capabilities.kakao_pay_payments.requested": {
			Type:        "boolean",
			Description: "Passing true requests the capability for the account, if it is not already requested. A requested capability may not immediately become active. Any requirements to activate the capability are returned in the `requirements` arrays.",
		},
		"individual.ssn_last_4": {
			Type:        "string",
			Description: "The last four digits of the individual's Social Security Number (U.S. only).",
		},
		"capabilities.zip_payments.requested": {
			Type:        "boolean",
			Description: "Passing true requests the capability for the account, if it is not already requested. A requested capability may not immediately become active. Any requirements to activate the capability are returned in the `requirements` arrays.",
		},
		"individual.verification.document.back": {
			Type:        "string",
			Description: "The back of an ID returned by a [file upload](https://api.stripe.com#create_file) with a `purpose` value of `identity_document`. The uploaded file needs to be a color image (smaller than 8,000px by 8,000px), in JPG, PNG, or PDF format, and less than 10 MB in size.",
		},
		"individual.registered_address.state": {
			Type:        "string",
			Description: "State, county, province, or region ([ISO 3166-2](https://en.wikipedia.org/wiki/ISO_3166-2)).",
		},
		"individual.address_kana.line1": {
			Type:        "string",
			Description: "Block or building number.",
		},
		"individual.address_kana.city": {
			Type:        "string",
			Description: "City or ward.",
		},
		"settings.card_issuing.tos_acceptance.ip": {
			Type:        "string",
			Description: "The IP address from which the account representative accepted the service agreement.",
		},
		"capabilities.fpx_payments.requested": {
			Type:        "boolean",
			Description: "Passing true requests the capability for the account, if it is not already requested. A requested capability may not immediately become active. Any requirements to activate the capability are returned in the `requirements` arrays.",
		},
		"business_profile.support_phone": {
			Type:        "string",
			Description: "A publicly available phone number to call with support issues.",
		},
		"company.ownership_declaration.user_agent": {
			Type:        "string",
			Description: "The user agent of the browser from which the beneficial owner attestation was made.",
		},
		"company.address.city": {
			Type:        "string",
			Description: "City, district, suburb, town, or village.",
		},
		"account_token": {
			Type:        "string",
			Description: "An [account token](https://api.stripe.com#create_account_token), used to securely provide details to the account.",
		},
		"capabilities.samsung_pay_payments.requested": {
			Type:        "boolean",
			Description: "Passing true requests the capability for the account, if it is not already requested. A requested capability may not immediately become active. Any requirements to activate the capability are returned in the `requirements` arrays.",
		},
		"capabilities.multibanco_payments.requested": {
			Type:        "boolean",
			Description: "Passing true requests the capability for the account, if it is not already requested. A requested capability may not immediately become active. Any requirements to activate the capability are returned in the `requirements` arrays.",
		},
		"capabilities.paynow_payments.requested": {
			Type:        "boolean",
			Description: "Passing true requests the capability for the account, if it is not already requested. A requested capability may not immediately become active. Any requirements to activate the capability are returned in the `requirements` arrays.",
		},
		"company.export_purpose_code": {
			Type:        "string",
			Description: "The purpose code to use for export transactions (India only).",
		},
		"company.address_kanji.line2": {
			Type:        "string",
			Description: "Building details.",
		},
		"business_profile.support_address.state": {
			Type:        "string",
			Description: "State, county, province, or region ([ISO 3166-2](https://en.wikipedia.org/wiki/ISO_3166-2)).",
		},
		"capabilities.gb_bank_transfer_payments.requested": {
			Type:        "boolean",
			Description: "Passing true requests the capability for the account, if it is not already requested. A requested capability may not immediately become active. Any requirements to activate the capability are returned in the `requirements` arrays.",
		},
		"individual.email": {
			Type:        "string",
			Description: "The individual's email address.",
		},
		"company.address_kanji.country": {
			Type:        "string",
			Description: "Two-letter country code ([ISO 3166-1 alpha-2](https://en.wikipedia.org/wiki/ISO_3166-1_alpha-2)).",
		},
		"settings.payouts.schedule.delay_days": {
			Type:        "string",
			Description: "The number of days charge funds are held before being paid out. May also be set to `minimum`, representing the lowest available value for the account country. Default is `minimum`. The `delay_days` parameter remains at the last configured value if `interval` is `manual`. [Learn more about controlling payout delay days](/connect/manage-payout-schedule).",
		},
		"groups.payments_pricing": {
			Type:        "string",
			Description: "The group the account is in to determine their payments pricing, and null if the account is on customized pricing. [See the Platform pricing tool documentation](https://docs.stripe.com/connect/platform-pricing-tools) for details.",
		},
		"company.verification.document.back": {
			Type:        "string",
			Description: "The back of a document returned by a [file upload](https://api.stripe.com#create_file) with a `purpose` value of `additional_verification`. The uploaded file needs to be a color image (smaller than 8,000px by 8,000px), in JPG, PNG, or PDF format, and less than 10 MB in size.",
		},
		"capabilities.afterpay_clearpay_payments.requested": {
			Type:        "boolean",
			Description: "Passing true requests the capability for the account, if it is not already requested. A requested capability may not immediately become active. Any requirements to activate the capability are returned in the `requirements` arrays.",
		},
		"tos_acceptance.user_agent": {
			Type:        "string",
			Description: "The user agent of the browser from which the account representative accepted their service agreement.",
		},
		"company.address.line2": {
			Type:        "string",
			Description: "Address line 2, such as the apartment, suite, unit, or building.",
		},
		"capabilities.nz_bank_account_becs_debit_payments.requested": {
			Type:        "boolean",
			Description: "Passing true requests the capability for the account, if it is not already requested. A requested capability may not immediately become active. Any requirements to activate the capability are returned in the `requirements` arrays.",
		},
		"capabilities.sepa_debit_payments.requested": {
			Type:        "boolean",
			Description: "Passing true requests the capability for the account, if it is not already requested. A requested capability may not immediately become active. Any requirements to activate the capability are returned in the `requirements` arrays.",
		},
		"company.directorship_declaration.date": {
			Type:        "integer",
			Description: "The Unix timestamp marking when the directorship declaration attestation was made.",
			Format:      "unix-time",
		},
		"capabilities.oxxo_payments.requested": {
			Type:        "boolean",
			Description: "Passing true requests the capability for the account, if it is not already requested. A requested capability may not immediately become active. Any requirements to activate the capability are returned in the `requirements` arrays.",
		},
		"company.address_kana.state": {
			Type:        "string",
			Description: "Prefecture.",
		},
		"company.owners_provided": {
			Type:        "boolean",
			Description: "Whether the company's owners have been provided. Set this Boolean to `true` after creating all the company's owners with [the Persons API](/api/persons) for accounts with a `relationship.owner` requirement.",
		},
		"capabilities.affirm_payments.requested": {
			Type:        "boolean",
			Description: "Passing true requests the capability for the account, if it is not already requested. A requested capability may not immediately become active. Any requirements to activate the capability are returned in the `requirements` arrays.",
		},
		"capabilities.twint_payments.requested": {
			Type:        "boolean",
			Description: "Passing true requests the capability for the account, if it is not already requested. A requested capability may not immediately become active. Any requirements to activate the capability are returned in the `requirements` arrays.",
		},
		"individual.verification.document.front": {
			Type:        "string",
			Description: "The front of an ID returned by a [file upload](https://api.stripe.com#create_file) with a `purpose` value of `identity_document`. The uploaded file needs to be a color image (smaller than 8,000px by 8,000px), in JPG, PNG, or PDF format, and less than 10 MB in size.",
		},
		"individual.last_name": {
			Type:        "string",
			Description: "The individual's last name.",
		},
		"individual.last_name_kanji": {
			Type:        "string",
			Description: "The Kanji variation of the individual's last name (Japan only).",
		},
		"company.executives_provided": {
			Type:        "boolean",
			Description: "Whether the company's executives have been provided. Set this Boolean to `true` after creating all the company's executives with [the Persons API](/api/persons) for accounts with a `relationship.executive` requirement.",
		},
		"business_profile.url": {
			Type:        "string",
			Description: "The business's publicly available website.",
		},
		"capabilities.tax_reporting_us_1099_misc.requested": {
			Type:        "boolean",
			Description: "Passing true requests the capability for the account, if it is not already requested. A requested capability may not immediately become active. Any requirements to activate the capability are returned in the `requirements` arrays.",
		},
		"capabilities.card_payments.requested": {
			Type:        "boolean",
			Description: "Passing true requests the capability for the account, if it is not already requested. A requested capability may not immediately become active. Any requirements to activate the capability are returned in the `requirements` arrays.",
		},
		"capabilities.au_becs_debit_payments.requested": {
			Type:        "boolean",
			Description: "Passing true requests the capability for the account, if it is not already requested. A requested capability may not immediately become active. Any requirements to activate the capability are returned in the `requirements` arrays.",
		},
		"capabilities.pix_payments.requested": {
			Type:        "boolean",
			Description: "Passing true requests the capability for the account, if it is not already requested. A requested capability may not immediately become active. Any requirements to activate the capability are returned in the `requirements` arrays.",
		},
		"settings.payouts.schedule.interval": {
			Type:        "string",
			Description: "How frequently available funds are paid out. One of: `daily`, `manual`, `weekly`, or `monthly`. Default is `daily`.",
			Enum: []resource.EnumSpec{
				{Value: "daily"},
				{Value: "manual"},
				{Value: "monthly"},
				{Value: "weekly"},
			},
		},
		"capabilities.giropay_payments.requested": {
			Type:        "boolean",
			Description: "Passing true requests the capability for the account, if it is not already requested. A requested capability may not immediately become active. Any requirements to activate the capability are returned in the `requirements` arrays.",
		},
		"individual.id_number_secondary": {
			Type:        "string",
			Description: "The government-issued secondary ID number of the individual, as appropriate for the representative's country, will be used for enhanced verification checks. In Thailand, this would be the laser code found on the back of an ID card. Instead of the number itself, you can also provide a [PII token created with Stripe.js](/js/tokens/create_token?type=pii).",
		},
		"settings.card_issuing.tos_acceptance.date": {
			Type:        "integer",
			Description: "The Unix timestamp marking when the account representative accepted the service agreement.",
			Format:      "unix-time",
		},
		"company.address_kana.postal_code": {
			Type:        "string",
			Description: "Postal code.",
		},
		"capabilities.sepa_bank_transfer_payments.requested": {
			Type:        "boolean",
			Description: "Passing true requests the capability for the account, if it is not already requested. A requested capability may not immediately become active. Any requirements to activate the capability are returned in the `requirements` arrays.",
		},
		"individual.address.country": {
			Type:        "string",
			Description: "Two-letter country code ([ISO 3166-1 alpha-2](https://en.wikipedia.org/wiki/ISO_3166-1_alpha-2)).",
		},
		"individual.registered_address.country": {
			Type:        "string",
			Description: "Two-letter country code ([ISO 3166-1 alpha-2](https://en.wikipedia.org/wiki/ISO_3166-1_alpha-2)).",
		},
	},
}

var V1ApplePayDomainsDelete = resource.OperationSpec{
	Name:   "delete",
	Path:   "/v1/apple_pay/domains/{domain}",
	Method: "DELETE",
}

var V1ApplePayDomainsList = resource.OperationSpec{
	Name:   "list",
	Path:   "/v1/apple_pay/domains",
	Method: "GET",
	Params: map[string]*resource.ParamSpec{
		"domain_name": {
			Type: "string",
		},
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

var V1ApplePayDomainsRetrieve = resource.OperationSpec{
	Name:   "retrieve",
	Path:   "/v1/apple_pay/domains/{domain}",
	Method: "GET",
}

var V1ApplePayDomainsCreate = resource.OperationSpec{
	Name:   "create",
	Path:   "/v1/apple_pay/domains",
	Method: "POST",
	Params: map[string]*resource.ParamSpec{
		"domain_name": {
			Type:     "string",
			Required: true,
		},
	},
}

var V1TaxCodesList = resource.OperationSpec{
	Name:    "list",
	Path:    "/v1/tax_codes",
	Method:  "GET",
	Summary: "List all tax codes",
	Params: map[string]*resource.ParamSpec{
		"limit": {
			Type:        "integer",
			Description: "A limit on the number of objects to be returned. Limit can range between 1 and 100, and the default is 10.",
		},
		"starting_after": {
			Type:        "string",
			Description: "A cursor for use in pagination. `starting_after` is an object ID that defines your place in the list. For instance, if you make a list request and receive 100 objects, ending with `obj_foo`, your subsequent call can include `starting_after=obj_foo` in order to fetch the next page of the list.",
		},
		"ending_before": {
			Type:        "string",
			Description: "A cursor for use in pagination. `ending_before` is an object ID that defines your place in the list. For instance, if you make a list request and receive 100 objects, starting with `obj_bar`, your subsequent call can include `ending_before=obj_bar` in order to fetch the previous page of the list.",
		},
	},
}

var V1TaxCodesRetrieve = resource.OperationSpec{
	Name:    "retrieve",
	Path:    "/v1/tax_codes/{id}",
	Method:  "GET",
	Summary: "Retrieve a tax code",
}

var V1ReviewsList = resource.OperationSpec{
	Name:    "list",
	Path:    "/v1/reviews",
	Method:  "GET",
	Summary: "List all open reviews",
	Params: map[string]*resource.ParamSpec{
		"limit": {
			Type:        "integer",
			Description: "A limit on the number of objects to be returned. Limit can range between 1 and 100, and the default is 10.",
		},
		"starting_after": {
			Type:        "string",
			Description: "A cursor for use in pagination. `starting_after` is an object ID that defines your place in the list. For instance, if you make a list request and receive 100 objects, ending with `obj_foo`, your subsequent call can include `starting_after=obj_foo` in order to fetch the next page of the list.",
		},
		"created": {
			Type:        "integer",
			Description: "Only return reviews that were created during the given date interval.",
		},
		"ending_before": {
			Type:        "string",
			Description: "A cursor for use in pagination. `ending_before` is an object ID that defines your place in the list. For instance, if you make a list request and receive 100 objects, starting with `obj_bar`, your subsequent call can include `ending_before=obj_bar` in order to fetch the previous page of the list.",
		},
	},
}

var V1ReviewsRetrieve = resource.OperationSpec{
	Name:    "retrieve",
	Path:    "/v1/reviews/{review}",
	Method:  "GET",
	Summary: "Retrieve a review",
}

var V1ReviewsApprove = resource.OperationSpec{
	Name:    "approve",
	Path:    "/v1/reviews/{review}/approve",
	Method:  "POST",
	Summary: "Approve a review",
}

var V1PaymentIntentsList = resource.OperationSpec{
	Name:    "list",
	Path:    "/v1/payment_intents",
	Method:  "GET",
	Summary: "List all PaymentIntents",
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
		"created": {
			Type:        "integer",
			Description: "A filter on the list, based on the object `created` field. The value can be a string with an integer Unix timestamp or a dictionary with a number of different query options.",
		},
		"customer": {
			Type:        "string",
			Description: "Only return PaymentIntents for the customer that this customer ID specifies.",
		},
		"customer_account": {
			Type:        "string",
			Description: "Only return PaymentIntents for the account representing the customer that this ID specifies.",
		},
	},
}

var V1PaymentIntentsSearch = resource.OperationSpec{
	Name:    "search",
	Path:    "/v1/payment_intents/search",
	Method:  "GET",
	Summary: "Search PaymentIntents",
	Params: map[string]*resource.ParamSpec{
		"limit": {
			Type:        "integer",
			Description: "A limit on the number of objects to be returned. Limit can range between 1 and 100, and the default is 10.",
		},
		"page": {
			Type:        "string",
			Description: "A cursor for pagination across multiple pages of results. Don't include this parameter on the first call. Use the next_page value returned in a previous response to request subsequent results.",
		},
		"query": {
			Type:        "string",
			Description: "The search query string. See [search query language](https://docs.stripe.com/search#search-query-language) and the list of supported [query fields for payment intents](https://docs.stripe.com/search#query-fields-for-payment-intents).",
			Required:    true,
		},
	},
}

var V1PaymentIntentsCreate = resource.OperationSpec{
	Name:    "create",
	Path:    "/v1/payment_intents",
	Method:  "POST",
	Summary: "Create a PaymentIntent",
	Params: map[string]*resource.ParamSpec{
		"confirmation_token": {
			Type:        "string",
			Description: "ID of the ConfirmationToken used to confirm this PaymentIntent.\n\nIf the provided ConfirmationToken contains properties that are also being provided in this request, such as `payment_method`, then the values in this request will take precedence.",
		},
		"return_url": {
			Type:        "string",
			Description: "The URL to redirect your customer back to after they authenticate or cancel their payment on the payment method's app or site. If you'd prefer to redirect to a mobile application, you can alternatively supply an application URI scheme. This parameter can only be used with [`confirm=true`](https://docs.stripe.com/api/payment_intents/create#create_payment_intent-confirm).",
		},
		"mandate": {
			Type:        "string",
			Description: "ID of the mandate that's used for this payment. This parameter can only be used with [`confirm=true`](https://docs.stripe.com/api/payment_intents/create#create_payment_intent-confirm).",
		},
		"payment_method_data.acss_debit.transit_number": {
			Type:        "string",
			Description: "Transit number of the customer's bank.",
			Required:    true,
		},
		"payment_method_data.nz_bank_account.suffix": {
			Type:        "string",
			Description: "The suffix of the bank account number.",
			Required:    true,
		},
		"hooks.inputs.tax.calculation": {
			Type:        "string",
			Description: "The [TaxCalculation](https://docs.stripe.com/api/tax/calculations) id",
			Required:    true,
		},
		"payment_method": {
			Type:        "string",
			Description: "ID of the payment method (a PaymentMethod, Card, or [compatible Source](https://docs.stripe.com/payments/payment-methods#compatibility) object) to attach to this PaymentIntent.\n\nIf you don't provide the `payment_method` parameter or the `source` parameter with `confirm=true`, `source` automatically populates with `customer.default_source` to improve migration for users of the Charges API. We recommend that you explicitly provide the `payment_method` moving forward.\nIf the payment method is attached to a Customer, you must also provide the ID of that Customer as the [customer](https://api.stripe.com#create_payment_intent-customer) parameter of this PaymentIntent.\nend",
		},
		"payment_method_data.sepa_debit.iban": {
			Type:        "string",
			Description: "IBAN of the bank account.",
			Required:    true,
		},
		"setup_future_usage": {
			Type:        "string",
			Description: "Indicates that you intend to make future payments with this PaymentIntent's payment method.\n\nIf you provide a Customer with the PaymentIntent, you can use this parameter to [attach the payment method](/payments/save-during-payment) to the Customer after the PaymentIntent is confirmed and the customer completes any required actions. If you don't provide a Customer, you can still [attach](/api/payment_methods/attach) the payment method to a Customer after the transaction completes.\n\nIf the payment method is `card_present` and isn't a digital wallet, Stripe creates and attaches a [generated_card](/api/charges/object#charge_object-payment_method_details-card_present-generated_card) payment method representing the card to the Customer instead.\n\nWhen processing card payments, Stripe uses `setup_future_usage` to help you comply with regional legislation and network rules, such as [SCA](/strong-customer-authentication).",
			Enum: []resource.EnumSpec{
				{Value: "off_session"},
				{Value: "on_session"},
			},
		},
		"shipping.name": {
			Type:        "string",
			Description: "Recipient name.",
			Required:    true,
		},
		"shipping.phone": {
			Type:        "string",
			Description: "Recipient phone (including extension).",
		},
		"payment_method_data.nz_bank_account.branch_code": {
			Type:        "string",
			Description: "The numeric code for the bank account's bank branch.",
			Required:    true,
		},
		"payment_method_data.allow_redisplay": {
			Type:        "string",
			Description: "This field indicates whether this payment method can be shown again to its customer in a checkout flow. Stripe products such as Checkout and Elements use this field to determine whether a payment method can be shown as a saved payment method in a checkout flow. The field defaults to `unspecified`.",
			Enum: []resource.EnumSpec{
				{Value: "always"},
				{Value: "limited"},
				{Value: "unspecified"},
			},
		},
		"shipping.tracking_number": {
			Type:        "string",
			Description: "The tracking number for a physical product, obtained from the delivery service. If multiple tracking numbers were generated for this purchase, please separate them with commas.",
		},
		"transfer_group": {
			Type:        "string",
			Description: "A string that identifies the resulting payment as part of a group. Learn more about the [use case for connected accounts](https://docs.stripe.com/connect/separate-charges-and-transfers).",
		},
		"automatic_payment_methods.enabled": {
			Type:        "boolean",
			Description: "Whether this feature is enabled.",
			Required:    true,
		},
		"amount_details.discount_amount": {
			Type:        "integer",
			Description: "The total discount applied on the transaction represented in the [smallest currency unit](https://docs.stripe.com/currencies#zero-decimal). An integer greater than 0.\n\nThis field is mutually exclusive with the `amount_details[line_items][#][discount_amount]` field.",
		},
		"payment_method_data.fpx.account_holder_type": {
			Type:        "string",
			Description: "Account holder type for FPX transaction",
			Enum: []resource.EnumSpec{
				{Value: "company"},
				{Value: "individual"},
			},
		},
		"payment_method_data.nz_bank_account.bank_code": {
			Type:        "string",
			Description: "The numeric code for the bank account's bank.",
			Required:    true,
		},
		"payment_method_data.billing_details.name": {
			Type:        "string",
			Description: "Full name.",
		},
		"payment_method_data.us_bank_account.account_holder_type": {
			Type:        "string",
			Description: "Account holder type: individual or company.",
			Enum: []resource.EnumSpec{
				{Value: "company"},
				{Value: "individual"},
			},
		},
		"use_stripe_sdk": {
			Type:        "boolean",
			Description: "Set to `true` when confirming server-side and using Stripe.js, iOS, or Android client-side SDKs to handle the next actions.",
		},
		"excluded_payment_method_types": {
			Type:        "array",
			Description: "The list of payment method types to exclude from use with this payment.",
		},
		"on_behalf_of": {
			Type:        "string",
			Description: "The Stripe account ID that these funds are intended for. Learn more about the [use case for connected accounts](https://docs.stripe.com/payments/connected-accounts).",
		},
		"amount": {
			Type:        "integer",
			Description: "Amount intended to be collected by this PaymentIntent. A positive integer representing how much to charge in the [smallest currency unit](https://docs.stripe.com/currencies#zero-decimal) (e.g., 100 cents to charge $1.00 or 100 to charge ¥100, a zero-decimal currency). The minimum amount is $0.50 US or [equivalent in charge currency](https://docs.stripe.com/currencies#minimum-and-maximum-charge-amounts). The amount value supports up to eight digits (e.g., a value of 99999999 for a USD charge of $999,999.99).",
			Required:    true,
		},
		"payment_method_data.fpx.bank": {
			Type:        "string",
			Description: "The customer's bank.",
			Required:    true,
			Enum: []resource.EnumSpec{
				{Value: "affin_bank"},
				{Value: "agrobank"},
				{Value: "alliance_bank"},
				{Value: "ambank"},
				{Value: "bank_islam"},
				{Value: "bank_muamalat"},
				{Value: "bank_of_china"},
				{Value: "bank_rakyat"},
				{Value: "bsn"},
				{Value: "cimb"},
				{Value: "deutsche_bank"},
				{Value: "hong_leong_bank"},
				{Value: "hsbc"},
				{Value: "kfh"},
				{Value: "maybank2e"},
				{Value: "maybank2u"},
				{Value: "ocbc"},
				{Value: "pb_enterprise"},
				{Value: "public_bank"},
				{Value: "rhb"},
				{Value: "standard_chartered"},
				{Value: "uob"},
			},
		},
		"payment_method_data.nz_bank_account.account_holder_name": {
			Type:        "string",
			Description: "The name on the bank account. Only required if the account holder name is different from the name of the authorized signatory collected in the PaymentMethod’s billing details.",
		},
		"payment_method_data.payto.account_number": {
			Type:        "string",
			Description: "The account number for the bank account.",
		},
		"payment_method_data.billing_details.email": {
			Type:        "string",
			Description: "Email address.",
		},
		"payment_method_data.bacs_debit.sort_code": {
			Type:        "string",
			Description: "Sort code of the bank account. (e.g., `10-20-30`)",
		},
		"confirm": {
			Type:        "boolean",
			Description: "Set to `true` to attempt to [confirm this PaymentIntent](https://docs.stripe.com/api/payment_intents/confirm) immediately. This parameter defaults to `false`. When creating and confirming a PaymentIntent at the same time, you can also provide the parameters available in the [Confirm API](https://docs.stripe.com/api/payment_intents/confirm).",
		},
		"customer_account": {
			Type:        "string",
			Description: "ID of the Account representing the customer that this PaymentIntent belongs to, if one exists.\n\nPayment methods attached to other Accounts cannot be used with this PaymentIntent.\n\nIf [setup_future_usage](https://api.stripe.com#payment_intent_object-setup_future_usage) is set and this PaymentIntent's payment method is not `card_present`, then the payment method attaches to the Account after the PaymentIntent has been confirmed and any required actions from the user are complete. If the payment method is `card_present` and isn't a digital wallet, then a [generated_card](https://docs.stripe.com/api/charges/object#charge_object-payment_method_details-card_present-generated_card) payment method representing the card is created and attached to the Account instead.",
		},
		"customer": {
			Type:        "string",
			Description: "ID of the Customer this PaymentIntent belongs to, if one exists.\n\nPayment methods attached to other Customers cannot be used with this PaymentIntent.\n\nIf [setup_future_usage](https://api.stripe.com#payment_intent_object-setup_future_usage) is set and this PaymentIntent's payment method is not `card_present`, then the payment method attaches to the Customer after the PaymentIntent has been confirmed and any required actions from the user are complete. If the payment method is `card_present` and isn't a digital wallet, then a [generated_card](https://docs.stripe.com/api/charges/object#charge_object-payment_method_details-card_present-generated_card) payment method representing the card is created and attached to the Customer instead.",
		},
		"payment_method_data.type": {
			Type:        "string",
			Description: "The type of the PaymentMethod. An additional hash is included on the PaymentMethod with a name matching this value. It contains additional information specific to the PaymentMethod type.",
			Required:    true,
			Enum: []resource.EnumSpec{
				{Value: "acss_debit"},
				{Value: "affirm"},
				{Value: "afterpay_clearpay"},
				{Value: "alipay"},
				{Value: "alma"},
				{Value: "amazon_pay"},
				{Value: "au_becs_debit"},
				{Value: "bacs_debit"},
				{Value: "bancontact"},
				{Value: "billie"},
				{Value: "blik"},
				{Value: "boleto"},
				{Value: "cashapp"},
				{Value: "crypto"},
				{Value: "customer_balance"},
				{Value: "eps"},
				{Value: "fpx"},
				{Value: "giropay"},
				{Value: "grabpay"},
				{Value: "ideal"},
				{Value: "kakao_pay"},
				{Value: "klarna"},
				{Value: "konbini"},
				{Value: "kr_card"},
				{Value: "link"},
				{Value: "mb_way"},
				{Value: "mobilepay"},
				{Value: "multibanco"},
				{Value: "naver_pay"},
				{Value: "nz_bank_account"},
				{Value: "oxxo"},
				{Value: "p24"},
				{Value: "pay_by_bank"},
				{Value: "payco"},
				{Value: "paynow"},
				{Value: "paypal"},
				{Value: "payto"},
				{Value: "pix"},
				{Value: "promptpay"},
				{Value: "revolut_pay"},
				{Value: "samsung_pay"},
				{Value: "satispay"},
				{Value: "sepa_debit"},
				{Value: "sofort"},
				{Value: "swish"},
				{Value: "twint"},
				{Value: "us_bank_account"},
				{Value: "wechat_pay"},
				{Value: "zip"},
			},
		},
		"payment_method_data.us_bank_account.account_type": {
			Type:        "string",
			Description: "Account type: checkings or savings. Defaults to checking if omitted.",
			Enum: []resource.EnumSpec{
				{Value: "checking"},
				{Value: "savings"},
			},
		},
		"payment_method_data.us_bank_account.financial_connections_account": {
			Type:        "string",
			Description: "The ID of a Financial Connections Account to use as a payment method.",
		},
		"payment_method_data.us_bank_account.account_number": {
			Type:        "string",
			Description: "Account number of the bank account.",
		},
		"payment_method_data.eps.bank": {
			Type:        "string",
			Description: "The customer's bank.",
			Enum: []resource.EnumSpec{
				{Value: "arzte_und_apotheker_bank"},
				{Value: "austrian_anadi_bank_ag"},
				{Value: "bank_austria"},
				{Value: "bankhaus_carl_spangler"},
				{Value: "bankhaus_schelhammer_und_schattera_ag"},
				{Value: "bawag_psk_ag"},
				{Value: "bks_bank_ag"},
				{Value: "brull_kallmus_bank_ag"},
				{Value: "btv_vier_lander_bank"},
				{Value: "capital_bank_grawe_gruppe_ag"},
				{Value: "deutsche_bank_ag"},
				{Value: "dolomitenbank"},
				{Value: "easybank_ag"},
				{Value: "erste_bank_und_sparkassen"},
				{Value: "hypo_alpeadriabank_international_ag"},
				{Value: "hypo_bank_burgenland_aktiengesellschaft"},
				{Value: "hypo_noe_lb_fur_niederosterreich_u_wien"},
				{Value: "hypo_oberosterreich_salzburg_steiermark"},
				{Value: "hypo_tirol_bank_ag"},
				{Value: "hypo_vorarlberg_bank_ag"},
				{Value: "marchfelder_bank"},
				{Value: "oberbank_ag"},
				{Value: "raiffeisen_bankengruppe_osterreich"},
				{Value: "schoellerbank_ag"},
				{Value: "sparda_bank_wien"},
				{Value: "volksbank_gruppe"},
				{Value: "volkskreditbank_ag"},
				{Value: "vr_bank_braunau"},
			},
		},
		"payment_method_data.p24.bank": {
			Type:        "string",
			Description: "The customer's bank.",
			Enum: []resource.EnumSpec{
				{Value: "alior_bank"},
				{Value: "bank_millennium"},
				{Value: "bank_nowy_bfg_sa"},
				{Value: "bank_pekao_sa"},
				{Value: "banki_spbdzielcze"},
				{Value: "blik"},
				{Value: "bnp_paribas"},
				{Value: "boz"},
				{Value: "citi_handlowy"},
				{Value: "credit_agricole"},
				{Value: "envelobank"},
				{Value: "etransfer_pocztowy24"},
				{Value: "getin_bank"},
				{Value: "ideabank"},
				{Value: "ing"},
				{Value: "inteligo"},
				{Value: "mbank_mtransfer"},
				{Value: "nest_przelew"},
				{Value: "noble_pay"},
				{Value: "pbac_z_ipko"},
				{Value: "plus_bank"},
				{Value: "santander_przelew24"},
				{Value: "tmobile_usbugi_bankowe"},
				{Value: "toyota_bank"},
				{Value: "velobank"},
				{Value: "volkswagen_bank"},
			},
		},
		"statement_descriptor": {
			Type:        "string",
			Description: "Text that appears on the customer's statement as the statement descriptor for a non-card charge. This value overrides the account's default statement descriptor. For information about requirements, including the 22-character limit, see [the Statement Descriptor docs](https://docs.stripe.com/get-started/account/statement-descriptors).\n\nSetting this value for a card charge returns an error. For card charges, set the [statement_descriptor_suffix](https://docs.stripe.com/get-started/account/statement-descriptors#dynamic) instead.",
		},
		"payment_method_data.billing_details.phone": {
			Type:        "string",
			Description: "Billing phone number (including extension).",
		},
		"receipt_email": {
			Type:        "string",
			Description: "Email address to send the receipt to. If you specify `receipt_email` for a payment in live mode, you send a receipt regardless of your [email settings](https://dashboard.stripe.com/account/emails).",
		},
		"transfer_data.amount": {
			Type:        "integer",
			Description: "The amount that will be transferred automatically when a charge succeeds.\nThe amount is capped at the total transaction amount and if no amount is set,\nthe full amount is transferred.\n\nIf you intend to collect a fee and you need a more robust reporting experience, using\n[application_fee_amount](https://docs.stripe.com/api/payment_intents/create#create_payment_intent-application_fee_amount)\nmight be a better fit for your integration.",
		},
		"confirmation_method": {
			Type:        "string",
			Description: "Describes whether we can confirm this PaymentIntent automatically, or if it requires customer action to confirm the payment.",
			Enum: []resource.EnumSpec{
				{Value: "automatic"},
				{Value: "manual"},
			},
		},
		"shipping.address.city": {
			Type:        "string",
			Description: "City, district, suburb, town, or village.",
		},
		"shipping.address.line1": {
			Type:        "string",
			Description: "Address line 1, such as the street, PO Box, or company name.",
		},
		"shipping.address.line2": {
			Type:        "string",
			Description: "Address line 2, such as the apartment, suite, unit, or building.",
		},
		"shipping.address.postal_code": {
			Type:        "string",
			Description: "ZIP or postal code.",
		},
		"payment_method_data.klarna.dob.day": {
			Type:        "integer",
			Description: "The day of birth, between 1 and 31.",
			Required:    true,
		},
		"payment_method_data.us_bank_account.routing_number": {
			Type:        "string",
			Description: "Routing number of the bank account.",
		},
		"payment_method_data.naver_pay.funding": {
			Type:        "string",
			Description: "Whether to use Naver Pay points or a card to fund this transaction. If not provided, this defaults to `card`.",
			Enum: []resource.EnumSpec{
				{Value: "card"},
				{Value: "points"},
			},
		},
		"payment_details.customer_reference": {
			Type:        "string",
			Description: "A unique value to identify the customer. This field is available only for card payments.\n\nThis field is truncated to 25 alphanumeric characters, excluding spaces, before being sent to card networks.",
		},
		"payment_details.order_reference": {
			Type:        "string",
			Description: "A unique value assigned by the business to identify the transaction. Required for L2 and L3 rates.\n\nRequired when the Payment Method Types array contains `card`, including when [automatic_payment_methods.enabled](/api/payment_intents/create#create_payment_intent-automatic_payment_methods-enabled) is set to `true`.\n\nFor Cards, this field is truncated to 25 alphanumeric characters, excluding spaces, before being sent to card networks. For Klarna, this field is truncated to 255 characters and is visible to customers when they view the order in the Klarna app.",
		},
		"description": {
			Type:        "string",
			Description: "An arbitrary string attached to the object. Often useful for displaying to users.",
		},
		"payment_method_data.acss_debit.account_number": {
			Type:        "string",
			Description: "Customer's bank account number.",
			Required:    true,
		},
		"payment_method_data.billing_details.tax_id": {
			Type:        "string",
			Description: "Taxpayer identification number. Used only for transactions between LATAM buyers and non-LATAM sellers.",
		},
		"payment_method_data.au_becs_debit.account_number": {
			Type:        "string",
			Description: "The account number for the bank account.",
			Required:    true,
		},
		"payment_method_data.ideal.bank": {
			Type:        "string",
			Description: "The customer's bank. Only use this parameter for existing customers. Don't use it for new customers.",
			Enum: []resource.EnumSpec{
				{Value: "abn_amro"},
				{Value: "adyen"},
				{Value: "asn_bank"},
				{Value: "bunq"},
				{Value: "buut"},
				{Value: "finom"},
				{Value: "handelsbanken"},
				{Value: "ing"},
				{Value: "knab"},
				{Value: "mollie"},
				{Value: "moneyou"},
				{Value: "n26"},
				{Value: "nn"},
				{Value: "rabobank"},
				{Value: "regiobank"},
				{Value: "revolut"},
				{Value: "sns_bank"},
				{Value: "triodos_bank"},
				{Value: "van_lanschot"},
				{Value: "yoursafe"},
			},
		},
		"off_session": {
			Type:        "boolean",
			Description: "Set to `true` to indicate that the customer isn't in your checkout flow during this payment attempt and can't authenticate. Use this parameter in scenarios where you collect card details and [charge them later](https://docs.stripe.com/payments/cards/charging-saved-cards). This parameter can only be used with [`confirm=true`](https://docs.stripe.com/api/payment_intents/create#create_payment_intent-confirm).",
		},
		"shipping.address.state": {
			Type:        "string",
			Description: "State, county, province, or region ([ISO 3166-2](https://en.wikipedia.org/wiki/ISO_3166-2)).",
		},
		"automatic_payment_methods.allow_redirects": {
			Type:        "string",
			Description: "Controls whether this PaymentIntent will accept redirect-based payment methods.\n\nRedirect-based payment methods may require your customer to be redirected to a payment method's app or site for authentication or additional steps. To [confirm](https://docs.stripe.com/api/payment_intents/confirm) this PaymentIntent, you may be required to provide a `return_url` to redirect customers back to your site after they authenticate or complete the payment.",
			Enum: []resource.EnumSpec{
				{Value: "always"},
				{Value: "never"},
			},
		},
		"payment_method_data.acss_debit.institution_number": {
			Type:        "string",
			Description: "Institution number of the customer's bank.",
			Required:    true,
		},
		"payment_method_data.klarna.dob.year": {
			Type:        "integer",
			Description: "The four-digit year of birth.",
			Required:    true,
		},
		"payment_method_data.sofort.country": {
			Type:        "string",
			Description: "Two-letter ISO code representing the country the bank account is located in.",
			Required:    true,
			Enum: []resource.EnumSpec{
				{Value: "AT"},
				{Value: "BE"},
				{Value: "DE"},
				{Value: "ES"},
				{Value: "IT"},
				{Value: "NL"},
			},
		},
		"transfer_data.destination": {
			Type:        "string",
			Description: "If specified, successful charges will be attributed to the destination\naccount for tax reporting, and the funds from charges will be transferred\nto the destination account. The ID of the resulting transfer will be\nreturned on the successful charge's `transfer` field.",
			Required:    true,
		},
		"shipping.address.country": {
			Type:        "string",
			Description: "Two-letter country code ([ISO 3166-1 alpha-2](https://en.wikipedia.org/wiki/ISO_3166-1_alpha-2)).",
		},
		"amount_details.enforce_arithmetic_validation": {
			Type:        "boolean",
			Description: "Set to `false` to return arithmetic validation errors in the response without failing the request. Use this when you want the operation to proceed regardless of arithmetic errors in the line item data.\n\nOmit or set to `true` to immediately return a 400 error when arithmetic validation fails. Use this for strict validation that prevents processing with line item data that has arithmetic inconsistencies.\n\nFor card payments, Stripe doesn't send line item data to card networks if there's an arithmetic validation error.",
		},
		"application_fee_amount": {
			Type:        "integer",
			Description: "The amount of the application fee (if any) that will be requested to be applied to the payment and transferred to the application owner's Stripe account. The amount of the application fee collected will be capped at the total amount captured. For more information, see the PaymentIntents [use case for connected accounts](https://docs.stripe.com/payments/connected-accounts).",
		},
		"payment_method_data.nz_bank_account.reference": {
			Type: "string",
		},
		"payment_method_data.nz_bank_account.account_number": {
			Type:        "string",
			Description: "The account number for the bank account.",
			Required:    true,
		},
		"payment_method_data.payto.bsb_number": {
			Type:        "string",
			Description: "Bank-State-Branch number of the bank account.",
		},
		"payment_method_data.payto.pay_id": {
			Type:        "string",
			Description: "The PayID alias for the bank account.",
		},
		"payment_method_data.au_becs_debit.bsb_number": {
			Type:        "string",
			Description: "Bank-State-Branch number of the bank account.",
			Required:    true,
		},
		"error_on_requires_action": {
			Type:        "boolean",
			Description: "Set to `true` to fail the payment attempt if the PaymentIntent transitions into `requires_action`. Use this parameter for simpler integrations that don't handle customer actions, such as [saving cards without authentication](https://docs.stripe.com/payments/save-card-without-authentication). This parameter can only be used with [`confirm=true`](https://docs.stripe.com/api/payment_intents/create#create_payment_intent-confirm).",
		},
		"capture_method": {
			Type:        "string",
			Description: "Controls when the funds will be captured from the customer's account.",
			Enum: []resource.EnumSpec{
				{Value: "automatic"},
				{Value: "automatic_async"},
				{Value: "manual"},
			},
		},
		"payment_method_data.boleto.tax_id": {
			Type:        "string",
			Description: "The tax ID of the customer (CPF for individual consumers or CNPJ for businesses consumers)",
			Required:    true,
		},
		"payment_method_data.radar_options.session": {
			Type:        "string",
			Description: "A [Radar Session](https://docs.stripe.com/radar/radar-session) is a snapshot of the browser metadata and device details that help Radar make more accurate predictions on your payments.",
		},
		"payment_method_types": {
			Type:        "array",
			Description: "The list of payment method types (for example, a card) that this PaymentIntent can use. If you don't provide this, Stripe will dynamically show relevant payment methods from your [payment method settings](https://dashboard.stripe.com/settings/payment_methods). A list of valid payment method types can be found [here](https://docs.stripe.com/api/payment_methods/object#payment_method_object-type).",
		},
		"payment_method_configuration": {
			Type:        "string",
			Description: "The ID of the [payment method configuration](https://docs.stripe.com/api/payment_method_configurations) to use with this PaymentIntent.",
		},
		"shipping.carrier": {
			Type:        "string",
			Description: "The delivery service that shipped a physical product, such as Fedex, UPS, USPS, etc.",
		},
		"statement_descriptor_suffix": {
			Type:        "string",
			Description: "Provides information about a card charge. Concatenated to the account's [statement descriptor prefix](https://docs.stripe.com/get-started/account/statement-descriptors#static) to form the complete statement descriptor that appears on the customer's statement.",
		},
		"payment_method_data.klarna.dob.month": {
			Type:        "integer",
			Description: "The month of birth, between 1 and 12.",
			Required:    true,
		},
		"payment_method_data.bacs_debit.account_number": {
			Type:        "string",
			Description: "Account number of the bank account that the funds will be debited from.",
		},
		"currency": {
			Type:        "string",
			Description: "Three-letter [ISO currency code](https://www.iso.org/iso-4217-currency-codes.html), in lowercase. Must be a [supported currency](https://stripe.com/docs/currencies).",
			Required:    true,
			Format:      "currency",
		},
		"radar_options.session": {
			Type:        "string",
			Description: "A [Radar Session](https://docs.stripe.com/radar/radar-session) is a snapshot of the browser metadata and device details that help Radar make more accurate predictions on your payments.",
		},
	},
}

var V1PaymentIntentsIncrementAuthorization = resource.OperationSpec{
	Name:    "increment_authorization",
	Path:    "/v1/payment_intents/{intent}/increment_authorization",
	Method:  "POST",
	Summary: "Increment an authorization",
	Params: map[string]*resource.ParamSpec{
		"amount_details.discount_amount": {
			Type:        "integer",
			Description: "The total discount applied on the transaction represented in the [smallest currency unit](https://docs.stripe.com/currencies#zero-decimal). An integer greater than 0.\n\nThis field is mutually exclusive with the `amount_details[line_items][#][discount_amount]` field.",
		},
		"amount_details.enforce_arithmetic_validation": {
			Type:        "boolean",
			Description: "Set to `false` to return arithmetic validation errors in the response without failing the request. Use this when you want the operation to proceed regardless of arithmetic errors in the line item data.\n\nOmit or set to `true` to immediately return a 400 error when arithmetic validation fails. Use this for strict validation that prevents processing with line item data that has arithmetic inconsistencies.\n\nFor card payments, Stripe doesn't send line item data to card networks if there's an arithmetic validation error.",
		},
		"payment_details.customer_reference": {
			Type:        "string",
			Description: "A unique value to identify the customer. This field is available only for card payments.\n\nThis field is truncated to 25 alphanumeric characters, excluding spaces, before being sent to card networks.",
		},
		"payment_details.order_reference": {
			Type:        "string",
			Description: "A unique value assigned by the business to identify the transaction. Required for L2 and L3 rates.\n\nRequired when the Payment Method Types array contains `card`, including when [automatic_payment_methods.enabled](/api/payment_intents/create#create_payment_intent-automatic_payment_methods-enabled) is set to `true`.\n\nFor Cards, this field is truncated to 25 alphanumeric characters, excluding spaces, before being sent to card networks. For Klarna, this field is truncated to 255 characters and is visible to customers when they view the order in the Klarna app.",
		},
		"amount": {
			Type:        "integer",
			Description: "The updated total amount that you intend to collect from the cardholder. This amount must be greater than the currently authorized amount.",
			Required:    true,
		},
		"application_fee_amount": {
			Type:        "integer",
			Description: "The amount of the application fee (if any) that will be requested to be applied to the payment and transferred to the application owner's Stripe account. The amount of the application fee collected will be capped at the total amount captured. For more information, see the PaymentIntents [use case for connected accounts](https://docs.stripe.com/payments/connected-accounts).",
		},
		"hooks.inputs.tax.calculation": {
			Type:        "string",
			Description: "The [TaxCalculation](https://docs.stripe.com/api/tax/calculations) id",
			Required:    true,
		},
		"statement_descriptor": {
			Type:        "string",
			Description: "Text that appears on the customer's statement as the statement descriptor for a non-card or card charge. This value overrides the account's default statement descriptor. For information about requirements, including the 22-character limit, see [the Statement Descriptor docs](https://docs.stripe.com/get-started/account/statement-descriptors).",
		},
		"transfer_data.amount": {
			Type:        "integer",
			Description: "The amount that will be transferred automatically when a charge succeeds.",
		},
		"description": {
			Type:        "string",
			Description: "An arbitrary string attached to the object. Often useful for displaying to users.",
		},
	},
}

var V1PaymentIntentsRetrieve = resource.OperationSpec{
	Name:    "retrieve",
	Path:    "/v1/payment_intents/{intent}",
	Method:  "GET",
	Summary: "Retrieve a PaymentIntent",
	Params: map[string]*resource.ParamSpec{
		"client_secret": {
			Type:        "string",
			Description: "The client secret of the PaymentIntent. We require it if you use a publishable key to retrieve the source.",
		},
	},
}

var V1PaymentIntentsUpdate = resource.OperationSpec{
	Name:    "update",
	Path:    "/v1/payment_intents/{intent}",
	Method:  "POST",
	Summary: "Update a PaymentIntent",
	Params: map[string]*resource.ParamSpec{
		"payment_method_data.au_becs_debit.bsb_number": {
			Type:        "string",
			Description: "Bank-State-Branch number of the bank account.",
			Required:    true,
		},
		"payment_method_data.us_bank_account.financial_connections_account": {
			Type:        "string",
			Description: "The ID of a Financial Connections Account to use as a payment method.",
		},
		"payment_method_data.sepa_debit.iban": {
			Type:        "string",
			Description: "IBAN of the bank account.",
			Required:    true,
		},
		"transfer_group": {
			Type:        "string",
			Description: "A string that identifies the resulting payment as part of a group. You can only provide `transfer_group` if it hasn't been set. Learn more about the [use case for connected accounts](https://docs.stripe.com/payments/connected-accounts).",
		},
		"capture_method": {
			Type:        "string",
			Description: "Controls when the funds will be captured from the customer's account.",
			Enum: []resource.EnumSpec{
				{Value: "automatic"},
				{Value: "automatic_async"},
				{Value: "manual"},
			},
		},
		"payment_method_types": {
			Type:        "array",
			Description: "The list of payment method types (for example, card) that this PaymentIntent can use. Use `automatic_payment_methods` to manage payment methods from the [Stripe Dashboard](https://dashboard.stripe.com/settings/payment_methods). A list of valid payment method types can be found [here](https://docs.stripe.com/api/payment_methods/object#payment_method_object-type).",
		},
		"payment_method_data.nz_bank_account.bank_code": {
			Type:        "string",
			Description: "The numeric code for the bank account's bank.",
			Required:    true,
		},
		"payment_method_data.nz_bank_account.reference": {
			Type: "string",
		},
		"payment_method_data.ideal.bank": {
			Type:        "string",
			Description: "The customer's bank. Only use this parameter for existing customers. Don't use it for new customers.",
			Enum: []resource.EnumSpec{
				{Value: "abn_amro"},
				{Value: "adyen"},
				{Value: "asn_bank"},
				{Value: "bunq"},
				{Value: "buut"},
				{Value: "finom"},
				{Value: "handelsbanken"},
				{Value: "ing"},
				{Value: "knab"},
				{Value: "mollie"},
				{Value: "moneyou"},
				{Value: "n26"},
				{Value: "nn"},
				{Value: "rabobank"},
				{Value: "regiobank"},
				{Value: "revolut"},
				{Value: "sns_bank"},
				{Value: "triodos_bank"},
				{Value: "van_lanschot"},
				{Value: "yoursafe"},
			},
		},
		"payment_method_data.us_bank_account.account_number": {
			Type:        "string",
			Description: "Account number of the bank account.",
		},
		"payment_method_data.klarna.dob.day": {
			Type:        "integer",
			Description: "The day of birth, between 1 and 31.",
			Required:    true,
		},
		"payment_method_data.klarna.dob.year": {
			Type:        "integer",
			Description: "The four-digit year of birth.",
			Required:    true,
		},
		"statement_descriptor": {
			Type:        "string",
			Description: "Text that appears on the customer's statement as the statement descriptor for a non-card charge. This value overrides the account's default statement descriptor. For information about requirements, including the 22-character limit, see [the Statement Descriptor docs](https://docs.stripe.com/get-started/account/statement-descriptors).\n\nSetting this value for a card charge returns an error. For card charges, set the [statement_descriptor_suffix](https://docs.stripe.com/get-started/account/statement-descriptors#dynamic) instead.",
		},
		"payment_method_data.billing_details.email": {
			Type:        "string",
			Description: "Email address.",
		},
		"description": {
			Type:        "string",
			Description: "An arbitrary string attached to the object. Often useful for displaying to users.",
		},
		"payment_method_data.billing_details.name": {
			Type:        "string",
			Description: "Full name.",
		},
		"payment_method_data.nz_bank_account.account_number": {
			Type:        "string",
			Description: "The account number for the bank account.",
			Required:    true,
		},
		"payment_method_data.acss_debit.transit_number": {
			Type:        "string",
			Description: "Transit number of the customer's bank.",
			Required:    true,
		},
		"payment_method_data.fpx.account_holder_type": {
			Type:        "string",
			Description: "Account holder type for FPX transaction",
			Enum: []resource.EnumSpec{
				{Value: "company"},
				{Value: "individual"},
			},
		},
		"payment_method_data.nz_bank_account.branch_code": {
			Type:        "string",
			Description: "The numeric code for the bank account's bank branch.",
			Required:    true,
		},
		"payment_method_data.billing_details.tax_id": {
			Type:        "string",
			Description: "Taxpayer identification number. Used only for transactions between LATAM buyers and non-LATAM sellers.",
		},
		"amount": {
			Type:        "integer",
			Description: "Amount intended to be collected by this PaymentIntent. A positive integer representing how much to charge in the [smallest currency unit](https://docs.stripe.com/currencies#zero-decimal) (e.g., 100 cents to charge $1.00 or 100 to charge ¥100, a zero-decimal currency). The minimum amount is $0.50 US or [equivalent in charge currency](https://docs.stripe.com/currencies#minimum-and-maximum-charge-amounts). The amount value supports up to eight digits (e.g., a value of 99999999 for a USD charge of $999,999.99).",
		},
		"hooks.inputs.tax.calculation": {
			Type:        "string",
			Description: "The [TaxCalculation](https://docs.stripe.com/api/tax/calculations) id",
			Required:    true,
		},
		"currency": {
			Type:        "string",
			Description: "Three-letter [ISO currency code](https://www.iso.org/iso-4217-currency-codes.html), in lowercase. Must be a [supported currency](https://stripe.com/docs/currencies).",
			Format:      "currency",
		},
		"payment_method_data.billing_details.phone": {
			Type:        "string",
			Description: "Billing phone number (including extension).",
		},
		"payment_method_data.us_bank_account.routing_number": {
			Type:        "string",
			Description: "Routing number of the bank account.",
		},
		"payment_method_data.klarna.dob.month": {
			Type:        "integer",
			Description: "The month of birth, between 1 and 12.",
			Required:    true,
		},
		"payment_method_data.eps.bank": {
			Type:        "string",
			Description: "The customer's bank.",
			Enum: []resource.EnumSpec{
				{Value: "arzte_und_apotheker_bank"},
				{Value: "austrian_anadi_bank_ag"},
				{Value: "bank_austria"},
				{Value: "bankhaus_carl_spangler"},
				{Value: "bankhaus_schelhammer_und_schattera_ag"},
				{Value: "bawag_psk_ag"},
				{Value: "bks_bank_ag"},
				{Value: "brull_kallmus_bank_ag"},
				{Value: "btv_vier_lander_bank"},
				{Value: "capital_bank_grawe_gruppe_ag"},
				{Value: "deutsche_bank_ag"},
				{Value: "dolomitenbank"},
				{Value: "easybank_ag"},
				{Value: "erste_bank_und_sparkassen"},
				{Value: "hypo_alpeadriabank_international_ag"},
				{Value: "hypo_bank_burgenland_aktiengesellschaft"},
				{Value: "hypo_noe_lb_fur_niederosterreich_u_wien"},
				{Value: "hypo_oberosterreich_salzburg_steiermark"},
				{Value: "hypo_tirol_bank_ag"},
				{Value: "hypo_vorarlberg_bank_ag"},
				{Value: "marchfelder_bank"},
				{Value: "oberbank_ag"},
				{Value: "raiffeisen_bankengruppe_osterreich"},
				{Value: "schoellerbank_ag"},
				{Value: "sparda_bank_wien"},
				{Value: "volksbank_gruppe"},
				{Value: "volkskreditbank_ag"},
				{Value: "vr_bank_braunau"},
			},
		},
		"payment_method_configuration": {
			Type:        "string",
			Description: "The ID of the [payment method configuration](https://docs.stripe.com/api/payment_method_configurations) to use with this PaymentIntent.",
		},
		"payment_method": {
			Type:        "string",
			Description: "ID of the payment method (a PaymentMethod, Card, or [compatible Source](https://docs.stripe.com/payments/payment-methods/transitioning#compatibility) object) to attach to this PaymentIntent. To unset this field to null, pass in an empty string.",
		},
		"excluded_payment_method_types": {
			Type:        "array",
			Description: "The list of payment method types to exclude from use with this payment.",
		},
		"payment_method_data.us_bank_account.account_holder_type": {
			Type:        "string",
			Description: "Account holder type: individual or company.",
			Enum: []resource.EnumSpec{
				{Value: "company"},
				{Value: "individual"},
			},
		},
		"payment_method_data.payto.pay_id": {
			Type:        "string",
			Description: "The PayID alias for the bank account.",
		},
		"payment_method_data.type": {
			Type:        "string",
			Description: "The type of the PaymentMethod. An additional hash is included on the PaymentMethod with a name matching this value. It contains additional information specific to the PaymentMethod type.",
			Required:    true,
			Enum: []resource.EnumSpec{
				{Value: "acss_debit"},
				{Value: "affirm"},
				{Value: "afterpay_clearpay"},
				{Value: "alipay"},
				{Value: "alma"},
				{Value: "amazon_pay"},
				{Value: "au_becs_debit"},
				{Value: "bacs_debit"},
				{Value: "bancontact"},
				{Value: "billie"},
				{Value: "blik"},
				{Value: "boleto"},
				{Value: "cashapp"},
				{Value: "crypto"},
				{Value: "customer_balance"},
				{Value: "eps"},
				{Value: "fpx"},
				{Value: "giropay"},
				{Value: "grabpay"},
				{Value: "ideal"},
				{Value: "kakao_pay"},
				{Value: "klarna"},
				{Value: "konbini"},
				{Value: "kr_card"},
				{Value: "link"},
				{Value: "mb_way"},
				{Value: "mobilepay"},
				{Value: "multibanco"},
				{Value: "naver_pay"},
				{Value: "nz_bank_account"},
				{Value: "oxxo"},
				{Value: "p24"},
				{Value: "pay_by_bank"},
				{Value: "payco"},
				{Value: "paynow"},
				{Value: "paypal"},
				{Value: "payto"},
				{Value: "pix"},
				{Value: "promptpay"},
				{Value: "revolut_pay"},
				{Value: "samsung_pay"},
				{Value: "satispay"},
				{Value: "sepa_debit"},
				{Value: "sofort"},
				{Value: "swish"},
				{Value: "twint"},
				{Value: "us_bank_account"},
				{Value: "wechat_pay"},
				{Value: "zip"},
			},
		},
		"payment_method_data.bacs_debit.account_number": {
			Type:        "string",
			Description: "Account number of the bank account that the funds will be debited from.",
		},
		"payment_method_data.radar_options.session": {
			Type:        "string",
			Description: "A [Radar Session](https://docs.stripe.com/radar/radar-session) is a snapshot of the browser metadata and device details that help Radar make more accurate predictions on your payments.",
		},
		"payment_method_data.sofort.country": {
			Type:        "string",
			Description: "Two-letter ISO code representing the country the bank account is located in.",
			Required:    true,
			Enum: []resource.EnumSpec{
				{Value: "AT"},
				{Value: "BE"},
				{Value: "DE"},
				{Value: "ES"},
				{Value: "IT"},
				{Value: "NL"},
			},
		},
		"payment_method_data.acss_debit.institution_number": {
			Type:        "string",
			Description: "Institution number of the customer's bank.",
			Required:    true,
		},
		"payment_method_data.allow_redisplay": {
			Type:        "string",
			Description: "This field indicates whether this payment method can be shown again to its customer in a checkout flow. Stripe products such as Checkout and Elements use this field to determine whether a payment method can be shown as a saved payment method in a checkout flow. The field defaults to `unspecified`.",
			Enum: []resource.EnumSpec{
				{Value: "always"},
				{Value: "limited"},
				{Value: "unspecified"},
			},
		},
		"receipt_email": {
			Type:        "string",
			Description: "Email address that the receipt for the resulting payment will be sent to. If `receipt_email` is specified for a payment in live mode, a receipt will be sent regardless of your [email settings](https://dashboard.stripe.com/account/emails).",
		},
		"transfer_data.amount": {
			Type:        "integer",
			Description: "The amount that will be transferred automatically when a charge succeeds.",
		},
		"payment_method_data.payto.account_number": {
			Type:        "string",
			Description: "The account number for the bank account.",
		},
		"statement_descriptor_suffix": {
			Type:        "string",
			Description: "Provides information about a card charge. Concatenated to the account's [statement descriptor prefix](https://docs.stripe.com/get-started/account/statement-descriptors#static) to form the complete statement descriptor that appears on the customer's statement.",
		},
		"application_fee_amount": {
			Type:        "integer",
			Description: "The amount of the application fee (if any) that will be requested to be applied to the payment and transferred to the application owner's Stripe account. The amount of the application fee collected will be capped at the total amount captured. For more information, see the PaymentIntents [use case for connected accounts](https://docs.stripe.com/payments/connected-accounts).",
		},
		"payment_method_data.nz_bank_account.account_holder_name": {
			Type:        "string",
			Description: "The name on the bank account. Only required if the account holder name is different from the name of the authorized signatory collected in the PaymentMethod’s billing details.",
		},
		"setup_future_usage": {
			Type:        "string",
			Description: "Indicates that you intend to make future payments with this PaymentIntent's payment method.\n\nIf you provide a Customer with the PaymentIntent, you can use this parameter to [attach the payment method](/payments/save-during-payment) to the Customer after the PaymentIntent is confirmed and the customer completes any required actions. If you don't provide a Customer, you can still [attach](/api/payment_methods/attach) the payment method to a Customer after the transaction completes.\n\nIf the payment method is `card_present` and isn't a digital wallet, Stripe creates and attaches a [generated_card](/api/charges/object#charge_object-payment_method_details-card_present-generated_card) payment method representing the card to the Customer instead.\n\nWhen processing card payments, Stripe uses `setup_future_usage` to help you comply with regional legislation and network rules, such as [SCA](/strong-customer-authentication).\n\nIf you've already set `setup_future_usage` and you're performing a request using a publishable key, you can only update the value from `on_session` to `off_session`.",
			Enum: []resource.EnumSpec{
				{Value: "off_session"},
				{Value: "on_session"},
			},
		},
		"payment_method_data.bacs_debit.sort_code": {
			Type:        "string",
			Description: "Sort code of the bank account. (e.g., `10-20-30`)",
		},
		"payment_method_data.au_becs_debit.account_number": {
			Type:        "string",
			Description: "The account number for the bank account.",
			Required:    true,
		},
		"payment_method_data.us_bank_account.account_type": {
			Type:        "string",
			Description: "Account type: checkings or savings. Defaults to checking if omitted.",
			Enum: []resource.EnumSpec{
				{Value: "checking"},
				{Value: "savings"},
			},
		},
		"customer_account": {
			Type:        "string",
			Description: "ID of the Account representing the customer that this PaymentIntent belongs to, if one exists.\n\nPayment methods attached to other Accounts cannot be used with this PaymentIntent.\n\nIf [setup_future_usage](https://api.stripe.com#payment_intent_object-setup_future_usage) is set and this PaymentIntent's payment method is not `card_present`, then the payment method attaches to the Account after the PaymentIntent has been confirmed and any required actions from the user are complete. If the payment method is `card_present` and isn't a digital wallet, then a [generated_card](https://docs.stripe.com/api/charges/object#charge_object-payment_method_details-card_present-generated_card) payment method representing the card is created and attached to the Account instead.",
		},
		"payment_method_data.nz_bank_account.suffix": {
			Type:        "string",
			Description: "The suffix of the bank account number.",
			Required:    true,
		},
		"payment_method_data.p24.bank": {
			Type:        "string",
			Description: "The customer's bank.",
			Enum: []resource.EnumSpec{
				{Value: "alior_bank"},
				{Value: "bank_millennium"},
				{Value: "bank_nowy_bfg_sa"},
				{Value: "bank_pekao_sa"},
				{Value: "banki_spbdzielcze"},
				{Value: "blik"},
				{Value: "bnp_paribas"},
				{Value: "boz"},
				{Value: "citi_handlowy"},
				{Value: "credit_agricole"},
				{Value: "envelobank"},
				{Value: "etransfer_pocztowy24"},
				{Value: "getin_bank"},
				{Value: "ideabank"},
				{Value: "ing"},
				{Value: "inteligo"},
				{Value: "mbank_mtransfer"},
				{Value: "nest_przelew"},
				{Value: "noble_pay"},
				{Value: "pbac_z_ipko"},
				{Value: "plus_bank"},
				{Value: "santander_przelew24"},
				{Value: "tmobile_usbugi_bankowe"},
				{Value: "toyota_bank"},
				{Value: "velobank"},
				{Value: "volkswagen_bank"},
			},
		},
		"payment_method_data.naver_pay.funding": {
			Type:        "string",
			Description: "Whether to use Naver Pay points or a card to fund this transaction. If not provided, this defaults to `card`.",
			Enum: []resource.EnumSpec{
				{Value: "card"},
				{Value: "points"},
			},
		},
		"payment_method_data.boleto.tax_id": {
			Type:        "string",
			Description: "The tax ID of the customer (CPF for individual consumers or CNPJ for businesses consumers)",
			Required:    true,
		},
		"payment_method_data.payto.bsb_number": {
			Type:        "string",
			Description: "Bank-State-Branch number of the bank account.",
		},
		"payment_method_data.acss_debit.account_number": {
			Type:        "string",
			Description: "Customer's bank account number.",
			Required:    true,
		},
		"payment_method_data.fpx.bank": {
			Type:        "string",
			Description: "The customer's bank.",
			Required:    true,
			Enum: []resource.EnumSpec{
				{Value: "affin_bank"},
				{Value: "agrobank"},
				{Value: "alliance_bank"},
				{Value: "ambank"},
				{Value: "bank_islam"},
				{Value: "bank_muamalat"},
				{Value: "bank_of_china"},
				{Value: "bank_rakyat"},
				{Value: "bsn"},
				{Value: "cimb"},
				{Value: "deutsche_bank"},
				{Value: "hong_leong_bank"},
				{Value: "hsbc"},
				{Value: "kfh"},
				{Value: "maybank2e"},
				{Value: "maybank2u"},
				{Value: "ocbc"},
				{Value: "pb_enterprise"},
				{Value: "public_bank"},
				{Value: "rhb"},
				{Value: "standard_chartered"},
				{Value: "uob"},
			},
		},
		"customer": {
			Type:        "string",
			Description: "ID of the Customer this PaymentIntent belongs to, if one exists.\n\nPayment methods attached to other Customers cannot be used with this PaymentIntent.\n\nIf [setup_future_usage](https://api.stripe.com#payment_intent_object-setup_future_usage) is set and this PaymentIntent's payment method is not `card_present`, then the payment method attaches to the Customer after the PaymentIntent has been confirmed and any required actions from the user are complete. If the payment method is `card_present` and isn't a digital wallet, then a [generated_card](https://docs.stripe.com/api/charges/object#charge_object-payment_method_details-card_present-generated_card) payment method representing the card is created and attached to the Customer instead.",
		},
	},
}

var V1PaymentIntentsApplyCustomerBalance = resource.OperationSpec{
	Name:    "apply_customer_balance",
	Path:    "/v1/payment_intents/{intent}/apply_customer_balance",
	Method:  "POST",
	Summary: "Reconcile a customer_balance PaymentIntent",
	Params: map[string]*resource.ParamSpec{
		"amount": {
			Type:        "integer",
			Description: "Amount that you intend to apply to this PaymentIntent from the customer’s cash balance. If the PaymentIntent was created by an Invoice, the full amount of the PaymentIntent is applied regardless of this parameter.\n\nA positive integer representing how much to charge in the [smallest currency unit](https://docs.stripe.com/currencies#zero-decimal) (for example, 100 cents to charge 1 USD or 100 to charge 100 JPY, a zero-decimal currency). The maximum amount is the amount of the PaymentIntent.\n\nWhen you omit the amount, it defaults to the remaining amount requested on the PaymentIntent.",
		},
		"currency": {
			Type:        "string",
			Description: "Three-letter [ISO currency code](https://www.iso.org/iso-4217-currency-codes.html), in lowercase. Must be a [supported currency](https://stripe.com/docs/currencies).",
			Format:      "currency",
		},
	},
}

var V1PaymentIntentsCancel = resource.OperationSpec{
	Name:    "cancel",
	Path:    "/v1/payment_intents/{intent}/cancel",
	Method:  "POST",
	Summary: "Cancel a PaymentIntent",
	Params: map[string]*resource.ParamSpec{
		"cancellation_reason": {
			Type:        "string",
			Description: "Reason for canceling this PaymentIntent. Possible values are: `duplicate`, `fraudulent`, `requested_by_customer`, or `abandoned`",
			Enum: []resource.EnumSpec{
				{Value: "abandoned"},
				{Value: "duplicate"},
				{Value: "fraudulent"},
				{Value: "requested_by_customer"},
			},
		},
	},
}

var V1PaymentIntentsCapture = resource.OperationSpec{
	Name:    "capture",
	Path:    "/v1/payment_intents/{intent}/capture",
	Method:  "POST",
	Summary: "Capture a PaymentIntent",
	Params: map[string]*resource.ParamSpec{
		"transfer_data.amount": {
			Type:        "integer",
			Description: "The amount that will be transferred automatically when a charge succeeds.",
		},
		"amount_details.enforce_arithmetic_validation": {
			Type:        "boolean",
			Description: "Set to `false` to return arithmetic validation errors in the response without failing the request. Use this when you want the operation to proceed regardless of arithmetic errors in the line item data.\n\nOmit or set to `true` to immediately return a 400 error when arithmetic validation fails. Use this for strict validation that prevents processing with line item data that has arithmetic inconsistencies.\n\nFor card payments, Stripe doesn't send line item data to card networks if there's an arithmetic validation error.",
		},
		"amount_details.discount_amount": {
			Type:        "integer",
			Description: "The total discount applied on the transaction represented in the [smallest currency unit](https://docs.stripe.com/currencies#zero-decimal). An integer greater than 0.\n\nThis field is mutually exclusive with the `amount_details[line_items][#][discount_amount]` field.",
		},
		"application_fee_amount": {
			Type:        "integer",
			Description: "The amount of the application fee (if any) that will be requested to be applied to the payment and transferred to the application owner's Stripe account. The amount of the application fee collected will be capped at the total amount captured. For more information, see the PaymentIntents [use case for connected accounts](https://docs.stripe.com/payments/connected-accounts).",
		},
		"amount_to_capture": {
			Type:        "integer",
			Description: "The amount to capture from the PaymentIntent, which must be less than or equal to the original amount. Defaults to the full `amount_capturable` if it's not provided.",
		},
		"final_capture": {
			Type:        "boolean",
			Description: "Defaults to `true`. When capturing a PaymentIntent, setting `final_capture` to `false` notifies Stripe to not release the remaining uncaptured funds to make sure that they're captured in future requests. You can only use this setting when [multicapture](https://docs.stripe.com/payments/multicapture) is available for PaymentIntents.",
		},
		"hooks.inputs.tax.calculation": {
			Type:        "string",
			Description: "The [TaxCalculation](https://docs.stripe.com/api/tax/calculations) id",
			Required:    true,
		},
		"statement_descriptor": {
			Type:        "string",
			Description: "Text that appears on the customer's statement as the statement descriptor for a non-card charge. This value overrides the account's default statement descriptor. For information about requirements, including the 22-character limit, see [the Statement Descriptor docs](https://docs.stripe.com/get-started/account/statement-descriptors).\n\nSetting this value for a card charge returns an error. For card charges, set the [statement_descriptor_suffix](https://docs.stripe.com/get-started/account/statement-descriptors#dynamic) instead.",
		},
		"statement_descriptor_suffix": {
			Type:        "string",
			Description: "Provides information about a card charge. Concatenated to the account's [statement descriptor prefix](https://docs.stripe.com/get-started/account/statement-descriptors#static) to form the complete statement descriptor that appears on the customer's statement.",
		},
	},
}

var V1PaymentIntentsConfirm = resource.OperationSpec{
	Name:    "confirm",
	Path:    "/v1/payment_intents/{intent}/confirm",
	Method:  "POST",
	Summary: "Confirm a PaymentIntent",
	Params: map[string]*resource.ParamSpec{
		"payment_method_data.billing_details.name": {
			Type:        "string",
			Description: "Full name.",
		},
		"payment_method_data.billing_details.tax_id": {
			Type:        "string",
			Description: "Taxpayer identification number. Used only for transactions between LATAM buyers and non-LATAM sellers.",
		},
		"payment_method_data.type": {
			Type:        "string",
			Description: "The type of the PaymentMethod. An additional hash is included on the PaymentMethod with a name matching this value. It contains additional information specific to the PaymentMethod type.",
			Required:    true,
			Enum: []resource.EnumSpec{
				{Value: "acss_debit"},
				{Value: "affirm"},
				{Value: "afterpay_clearpay"},
				{Value: "alipay"},
				{Value: "alma"},
				{Value: "amazon_pay"},
				{Value: "au_becs_debit"},
				{Value: "bacs_debit"},
				{Value: "bancontact"},
				{Value: "billie"},
				{Value: "blik"},
				{Value: "boleto"},
				{Value: "cashapp"},
				{Value: "crypto"},
				{Value: "customer_balance"},
				{Value: "eps"},
				{Value: "fpx"},
				{Value: "giropay"},
				{Value: "grabpay"},
				{Value: "ideal"},
				{Value: "kakao_pay"},
				{Value: "klarna"},
				{Value: "konbini"},
				{Value: "kr_card"},
				{Value: "link"},
				{Value: "mb_way"},
				{Value: "mobilepay"},
				{Value: "multibanco"},
				{Value: "naver_pay"},
				{Value: "nz_bank_account"},
				{Value: "oxxo"},
				{Value: "p24"},
				{Value: "pay_by_bank"},
				{Value: "payco"},
				{Value: "paynow"},
				{Value: "paypal"},
				{Value: "payto"},
				{Value: "pix"},
				{Value: "promptpay"},
				{Value: "revolut_pay"},
				{Value: "samsung_pay"},
				{Value: "satispay"},
				{Value: "sepa_debit"},
				{Value: "sofort"},
				{Value: "swish"},
				{Value: "twint"},
				{Value: "us_bank_account"},
				{Value: "wechat_pay"},
				{Value: "zip"},
			},
		},
		"radar_options.session": {
			Type:        "string",
			Description: "A [Radar Session](https://docs.stripe.com/radar/radar-session) is a snapshot of the browser metadata and device details that help Radar make more accurate predictions on your payments.",
		},
		"capture_method": {
			Type:        "string",
			Description: "Controls when the funds will be captured from the customer's account.",
			Enum: []resource.EnumSpec{
				{Value: "automatic"},
				{Value: "automatic_async"},
				{Value: "manual"},
			},
		},
		"payment_method_data.klarna.dob.month": {
			Type:        "integer",
			Description: "The month of birth, between 1 and 12.",
			Required:    true,
		},
		"payment_method_data.us_bank_account.account_type": {
			Type:        "string",
			Description: "Account type: checkings or savings. Defaults to checking if omitted.",
			Enum: []resource.EnumSpec{
				{Value: "checking"},
				{Value: "savings"},
			},
		},
		"payment_method_data.fpx.account_holder_type": {
			Type:        "string",
			Description: "Account holder type for FPX transaction",
			Enum: []resource.EnumSpec{
				{Value: "company"},
				{Value: "individual"},
			},
		},
		"payment_method_data.fpx.bank": {
			Type:        "string",
			Description: "The customer's bank.",
			Required:    true,
			Enum: []resource.EnumSpec{
				{Value: "affin_bank"},
				{Value: "agrobank"},
				{Value: "alliance_bank"},
				{Value: "ambank"},
				{Value: "bank_islam"},
				{Value: "bank_muamalat"},
				{Value: "bank_of_china"},
				{Value: "bank_rakyat"},
				{Value: "bsn"},
				{Value: "cimb"},
				{Value: "deutsche_bank"},
				{Value: "hong_leong_bank"},
				{Value: "hsbc"},
				{Value: "kfh"},
				{Value: "maybank2e"},
				{Value: "maybank2u"},
				{Value: "ocbc"},
				{Value: "pb_enterprise"},
				{Value: "public_bank"},
				{Value: "rhb"},
				{Value: "standard_chartered"},
				{Value: "uob"},
			},
		},
		"payment_method_data.nz_bank_account.suffix": {
			Type:        "string",
			Description: "The suffix of the bank account number.",
			Required:    true,
		},
		"hooks.inputs.tax.calculation": {
			Type:        "string",
			Description: "The [TaxCalculation](https://docs.stripe.com/api/tax/calculations) id",
			Required:    true,
		},
		"payment_method_data.us_bank_account.routing_number": {
			Type:        "string",
			Description: "Routing number of the bank account.",
		},
		"payment_method_data.acss_debit.account_number": {
			Type:        "string",
			Description: "Customer's bank account number.",
			Required:    true,
		},
		"payment_method_data.sepa_debit.iban": {
			Type:        "string",
			Description: "IBAN of the bank account.",
			Required:    true,
		},
		"payment_method_data.p24.bank": {
			Type:        "string",
			Description: "The customer's bank.",
			Enum: []resource.EnumSpec{
				{Value: "alior_bank"},
				{Value: "bank_millennium"},
				{Value: "bank_nowy_bfg_sa"},
				{Value: "bank_pekao_sa"},
				{Value: "banki_spbdzielcze"},
				{Value: "blik"},
				{Value: "bnp_paribas"},
				{Value: "boz"},
				{Value: "citi_handlowy"},
				{Value: "credit_agricole"},
				{Value: "envelobank"},
				{Value: "etransfer_pocztowy24"},
				{Value: "getin_bank"},
				{Value: "ideabank"},
				{Value: "ing"},
				{Value: "inteligo"},
				{Value: "mbank_mtransfer"},
				{Value: "nest_przelew"},
				{Value: "noble_pay"},
				{Value: "pbac_z_ipko"},
				{Value: "plus_bank"},
				{Value: "santander_przelew24"},
				{Value: "tmobile_usbugi_bankowe"},
				{Value: "toyota_bank"},
				{Value: "velobank"},
				{Value: "volkswagen_bank"},
			},
		},
		"payment_method_data.payto.bsb_number": {
			Type:        "string",
			Description: "Bank-State-Branch number of the bank account.",
		},
		"payment_method_data.nz_bank_account.branch_code": {
			Type:        "string",
			Description: "The numeric code for the bank account's bank branch.",
			Required:    true,
		},
		"payment_method_data.nz_bank_account.reference": {
			Type: "string",
		},
		"excluded_payment_method_types": {
			Type:        "array",
			Description: "The list of payment method types to exclude from use with this payment.",
		},
		"payment_method_data.boleto.tax_id": {
			Type:        "string",
			Description: "The tax ID of the customer (CPF for individual consumers or CNPJ for businesses consumers)",
			Required:    true,
		},
		"payment_method_data.radar_options.session": {
			Type:        "string",
			Description: "A [Radar Session](https://docs.stripe.com/radar/radar-session) is a snapshot of the browser metadata and device details that help Radar make more accurate predictions on your payments.",
		},
		"payment_method_data.allow_redisplay": {
			Type:        "string",
			Description: "This field indicates whether this payment method can be shown again to its customer in a checkout flow. Stripe products such as Checkout and Elements use this field to determine whether a payment method can be shown as a saved payment method in a checkout flow. The field defaults to `unspecified`.",
			Enum: []resource.EnumSpec{
				{Value: "always"},
				{Value: "limited"},
				{Value: "unspecified"},
			},
		},
		"payment_method_data.us_bank_account.account_holder_type": {
			Type:        "string",
			Description: "Account holder type: individual or company.",
			Enum: []resource.EnumSpec{
				{Value: "company"},
				{Value: "individual"},
			},
		},
		"payment_method_data.acss_debit.transit_number": {
			Type:        "string",
			Description: "Transit number of the customer's bank.",
			Required:    true,
		},
		"payment_method_data.billing_details.phone": {
			Type:        "string",
			Description: "Billing phone number (including extension).",
		},
		"payment_method_data.eps.bank": {
			Type:        "string",
			Description: "The customer's bank.",
			Enum: []resource.EnumSpec{
				{Value: "arzte_und_apotheker_bank"},
				{Value: "austrian_anadi_bank_ag"},
				{Value: "bank_austria"},
				{Value: "bankhaus_carl_spangler"},
				{Value: "bankhaus_schelhammer_und_schattera_ag"},
				{Value: "bawag_psk_ag"},
				{Value: "bks_bank_ag"},
				{Value: "brull_kallmus_bank_ag"},
				{Value: "btv_vier_lander_bank"},
				{Value: "capital_bank_grawe_gruppe_ag"},
				{Value: "deutsche_bank_ag"},
				{Value: "dolomitenbank"},
				{Value: "easybank_ag"},
				{Value: "erste_bank_und_sparkassen"},
				{Value: "hypo_alpeadriabank_international_ag"},
				{Value: "hypo_bank_burgenland_aktiengesellschaft"},
				{Value: "hypo_noe_lb_fur_niederosterreich_u_wien"},
				{Value: "hypo_oberosterreich_salzburg_steiermark"},
				{Value: "hypo_tirol_bank_ag"},
				{Value: "hypo_vorarlberg_bank_ag"},
				{Value: "marchfelder_bank"},
				{Value: "oberbank_ag"},
				{Value: "raiffeisen_bankengruppe_osterreich"},
				{Value: "schoellerbank_ag"},
				{Value: "sparda_bank_wien"},
				{Value: "volksbank_gruppe"},
				{Value: "volkskreditbank_ag"},
				{Value: "vr_bank_braunau"},
			},
		},
		"mandate": {
			Type:        "string",
			Description: "ID of the mandate that's used for this payment.",
		},
		"payment_method_data.payto.pay_id": {
			Type:        "string",
			Description: "The PayID alias for the bank account.",
		},
		"payment_method_data.nz_bank_account.account_number": {
			Type:        "string",
			Description: "The account number for the bank account.",
			Required:    true,
		},
		"payment_method_data.nz_bank_account.bank_code": {
			Type:        "string",
			Description: "The numeric code for the bank account's bank.",
			Required:    true,
		},
		"payment_method_data.nz_bank_account.account_holder_name": {
			Type:        "string",
			Description: "The name on the bank account. Only required if the account holder name is different from the name of the authorized signatory collected in the PaymentMethod’s billing details.",
		},
		"payment_method": {
			Type:        "string",
			Description: "ID of the payment method (a PaymentMethod, Card, or [compatible Source](https://docs.stripe.com/payments/payment-methods/transitioning#compatibility) object) to attach to this PaymentIntent.\nIf the payment method is attached to a Customer, it must match the [customer](https://api.stripe.com#create_payment_intent-customer) that is set on this PaymentIntent.",
		},
		"payment_method_data.sofort.country": {
			Type:        "string",
			Description: "Two-letter ISO code representing the country the bank account is located in.",
			Required:    true,
			Enum: []resource.EnumSpec{
				{Value: "AT"},
				{Value: "BE"},
				{Value: "DE"},
				{Value: "ES"},
				{Value: "IT"},
				{Value: "NL"},
			},
		},
		"confirmation_token": {
			Type:        "string",
			Description: "ID of the ConfirmationToken used to confirm this PaymentIntent.\n\nIf the provided ConfirmationToken contains properties that are also being provided in this request, such as `payment_method`, then the values in this request will take precedence.",
		},
		"error_on_requires_action": {
			Type:        "boolean",
			Description: "Set to `true` to fail the payment attempt if the PaymentIntent transitions into `requires_action`. This parameter is intended for simpler integrations that do not handle customer actions, like [saving cards without authentication](https://docs.stripe.com/payments/save-card-without-authentication).",
		},
		"payment_method_data.klarna.dob.year": {
			Type:        "integer",
			Description: "The four-digit year of birth.",
			Required:    true,
		},
		"payment_method_data.us_bank_account.account_number": {
			Type:        "string",
			Description: "Account number of the bank account.",
		},
		"payment_method_data.us_bank_account.financial_connections_account": {
			Type:        "string",
			Description: "The ID of a Financial Connections Account to use as a payment method.",
		},
		"payment_method_data.au_becs_debit.account_number": {
			Type:        "string",
			Description: "The account number for the bank account.",
			Required:    true,
		},
		"payment_method_data.au_becs_debit.bsb_number": {
			Type:        "string",
			Description: "Bank-State-Branch number of the bank account.",
			Required:    true,
		},
		"return_url": {
			Type:        "string",
			Description: "The URL to redirect your customer back to after they authenticate or cancel their payment on the payment method's app or site.\nIf you'd prefer to redirect to a mobile application, you can alternatively supply an application URI scheme.\nThis parameter is only used for cards and other redirect-based payment methods.",
		},
		"payment_method_data.klarna.dob.day": {
			Type:        "integer",
			Description: "The day of birth, between 1 and 31.",
			Required:    true,
		},
		"payment_method_data.ideal.bank": {
			Type:        "string",
			Description: "The customer's bank. Only use this parameter for existing customers. Don't use it for new customers.",
			Enum: []resource.EnumSpec{
				{Value: "abn_amro"},
				{Value: "adyen"},
				{Value: "asn_bank"},
				{Value: "bunq"},
				{Value: "buut"},
				{Value: "finom"},
				{Value: "handelsbanken"},
				{Value: "ing"},
				{Value: "knab"},
				{Value: "mollie"},
				{Value: "moneyou"},
				{Value: "n26"},
				{Value: "nn"},
				{Value: "rabobank"},
				{Value: "regiobank"},
				{Value: "revolut"},
				{Value: "sns_bank"},
				{Value: "triodos_bank"},
				{Value: "van_lanschot"},
				{Value: "yoursafe"},
			},
		},
		"payment_method_data.payto.account_number": {
			Type:        "string",
			Description: "The account number for the bank account.",
		},
		"use_stripe_sdk": {
			Type:        "boolean",
			Description: "Set to `true` when confirming server-side and using Stripe.js, iOS, or Android client-side SDKs to handle the next actions.",
		},
		"payment_method_data.bacs_debit.sort_code": {
			Type:        "string",
			Description: "Sort code of the bank account. (e.g., `10-20-30`)",
		},
		"setup_future_usage": {
			Type:        "string",
			Description: "Indicates that you intend to make future payments with this PaymentIntent's payment method.\n\nIf you provide a Customer with the PaymentIntent, you can use this parameter to [attach the payment method](/payments/save-during-payment) to the Customer after the PaymentIntent is confirmed and the customer completes any required actions. If you don't provide a Customer, you can still [attach](/api/payment_methods/attach) the payment method to a Customer after the transaction completes.\n\nIf the payment method is `card_present` and isn't a digital wallet, Stripe creates and attaches a [generated_card](/api/charges/object#charge_object-payment_method_details-card_present-generated_card) payment method representing the card to the Customer instead.\n\nWhen processing card payments, Stripe uses `setup_future_usage` to help you comply with regional legislation and network rules, such as [SCA](/strong-customer-authentication).\n\nIf you've already set `setup_future_usage` and you're performing a request using a publishable key, you can only update the value from `on_session` to `off_session`.",
			Enum: []resource.EnumSpec{
				{Value: "off_session"},
				{Value: "on_session"},
			},
		},
		"payment_method_data.acss_debit.institution_number": {
			Type:        "string",
			Description: "Institution number of the customer's bank.",
			Required:    true,
		},
		"payment_method_data.billing_details.email": {
			Type:        "string",
			Description: "Email address.",
		},
		"payment_method_data.naver_pay.funding": {
			Type:        "string",
			Description: "Whether to use Naver Pay points or a card to fund this transaction. If not provided, this defaults to `card`.",
			Enum: []resource.EnumSpec{
				{Value: "card"},
				{Value: "points"},
			},
		},
		"payment_method_types": {
			Type:        "array",
			Description: "The list of payment method types (for example, a card) that this PaymentIntent can use. Use `automatic_payment_methods` to manage payment methods from the [Stripe Dashboard](https://dashboard.stripe.com/settings/payment_methods). A list of valid payment method types can be found [here](https://docs.stripe.com/api/payment_methods/object#payment_method_object-type).",
		},
		"receipt_email": {
			Type:        "string",
			Description: "Email address that the receipt for the resulting payment will be sent to. If `receipt_email` is specified for a payment in live mode, a receipt will be sent regardless of your [email settings](https://dashboard.stripe.com/account/emails).",
		},
		"off_session": {
			Type:        "boolean",
			Description: "Set to `true` to indicate that the customer isn't in your checkout flow during this payment attempt and can't authenticate. Use this parameter in scenarios where you collect card details and [charge them later](https://docs.stripe.com/payments/cards/charging-saved-cards).",
		},
		"payment_method_data.bacs_debit.account_number": {
			Type:        "string",
			Description: "Account number of the bank account that the funds will be debited from.",
		},
	},
}

var V1PaymentIntentsVerifyMicrodeposits = resource.OperationSpec{
	Name:    "verify_microdeposits",
	Path:    "/v1/payment_intents/{intent}/verify_microdeposits",
	Method:  "POST",
	Summary: "Verify microdeposits on a PaymentIntent",
	Params: map[string]*resource.ParamSpec{
		"amounts": {
			Type:        "array",
			Description: "Two positive integers, in *cents*, equal to the values of the microdeposits sent to the bank account.",
		},
		"descriptor_code": {
			Type:        "string",
			Description: "A six-character code starting with SM present in the microdeposit sent to the bank account.",
		},
	},
}

var V1PaymentIntentAmountDetailsLineItemsList = resource.OperationSpec{
	Name:    "list",
	Path:    "/v1/payment_intents/{intent}/amount_details_line_items",
	Method:  "GET",
	Summary: "List all PaymentIntent LineItems",
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

var V1QuotesList = resource.OperationSpec{
	Name:    "list",
	Path:    "/v1/quotes",
	Method:  "GET",
	Summary: "List all quotes",
	Params: map[string]*resource.ParamSpec{
		"test_clock": {
			Type:        "string",
			Description: "Provides a list of quotes that are associated with the specified test clock. The response will not include quotes with test clocks if this and the customer parameter is not set.",
		},
		"customer": {
			Type:        "string",
			Description: "The ID of the customer whose quotes you're retrieving.",
		},
		"customer_account": {
			Type:        "string",
			Description: "The ID of the account representing the customer whose quotes you're retrieving.",
		},
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
		"status": {
			Type:        "string",
			Description: "The status of the quote.",
			Enum: []resource.EnumSpec{
				{Value: "accepted"},
				{Value: "canceled"},
				{Value: "draft"},
				{Value: "open"},
			},
		},
	},
}

var V1QuotesListLineItems = resource.OperationSpec{
	Name:    "list_line_items",
	Path:    "/v1/quotes/{quote}/line_items",
	Method:  "GET",
	Summary: "Retrieve a quote's line items",
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

var V1QuotesCreate = resource.OperationSpec{
	Name:    "create",
	Path:    "/v1/quotes",
	Method:  "POST",
	Summary: "Create a quote",
	Params: map[string]*resource.ParamSpec{
		"invoice_settings.days_until_due": {
			Type:        "integer",
			Description: "Number of days within which a customer must pay the invoice generated by this quote. This value will be `null` for quotes where `collection_method=charge_automatically`.",
		},
		"application_fee_percent": {
			Type:        "number",
			Description: "A non-negative decimal between 0 and 100, with at most two decimal places. This represents the percentage of the subscription invoice total that will be transferred to the application owner's Stripe account. There must be at least 1 line item with a recurring price to use this field.",
		},
		"header": {
			Type:        "string",
			Description: "A header that will be displayed on the quote PDF. If no value is passed, the default header configured in your [quote template settings](https://dashboard.stripe.com/settings/billing/quote) will be used.",
		},
		"application_fee_amount": {
			Type:        "integer",
			Description: "The amount of the application fee (if any) that will be requested to be applied to the payment and transferred to the application owner's Stripe account. There cannot be any line items with recurring prices when using this field.",
		},
		"description": {
			Type:        "string",
			Description: "A description that will be displayed on the quote PDF. If no value is passed, the default description configured in your [quote template settings](https://dashboard.stripe.com/settings/billing/quote) will be used.",
		},
		"expires_at": {
			Type:        "integer",
			Description: "A future timestamp on which the quote will be canceled if in `open` or `draft` status. Measured in seconds since the Unix epoch. If no value is passed, the default expiration date configured in your [quote template settings](https://dashboard.stripe.com/settings/billing/quote) will be used.",
			Format:      "unix-time",
		},
		"from_quote.quote": {
			Type:        "string",
			Description: "The `id` of the quote that will be cloned.",
			Required:    true,
		},
		"automatic_tax.liability.account": {
			Type:        "string",
			Description: "The connected account being referenced when `type` is `account`.",
		},
		"invoice_settings.issuer.account": {
			Type:        "string",
			Description: "The connected account being referenced when `type` is `account`.",
		},
		"invoice_settings.issuer.type": {
			Type:        "string",
			Description: "Type of the account referenced in the request.",
			Required:    true,
			Enum: []resource.EnumSpec{
				{Value: "account"},
				{Value: "self"},
			},
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
		"customer_account": {
			Type:        "string",
			Description: "The account for which this quote belongs to. A customer or account is required before finalizing the quote. Once specified, it cannot be changed.",
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
		"subscription_data.description": {
			Type:        "string",
			Description: "The subscription's description, meant to be displayable to the customer. Use this field to optionally store an explanation of the subscription for rendering in Stripe surfaces and certain local payment methods UIs.",
		},
		"subscription_data.trial_period_days": {
			Type:        "integer",
			Description: "Integer representing the number of trial period days before the customer is charged for the first time.",
		},
		"footer": {
			Type:        "string",
			Description: "A footer that will be displayed on the quote PDF. If no value is passed, the default footer configured in your [quote template settings](https://dashboard.stripe.com/settings/billing/quote) will be used.",
		},
		"on_behalf_of": {
			Type:        "string",
			Description: "The account on behalf of which to charge.",
		},
		"customer": {
			Type:        "string",
			Description: "The customer for which this quote belongs to. A customer is required before finalizing the quote. Once specified, it cannot be changed.",
		},
		"collection_method": {
			Type:        "string",
			Description: "Either `charge_automatically`, or `send_invoice`. When charging automatically, Stripe will attempt to pay invoices at the end of the subscription cycle or at invoice finalization using the default payment method attached to the subscription or customer. When sending an invoice, Stripe will email your customer an invoice with payment instructions and mark the subscription as `active`. Defaults to `charge_automatically`.",
			Enum: []resource.EnumSpec{
				{Value: "charge_automatically"},
				{Value: "send_invoice"},
			},
		},
		"subscription_data.billing_mode.flexible.proration_discounts": {
			Type:        "string",
			Description: "Controls how invoices and invoice items display proration amounts and discount amounts.",
			Enum: []resource.EnumSpec{
				{Value: "included"},
				{Value: "itemized"},
			},
		},
		"from_quote.is_revision": {
			Type:        "boolean",
			Description: "Whether this quote is a revision of the previous quote.",
		},
		"default_tax_rates": {
			Type:        "array",
			Description: "The tax rates that will apply to any line item that does not have `tax_rates` set.",
		},
		"test_clock": {
			Type:        "string",
			Description: "ID of the test clock to attach to the quote.",
		},
		"automatic_tax.enabled": {
			Type:        "boolean",
			Description: "Controls whether Stripe will automatically compute tax on the resulting invoices or subscriptions as well as the quote itself.",
			Required:    true,
		},
		"subscription_data.effective_date": {
			Type:        "string",
			Description: "When creating a new subscription, the date of which the subscription schedule will start after the quote is accepted. The `effective_date` is ignored if it is in the past when the quote is accepted.",
		},
	},
}

var V1QuotesUpdate = resource.OperationSpec{
	Name:    "update",
	Path:    "/v1/quotes/{quote}",
	Method:  "POST",
	Summary: "Update a quote",
	Params: map[string]*resource.ParamSpec{
		"application_fee_percent": {
			Type:        "number",
			Description: "A non-negative decimal between 0 and 100, with at most two decimal places. This represents the percentage of the subscription invoice total that will be transferred to the application owner's Stripe account. There must be at least 1 line item with a recurring price to use this field.",
		},
		"subscription_data.effective_date": {
			Type:        "string",
			Description: "When creating a new subscription, the date of which the subscription schedule will start after the quote is accepted. The `effective_date` is ignored if it is in the past when the quote is accepted.",
		},
		"invoice_settings.days_until_due": {
			Type:        "integer",
			Description: "Number of days within which a customer must pay the invoice generated by this quote. This value will be `null` for quotes where `collection_method=charge_automatically`.",
		},
		"expires_at": {
			Type:        "integer",
			Description: "A future timestamp on which the quote will be canceled if in `open` or `draft` status. Measured in seconds since the Unix epoch.",
			Format:      "unix-time",
		},
		"footer": {
			Type:        "string",
			Description: "A footer that will be displayed on the quote PDF.",
		},
		"on_behalf_of": {
			Type:        "string",
			Description: "The account on behalf of which to charge.",
		},
		"automatic_tax.enabled": {
			Type:        "boolean",
			Description: "Controls whether Stripe will automatically compute tax on the resulting invoices or subscriptions as well as the quote itself.",
			Required:    true,
		},
		"automatic_tax.liability.account": {
			Type:        "string",
			Description: "The connected account being referenced when `type` is `account`.",
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
		"subscription_data.description": {
			Type:        "string",
			Description: "The subscription's description, meant to be displayable to the customer. Use this field to optionally store an explanation of the subscription for rendering in Stripe surfaces and certain local payment methods UIs.",
		},
		"customer": {
			Type:        "string",
			Description: "The customer for which this quote belongs to. A customer is required before finalizing the quote. Once specified, it cannot be changed.",
		},
		"header": {
			Type:        "string",
			Description: "A header that will be displayed on the quote PDF.",
		},
		"subscription_data.trial_period_days": {
			Type:        "integer",
			Description: "Integer representing the number of trial period days before the customer is charged for the first time.",
		},
		"invoice_settings.issuer.type": {
			Type:        "string",
			Description: "Type of the account referenced in the request.",
			Required:    true,
			Enum: []resource.EnumSpec{
				{Value: "account"},
				{Value: "self"},
			},
		},
		"customer_account": {
			Type:        "string",
			Description: "The account for which this quote belongs to. A customer or account is required before finalizing the quote. Once specified, it cannot be changed.",
		},
		"description": {
			Type:        "string",
			Description: "A description that will be displayed on the quote PDF.",
		},
		"application_fee_amount": {
			Type:        "integer",
			Description: "The amount of the application fee (if any) that will be requested to be applied to the payment and transferred to the application owner's Stripe account. There cannot be any line items with recurring prices when using this field.",
		},
		"default_tax_rates": {
			Type:        "array",
			Description: "The tax rates that will apply to any line item that does not have `tax_rates` set.",
		},
		"invoice_settings.issuer.account": {
			Type:        "string",
			Description: "The connected account being referenced when `type` is `account`.",
		},
		"collection_method": {
			Type:        "string",
			Description: "Either `charge_automatically`, or `send_invoice`. When charging automatically, Stripe will attempt to pay invoices at the end of the subscription cycle or at invoice finalization using the default payment method attached to the subscription or customer. When sending an invoice, Stripe will email your customer an invoice with payment instructions and mark the subscription as `active`. Defaults to `charge_automatically`.",
			Enum: []resource.EnumSpec{
				{Value: "charge_automatically"},
				{Value: "send_invoice"},
			},
		},
	},
}

var V1QuotesFinalizeQuote = resource.OperationSpec{
	Name:    "finalize_quote",
	Path:    "/v1/quotes/{quote}/finalize",
	Method:  "POST",
	Summary: "Finalize a quote",
	Params: map[string]*resource.ParamSpec{
		"expires_at": {
			Type:        "integer",
			Description: "A future timestamp on which the quote will be canceled if in `open` or `draft` status. Measured in seconds since the Unix epoch.",
			Format:      "unix-time",
		},
	},
}

var V1QuotesRetrieve = resource.OperationSpec{
	Name:    "retrieve",
	Path:    "/v1/quotes/{quote}",
	Method:  "GET",
	Summary: "Retrieve a quote",
}

var V1QuotesListComputedUpfrontLineItems = resource.OperationSpec{
	Name:    "list_computed_upfront_line_items",
	Path:    "/v1/quotes/{quote}/computed_upfront_line_items",
	Method:  "GET",
	Summary: "Retrieve a quote's upfront line items",
	Params: map[string]*resource.ParamSpec{
		"starting_after": {
			Type:        "string",
			Description: "A cursor for use in pagination. `starting_after` is an object ID that defines your place in the list. For instance, if you make a list request and receive 100 objects, ending with `obj_foo`, your subsequent call can include `starting_after=obj_foo` in order to fetch the next page of the list.",
		},
		"ending_before": {
			Type:        "string",
			Description: "A cursor for use in pagination. `ending_before` is an object ID that defines your place in the list. For instance, if you make a list request and receive 100 objects, starting with `obj_bar`, your subsequent call can include `ending_before=obj_bar` in order to fetch the previous page of the list.",
		},
		"limit": {
			Type:        "integer",
			Description: "A limit on the number of objects to be returned. Limit can range between 1 and 100, and the default is 10.",
		},
	},
}

var V1QuotesAccept = resource.OperationSpec{
	Name:    "accept",
	Path:    "/v1/quotes/{quote}/accept",
	Method:  "POST",
	Summary: "Accept a quote",
}

var V1QuotesCancel = resource.OperationSpec{
	Name:    "cancel",
	Path:    "/v1/quotes/{quote}/cancel",
	Method:  "POST",
	Summary: "Cancel a quote",
}

var V1QuotesPdf = resource.OperationSpec{
	Name:      "pdf",
	Path:      "/v1/quotes/{quote}/pdf",
	Method:    "GET",
	ServerURL: "https://files.stripe.com/",
	Summary:   "Download quote PDF",
}

var V1PaymentLinksList = resource.OperationSpec{
	Name:    "list",
	Path:    "/v1/payment_links",
	Method:  "GET",
	Summary: "List all payment links",
	Params: map[string]*resource.ParamSpec{
		"starting_after": {
			Type:        "string",
			Description: "A cursor for use in pagination. `starting_after` is an object ID that defines your place in the list. For instance, if you make a list request and receive 100 objects, ending with `obj_foo`, your subsequent call can include `starting_after=obj_foo` in order to fetch the next page of the list.",
		},
		"active": {
			Type:        "boolean",
			Description: "Only return payment links that are active or inactive (e.g., pass `false` to list all inactive payment links).",
		},
		"ending_before": {
			Type:        "string",
			Description: "A cursor for use in pagination. `ending_before` is an object ID that defines your place in the list. For instance, if you make a list request and receive 100 objects, starting with `obj_bar`, your subsequent call can include `ending_before=obj_bar` in order to fetch the previous page of the list.",
		},
		"limit": {
			Type:        "integer",
			Description: "A limit on the number of objects to be returned. Limit can range between 1 and 100, and the default is 10.",
		},
	},
}

var V1PaymentLinksRetrieve = resource.OperationSpec{
	Name:    "retrieve",
	Path:    "/v1/payment_links/{payment_link}",
	Method:  "GET",
	Summary: "Retrieve payment link",
}

var V1PaymentLinksListLineItems = resource.OperationSpec{
	Name:    "list_line_items",
	Path:    "/v1/payment_links/{payment_link}/line_items",
	Method:  "GET",
	Summary: "Retrieve a payment link's line items",
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

var V1PaymentLinksCreate = resource.OperationSpec{
	Name:    "create",
	Path:    "/v1/payment_links",
	Method:  "POST",
	Summary: "Create a payment link",
	Params: map[string]*resource.ParamSpec{
		"application_fee_percent": {
			Type:        "number",
			Description: "A non-negative decimal between 0 and 100, with at most two decimal places. This represents the percentage of the subscription invoice total that will be transferred to the application owner's Stripe account. There must be at least 1 line item with a recurring price to use this field.",
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
		"name_collection.individual.enabled": {
			Type:        "boolean",
			Description: "Enable individual name collection on the payment link. Defaults to `false`.",
			Required:    true,
		},
		"invoice_creation.invoice_data.issuer.account": {
			Type:        "string",
			Description: "The connected account being referenced when `type` is `account`.",
		},
		"currency": {
			Type:        "string",
			Description: "Three-letter [ISO currency code](https://www.iso.org/iso-4217-currency-codes.html), in lowercase. Must be a [supported currency](https://stripe.com/docs/currencies) and supported by each line item's price.",
			Format:      "currency",
		},
		"application_fee_amount": {
			Type:        "integer",
			Description: "The amount of the application fee (if any) that will be requested to be applied to the payment and transferred to the application owner's Stripe account. Can only be applied when there are no line items with recurring prices.",
		},
		"on_behalf_of": {
			Type:        "string",
			Description: "The account on behalf of which to charge.",
		},
		"automatic_tax.liability.account": {
			Type:        "string",
			Description: "The connected account being referenced when `type` is `account`.",
		},
		"inactive_message": {
			Type:        "string",
			Description: "The custom message to be displayed to a customer when a payment link is no longer active.",
		},
		"allow_promotion_codes": {
			Type:        "boolean",
			Description: "Enables user redeemable promotion codes.",
		},
		"after_completion.type": {
			Type:        "string",
			Description: "The specified behavior after the purchase is complete. Either `redirect` or `hosted_confirmation`.",
			Required:    true,
			Enum: []resource.EnumSpec{
				{Value: "hosted_confirmation"},
				{Value: "redirect"},
			},
		},
		"payment_intent_data.statement_descriptor": {
			Type:        "string",
			Description: "Text that appears on the customer's statement as the statement descriptor for a non-card charge. This value overrides the account's default statement descriptor. For information about requirements, including the 22-character limit, see [the Statement Descriptor docs](https://docs.stripe.com/get-started/account/statement-descriptors).\n\nSetting this value for a card charge returns an error. For card charges, set the [statement_descriptor_suffix](https://docs.stripe.com/get-started/account/statement-descriptors#dynamic) instead.",
		},
		"transfer_data.amount": {
			Type:        "integer",
			Description: "The amount that will be transferred automatically when a charge succeeds.",
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
		"consent_collection.terms_of_service": {
			Type:        "string",
			Description: "If set to `required`, it requires customers to check a terms of service checkbox before being able to pay.\nThere must be a valid terms of service URL set in your [Dashboard settings](https://dashboard.stripe.com/settings/public).",
			Enum: []resource.EnumSpec{
				{Value: "none"},
				{Value: "required"},
			},
		},
		"billing_address_collection": {
			Type:        "string",
			Description: "Configuration for collecting the customer's billing address. Defaults to `auto`.",
			Enum: []resource.EnumSpec{
				{Value: "auto"},
				{Value: "required"},
			},
		},
		"name_collection.business.enabled": {
			Type:        "boolean",
			Description: "Enable business name collection on the payment link. Defaults to `false`.",
			Required:    true,
		},
		"subscription_data.trial_period_days": {
			Type:        "integer",
			Description: "Integer representing the number of trial period days before the customer is charged for the first time. Has to be at least 1.",
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
		"transfer_data.destination": {
			Type:        "string",
			Description: "If specified, successful charges will be attributed to the destination\naccount for tax reporting, and the funds from charges will be transferred\nto the destination account. The ID of the resulting transfer will be\nreturned on the successful charge's `transfer` field.",
			Required:    true,
		},
		"tax_id_collection.required": {
			Type:        "string",
			Description: "Describes whether a tax ID is required during checkout. Defaults to `never`. You can't set this parameter if `ui_mode` is `custom`.",
			Enum: []resource.EnumSpec{
				{Value: "if_supported"},
				{Value: "never"},
			},
		},
		"payment_method_types": {
			Type:        "array",
			Description: "The list of payment method types that customers can use. If no value is passed, Stripe will dynamically show relevant payment methods from your [payment method settings](https://dashboard.stripe.com/settings/payment_methods) (20+ payment methods [supported](https://docs.stripe.com/payments/payment-methods/integration-options#payment-method-product-support)).",
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
		"submit_type": {
			Type:        "string",
			Description: "Describes the type of transaction being performed in order to customize relevant text on the page, such as the submit button. Changing this value will also affect the hostname in the [url](https://docs.stripe.com/api/payment_links/payment_links/object#url) property (example: `donate.stripe.com`).",
			Enum: []resource.EnumSpec{
				{Value: "auto"},
				{Value: "book"},
				{Value: "donate"},
				{Value: "pay"},
				{Value: "subscribe"},
			},
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
		"payment_method_collection": {
			Type:        "string",
			Description: "Specify whether Checkout should collect a payment method. When set to `if_required`, Checkout will not collect a payment method when the total due for the session is 0.This may occur if the Checkout Session includes a free trial or a discount.\n\nCan only be set in `subscription` mode. Defaults to `always`.\n\nIf you'd like information on how to collect a payment method outside of Checkout, read the guide on [configuring subscriptions with a free trial](https://docs.stripe.com/payments/checkout/free-trials).",
			Enum: []resource.EnumSpec{
				{Value: "always"},
				{Value: "if_required"},
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
		"shipping_address_collection.allowed_countries": {
			Type:        "array",
			Description: "An array of two-letter ISO country codes representing which countries Checkout should provide as options for\nshipping locations.",
			Required:    true,
		},
		"subscription_data.description": {
			Type:        "string",
			Description: "The subscription's description, meant to be displayable to the customer. Use this field to optionally store an explanation of the subscription for rendering in Stripe surfaces and certain local payment methods UIs.",
		},
		"invoice_creation.invoice_data.footer": {
			Type:        "string",
			Description: "Default footer to be displayed on invoices for this customer.",
		},
		"restrictions.completed_sessions.limit": {
			Type:        "integer",
			Description: "The maximum number of checkout sessions that can be completed for the `completed_sessions` restriction to be met.",
			Required:    true,
		},
		"after_completion.hosted_confirmation.custom_message": {
			Type:        "string",
			Description: "A custom message to display to the customer after the purchase is complete.",
		},
		"after_completion.redirect.url": {
			Type:        "string",
			Description: "The URL the customer will be redirected to after the purchase is complete. You can embed `{CHECKOUT_SESSION_ID}` into the URL to have the `id` of the completed [checkout session](https://docs.stripe.com/api/checkout/sessions/object#checkout_session_object-id) included.",
			Required:    true,
		},
		"payment_intent_data.statement_descriptor_suffix": {
			Type:        "string",
			Description: "Provides information about a card charge. Concatenated to the account's [statement descriptor prefix](https://docs.stripe.com/get-started/account/statement-descriptors#static) to form the complete statement descriptor that appears on the customer's statement.",
		},
		"payment_intent_data.transfer_group": {
			Type:        "string",
			Description: "A string that identifies the resulting payment as part of a group. See the PaymentIntents [use case for connected accounts](https://docs.stripe.com/connect/separate-charges-and-transfers) for details.",
		},
		"phone_number_collection.enabled": {
			Type:        "boolean",
			Description: "Set to `true` to enable phone number collection.",
			Required:    true,
		},
		"tax_id_collection.enabled": {
			Type:        "boolean",
			Description: "Enable tax ID collection during checkout. Defaults to `false`.",
			Required:    true,
		},
		"automatic_tax.enabled": {
			Type:        "boolean",
			Description: "Set to `true` to [calculate tax automatically](https://docs.stripe.com/tax) using the customer's location.\n\nEnabling this parameter causes the payment link to collect any billing address information necessary for tax calculation.",
			Required:    true,
		},
		"name_collection.business.optional": {
			Type:        "boolean",
			Description: "Whether the customer is required to provide their business name before checking out. Defaults to `false`.",
		},
		"name_collection.individual.optional": {
			Type:        "boolean",
			Description: "Whether the customer is required to provide their full name before checking out. Defaults to `false`.",
		},
		"customer_creation": {
			Type:        "string",
			Description: "Configures whether [checkout sessions](https://docs.stripe.com/api/checkout/sessions) created by this payment link create a [Customer](https://docs.stripe.com/api/customers).",
			Enum: []resource.EnumSpec{
				{Value: "always"},
				{Value: "if_required"},
			},
		},
		"invoice_creation.enabled": {
			Type:        "boolean",
			Description: "Whether the feature is enabled",
			Required:    true,
		},
		"invoice_creation.invoice_data.description": {
			Type:        "string",
			Description: "An arbitrary string attached to the object. Often useful for displaying to users.",
		},
		"invoice_creation.invoice_data.account_tax_ids": {
			Type:        "array",
			Description: "The account tax IDs associated with the invoice.",
		},
		"subscription_data.invoice_settings.issuer.account": {
			Type:        "string",
			Description: "The connected account being referenced when `type` is `account`.",
		},
		"payment_intent_data.description": {
			Type:        "string",
			Description: "An arbitrary string attached to the object. Often useful for displaying to users.",
		},
		"payment_intent_data.setup_future_usage": {
			Type:        "string",
			Description: "Indicates that you intend to [make future payments](https://docs.stripe.com/payments/payment-intents#future-usage) with the payment method collected by this Checkout Session.\n\nWhen setting this to `on_session`, Checkout will show a notice to the customer that their payment details will be saved.\n\nWhen setting this to `off_session`, Checkout will show a notice to the customer that their payment details will be saved and used for future payments.\n\nIf a Customer has been provided or Checkout creates a new Customer,Checkout will attach the payment method to the Customer.\n\nIf Checkout does not create a Customer, the payment method is not attached to a Customer. To reuse the payment method, you can retrieve it from the Checkout Session's PaymentIntent.\n\nWhen processing card payments, Checkout also uses `setup_future_usage` to dynamically optimize your payment flow and comply with regional legislation and network rules, such as SCA.",
			Enum: []resource.EnumSpec{
				{Value: "off_session"},
				{Value: "on_session"},
			},
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
	},
}

var V1PaymentLinksUpdate = resource.OperationSpec{
	Name:    "update",
	Path:    "/v1/payment_links/{payment_link}",
	Method:  "POST",
	Summary: "Update a payment link",
	Params: map[string]*resource.ParamSpec{
		"invoice_creation.invoice_data.issuer.type": {
			Type:        "string",
			Description: "Type of the account referenced in the request.",
			Required:    true,
			Enum: []resource.EnumSpec{
				{Value: "account"},
				{Value: "self"},
			},
		},
		"payment_method_collection": {
			Type:        "string",
			Description: "Specify whether Checkout should collect a payment method. When set to `if_required`, Checkout will not collect a payment method when the total due for the session is 0.This may occur if the Checkout Session includes a free trial or a discount.\n\nCan only be set in `subscription` mode. Defaults to `always`.\n\nIf you'd like information on how to collect a payment method outside of Checkout, read the guide on [configuring subscriptions with a free trial](https://docs.stripe.com/payments/checkout/free-trials).",
			Enum: []resource.EnumSpec{
				{Value: "always"},
				{Value: "if_required"},
			},
		},
		"after_completion.hosted_confirmation.custom_message": {
			Type:        "string",
			Description: "A custom message to display to the customer after the purchase is complete.",
		},
		"invoice_creation.invoice_data.footer": {
			Type:        "string",
			Description: "Default footer to be displayed on invoices for this customer.",
		},
		"allow_promotion_codes": {
			Type:        "boolean",
			Description: "Enables user redeemable promotion codes.",
		},
		"submit_type": {
			Type:        "string",
			Description: "Describes the type of transaction being performed in order to customize relevant text on the page, such as the submit button. Changing this value will also affect the hostname in the [url](https://docs.stripe.com/api/payment_links/payment_links/object#url) property (example: `donate.stripe.com`).",
			Enum: []resource.EnumSpec{
				{Value: "auto"},
				{Value: "book"},
				{Value: "donate"},
				{Value: "pay"},
				{Value: "subscribe"},
			},
		},
		"invoice_creation.enabled": {
			Type:        "boolean",
			Description: "Whether the feature is enabled",
			Required:    true,
		},
		"invoice_creation.invoice_data.issuer.account": {
			Type:        "string",
			Description: "The connected account being referenced when `type` is `account`.",
		},
		"tax_id_collection.enabled": {
			Type:        "boolean",
			Description: "Enable tax ID collection during checkout. Defaults to `false`.",
			Required:    true,
		},
		"automatic_tax.liability.account": {
			Type:        "string",
			Description: "The connected account being referenced when `type` is `account`.",
		},
		"payment_intent_data.description": {
			Type:        "string",
			Description: "An arbitrary string attached to the object. Often useful for displaying to users.",
		},
		"payment_intent_data.statement_descriptor": {
			Type:        "string",
			Description: "Text that appears on the customer's statement as the statement descriptor for a non-card charge. This value overrides the account's default statement descriptor. For information about requirements, including the 22-character limit, see [the Statement Descriptor docs](https://docs.stripe.com/get-started/account/statement-descriptors).\n\nSetting this value for a card charge returns an error. For card charges, set the [statement_descriptor_suffix](https://docs.stripe.com/get-started/account/statement-descriptors#dynamic) instead.",
		},
		"invoice_creation.invoice_data.description": {
			Type:        "string",
			Description: "An arbitrary string attached to the object. Often useful for displaying to users.",
		},
		"subscription_data.trial_period_days": {
			Type:        "integer",
			Description: "Integer representing the number of trial period days before the customer is charged for the first time. Has to be at least 1.",
		},
		"automatic_tax.enabled": {
			Type:        "boolean",
			Description: "Set to `true` to [calculate tax automatically](https://docs.stripe.com/tax) using the customer's location.\n\nEnabling this parameter causes the payment link to collect any billing address information necessary for tax calculation.",
			Required:    true,
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
		"customer_creation": {
			Type:        "string",
			Description: "Configures whether [checkout sessions](https://docs.stripe.com/api/checkout/sessions) created by this payment link create a [Customer](https://docs.stripe.com/api/customers).",
			Enum: []resource.EnumSpec{
				{Value: "always"},
				{Value: "if_required"},
			},
		},
		"invoice_creation.invoice_data.account_tax_ids": {
			Type:        "array",
			Description: "The account tax IDs associated with the invoice.",
		},
		"payment_method_types": {
			Type:        "array",
			Description: "The list of payment method types that customers can use. Pass an empty string to enable dynamic payment methods that use your [payment method settings](https://dashboard.stripe.com/settings/payment_methods).",
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
		"after_completion.redirect.url": {
			Type:        "string",
			Description: "The URL the customer will be redirected to after the purchase is complete. You can embed `{CHECKOUT_SESSION_ID}` into the URL to have the `id` of the completed [checkout session](https://docs.stripe.com/api/checkout/sessions/object#checkout_session_object-id) included.",
			Required:    true,
		},
		"tax_id_collection.required": {
			Type:        "string",
			Description: "Describes whether a tax ID is required during checkout. Defaults to `never`. You can't set this parameter if `ui_mode` is `custom`.",
			Enum: []resource.EnumSpec{
				{Value: "if_supported"},
				{Value: "never"},
			},
		},
		"subscription_data.invoice_settings.issuer.account": {
			Type:        "string",
			Description: "The connected account being referenced when `type` is `account`.",
		},
		"active": {
			Type:        "boolean",
			Description: "Whether the payment link's `url` is active. If `false`, customers visiting the URL will be shown a page saying that the link has been deactivated.",
		},
		"after_completion.type": {
			Type:        "string",
			Description: "The specified behavior after the purchase is complete. Either `redirect` or `hosted_confirmation`.",
			Required:    true,
			Enum: []resource.EnumSpec{
				{Value: "hosted_confirmation"},
				{Value: "redirect"},
			},
		},
		"payment_intent_data.statement_descriptor_suffix": {
			Type:        "string",
			Description: "Provides information about a card charge. Concatenated to the account's [statement descriptor prefix](https://docs.stripe.com/get-started/account/statement-descriptors#static) to form the complete statement descriptor that appears on the customer's statement.",
		},
		"phone_number_collection.enabled": {
			Type:        "boolean",
			Description: "Set to `true` to enable phone number collection.",
			Required:    true,
		},
		"payment_intent_data.transfer_group": {
			Type:        "string",
			Description: "A string that identifies the resulting payment as part of a group. See the PaymentIntents [use case for connected accounts](https://docs.stripe.com/connect/separate-charges-and-transfers) for details.",
		},
		"billing_address_collection": {
			Type:        "string",
			Description: "Configuration for collecting the customer's billing address. Defaults to `auto`.",
			Enum: []resource.EnumSpec{
				{Value: "auto"},
				{Value: "required"},
			},
		},
		"inactive_message": {
			Type:        "string",
			Description: "The custom message to be displayed to a customer when a payment link is no longer active.",
		},
	},
}

var V1SubscriptionSchedulesCreate = resource.OperationSpec{
	Name:    "create",
	Path:    "/v1/subscription_schedules",
	Method:  "POST",
	Summary: "Create a schedule",
	Params: map[string]*resource.ParamSpec{
		"customer": {
			Type:        "string",
			Description: "The identifier of the customer to create the subscription schedule for.",
		},
		"default_settings.description": {
			Type:        "string",
			Description: "Subscription description, meant to be displayable to the customer. Use this field to optionally store an explanation of the subscription for rendering in Stripe surfaces and certain local payment methods UIs.",
		},
		"default_settings.default_payment_method": {
			Type:        "string",
			Description: "ID of the default payment method for the subscription schedule. It must belong to the customer associated with the subscription schedule. If not set, invoices will use the default payment method in the customer's invoice settings.",
		},
		"end_behavior": {
			Type:        "string",
			Description: "Behavior of the subscription schedule and underlying subscription when it ends. Possible values are `release` or `cancel` with the default being `release`. `release` will end the subscription schedule and keep the underlying subscription running. `cancel` will end the subscription schedule and cancel the underlying subscription.",
			Enum: []resource.EnumSpec{
				{Value: "cancel"},
				{Value: "none"},
				{Value: "release"},
				{Value: "renew"},
			},
		},
		"start_date": {
			Type:        "integer",
			Description: "When the subscription schedule starts. We recommend using `now` so that it starts the subscription immediately. You can also use a Unix timestamp to backdate the subscription so that it starts on a past date, or set a future date for the subscription to start on.",
		},
		"customer_account": {
			Type:        "string",
			Description: "The identifier of the account to create the subscription schedule for.",
		},
		"default_settings.automatic_tax.liability.type": {
			Type:        "string",
			Description: "Type of the account referenced in the request.",
			Required:    true,
			Enum: []resource.EnumSpec{
				{Value: "account"},
				{Value: "self"},
			},
		},
		"default_settings.invoice_settings.account_tax_ids": {
			Type:        "array",
			Description: "The account tax IDs associated with the subscription schedule. Will be set on invoices generated by the subscription schedule.",
		},
		"default_settings.invoice_settings.days_until_due": {
			Type:        "integer",
			Description: "Number of days within which a customer must pay invoices generated by this subscription schedule. This value will be `null` for subscription schedules where `collection_method=charge_automatically`.",
		},
		"from_subscription": {
			Type:        "string",
			Description: "Migrate an existing subscription to be managed by a subscription schedule. If this parameter is set, a subscription schedule will be created using the subscription's item(s), set to auto-renew using the subscription's interval. When using this parameter, other parameters (such as phase values) cannot be set. To create a subscription schedule with other modifications, we recommend making two separate API calls.",
		},
		"billing_mode.flexible.proration_discounts": {
			Type:        "string",
			Description: "Controls how invoices and invoice items display proration amounts and discount amounts.",
			Enum: []resource.EnumSpec{
				{Value: "included"},
				{Value: "itemized"},
			},
		},
		"default_settings.collection_method": {
			Type:        "string",
			Description: "Either `charge_automatically`, or `send_invoice`. When charging automatically, Stripe will attempt to pay the underlying subscription at the end of each billing cycle using the default source attached to the customer. When sending an invoice, Stripe will email your customer an invoice with payment instructions and mark the subscription as `active`. Defaults to `charge_automatically` on creation.",
			Enum: []resource.EnumSpec{
				{Value: "charge_automatically"},
				{Value: "send_invoice"},
			},
		},
		"default_settings.on_behalf_of": {
			Type:        "string",
			Description: "The account on behalf of which to charge, for each of the associated subscription's invoices.",
		},
		"default_settings.automatic_tax.liability.account": {
			Type:        "string",
			Description: "The connected account being referenced when `type` is `account`.",
		},
		"default_settings.application_fee_percent": {
			Type:        "number",
			Description: "A non-negative decimal between 0 and 100, with at most two decimal places. This represents the percentage of the subscription invoice total that will be transferred to the application owner's Stripe account. The request must be made by a platform account on a connected account in order to set an application fee percentage. For more information, see the application fees [documentation](https://stripe.com/docs/connect/subscriptions#collecting-fees-on-subscriptions).",
		},
		"default_settings.automatic_tax.enabled": {
			Type:        "boolean",
			Description: "Enabled automatic tax calculation which will automatically compute tax rates on all invoices generated by the subscription.",
			Required:    true,
		},
		"default_settings.billing_cycle_anchor": {
			Type:        "string",
			Description: "Can be set to `phase_start` to set the anchor to the start of the phase or `automatic` to automatically change it if needed. Cannot be set to `phase_start` if this phase specifies a trial. For more information, see the billing cycle [documentation](https://docs.stripe.com/billing/subscriptions/billing-cycle).",
			Enum: []resource.EnumSpec{
				{Value: "automatic"},
				{Value: "phase_start"},
			},
		},
		"default_settings.invoice_settings.issuer.account": {
			Type:        "string",
			Description: "The connected account being referenced when `type` is `account`.",
		},
		"default_settings.invoice_settings.issuer.type": {
			Type:        "string",
			Description: "Type of the account referenced in the request.",
			Required:    true,
			Enum: []resource.EnumSpec{
				{Value: "account"},
				{Value: "self"},
			},
		},
		"billing_mode.type": {
			Type:        "string",
			Description: "Controls the calculation and orchestration of prorations and invoices for subscriptions. If no value is passed, the default is `flexible`.",
			Required:    true,
			Enum: []resource.EnumSpec{
				{Value: "classic"},
				{Value: "flexible"},
			},
		},
	},
}

var V1SubscriptionSchedulesUpdate = resource.OperationSpec{
	Name:    "update",
	Path:    "/v1/subscription_schedules/{schedule}",
	Method:  "POST",
	Summary: "Update a schedule",
	Params: map[string]*resource.ParamSpec{
		"default_settings.invoice_settings.issuer.account": {
			Type:        "string",
			Description: "The connected account being referenced when `type` is `account`.",
		},
		"default_settings.invoice_settings.issuer.type": {
			Type:        "string",
			Description: "Type of the account referenced in the request.",
			Required:    true,
			Enum: []resource.EnumSpec{
				{Value: "account"},
				{Value: "self"},
			},
		},
		"default_settings.collection_method": {
			Type:        "string",
			Description: "Either `charge_automatically`, or `send_invoice`. When charging automatically, Stripe will attempt to pay the underlying subscription at the end of each billing cycle using the default source attached to the customer. When sending an invoice, Stripe will email your customer an invoice with payment instructions and mark the subscription as `active`. Defaults to `charge_automatically` on creation.",
			Enum: []resource.EnumSpec{
				{Value: "charge_automatically"},
				{Value: "send_invoice"},
			},
		},
		"proration_behavior": {
			Type:        "string",
			Description: "If the update changes the billing configuration (item price, quantity, etc.) of the current phase, indicates how prorations from this change should be handled. The default value is `create_prorations`.",
			Enum: []resource.EnumSpec{
				{Value: "always_invoice"},
				{Value: "create_prorations"},
				{Value: "none"},
			},
		},
		"default_settings.automatic_tax.enabled": {
			Type:        "boolean",
			Description: "Enabled automatic tax calculation which will automatically compute tax rates on all invoices generated by the subscription.",
			Required:    true,
		},
		"default_settings.automatic_tax.liability.type": {
			Type:        "string",
			Description: "Type of the account referenced in the request.",
			Required:    true,
			Enum: []resource.EnumSpec{
				{Value: "account"},
				{Value: "self"},
			},
		},
		"default_settings.billing_cycle_anchor": {
			Type:        "string",
			Description: "Can be set to `phase_start` to set the anchor to the start of the phase or `automatic` to automatically change it if needed. Cannot be set to `phase_start` if this phase specifies a trial. For more information, see the billing cycle [documentation](https://docs.stripe.com/billing/subscriptions/billing-cycle).",
			Enum: []resource.EnumSpec{
				{Value: "automatic"},
				{Value: "phase_start"},
			},
		},
		"default_settings.invoice_settings.account_tax_ids": {
			Type:        "array",
			Description: "The account tax IDs associated with the subscription schedule. Will be set on invoices generated by the subscription schedule.",
		},
		"default_settings.default_payment_method": {
			Type:        "string",
			Description: "ID of the default payment method for the subscription schedule. It must belong to the customer associated with the subscription schedule. If not set, invoices will use the default payment method in the customer's invoice settings.",
		},
		"default_settings.on_behalf_of": {
			Type:        "string",
			Description: "The account on behalf of which to charge, for each of the associated subscription's invoices.",
		},
		"default_settings.description": {
			Type:        "string",
			Description: "Subscription description, meant to be displayable to the customer. Use this field to optionally store an explanation of the subscription for rendering in Stripe surfaces and certain local payment methods UIs.",
		},
		"default_settings.application_fee_percent": {
			Type:        "number",
			Description: "A non-negative decimal between 0 and 100, with at most two decimal places. This represents the percentage of the subscription invoice total that will be transferred to the application owner's Stripe account. The request must be made by a platform account on a connected account in order to set an application fee percentage. For more information, see the application fees [documentation](https://stripe.com/docs/connect/subscriptions#collecting-fees-on-subscriptions).",
		},
		"end_behavior": {
			Type:        "string",
			Description: "Behavior of the subscription schedule and underlying subscription when it ends. Possible values are `release` or `cancel` with the default being `release`. `release` will end the subscription schedule and keep the underlying subscription running. `cancel` will end the subscription schedule and cancel the underlying subscription.",
			Enum: []resource.EnumSpec{
				{Value: "cancel"},
				{Value: "none"},
				{Value: "release"},
				{Value: "renew"},
			},
		},
		"default_settings.automatic_tax.liability.account": {
			Type:        "string",
			Description: "The connected account being referenced when `type` is `account`.",
		},
		"default_settings.invoice_settings.days_until_due": {
			Type:        "integer",
			Description: "Number of days within which a customer must pay invoices generated by this subscription schedule. This value will be `null` for subscription schedules where `collection_method=charge_automatically`.",
		},
	},
}

var V1SubscriptionSchedulesCancel = resource.OperationSpec{
	Name:    "cancel",
	Path:    "/v1/subscription_schedules/{schedule}/cancel",
	Method:  "POST",
	Summary: "Cancel a schedule",
	Params: map[string]*resource.ParamSpec{
		"invoice_now": {
			Type:        "boolean",
			Description: "If the subscription schedule is `active`, indicates if a final invoice will be generated that contains any un-invoiced metered usage and new/pending proration invoice items. Defaults to `true`.",
		},
		"prorate": {
			Type:        "boolean",
			Description: "If the subscription schedule is `active`, indicates if the cancellation should be prorated. Defaults to `true`.",
		},
	},
}

var V1SubscriptionSchedulesRelease = resource.OperationSpec{
	Name:    "release",
	Path:    "/v1/subscription_schedules/{schedule}/release",
	Method:  "POST",
	Summary: "Release a schedule",
	Params: map[string]*resource.ParamSpec{
		"preserve_cancel_date": {
			Type:        "boolean",
			Description: "Keep any cancellation on the subscription that the schedule has set",
		},
	},
}

var V1SubscriptionSchedulesList = resource.OperationSpec{
	Name:    "list",
	Path:    "/v1/subscription_schedules",
	Method:  "GET",
	Summary: "List all schedules",
	Params: map[string]*resource.ParamSpec{
		"customer": {
			Type:        "string",
			Description: "Only return subscription schedules for the given customer.",
		},
		"customer_account": {
			Type:        "string",
			Description: "Only return subscription schedules for the given account.",
		},
		"ending_before": {
			Type:        "string",
			Description: "A cursor for use in pagination. `ending_before` is an object ID that defines your place in the list. For instance, if you make a list request and receive 100 objects, starting with `obj_bar`, your subsequent call can include `ending_before=obj_bar` in order to fetch the previous page of the list.",
		},
		"limit": {
			Type:        "integer",
			Description: "A limit on the number of objects to be returned. Limit can range between 1 and 100, and the default is 10.",
		},
		"released_at": {
			Type:        "integer",
			Description: "Only return subscription schedules that were released during the given date interval.",
		},
		"canceled_at": {
			Type:        "integer",
			Description: "Only return subscription schedules that were created canceled the given date interval.",
		},
		"completed_at": {
			Type:        "integer",
			Description: "Only return subscription schedules that completed during the given date interval.",
		},
		"created": {
			Type:        "integer",
			Description: "Only return subscription schedules that were created during the given date interval.",
		},
		"scheduled": {
			Type:        "boolean",
			Description: "Only return subscription schedules that have not started yet.",
		},
		"starting_after": {
			Type:        "string",
			Description: "A cursor for use in pagination. `starting_after` is an object ID that defines your place in the list. For instance, if you make a list request and receive 100 objects, ending with `obj_foo`, your subsequent call can include `starting_after=obj_foo` in order to fetch the next page of the list.",
		},
	},
}

var V1SubscriptionSchedulesRetrieve = resource.OperationSpec{
	Name:    "retrieve",
	Path:    "/v1/subscription_schedules/{schedule}",
	Method:  "GET",
	Summary: "Retrieve a schedule",
}

var V1TopupsCancel = resource.OperationSpec{
	Name:    "cancel",
	Path:    "/v1/topups/{topup}/cancel",
	Method:  "POST",
	Summary: "Cancel a top-up",
}

var V1TopupsList = resource.OperationSpec{
	Name:    "list",
	Path:    "/v1/topups",
	Method:  "GET",
	Summary: "List all top-ups",
	Params: map[string]*resource.ParamSpec{
		"created": {
			Type:        "integer",
			Description: "A filter on the list, based on the object `created` field. The value can be a string with an integer Unix timestamp, or it can be a dictionary with a number of different query options.",
		},
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
		"status": {
			Type:        "string",
			Description: "Only return top-ups that have the given status. One of `canceled`, `failed`, `pending` or `succeeded`.",
			Enum: []resource.EnumSpec{
				{Value: "canceled"},
				{Value: "failed"},
				{Value: "pending"},
				{Value: "succeeded"},
			},
		},
		"amount": {
			Type:        "integer",
			Description: "A positive integer representing how much to transfer.",
		},
	},
}

var V1TopupsRetrieve = resource.OperationSpec{
	Name:    "retrieve",
	Path:    "/v1/topups/{topup}",
	Method:  "GET",
	Summary: "Retrieve a top-up",
}

var V1TopupsCreate = resource.OperationSpec{
	Name:    "create",
	Path:    "/v1/topups",
	Method:  "POST",
	Summary: "Create a top-up",
	Params: map[string]*resource.ParamSpec{
		"source": {
			Type:        "string",
			Description: "The ID of a source to transfer funds from. For most users, this should be left unspecified which will use the bank account that was set up in the dashboard for the specified currency. In test mode, this can be a test bank token (see [Testing Top-ups](https://docs.stripe.com/connect/testing#testing-top-ups)).",
		},
		"statement_descriptor": {
			Type:        "string",
			Description: "Extra information about a top-up for the source's bank statement. Limited to 15 ASCII characters.",
		},
		"transfer_group": {
			Type:        "string",
			Description: "A string that identifies this top-up as part of a group.",
		},
		"amount": {
			Type:        "integer",
			Description: "A positive integer representing how much to transfer.",
			Required:    true,
		},
		"currency": {
			Type:        "string",
			Description: "Three-letter [ISO currency code](https://www.iso.org/iso-4217-currency-codes.html), in lowercase. Must be a [supported currency](https://stripe.com/docs/currencies).",
			Required:    true,
		},
		"description": {
			Type:        "string",
			Description: "An arbitrary string attached to the object. Often useful for displaying to users.",
		},
	},
}

var V1TopupsUpdate = resource.OperationSpec{
	Name:    "update",
	Path:    "/v1/topups/{topup}",
	Method:  "POST",
	Summary: "Update a top-up",
	Params: map[string]*resource.ParamSpec{
		"description": {
			Type:        "string",
			Description: "An arbitrary string attached to the object. Often useful for displaying to users.",
		},
	},
}
