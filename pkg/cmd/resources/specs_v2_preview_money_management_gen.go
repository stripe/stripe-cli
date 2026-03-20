// This file is generated; DO NOT EDIT.

package resources

import "github.com/stripe/stripe-cli/pkg/cmd/resource"

var V2PreviewMoneyManagementOutboundSetupIntentsRetrieve = resource.OperationSpec{
	Name:      "retrieve",
	Path:      "/v2/money_management/outbound_setup_intents/{id}",
	Method:    "GET",
	IsPreview: true,
	Summary:   "Retrieve an Outbound Setup Intent",
}

var V2PreviewMoneyManagementOutboundSetupIntentsUpdate = resource.OperationSpec{
	Name:      "update",
	Path:      "/v2/money_management/outbound_setup_intents/{id}",
	Method:    "POST",
	IsPreview: true,
	Summary:   "Update an Outbound Setup Intent",
	Params: map[string]*resource.ParamSpec{
		"payout_method": {
			Type:        "string",
			Description: "If provided, the existing payout method resource to link to this outbound setup intent.",
		},
		"payout_method_data.bank_account.branch_number": {
			Type:        "string",
			Description: "The branch number of the bank account, if present.",
		},
		"payout_method_data.bank_account.routing_number": {
			Type:        "string",
			Description: "The routing number of the bank account, if present.",
		},
		"payout_method_data.card.exp_month": {
			Type:        "string",
			Description: "The expiration month of the card.",
		},
		"payout_method_data.card.number": {
			Type:        "string",
			Description: "The card number. This can only be passed when creating a new credential on an outbound setup intent in the requires_payout_method state.",
		},
		"payout_method_data.bank_account.country": {
			Type:        "string",
			Description: "The country code of the bank account.",
			Required:    true,
		},
		"payout_method_data.bank_account.swift_code": {
			Type:        "string",
			Description: "The swift code of the bank account, if present.",
		},
		"payout_method_data.bank_account.account_number": {
			Type:        "string",
			Description: "The account number or IBAN of the bank account.",
			Required:    true,
		},
		"payout_method_data.bank_account.bank_account_type": {
			Type:        "string",
			Description: "Closed Enum. The type of the bank account (checking or savings).",
			Enum: []resource.EnumSpec{
				{Value: "checking"},
				{Value: "savings"},
			},
		},
		"payout_method_data.card.exp_year": {
			Type:        "string",
			Description: "The expiration year of the card.",
		},
		"payout_method_data.type": {
			Type:        "string",
			Description: "Closed Enum. The type of payout method to be created/updated.",
			Required:    true,
			Enum: []resource.EnumSpec{
				{Value: "bank_account"},
				{Value: "card"},
				{Value: "crypto_wallet"},
			},
		},
	},
}

var V2PreviewMoneyManagementOutboundSetupIntentsCancel = resource.OperationSpec{
	Name:      "cancel",
	Path:      "/v2/money_management/outbound_setup_intents/{id}/cancel",
	Method:    "POST",
	IsPreview: true,
	Summary:   "Cancel an Outbound Setup Intent",
}

var V2PreviewMoneyManagementOutboundSetupIntentsList = resource.OperationSpec{
	Name:      "list",
	Path:      "/v2/money_management/outbound_setup_intents",
	Method:    "GET",
	IsPreview: true,
	Summary:   "List Outbound Setup Intents",
	Params: map[string]*resource.ParamSpec{
		"limit": {
			Type:        "integer",
			Description: "The page size.",
		},
		"page": {
			Type:        "string",
			Description: "The requested page.",
		},
	},
}

var V2PreviewMoneyManagementOutboundSetupIntentsCreate = resource.OperationSpec{
	Name:      "create",
	Path:      "/v2/money_management/outbound_setup_intents",
	Method:    "POST",
	IsPreview: true,
	Summary:   "Create an Outbound Setup Intent",
	Params: map[string]*resource.ParamSpec{
		"payout_method": {
			Type:        "string",
			Description: "If provided, the existing payout method resource to link to this setup intent.\nAny payout_method_data provided is used to update information on this linked payout method resource.",
		},
		"payout_method_data.card.number": {
			Type:        "string",
			Description: "The card number.",
			Required:    true,
		},
		"payout_method_data.card.exp_month": {
			Type:        "string",
			Description: "The expiration month of the card.",
			Required:    true,
		},
		"payout_method_data.card.exp_year": {
			Type:        "string",
			Description: "The expiration year of the card.",
			Required:    true,
		},
		"payout_method_data.bank_account.routing_number": {
			Type:        "string",
			Description: "The routing number of the bank account, if present.",
		},
		"payout_method_data.bank_account.branch_number": {
			Type:        "string",
			Description: "The branch number of the bank account, if present.",
		},
		"payout_method_data.type": {
			Type:        "string",
			Description: "Closed Enum. The type of payout method to be created.",
			Required:    true,
			Enum: []resource.EnumSpec{
				{Value: "bank_account"},
				{Value: "card"},
				{Value: "crypto_wallet"},
			},
		},
		"payout_method_data.bank_account.country": {
			Type:        "string",
			Description: "The country code of the bank account.",
			Required:    true,
		},
		"payout_method_data.bank_account.swift_code": {
			Type:        "string",
			Description: "The swift code of the bank account, if present.",
		},
		"payout_method_data.bank_account.account_number": {
			Type:        "string",
			Description: "The account number or IBAN of the bank account.",
			Required:    true,
		},
		"payout_method_data.bank_account.bank_account_type": {
			Type:        "string",
			Description: "Closed Enum. The type of the bank account (checking or savings).",
			Enum: []resource.EnumSpec{
				{Value: "checking"},
				{Value: "savings"},
			},
		},
		"usage_intent": {
			Type:        "string",
			Description: "Specify which type of outbound money movement this credential should be set up for (payment | transfer).\nIf not provided, defaults to payment.",
			Enum: []resource.EnumSpec{
				{Value: "payment"},
				{Value: "transfer"},
			},
		},
	},
}

var V2PreviewMoneyManagementPayoutMethodsBankAccountSpecsRetrieve = resource.OperationSpec{
	Name:      "retrieve",
	Path:      "/v2/money_management/payout_methods_bank_account_spec",
	Method:    "GET",
	IsPreview: true,
	Summary:   "Retrieve Bank Account Specification by country",
	Params: map[string]*resource.ParamSpec{
		"countries": {
			Type:        "array",
			Description: "The countries to fetch the bank account spec for.",
		},
	},
}

var V2PreviewMoneyManagementReceivedCreditsList = resource.OperationSpec{
	Name:      "list",
	Path:      "/v2/money_management/received_credits",
	Method:    "GET",
	IsPreview: true,
	Summary:   "List Received Credits",
	Params: map[string]*resource.ParamSpec{
		"created_lte": {
			Type:        "string",
			Description: "Filter for objects created on or before the specified timestamp.\nMust be an RFC 3339 date & time value, for example: 2022-09-18T13:22:00Z.",
			Format:      "date-time",
		},
		"limit": {
			Type:        "integer",
			Description: "The page limit.",
		},
		"page": {
			Type:        "string",
			Description: "The page token.",
		},
		"created": {
			Type:        "string",
			Description: "Filter for objects created at the specified timestamp.\nMust be an RFC 3339 date & time value, for example: 2022-09-18T13:22:00Z.",
			Format:      "date-time",
		},
		"created_gt": {
			Type:        "string",
			Description: "Filter for objects created after the specified timestamp.\nMust be an RFC 3339 date & time value, for example: 2022-09-18T13:22:00Z.",
			Format:      "date-time",
		},
		"created_gte": {
			Type:        "string",
			Description: "Filter for objects created on or after the specified timestamp.\nMust be an RFC 3339 date & time value, for example: 2022-09-18T13:22:00Z.",
			Format:      "date-time",
		},
		"created_lt": {
			Type:        "string",
			Description: "Filter for objects created before the specified timestamp.\nMust be an RFC 3339 date & time value, for example: 2022-09-18T13:22:00Z.",
			Format:      "date-time",
		},
	},
}

var V2PreviewMoneyManagementReceivedCreditsRetrieve = resource.OperationSpec{
	Name:      "retrieve",
	Path:      "/v2/money_management/received_credits/{id}",
	Method:    "GET",
	IsPreview: true,
	Summary:   "Retrieve a Received Credit",
}

var V2PreviewMoneyManagementOutboundTransfersRetrieve = resource.OperationSpec{
	Name:      "retrieve",
	Path:      "/v2/money_management/outbound_transfers/{id}",
	Method:    "GET",
	IsPreview: true,
	Summary:   "Retrieve an Outbound Transfer",
}

var V2PreviewMoneyManagementOutboundTransfersCancel = resource.OperationSpec{
	Name:      "cancel",
	Path:      "/v2/money_management/outbound_transfers/{id}/cancel",
	Method:    "POST",
	IsPreview: true,
	Summary:   "Cancel an Outbound Transfer",
}

var V2PreviewMoneyManagementOutboundTransfersList = resource.OperationSpec{
	Name:      "list",
	Path:      "/v2/money_management/outbound_transfers",
	Method:    "GET",
	IsPreview: true,
	Summary:   "List Outbound Transfers",
	Params: map[string]*resource.ParamSpec{
		"created": {
			Type:        "string",
			Description: "Filter for objects created at the specified timestamp.\nMust be an RFC 3339 date & time value, for example: 2022-09-18T13:22:00Z.",
			Format:      "date-time",
		},
		"created_gt": {
			Type:        "string",
			Description: "Filter for objects created after the specified timestamp.\nMust be an RFC 3339 date & time value, for example: 2022-09-18T13:22:00Z.",
			Format:      "date-time",
		},
		"created_gte": {
			Type:        "string",
			Description: "Filter for objects created on or after the specified timestamp.\nMust be an RFC 3339 date & time value, for example: 2022-09-18T13:22:00Z.",
			Format:      "date-time",
		},
		"created_lt": {
			Type:        "string",
			Description: "Filter for objects created before the specified timestamp.\nMust be an RFC 3339 date & time value, for example: 2022-09-18T13:22:00Z.",
			Format:      "date-time",
		},
		"created_lte": {
			Type:        "string",
			Description: "Filter for objects created on or before the specified timestamp.\nMust be an RFC 3339 date & time value, for example: 2022-09-18T13:22:00Z.",
			Format:      "date-time",
		},
		"limit": {
			Type:        "integer",
			Description: "The maximum number of results to return.",
		},
		"page": {
			Type:        "string",
			Description: "The page token to use to retrieve the page being requested.",
		},
		"status": {
			Type:        "array",
			Description: "Closed Enum. Only return OutboundTransfers with this status.",
		},
	},
}

var V2PreviewMoneyManagementOutboundTransfersCreate = resource.OperationSpec{
	Name:      "create",
	Path:      "/v2/money_management/outbound_transfers",
	Method:    "POST",
	IsPreview: true,
	Summary:   "Create an Outbound Transfer",
	Params: map[string]*resource.ParamSpec{
		"amount.value": {
			Type:        "integer",
			Description: "A non-negative integer representing how much to charge in the [smallest currency unit](https://docs.stripe.com/currencies#minor-units).",
			Required:    true,
		},
		"delivery_options.bank_account": {
			Type:        "string",
			Description: "Open Enum. Method for bank account.",
			Enum: []resource.EnumSpec{
				{Value: "automatic"},
				{Value: "local"},
				{Value: "wire"},
			},
		},
		"description": {
			Type:        "string",
			Description: "An arbitrary string attached to the OutboundTransfer. Often useful for displaying to users.",
		},
		"from.financial_account": {
			Type:        "string",
			Description: "The FinancialAccount that funds were pulled from.",
			Required:    true,
		},
		"from.currency": {
			Type:        "string",
			Description: "Describes the FinancialAmount's currency drawn from.",
			Required:    true,
		},
		"to.currency": {
			Type:        "string",
			Description: "Describes the currency to send to the recipient.\nIf included, this currency must match a currency supported by the destination.\nCan be omitted in the following cases:\n- destination only supports one currency\n- destination supports multiple currencies and one of the currencies matches the FA currency\n- destination supports multiple currencies and one of the currencies matches the presentment currency\nNote - when both FA currency and presentment currency are supported, we pick the FA currency to minimize FX.",
		},
		"to.payout_method": {
			Type:        "string",
			Description: "The payout method which the OutboundTransfer uses to send payout.",
			Required:    true,
		},
		"amount.currency": {
			Type:        "string",
			Description: "Three-letter [ISO currency code](https://www.iso.org/iso-4217-currency-codes.html), in lowercase. Must be a [supported currency](https://stripe.com/docs/currencies).",
			Required:    true,
		},
	},
}

var V2PreviewMoneyManagementFinancialAccountsList = resource.OperationSpec{
	Name:      "list",
	Path:      "/v2/money_management/financial_accounts",
	Method:    "GET",
	IsPreview: true,
	Summary:   "List Financial Accounts",
	Params: map[string]*resource.ParamSpec{
		"limit": {
			Type:        "integer",
			Description: "The page limit.",
		},
		"page": {
			Type:        "string",
			Description: "The page token.",
		},
		"status": {
			Type:        "string",
			Description: "The status of the FinancialAccount to filter by. By default, closed FinancialAccounts are not returned.",
			Enum: []resource.EnumSpec{
				{Value: "closed"},
				{Value: "open"},
				{Value: "pending"},
			},
		},
	},
}

var V2PreviewMoneyManagementFinancialAccountsCreate = resource.OperationSpec{
	Name:      "create",
	Path:      "/v2/money_management/financial_accounts",
	Method:    "POST",
	IsPreview: true,
	Summary:   "Create a Financial Account",
	Params: map[string]*resource.ParamSpec{
		"storage.holds_currencies": {
			Type:        "array",
			Description: "The currencies that this FinancialAccount can hold.",
			Required:    true,
		},
		"type": {
			Type:        "string",
			Description: "The type of FinancialAccount to create.",
			Required:    true,
			Enum: []resource.EnumSpec{
				{Value: "storage"},
			},
		},
		"display_name": {
			Type:        "string",
			Description: "A descriptive name for the FinancialAccount, up to 50 characters long. This name will be used in the Stripe Dashboard and embedded components.",
		},
	},
}

var V2PreviewMoneyManagementFinancialAccountsRetrieve = resource.OperationSpec{
	Name:      "retrieve",
	Path:      "/v2/money_management/financial_accounts/{id}",
	Method:    "GET",
	IsPreview: true,
	Summary:   "Retrieve a Financial Account",
}

var V2PreviewMoneyManagementFinancialAccountsUpdate = resource.OperationSpec{
	Name:      "update",
	Path:      "/v2/money_management/financial_accounts/{id}",
	Method:    "POST",
	IsPreview: true,
	Summary:   "Update a Financial Account",
	Params: map[string]*resource.ParamSpec{
		"display_name": {
			Type:        "string",
			Description: "A descriptive name for the FinancialAccount, up to 50 characters long. This name will be used in the Stripe Dashboard and embedded components.",
		},
	},
}

var V2PreviewMoneyManagementFinancialAccountsClose = resource.OperationSpec{
	Name:      "close",
	Path:      "/v2/money_management/financial_accounts/{id}/close",
	Method:    "POST",
	IsPreview: true,
	Summary:   "Close a Financial Account",
	Params: map[string]*resource.ParamSpec{
		"forwarding_settings.payment_method": {
			Type:        "string",
			Description: "The address to send forwarded payments to.",
		},
		"forwarding_settings.payout_method": {
			Type:        "string",
			Description: "The address to send forwarded payouts to.",
		},
	},
}

var V2PreviewMoneyManagementInboundTransfersList = resource.OperationSpec{
	Name:      "list",
	Path:      "/v2/money_management/inbound_transfers",
	Method:    "GET",
	IsPreview: true,
	Summary:   "List Inbound Transfers",
	Params: map[string]*resource.ParamSpec{
		"limit": {
			Type:        "integer",
			Description: "The page limit.",
		},
		"page": {
			Type:        "string",
			Description: "The page token.",
		},
		"created": {
			Type:        "string",
			Description: "Filter for objects created at the specified timestamp.\nMust be an RFC 3339 date & time value, for example: 2022-09-18T13:22:00Z.",
			Format:      "date-time",
		},
		"created_gt": {
			Type:        "string",
			Description: "Filter for objects created after the specified timestamp.\nMust be an RFC 3339 date & time value, for example: 2022-09-18T13:22:00Z.",
			Format:      "date-time",
		},
		"created_gte": {
			Type:        "string",
			Description: "Filter for objects created on or after the specified timestamp.\nMust be an RFC 3339 date & time value, for example: 2022-09-18T13:22:00Z.",
			Format:      "date-time",
		},
		"created_lt": {
			Type:        "string",
			Description: "Filter for objects created before the specified timestamp.\nMust be an RFC 3339 date & time value, for example: 2022-09-18T13:22:00Z.",
			Format:      "date-time",
		},
		"created_lte": {
			Type:        "string",
			Description: "Filter for objects created on or before the specified timestamp.\nMust be an RFC 3339 date & time value, for example: 2022-09-18T13:22:00Z.",
			Format:      "date-time",
		},
	},
}

var V2PreviewMoneyManagementInboundTransfersCreate = resource.OperationSpec{
	Name:      "create",
	Path:      "/v2/money_management/inbound_transfers",
	Method:    "POST",
	IsPreview: true,
	Summary:   "Create an Inbound Transfer",
	Params: map[string]*resource.ParamSpec{
		"to.currency": {
			Type:        "string",
			Description: "The currency in which funds will land in.",
			Required:    true,
		},
		"to.financial_account": {
			Type:        "string",
			Description: "The FinancialAccount that funds will land in.",
			Required:    true,
		},
		"amount.currency": {
			Type:        "string",
			Description: "Three-letter [ISO currency code](https://www.iso.org/iso-4217-currency-codes.html), in lowercase. Must be a [supported currency](https://stripe.com/docs/currencies).",
			Required:    true,
		},
		"amount.value": {
			Type:        "integer",
			Description: "A non-negative integer representing how much to charge in the [smallest currency unit](https://docs.stripe.com/currencies#minor-units).",
			Required:    true,
		},
		"description": {
			Type:        "string",
			Description: "An optional, freeform description field intended to store metadata.",
		},
		"from.currency": {
			Type:        "string",
			Description: "An optional currency field used to specify which currency is debited from the Payment Method.\nSince many Payment Methods support only one currency, this field is optional.",
		},
		"from.payment_method": {
			Type:        "string",
			Description: "ID of the Payment Method using which IBT will be made.",
			Required:    true,
		},
	},
}

var V2PreviewMoneyManagementInboundTransfersRetrieve = resource.OperationSpec{
	Name:      "retrieve",
	Path:      "/v2/money_management/inbound_transfers/{id}",
	Method:    "GET",
	IsPreview: true,
	Summary:   "Retrieve an Inbound Transfer",
}

var V2PreviewMoneyManagementAdjustmentsList = resource.OperationSpec{
	Name:      "list",
	Path:      "/v2/money_management/adjustments",
	Method:    "GET",
	IsPreview: true,
	Summary:   "List Adjustments",
	Params: map[string]*resource.ParamSpec{
		"created_gt": {
			Type:        "string",
			Description: "Filter for objects created after the specified timestamp.\nMust be an RFC 3339 date & time value, for example: 2022-09-18T13:22:00Z.",
			Format:      "date-time",
		},
		"created_gte": {
			Type:        "string",
			Description: "Filter for objects created on or after the specified timestamp.\nMust be an RFC 3339 date & time value, for example: 2022-09-18T13:22:00Z.",
			Format:      "date-time",
		},
		"created_lt": {
			Type:        "string",
			Description: "Filter for objects created before the specified timestamp.\nMust be an RFC 3339 date & time value, for example: 2022-09-18T13:22:00Z.",
			Format:      "date-time",
		},
		"created_lte": {
			Type:        "string",
			Description: "Filter for objects created on or before the specified timestamp.\nMust be an RFC 3339 date & time value, for example: 2022-09-18T13:22:00Z.",
			Format:      "date-time",
		},
		"limit": {
			Type:        "integer",
			Description: "The page limit.",
		},
		"page": {
			Type:        "string",
			Description: "The page token.",
		},
		"adjusted_flow": {
			Type:        "string",
			Description: "Filter for Adjustments linked to a Flow.",
		},
		"created": {
			Type:        "string",
			Description: "Filter for objects created at the specified timestamp.\nMust be an RFC 3339 date & time value, for example: 2022-09-18T13:22:00Z.",
			Format:      "date-time",
		},
	},
}

var V2PreviewMoneyManagementAdjustmentsRetrieve = resource.OperationSpec{
	Name:      "retrieve",
	Path:      "/v2/money_management/adjustments/{id}",
	Method:    "GET",
	IsPreview: true,
	Summary:   "Retrieve an Adjustment",
}

var V2PreviewMoneyManagementFinancialAddresssList = resource.OperationSpec{
	Name:      "list",
	Path:      "/v2/money_management/financial_addresses",
	Method:    "GET",
	IsPreview: true,
	Summary:   "List Financial Addresses",
	Params: map[string]*resource.ParamSpec{
		"financial_account": {
			Type:        "string",
			Description: "The ID of the FinancialAccount for which FinancialAddresses are to be returned.",
		},
		"include": {
			Type:        "array",
			Description: "Open Enum. A list of fields to reveal in the FinancialAddresses returned.",
		},
		"limit": {
			Type:        "integer",
			Description: "The page limit.",
		},
		"page": {
			Type:        "string",
			Description: "The page token.",
		},
	},
}

var V2PreviewMoneyManagementFinancialAddresssCreate = resource.OperationSpec{
	Name:      "create",
	Path:      "/v2/money_management/financial_addresses",
	Method:    "POST",
	IsPreview: true,
	Summary:   "Create a Financial Address",
	Params: map[string]*resource.ParamSpec{
		"financial_account": {
			Type:        "string",
			Description: "The ID of the FinancialAccount the new FinancialAddress should be associated with.",
			Required:    true,
		},
		"type": {
			Type:        "string",
			Description: "The type of FinancialAddress details to provision.",
			Required:    true,
			Enum: []resource.EnumSpec{
				{Value: "gb_bank_account"},
				{Value: "us_bank_account"},
			},
		},
	},
}

var V2PreviewMoneyManagementFinancialAddresssRetrieve = resource.OperationSpec{
	Name:      "retrieve",
	Path:      "/v2/money_management/financial_addresses/{id}",
	Method:    "GET",
	IsPreview: true,
	Summary:   "Retrieve a Financial Address",
	Params: map[string]*resource.ParamSpec{
		"include": {
			Type:        "array",
			Description: "Open Enum. A list of fields to reveal in the FinancialAddresses returned.",
		},
	},
}

var V2PreviewMoneyManagementReceivedDebitsList = resource.OperationSpec{
	Name:      "list",
	Path:      "/v2/money_management/received_debits",
	Method:    "GET",
	IsPreview: true,
	Summary:   "List Received Debits",
	Params: map[string]*resource.ParamSpec{
		"limit": {
			Type:        "integer",
			Description: "The page limit.",
		},
		"page": {
			Type:        "string",
			Description: "The page token.",
		},
	},
}

var V2PreviewMoneyManagementReceivedDebitsRetrieve = resource.OperationSpec{
	Name:      "retrieve",
	Path:      "/v2/money_management/received_debits/{id}",
	Method:    "GET",
	IsPreview: true,
	Summary:   "Retrieve a Received Debit",
}

var V2PreviewMoneyManagementTransactionEntrysList = resource.OperationSpec{
	Name:      "list",
	Path:      "/v2/money_management/transaction_entries",
	Method:    "GET",
	IsPreview: true,
	Summary:   "List Transaction Entries",
	Params: map[string]*resource.ParamSpec{
		"page": {
			Type:        "string",
			Description: "The page token.",
		},
		"transaction": {
			Type:        "string",
			Description: "Filter for TransactionEntries belonging to a Transaction.",
		},
		"created": {
			Type:        "string",
			Description: "Filter for Transactions created at an exact time.",
			Format:      "date-time",
		},
		"created_gt": {
			Type:        "string",
			Description: "Filter for Transactions created after the specified timestamp.",
			Format:      "date-time",
		},
		"created_gte": {
			Type:        "string",
			Description: "Filter for Transactions created at or after the specified timestamp.",
			Format:      "date-time",
		},
		"created_lt": {
			Type:        "string",
			Description: "Filter for Transactions created before the specified timestamp.",
			Format:      "date-time",
		},
		"created_lte": {
			Type:        "string",
			Description: "Filter for Transactions created at or before the specified timestamp.",
			Format:      "date-time",
		},
		"limit": {
			Type:        "integer",
			Description: "The page limit.",
		},
	},
}

var V2PreviewMoneyManagementTransactionEntrysRetrieve = resource.OperationSpec{
	Name:      "retrieve",
	Path:      "/v2/money_management/transaction_entries/{id}",
	Method:    "GET",
	IsPreview: true,
	Summary:   "Retrieve a Transaction Entry",
}

var V2PreviewMoneyManagementPayoutMethodsList = resource.OperationSpec{
	Name:      "list",
	Path:      "/v2/money_management/payout_methods",
	Method:    "GET",
	IsPreview: true,
	Summary:   "List Payout Methods",
	Params: map[string]*resource.ParamSpec{
		"limit": {
			Type:        "integer",
			Description: "The page size.",
		},
		"page": {
			Type:        "string",
			Description: "The requested page.",
		},
	},
}

var V2PreviewMoneyManagementPayoutMethodsRetrieve = resource.OperationSpec{
	Name:      "retrieve",
	Path:      "/v2/money_management/payout_methods/{id}",
	Method:    "GET",
	IsPreview: true,
	Summary:   "Retrieve a Payout Method",
}

var V2PreviewMoneyManagementPayoutMethodsArchive = resource.OperationSpec{
	Name:      "archive",
	Path:      "/v2/money_management/payout_methods/{id}/archive",
	Method:    "POST",
	IsPreview: true,
	Summary:   "Archive a Payout Method",
}

var V2PreviewMoneyManagementPayoutMethodsUnarchive = resource.OperationSpec{
	Name:      "unarchive",
	Path:      "/v2/money_management/payout_methods/{id}/unarchive",
	Method:    "POST",
	IsPreview: true,
	Summary:   "Unarchive a Payout Method",
}

var V2PreviewMoneyManagementOutboundPaymentsList = resource.OperationSpec{
	Name:      "list",
	Path:      "/v2/money_management/outbound_payments",
	Method:    "GET",
	IsPreview: true,
	Summary:   "List Outbound Payments",
	Params: map[string]*resource.ParamSpec{
		"created_lt": {
			Type:        "string",
			Description: "Filter for objects created before the specified timestamp.\nMust be an RFC 3339 date & time value, for example: 2022-09-18T13:22:00Z.",
			Format:      "date-time",
		},
		"limit": {
			Type:        "integer",
			Description: "The maximum number of results to return.",
		},
		"page": {
			Type:        "string",
			Description: "The page token to use to retrieve the page being requested.",
		},
		"recipient": {
			Type:        "string",
			Description: "Only return OutboundPayments sent to this recipient.",
		},
		"created_gt": {
			Type:        "string",
			Description: "Filter for objects created after the specified timestamp.\nMust be an RFC 3339 date & time value, for example: 2022-09-18T13:22:00Z.",
			Format:      "date-time",
		},
		"created_gte": {
			Type:        "string",
			Description: "Filter for objects created on or after the specified timestamp.\nMust be an RFC 3339 date & time value, for example: 2022-09-18T13:22:00Z.",
			Format:      "date-time",
		},
		"created_lte": {
			Type:        "string",
			Description: "Filter for objects created on or before the specified timestamp.\nMust be an RFC 3339 date & time value, for example: 2022-09-18T13:22:00Z.",
			Format:      "date-time",
		},
		"status": {
			Type:        "array",
			Description: "Closed Enum. Only return OutboundPayments with this status.",
		},
		"created": {
			Type:        "string",
			Description: "Filter for objects created at the specified timestamp.\nMust be an RFC 3339 date & time value, for example: 2022-09-18T13:22:00Z.",
			Format:      "date-time",
		},
	},
}

var V2PreviewMoneyManagementOutboundPaymentsCreate = resource.OperationSpec{
	Name:      "create",
	Path:      "/v2/money_management/outbound_payments",
	Method:    "POST",
	IsPreview: true,
	Summary:   "Create an Outbound Payment",
	Params: map[string]*resource.ParamSpec{
		"to.currency": {
			Type:        "string",
			Description: "Describes the currency to send to the recipient.\nIf included, this currency must match a currency supported by the destination.\nCan be omitted in the following cases:\n- destination only supports one currency\n- destination supports multiple currencies and one of the currencies matches the FA currency\n- destination supports multiple currencies and one of the currencies matches the presentment currency\nNote - when both FA currency and presentment currency are supported, we pick the FA currency to minimize FX.",
		},
		"to.payout_method": {
			Type:        "string",
			Description: "The payout method which the OutboundPayment uses to send payout.",
		},
		"description": {
			Type:        "string",
			Description: "An arbitrary string attached to the OutboundPayment. Often useful for displaying to users.",
		},
		"to.recipient": {
			Type:        "string",
			Description: "To which account the OutboundPayment is sent.",
			Required:    true,
		},
		"amount.currency": {
			Type:        "string",
			Description: "Three-letter [ISO currency code](https://www.iso.org/iso-4217-currency-codes.html), in lowercase. Must be a [supported currency](https://stripe.com/docs/currencies).",
			Required:    true,
		},
		"amount.value": {
			Type:        "integer",
			Description: "A non-negative integer representing how much to charge in the [smallest currency unit](https://docs.stripe.com/currencies#minor-units).",
			Required:    true,
		},
		"delivery_options.bank_account": {
			Type:        "string",
			Description: "Open Enum. Method for bank account.",
			Enum: []resource.EnumSpec{
				{Value: "automatic"},
				{Value: "local"},
				{Value: "wire"},
			},
		},
		"from.financial_account": {
			Type:        "string",
			Description: "The FinancialAccount that funds were pulled from.",
			Required:    true,
		},
		"from.currency": {
			Type:        "string",
			Description: "Describes the FinancialAmount's currency drawn from.",
			Required:    true,
		},
		"recipient_notification.setting": {
			Type:        "string",
			Description: "Closed Enum. Configuration option to enable or disable notifications to recipients.\nDo not send notifications when setting is NONE. Default to account setting when setting is CONFIGURED or not set.",
			Required:    true,
			Enum: []resource.EnumSpec{
				{Value: "configured"},
				{Value: "none"},
			},
		},
	},
}

var V2PreviewMoneyManagementOutboundPaymentsRetrieve = resource.OperationSpec{
	Name:      "retrieve",
	Path:      "/v2/money_management/outbound_payments/{id}",
	Method:    "GET",
	IsPreview: true,
	Summary:   "Retrieve an Outbound Payment",
}

var V2PreviewMoneyManagementOutboundPaymentsCancel = resource.OperationSpec{
	Name:      "cancel",
	Path:      "/v2/money_management/outbound_payments/{id}/cancel",
	Method:    "POST",
	IsPreview: true,
	Summary:   "Cancel an Outbound Payment",
}

var V2PreviewMoneyManagementTransactionsList = resource.OperationSpec{
	Name:      "list",
	Path:      "/v2/money_management/transactions",
	Method:    "GET",
	IsPreview: true,
	Summary:   "List Transactions",
	Params: map[string]*resource.ParamSpec{
		"financial_account": {
			Type:        "string",
			Description: "Filter for Transactions belonging to a FinancialAccount.",
		},
		"flow": {
			Type:        "string",
			Description: "Filter for Transactions corresponding to a Flow.",
		},
		"created_gte": {
			Type:        "string",
			Description: "Filter for Transactions created at or after the specified timestamp.",
			Format:      "date-time",
		},
		"limit": {
			Type:        "integer",
			Description: "The page limit.",
		},
		"page": {
			Type:        "string",
			Description: "The page token.",
		},
		"created": {
			Type:        "string",
			Description: "Filter for Transactions created at an exact time.",
			Format:      "date-time",
		},
		"created_gt": {
			Type:        "string",
			Description: "Filter for Transactions created after the specified timestamp.",
			Format:      "date-time",
		},
		"created_lt": {
			Type:        "string",
			Description: "Filter for Transactions created before the specified timestamp.",
			Format:      "date-time",
		},
		"created_lte": {
			Type:        "string",
			Description: "Filter for Transactions created at or before the specified timestamp.",
			Format:      "date-time",
		},
	},
}

var V2PreviewMoneyManagementTransactionsRetrieve = resource.OperationSpec{
	Name:      "retrieve",
	Path:      "/v2/money_management/transactions/{id}",
	Method:    "GET",
	IsPreview: true,
	Summary:   "Retrieve a Transaction",
}

var V2PreviewMoneyManagementOutboundPaymentQuotesCreate = resource.OperationSpec{
	Name:      "create",
	Path:      "/v2/money_management/outbound_payment_quotes",
	Method:    "POST",
	IsPreview: true,
	Summary:   "Create an Outbound Payment Quote",
	Params: map[string]*resource.ParamSpec{
		"amount.value": {
			Type:        "integer",
			Description: "A non-negative integer representing how much to charge in the [smallest currency unit](https://docs.stripe.com/currencies#minor-units).",
			Required:    true,
		},
		"delivery_options.bank_account": {
			Type:        "string",
			Description: "Open Enum. Method for bank account.",
			Enum: []resource.EnumSpec{
				{Value: "automatic"},
				{Value: "local"},
				{Value: "wire"},
			},
		},
		"from.currency": {
			Type:        "string",
			Description: "Describes the FinancialAccount's currency drawn from.",
			Required:    true,
		},
		"from.financial_account": {
			Type:        "string",
			Description: "The FinancialAccount that funds were pulled from.",
			Required:    true,
		},
		"to.recipient": {
			Type:        "string",
			Description: "To which account the OutboundPayment is sent.",
			Required:    true,
		},
		"to.currency": {
			Type:        "string",
			Description: "Describes the currency to send to the recipient.\nIf included, this currency must match a currency supported by the destination.\nCan be omitted in the following cases:\n- destination only supports one currency\n- destination supports multiple currencies and one of the currencies matches the FA currency\n- destination supports multiple currencies and one of the currencies matches the presentment currency\nNote - when both FA currency and presentment currency are supported, we pick the FA currency to minimize FX.",
		},
		"to.payout_method": {
			Type:        "string",
			Description: "The payout method which the OutboundPayment uses to send payout.",
		},
		"amount.currency": {
			Type:        "string",
			Description: "Three-letter [ISO currency code](https://www.iso.org/iso-4217-currency-codes.html), in lowercase. Must be a [supported currency](https://stripe.com/docs/currencies).",
			Required:    true,
		},
	},
}

var V2PreviewMoneyManagementOutboundPaymentQuotesRetrieve = resource.OperationSpec{
	Name:      "retrieve",
	Path:      "/v2/money_management/outbound_payment_quotes/{id}",
	Method:    "GET",
	IsPreview: true,
	Summary:   "Retrieve an Outbound Payment Quote",
}
