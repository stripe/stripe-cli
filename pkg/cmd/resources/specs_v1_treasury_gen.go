// This file is generated; DO NOT EDIT.

package resources

import "github.com/stripe/stripe-cli/pkg/cmd/resource"

var V1TreasuryOutboundPaymentsCreate = resource.OperationSpec{
	Name:    "create",
	Path:    "/v1/treasury/outbound_payments",
	Method:  "POST",
	Summary: "Create an OutboundPayment",
	Params: map[string]*resource.ParamSpec{
		"description": {
			Type:        "string",
			Description: "An arbitrary string attached to the object. Often useful for displaying to users.",
		},
		"customer": {
			Type:        "string",
			Description: "ID of the customer to whom the OutboundPayment is sent. Must match the Customer attached to the `destination_payment_method` passed in.",
		},
		"financial_account": {
			Type:        "string",
			Description: "The FinancialAccount to pull funds from.",
			Required:    true,
		},
		"destination_payment_method_data.billing_details.email": {
			Type:        "string",
			Description: "Email address.",
		},
		"destination_payment_method_data.us_bank_account.routing_number": {
			Type:        "string",
			Description: "Routing number of the bank account.",
		},
		"amount": {
			Type:        "integer",
			Description: "Amount (in cents) to be transferred.",
			Required:    true,
		},
		"destination_payment_method": {
			Type:        "string",
			Description: "The PaymentMethod to use as the payment instrument for the OutboundPayment. Exclusive with `destination_payment_method_data`.",
		},
		"destination_payment_method_data.billing_details.name": {
			Type:        "string",
			Description: "Full name.",
		},
		"destination_payment_method_data.billing_details.phone": {
			Type:        "string",
			Description: "Billing phone number (including extension).",
		},
		"destination_payment_method_data.financial_account": {
			Type:        "string",
			Description: "Required if type is set to `financial_account`. The FinancialAccount ID to send funds to.",
		},
		"destination_payment_method_data.us_bank_account.account_holder_type": {
			Type:        "string",
			Description: "Account holder type: individual or company.",
			Enum: []resource.EnumSpec{
				{Value: "company"},
				{Value: "individual"},
			},
		},
		"destination_payment_method_data.us_bank_account.account_number": {
			Type:        "string",
			Description: "Account number of the bank account.",
		},
		"destination_payment_method_data.us_bank_account.account_type": {
			Type:        "string",
			Description: "Account type: checkings or savings. Defaults to checking if omitted.",
			Enum: []resource.EnumSpec{
				{Value: "checking"},
				{Value: "savings"},
			},
		},
		"end_user_details.present": {
			Type:        "boolean",
			Description: "`True` if the OutboundPayment creation request is being made on behalf of an end user by a platform. Otherwise, `false`.",
			Required:    true,
		},
		"statement_descriptor": {
			Type:        "string",
			Description: "The description that appears on the receiving end for this OutboundPayment (for example, bank statement for external bank transfer). Maximum 10 characters for `ach` payments, 140 characters for `us_domestic_wire` payments, or 500 characters for `stripe` network transfers. Can only include -#.$&*, spaces, and alphanumeric characters. The default value is \"payment\".",
		},
		"destination_payment_method_data.type": {
			Type:        "string",
			Description: "The type of the PaymentMethod. An additional hash is included on the PaymentMethod with a name matching this value. It contains additional information specific to the PaymentMethod type.",
			Required:    true,
			Enum: []resource.EnumSpec{
				{Value: "financial_account"},
				{Value: "us_bank_account"},
			},
		},
		"destination_payment_method_data.us_bank_account.financial_connections_account": {
			Type:        "string",
			Description: "The ID of a Financial Connections Account to use as a payment method.",
		},
		"end_user_details.ip_address": {
			Type:        "string",
			Description: "IP address of the user initiating the OutboundPayment. Must be supplied if `present` is set to `true`.",
		},
		"currency": {
			Type:        "string",
			Description: "Three-letter [ISO currency code](https://www.iso.org/iso-4217-currency-codes.html), in lowercase. Must be a [supported currency](https://stripe.com/docs/currencies).",
			Required:    true,
			Format:      "currency",
		},
	},
}

var V1TreasuryOutboundPaymentsCancel = resource.OperationSpec{
	Name:    "cancel",
	Path:    "/v1/treasury/outbound_payments/{id}/cancel",
	Method:  "POST",
	Summary: "Cancel an OutboundPayment",
}

var V1TreasuryOutboundPaymentsList = resource.OperationSpec{
	Name:    "list",
	Path:    "/v1/treasury/outbound_payments",
	Method:  "GET",
	Summary: "List all OutboundPayments",
	Params: map[string]*resource.ParamSpec{
		"created": {
			Type:        "integer",
			Description: "Only return OutboundPayments that were created during the given date interval.",
		},
		"customer": {
			Type:        "string",
			Description: "Only return OutboundPayments sent to this customer.",
		},
		"ending_before": {
			Type:        "string",
			Description: "A cursor for use in pagination. `ending_before` is an object ID that defines your place in the list. For instance, if you make a list request and receive 100 objects, starting with `obj_bar`, your subsequent call can include `ending_before=obj_bar` in order to fetch the previous page of the list.",
		},
		"financial_account": {
			Type:        "string",
			Description: "Returns objects associated with this FinancialAccount.",
			Required:    true,
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
			Description: "Only return OutboundPayments that have the given status: `processing`, `failed`, `posted`, `returned`, or `canceled`.",
			Enum: []resource.EnumSpec{
				{Value: "canceled"},
				{Value: "failed"},
				{Value: "posted"},
				{Value: "processing"},
				{Value: "returned"},
			},
		},
	},
}

var V1TreasuryOutboundPaymentsRetrieve = resource.OperationSpec{
	Name:    "retrieve",
	Path:    "/v1/treasury/outbound_payments/{id}",
	Method:  "GET",
	Summary: "Retrieve an OutboundPayment",
}

var V1TreasuryOutboundPaymentsTestHelpersUpdate = resource.OperationSpec{
	Name:    "update",
	Path:    "/v1/test_helpers/treasury/outbound_payments/{id}",
	Method:  "POST",
	Summary: "Test mode: Update an OutboundPayment",
	Params: map[string]*resource.ParamSpec{
		"tracking_details.ach.trace_id": {
			Type:        "string",
			Description: "ACH trace ID for funds sent over the `ach` network.",
			Required:    true,
		},
		"tracking_details.type": {
			Type:        "string",
			Description: "The US bank account network used to send funds.",
			Required:    true,
			Enum: []resource.EnumSpec{
				{Value: "ach"},
				{Value: "us_domestic_wire"},
			},
		},
		"tracking_details.us_domestic_wire.chips": {
			Type:        "string",
			Description: "CHIPS System Sequence Number (SSN) for funds sent over the `us_domestic_wire` network.",
		},
		"tracking_details.us_domestic_wire.imad": {
			Type:        "string",
			Description: "IMAD for funds sent over the `us_domestic_wire` network.",
		},
		"tracking_details.us_domestic_wire.omad": {
			Type:        "string",
			Description: "OMAD for funds sent over the `us_domestic_wire` network.",
		},
	},
}

var V1TreasuryOutboundPaymentsTestHelpersFail = resource.OperationSpec{
	Name:    "fail",
	Path:    "/v1/test_helpers/treasury/outbound_payments/{id}/fail",
	Method:  "POST",
	Summary: "Test mode: Fail an OutboundPayment",
}

var V1TreasuryOutboundPaymentsTestHelpersPost = resource.OperationSpec{
	Name:    "post",
	Path:    "/v1/test_helpers/treasury/outbound_payments/{id}/post",
	Method:  "POST",
	Summary: "Test mode: Post an OutboundPayment",
}

var V1TreasuryOutboundPaymentsTestHelpersReturnOutboundPayment = resource.OperationSpec{
	Name:    "return_outbound_payment",
	Path:    "/v1/test_helpers/treasury/outbound_payments/{id}/return",
	Method:  "POST",
	Summary: "Test mode: Return an OutboundPayment",
	Params: map[string]*resource.ParamSpec{
		"returned_details.code": {
			Type:        "string",
			Description: "The return code to be set on the OutboundPayment object.",
			Enum: []resource.EnumSpec{
				{Value: "account_closed"},
				{Value: "account_frozen"},
				{Value: "bank_account_restricted"},
				{Value: "bank_ownership_changed"},
				{Value: "declined"},
				{Value: "incorrect_account_holder_name"},
				{Value: "invalid_account_number"},
				{Value: "invalid_currency"},
				{Value: "no_account"},
				{Value: "other"},
			},
		},
	},
}

var V1TreasuryFinancialAccountsList = resource.OperationSpec{
	Name:    "list",
	Path:    "/v1/treasury/financial_accounts",
	Method:  "GET",
	Summary: "List all FinancialAccounts",
	Params: map[string]*resource.ParamSpec{
		"created": {
			Type:        "integer",
			Description: "Only return FinancialAccounts that were created during the given date interval.",
		},
		"ending_before": {
			Type:        "string",
			Description: "An object ID cursor for use in pagination.",
		},
		"limit": {
			Type:        "integer",
			Description: "A limit ranging from 1 to 100 (defaults to 10).",
		},
		"starting_after": {
			Type:        "string",
			Description: "An object ID cursor for use in pagination.",
		},
		"status": {
			Type:        "string",
			Description: "Only return FinancialAccounts that have the given status: `open` or `closed`",
			Enum: []resource.EnumSpec{
				{Value: "closed"},
				{Value: "open"},
			},
		},
	},
}

var V1TreasuryFinancialAccountsRetrieve = resource.OperationSpec{
	Name:    "retrieve",
	Path:    "/v1/treasury/financial_accounts/{financial_account}",
	Method:  "GET",
	Summary: "Retrieve a FinancialAccount",
}

var V1TreasuryFinancialAccountsRetrieveFeatures = resource.OperationSpec{
	Name:    "retrieve_features",
	Path:    "/v1/treasury/financial_accounts/{financial_account}/features",
	Method:  "GET",
	Summary: "Retrieve FinancialAccount Features",
}

var V1TreasuryFinancialAccountsCreate = resource.OperationSpec{
	Name:    "create",
	Path:    "/v1/treasury/financial_accounts",
	Method:  "POST",
	Summary: "Create a FinancialAccount",
	Params: map[string]*resource.ParamSpec{
		"nickname": {
			Type:        "string",
			Description: "The nickname for the FinancialAccount.",
		},
		"platform_restrictions.outbound_flows": {
			Type:        "string",
			Description: "Restricts all outbound money movement.",
			Enum: []resource.EnumSpec{
				{Value: "restricted"},
				{Value: "unrestricted"},
			},
		},
		"supported_currencies": {
			Type:        "array",
			Description: "The currencies the FinancialAccount can hold a balance in.",
			Required:    true,
		},
		"features.outbound_transfers.ach.requested": {
			Type:        "boolean",
			Description: "Whether the FinancialAccount should have the Feature.",
			Required:    true,
		},
		"features.outbound_transfers.us_domestic_wire.requested": {
			Type:        "boolean",
			Description: "Whether the FinancialAccount should have the Feature.",
			Required:    true,
		},
		"features.financial_addresses.aba.requested": {
			Type:        "boolean",
			Description: "Whether the FinancialAccount should have the Feature.",
			Required:    true,
		},
		"features.intra_stripe_flows.requested": {
			Type:        "boolean",
			Description: "Whether the FinancialAccount should have the Feature.",
			Required:    true,
		},
		"features.outbound_payments.ach.requested": {
			Type:        "boolean",
			Description: "Whether the FinancialAccount should have the Feature.",
			Required:    true,
		},
		"platform_restrictions.inbound_flows": {
			Type:        "string",
			Description: "Restricts all inbound money movement.",
			Enum: []resource.EnumSpec{
				{Value: "restricted"},
				{Value: "unrestricted"},
			},
		},
		"features.card_issuing.requested": {
			Type:        "boolean",
			Description: "Whether the FinancialAccount should have the Feature.",
			Required:    true,
		},
		"features.deposit_insurance.requested": {
			Type:        "boolean",
			Description: "Whether the FinancialAccount should have the Feature.",
			Required:    true,
		},
		"features.inbound_transfers.ach.requested": {
			Type:        "boolean",
			Description: "Whether the FinancialAccount should have the Feature.",
			Required:    true,
		},
		"features.outbound_payments.us_domestic_wire.requested": {
			Type:        "boolean",
			Description: "Whether the FinancialAccount should have the Feature.",
			Required:    true,
		},
	},
}

var V1TreasuryFinancialAccountsUpdate = resource.OperationSpec{
	Name:    "update",
	Path:    "/v1/treasury/financial_accounts/{financial_account}",
	Method:  "POST",
	Summary: "Update a FinancialAccount",
	Params: map[string]*resource.ParamSpec{
		"forwarding_settings.financial_account": {
			Type:        "string",
			Description: "The financial_account id",
		},
		"nickname": {
			Type:        "string",
			Description: "The nickname for the FinancialAccount.",
		},
		"platform_restrictions.outbound_flows": {
			Type:        "string",
			Description: "Restricts all outbound money movement.",
			Enum: []resource.EnumSpec{
				{Value: "restricted"},
				{Value: "unrestricted"},
			},
		},
		"features.outbound_transfers.us_domestic_wire.requested": {
			Type:        "boolean",
			Description: "Whether the FinancialAccount should have the Feature.",
			Required:    true,
		},
		"features.card_issuing.requested": {
			Type:        "boolean",
			Description: "Whether the FinancialAccount should have the Feature.",
			Required:    true,
		},
		"features.outbound_payments.us_domestic_wire.requested": {
			Type:        "boolean",
			Description: "Whether the FinancialAccount should have the Feature.",
			Required:    true,
		},
		"features.deposit_insurance.requested": {
			Type:        "boolean",
			Description: "Whether the FinancialAccount should have the Feature.",
			Required:    true,
		},
		"features.financial_addresses.aba.requested": {
			Type:        "boolean",
			Description: "Whether the FinancialAccount should have the Feature.",
			Required:    true,
		},
		"forwarding_settings.payment_method": {
			Type:        "string",
			Description: "The payment_method or bank account id. This needs to be a verified bank account.",
		},
		"platform_restrictions.inbound_flows": {
			Type:        "string",
			Description: "Restricts all inbound money movement.",
			Enum: []resource.EnumSpec{
				{Value: "restricted"},
				{Value: "unrestricted"},
			},
		},
		"features.outbound_transfers.ach.requested": {
			Type:        "boolean",
			Description: "Whether the FinancialAccount should have the Feature.",
			Required:    true,
		},
		"features.inbound_transfers.ach.requested": {
			Type:        "boolean",
			Description: "Whether the FinancialAccount should have the Feature.",
			Required:    true,
		},
		"features.intra_stripe_flows.requested": {
			Type:        "boolean",
			Description: "Whether the FinancialAccount should have the Feature.",
			Required:    true,
		},
		"features.outbound_payments.ach.requested": {
			Type:        "boolean",
			Description: "Whether the FinancialAccount should have the Feature.",
			Required:    true,
		},
		"forwarding_settings.type": {
			Type:        "string",
			Description: "The type of the bank account provided. This can be either \"financial_account\" or \"payment_method\"",
			Required:    true,
			Enum: []resource.EnumSpec{
				{Value: "financial_account"},
				{Value: "payment_method"},
			},
		},
	},
}

var V1TreasuryFinancialAccountsClose = resource.OperationSpec{
	Name:    "close",
	Path:    "/v1/treasury/financial_accounts/{financial_account}/close",
	Method:  "POST",
	Summary: "Close a FinancialAccount",
	Params: map[string]*resource.ParamSpec{
		"forwarding_settings.financial_account": {
			Type:        "string",
			Description: "The financial_account id",
		},
		"forwarding_settings.payment_method": {
			Type:        "string",
			Description: "The payment_method or bank account id. This needs to be a verified bank account.",
		},
		"forwarding_settings.type": {
			Type:        "string",
			Description: "The type of the bank account provided. This can be either \"financial_account\" or \"payment_method\"",
			Required:    true,
			Enum: []resource.EnumSpec{
				{Value: "financial_account"},
				{Value: "payment_method"},
			},
		},
	},
}

var V1TreasuryFinancialAccountsUpdateFeatures = resource.OperationSpec{
	Name:    "update_features",
	Path:    "/v1/treasury/financial_accounts/{financial_account}/features",
	Method:  "POST",
	Summary: "Update FinancialAccount Features",
	Params: map[string]*resource.ParamSpec{
		"outbound_transfers.ach.requested": {
			Type:        "boolean",
			Description: "Whether the FinancialAccount should have the Feature.",
			Required:    true,
		},
		"outbound_transfers.us_domestic_wire.requested": {
			Type:        "boolean",
			Description: "Whether the FinancialAccount should have the Feature.",
			Required:    true,
		},
		"card_issuing.requested": {
			Type:        "boolean",
			Description: "Whether the FinancialAccount should have the Feature.",
			Required:    true,
		},
		"intra_stripe_flows.requested": {
			Type:        "boolean",
			Description: "Whether the FinancialAccount should have the Feature.",
			Required:    true,
		},
		"outbound_payments.us_domestic_wire.requested": {
			Type:        "boolean",
			Description: "Whether the FinancialAccount should have the Feature.",
			Required:    true,
		},
		"deposit_insurance.requested": {
			Type:        "boolean",
			Description: "Whether the FinancialAccount should have the Feature.",
			Required:    true,
		},
		"financial_addresses.aba.requested": {
			Type:        "boolean",
			Description: "Whether the FinancialAccount should have the Feature.",
			Required:    true,
		},
		"inbound_transfers.ach.requested": {
			Type:        "boolean",
			Description: "Whether the FinancialAccount should have the Feature.",
			Required:    true,
		},
		"outbound_payments.ach.requested": {
			Type:        "boolean",
			Description: "Whether the FinancialAccount should have the Feature.",
			Required:    true,
		},
	},
}

var V1TreasuryTransactionsList = resource.OperationSpec{
	Name:    "list",
	Path:    "/v1/treasury/transactions",
	Method:  "GET",
	Summary: "List all Transactions",
	Params: map[string]*resource.ParamSpec{
		"financial_account": {
			Type:        "string",
			Description: "Returns objects associated with this FinancialAccount.",
			Required:    true,
		},
		"limit": {
			Type:        "integer",
			Description: "A limit on the number of objects to be returned. Limit can range between 1 and 100, and the default is 10.",
		},
		"order_by": {
			Type:        "string",
			Description: "The results are in reverse chronological order by `created` or `posted_at`. The default is `created`.",
			Enum: []resource.EnumSpec{
				{Value: "created"},
				{Value: "posted_at"},
			},
		},
		"starting_after": {
			Type:        "string",
			Description: "A cursor for use in pagination. `starting_after` is an object ID that defines your place in the list. For instance, if you make a list request and receive 100 objects, ending with `obj_foo`, your subsequent call can include `starting_after=obj_foo` in order to fetch the next page of the list.",
		},
		"status": {
			Type:        "string",
			Description: "Only return Transactions that have the given status: `open`, `posted`, or `void`.",
			Enum: []resource.EnumSpec{
				{Value: "open"},
				{Value: "posted"},
				{Value: "void"},
			},
		},
		"created": {
			Type:        "integer",
			Description: "Only return Transactions that were created during the given date interval.",
		},
		"ending_before": {
			Type:        "string",
			Description: "A cursor for use in pagination. `ending_before` is an object ID that defines your place in the list. For instance, if you make a list request and receive 100 objects, starting with `obj_bar`, your subsequent call can include `ending_before=obj_bar` in order to fetch the previous page of the list.",
		},
	},
}

var V1TreasuryTransactionsRetrieve = resource.OperationSpec{
	Name:    "retrieve",
	Path:    "/v1/treasury/transactions/{id}",
	Method:  "GET",
	Summary: "Retrieve a Transaction",
}

var V1TreasuryDebitReversalsRetrieve = resource.OperationSpec{
	Name:    "retrieve",
	Path:    "/v1/treasury/debit_reversals/{debit_reversal}",
	Method:  "GET",
	Summary: "Retrieve a DebitReversal",
}

var V1TreasuryDebitReversalsCreate = resource.OperationSpec{
	Name:    "create",
	Path:    "/v1/treasury/debit_reversals",
	Method:  "POST",
	Summary: "Create a DebitReversal",
	Params: map[string]*resource.ParamSpec{
		"received_debit": {
			Type:        "string",
			Description: "The ReceivedDebit to reverse.",
			Required:    true,
		},
	},
}

var V1TreasuryDebitReversalsList = resource.OperationSpec{
	Name:    "list",
	Path:    "/v1/treasury/debit_reversals",
	Method:  "GET",
	Summary: "List all DebitReversals",
	Params: map[string]*resource.ParamSpec{
		"ending_before": {
			Type:        "string",
			Description: "A cursor for use in pagination. `ending_before` is an object ID that defines your place in the list. For instance, if you make a list request and receive 100 objects, starting with `obj_bar`, your subsequent call can include `ending_before=obj_bar` in order to fetch the previous page of the list.",
		},
		"financial_account": {
			Type:        "string",
			Description: "Returns objects associated with this FinancialAccount.",
			Required:    true,
		},
		"limit": {
			Type:        "integer",
			Description: "A limit on the number of objects to be returned. Limit can range between 1 and 100, and the default is 10.",
		},
		"received_debit": {
			Type:        "string",
			Description: "Only return DebitReversals for the ReceivedDebit ID.",
		},
		"resolution": {
			Type:        "string",
			Description: "Only return DebitReversals for a given resolution.",
			Enum: []resource.EnumSpec{
				{Value: "lost"},
				{Value: "won"},
			},
		},
		"starting_after": {
			Type:        "string",
			Description: "A cursor for use in pagination. `starting_after` is an object ID that defines your place in the list. For instance, if you make a list request and receive 100 objects, ending with `obj_foo`, your subsequent call can include `starting_after=obj_foo` in order to fetch the next page of the list.",
		},
		"status": {
			Type:        "string",
			Description: "Only return DebitReversals for a given status.",
			Enum: []resource.EnumSpec{
				{Value: "canceled"},
				{Value: "completed"},
				{Value: "processing"},
			},
		},
	},
}

var V1TreasuryOutboundTransfersList = resource.OperationSpec{
	Name:    "list",
	Path:    "/v1/treasury/outbound_transfers",
	Method:  "GET",
	Summary: "List all OutboundTransfers",
	Params: map[string]*resource.ParamSpec{
		"financial_account": {
			Type:        "string",
			Description: "Returns objects associated with this FinancialAccount.",
			Required:    true,
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
			Description: "Only return OutboundTransfers that have the given status: `processing`, `canceled`, `failed`, `posted`, or `returned`.",
			Enum: []resource.EnumSpec{
				{Value: "canceled"},
				{Value: "failed"},
				{Value: "posted"},
				{Value: "processing"},
				{Value: "returned"},
			},
		},
		"ending_before": {
			Type:        "string",
			Description: "A cursor for use in pagination. `ending_before` is an object ID that defines your place in the list. For instance, if you make a list request and receive 100 objects, starting with `obj_bar`, your subsequent call can include `ending_before=obj_bar` in order to fetch the previous page of the list.",
		},
	},
}

var V1TreasuryOutboundTransfersRetrieve = resource.OperationSpec{
	Name:    "retrieve",
	Path:    "/v1/treasury/outbound_transfers/{outbound_transfer}",
	Method:  "GET",
	Summary: "Retrieve an OutboundTransfer",
}

var V1TreasuryOutboundTransfersCreate = resource.OperationSpec{
	Name:    "create",
	Path:    "/v1/treasury/outbound_transfers",
	Method:  "POST",
	Summary: "Create an OutboundTransfer",
	Params: map[string]*resource.ParamSpec{
		"financial_account": {
			Type:        "string",
			Description: "The FinancialAccount to pull funds from.",
			Required:    true,
		},
		"statement_descriptor": {
			Type:        "string",
			Description: "Statement descriptor to be shown on the receiving end of an OutboundTransfer. Maximum 10 characters for `ach` transfers or 140 characters for `us_domestic_wire` transfers. The default value is \"transfer\". Can only include -#.$&*, spaces, and alphanumeric characters.",
		},
		"amount": {
			Type:        "integer",
			Description: "Amount (in cents) to be transferred.",
			Required:    true,
		},
		"description": {
			Type:        "string",
			Description: "An arbitrary string attached to the object. Often useful for displaying to users.",
		},
		"destination_payment_method": {
			Type:        "string",
			Description: "The PaymentMethod to use as the payment instrument for the OutboundTransfer.",
		},
		"destination_payment_method_data.type": {
			Type:        "string",
			Description: "The type of the destination.",
			Required:    true,
			Enum: []resource.EnumSpec{
				{Value: "financial_account"},
			},
		},
		"destination_payment_method_data.financial_account": {
			Type:        "string",
			Description: "Required if type is set to `financial_account`. The FinancialAccount ID to send funds to.",
		},
		"currency": {
			Type:        "string",
			Description: "Three-letter [ISO currency code](https://www.iso.org/iso-4217-currency-codes.html), in lowercase. Must be a [supported currency](https://stripe.com/docs/currencies).",
			Required:    true,
			Format:      "currency",
		},
	},
}

var V1TreasuryOutboundTransfersCancel = resource.OperationSpec{
	Name:    "cancel",
	Path:    "/v1/treasury/outbound_transfers/{outbound_transfer}/cancel",
	Method:  "POST",
	Summary: "Cancel an OutboundTransfer",
}

var V1TreasuryOutboundTransfersTestHelpersUpdate = resource.OperationSpec{
	Name:    "update",
	Path:    "/v1/test_helpers/treasury/outbound_transfers/{outbound_transfer}",
	Method:  "POST",
	Summary: "Test mode: Update an OutboundTransfer",
	Params: map[string]*resource.ParamSpec{
		"tracking_details.us_domestic_wire.imad": {
			Type:        "string",
			Description: "IMAD for funds sent over the `us_domestic_wire` network.",
		},
		"tracking_details.us_domestic_wire.omad": {
			Type:        "string",
			Description: "OMAD for funds sent over the `us_domestic_wire` network.",
		},
		"tracking_details.ach.trace_id": {
			Type:        "string",
			Description: "ACH trace ID for funds sent over the `ach` network.",
			Required:    true,
		},
		"tracking_details.type": {
			Type:        "string",
			Description: "The US bank account network used to send funds.",
			Required:    true,
			Enum: []resource.EnumSpec{
				{Value: "ach"},
				{Value: "us_domestic_wire"},
			},
		},
		"tracking_details.us_domestic_wire.chips": {
			Type:        "string",
			Description: "CHIPS System Sequence Number (SSN) for funds sent over the `us_domestic_wire` network.",
		},
	},
}

var V1TreasuryOutboundTransfersTestHelpersFail = resource.OperationSpec{
	Name:    "fail",
	Path:    "/v1/test_helpers/treasury/outbound_transfers/{outbound_transfer}/fail",
	Method:  "POST",
	Summary: "Test mode: Fail an OutboundTransfer",
}

var V1TreasuryOutboundTransfersTestHelpersPost = resource.OperationSpec{
	Name:    "post",
	Path:    "/v1/test_helpers/treasury/outbound_transfers/{outbound_transfer}/post",
	Method:  "POST",
	Summary: "Test mode: Post an OutboundTransfer",
}

var V1TreasuryOutboundTransfersTestHelpersReturnOutboundTransfer = resource.OperationSpec{
	Name:    "return_outbound_transfer",
	Path:    "/v1/test_helpers/treasury/outbound_transfers/{outbound_transfer}/return",
	Method:  "POST",
	Summary: "Test mode: Return an OutboundTransfer",
	Params: map[string]*resource.ParamSpec{
		"returned_details.code": {
			Type:        "string",
			Description: "Reason for the return.",
			Enum: []resource.EnumSpec{
				{Value: "account_closed"},
				{Value: "account_frozen"},
				{Value: "bank_account_restricted"},
				{Value: "bank_ownership_changed"},
				{Value: "declined"},
				{Value: "incorrect_account_holder_name"},
				{Value: "invalid_account_number"},
				{Value: "invalid_currency"},
				{Value: "no_account"},
				{Value: "other"},
			},
		},
	},
}

var V1TreasuryInboundTransfersCancel = resource.OperationSpec{
	Name:    "cancel",
	Path:    "/v1/treasury/inbound_transfers/{inbound_transfer}/cancel",
	Method:  "POST",
	Summary: "Cancel an InboundTransfer",
}

var V1TreasuryInboundTransfersList = resource.OperationSpec{
	Name:    "list",
	Path:    "/v1/treasury/inbound_transfers",
	Method:  "GET",
	Summary: "List all InboundTransfers",
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
			Description: "Only return InboundTransfers that have the given status: `processing`, `succeeded`, `failed` or `canceled`.",
			Enum: []resource.EnumSpec{
				{Value: "canceled"},
				{Value: "failed"},
				{Value: "processing"},
				{Value: "succeeded"},
			},
		},
		"ending_before": {
			Type:        "string",
			Description: "A cursor for use in pagination. `ending_before` is an object ID that defines your place in the list. For instance, if you make a list request and receive 100 objects, starting with `obj_bar`, your subsequent call can include `ending_before=obj_bar` in order to fetch the previous page of the list.",
		},
		"financial_account": {
			Type:        "string",
			Description: "Returns objects associated with this FinancialAccount.",
			Required:    true,
		},
	},
}

var V1TreasuryInboundTransfersRetrieve = resource.OperationSpec{
	Name:    "retrieve",
	Path:    "/v1/treasury/inbound_transfers/{id}",
	Method:  "GET",
	Summary: "Retrieve an InboundTransfer",
}

var V1TreasuryInboundTransfersCreate = resource.OperationSpec{
	Name:    "create",
	Path:    "/v1/treasury/inbound_transfers",
	Method:  "POST",
	Summary: "Create an InboundTransfer",
	Params: map[string]*resource.ParamSpec{
		"statement_descriptor": {
			Type:        "string",
			Description: "The complete description that appears on your customers' statements. Maximum 10 characters. Can only include -#.$&*, spaces, and alphanumeric characters.",
		},
		"amount": {
			Type:        "integer",
			Description: "Amount (in cents) to be transferred.",
			Required:    true,
		},
		"currency": {
			Type:        "string",
			Description: "Three-letter [ISO currency code](https://www.iso.org/iso-4217-currency-codes.html), in lowercase. Must be a [supported currency](https://stripe.com/docs/currencies).",
			Required:    true,
			Format:      "currency",
		},
		"description": {
			Type:        "string",
			Description: "An arbitrary string attached to the object. Often useful for displaying to users.",
		},
		"financial_account": {
			Type:        "string",
			Description: "The FinancialAccount to send funds to.",
			Required:    true,
		},
		"origin_payment_method": {
			Type:        "string",
			Description: "The origin payment method to be debited for the InboundTransfer.",
			Required:    true,
		},
	},
}

var V1TreasuryInboundTransfersTestHelpersFail = resource.OperationSpec{
	Name:    "fail",
	Path:    "/v1/test_helpers/treasury/inbound_transfers/{id}/fail",
	Method:  "POST",
	Summary: "Test mode: Fail an InboundTransfer",
	Params: map[string]*resource.ParamSpec{
		"failure_details.code": {
			Type:        "string",
			Description: "Reason for the failure.",
			Enum: []resource.EnumSpec{
				{Value: "account_closed"},
				{Value: "account_frozen"},
				{Value: "bank_account_restricted"},
				{Value: "bank_ownership_changed"},
				{Value: "debit_not_authorized"},
				{Value: "incorrect_account_holder_address"},
				{Value: "incorrect_account_holder_name"},
				{Value: "incorrect_account_holder_tax_id"},
				{Value: "insufficient_funds"},
				{Value: "invalid_account_number"},
				{Value: "invalid_currency"},
				{Value: "no_account"},
				{Value: "other"},
			},
		},
	},
}

var V1TreasuryInboundTransfersTestHelpersReturnInboundTransfer = resource.OperationSpec{
	Name:    "return_inbound_transfer",
	Path:    "/v1/test_helpers/treasury/inbound_transfers/{id}/return",
	Method:  "POST",
	Summary: "Test mode: Return an InboundTransfer",
}

var V1TreasuryInboundTransfersTestHelpersSucceed = resource.OperationSpec{
	Name:    "succeed",
	Path:    "/v1/test_helpers/treasury/inbound_transfers/{id}/succeed",
	Method:  "POST",
	Summary: "Test mode: Succeed an InboundTransfer",
}

var V1TreasuryTransactionEntrysList = resource.OperationSpec{
	Name:    "list",
	Path:    "/v1/treasury/transaction_entries",
	Method:  "GET",
	Summary: "List all TransactionEntries",
	Params: map[string]*resource.ParamSpec{
		"ending_before": {
			Type:        "string",
			Description: "A cursor for use in pagination. `ending_before` is an object ID that defines your place in the list. For instance, if you make a list request and receive 100 objects, starting with `obj_bar`, your subsequent call can include `ending_before=obj_bar` in order to fetch the previous page of the list.",
		},
		"financial_account": {
			Type:        "string",
			Description: "Returns objects associated with this FinancialAccount.",
			Required:    true,
		},
		"limit": {
			Type:        "integer",
			Description: "A limit on the number of objects to be returned. Limit can range between 1 and 100, and the default is 10.",
		},
		"order_by": {
			Type:        "string",
			Description: "The results are in reverse chronological order by `created` or `effective_at`. The default is `created`.",
			Enum: []resource.EnumSpec{
				{Value: "created"},
				{Value: "effective_at"},
			},
		},
		"starting_after": {
			Type:        "string",
			Description: "A cursor for use in pagination. `starting_after` is an object ID that defines your place in the list. For instance, if you make a list request and receive 100 objects, ending with `obj_foo`, your subsequent call can include `starting_after=obj_foo` in order to fetch the next page of the list.",
		},
		"transaction": {
			Type:        "string",
			Description: "Only return TransactionEntries associated with this Transaction.",
		},
		"created": {
			Type:        "integer",
			Description: "Only return TransactionEntries that were created during the given date interval.",
		},
		"effective_at": {
			Type: "integer",
		},
	},
}

var V1TreasuryTransactionEntrysRetrieve = resource.OperationSpec{
	Name:    "retrieve",
	Path:    "/v1/treasury/transaction_entries/{id}",
	Method:  "GET",
	Summary: "Retrieve a TransactionEntry",
}

var V1TreasuryReceivedDebitsRetrieve = resource.OperationSpec{
	Name:    "retrieve",
	Path:    "/v1/treasury/received_debits/{id}",
	Method:  "GET",
	Summary: "Retrieve a ReceivedDebit",
}

var V1TreasuryReceivedDebitsList = resource.OperationSpec{
	Name:    "list",
	Path:    "/v1/treasury/received_debits",
	Method:  "GET",
	Summary: "List all ReceivedDebits",
	Params: map[string]*resource.ParamSpec{
		"starting_after": {
			Type:        "string",
			Description: "A cursor for use in pagination. `starting_after` is an object ID that defines your place in the list. For instance, if you make a list request and receive 100 objects, ending with `obj_foo`, your subsequent call can include `starting_after=obj_foo` in order to fetch the next page of the list.",
		},
		"status": {
			Type:        "string",
			Description: "Only return ReceivedDebits that have the given status: `succeeded` or `failed`.",
			Enum: []resource.EnumSpec{
				{Value: "failed"},
				{Value: "succeeded"},
			},
		},
		"ending_before": {
			Type:        "string",
			Description: "A cursor for use in pagination. `ending_before` is an object ID that defines your place in the list. For instance, if you make a list request and receive 100 objects, starting with `obj_bar`, your subsequent call can include `ending_before=obj_bar` in order to fetch the previous page of the list.",
		},
		"financial_account": {
			Type:        "string",
			Description: "The FinancialAccount that funds were pulled from.",
			Required:    true,
		},
		"limit": {
			Type:        "integer",
			Description: "A limit on the number of objects to be returned. Limit can range between 1 and 100, and the default is 10.",
		},
	},
}

var V1TreasuryReceivedDebitsTestHelpersCreate = resource.OperationSpec{
	Name:    "create",
	Path:    "/v1/test_helpers/treasury/received_debits",
	Method:  "POST",
	Summary: "Test mode: Create a ReceivedDebit",
	Params: map[string]*resource.ParamSpec{
		"amount": {
			Type:        "integer",
			Description: "Amount (in cents) to be transferred.",
			Required:    true,
		},
		"initiating_payment_method_details.type": {
			Type:        "string",
			Description: "The source type.",
			Required:    true,
			Enum: []resource.EnumSpec{
				{Value: "us_bank_account"},
			},
		},
		"initiating_payment_method_details.us_bank_account.account_holder_name": {
			Type:        "string",
			Description: "The bank account holder's name.",
		},
		"initiating_payment_method_details.us_bank_account.account_number": {
			Type:        "string",
			Description: "The bank account number.",
		},
		"initiating_payment_method_details.us_bank_account.routing_number": {
			Type:        "string",
			Description: "The bank account's routing number.",
		},
		"network": {
			Type:        "string",
			Description: "Specifies the network rails to be used. If not set, will default to the PaymentMethod's preferred network. See the [docs](https://docs.stripe.com/treasury/money-movement/timelines) to learn more about money movement timelines for each network type.",
			Required:    true,
			Enum: []resource.EnumSpec{
				{Value: "ach"},
			},
		},
		"currency": {
			Type:        "string",
			Description: "Three-letter [ISO currency code](https://www.iso.org/iso-4217-currency-codes.html), in lowercase. Must be a [supported currency](https://stripe.com/docs/currencies).",
			Required:    true,
			Format:      "currency",
		},
		"description": {
			Type:        "string",
			Description: "An arbitrary string attached to the object. Often useful for displaying to users.",
		},
		"financial_account": {
			Type:        "string",
			Description: "The FinancialAccount to pull funds from.",
			Required:    true,
		},
	},
}

var V1TreasuryCreditReversalsCreate = resource.OperationSpec{
	Name:    "create",
	Path:    "/v1/treasury/credit_reversals",
	Method:  "POST",
	Summary: "Create a CreditReversal",
	Params: map[string]*resource.ParamSpec{
		"received_credit": {
			Type:        "string",
			Description: "The ReceivedCredit to reverse.",
			Required:    true,
		},
	},
}

var V1TreasuryCreditReversalsList = resource.OperationSpec{
	Name:    "list",
	Path:    "/v1/treasury/credit_reversals",
	Method:  "GET",
	Summary: "List all CreditReversals",
	Params: map[string]*resource.ParamSpec{
		"limit": {
			Type:        "integer",
			Description: "A limit on the number of objects to be returned. Limit can range between 1 and 100, and the default is 10.",
		},
		"received_credit": {
			Type:        "string",
			Description: "Only return CreditReversals for the ReceivedCredit ID.",
		},
		"starting_after": {
			Type:        "string",
			Description: "A cursor for use in pagination. `starting_after` is an object ID that defines your place in the list. For instance, if you make a list request and receive 100 objects, ending with `obj_foo`, your subsequent call can include `starting_after=obj_foo` in order to fetch the next page of the list.",
		},
		"status": {
			Type:        "string",
			Description: "Only return CreditReversals for a given status.",
			Enum: []resource.EnumSpec{
				{Value: "canceled"},
				{Value: "posted"},
				{Value: "processing"},
			},
		},
		"ending_before": {
			Type:        "string",
			Description: "A cursor for use in pagination. `ending_before` is an object ID that defines your place in the list. For instance, if you make a list request and receive 100 objects, starting with `obj_bar`, your subsequent call can include `ending_before=obj_bar` in order to fetch the previous page of the list.",
		},
		"financial_account": {
			Type:        "string",
			Description: "Returns objects associated with this FinancialAccount.",
			Required:    true,
		},
	},
}

var V1TreasuryCreditReversalsRetrieve = resource.OperationSpec{
	Name:    "retrieve",
	Path:    "/v1/treasury/credit_reversals/{credit_reversal}",
	Method:  "GET",
	Summary: "Retrieve a CreditReversal",
}

var V1TreasuryReceivedCreditsList = resource.OperationSpec{
	Name:    "list",
	Path:    "/v1/treasury/received_credits",
	Method:  "GET",
	Summary: "List all ReceivedCredits",
	Params: map[string]*resource.ParamSpec{
		"starting_after": {
			Type:        "string",
			Description: "A cursor for use in pagination. `starting_after` is an object ID that defines your place in the list. For instance, if you make a list request and receive 100 objects, ending with `obj_foo`, your subsequent call can include `starting_after=obj_foo` in order to fetch the next page of the list.",
		},
		"status": {
			Type:        "string",
			Description: "Only return ReceivedCredits that have the given status: `succeeded` or `failed`.",
			Enum: []resource.EnumSpec{
				{Value: "failed"},
				{Value: "succeeded"},
			},
		},
		"ending_before": {
			Type:        "string",
			Description: "A cursor for use in pagination. `ending_before` is an object ID that defines your place in the list. For instance, if you make a list request and receive 100 objects, starting with `obj_bar`, your subsequent call can include `ending_before=obj_bar` in order to fetch the previous page of the list.",
		},
		"financial_account": {
			Type:        "string",
			Description: "The FinancialAccount that received the funds.",
			Required:    true,
		},
		"limit": {
			Type:        "integer",
			Description: "A limit on the number of objects to be returned. Limit can range between 1 and 100, and the default is 10.",
		},
	},
}

var V1TreasuryReceivedCreditsRetrieve = resource.OperationSpec{
	Name:    "retrieve",
	Path:    "/v1/treasury/received_credits/{id}",
	Method:  "GET",
	Summary: "Retrieve a ReceivedCredit",
}

var V1TreasuryReceivedCreditsTestHelpersCreate = resource.OperationSpec{
	Name:    "create",
	Path:    "/v1/test_helpers/treasury/received_credits",
	Method:  "POST",
	Summary: "Test mode: Create a ReceivedCredit",
	Params: map[string]*resource.ParamSpec{
		"amount": {
			Type:        "integer",
			Description: "Amount (in cents) to be transferred.",
			Required:    true,
		},
		"financial_account": {
			Type:        "string",
			Description: "The FinancialAccount to send funds to.",
			Required:    true,
		},
		"initiating_payment_method_details.type": {
			Type:        "string",
			Description: "The source type.",
			Required:    true,
			Enum: []resource.EnumSpec{
				{Value: "us_bank_account"},
			},
		},
		"initiating_payment_method_details.us_bank_account.routing_number": {
			Type:        "string",
			Description: "The bank account's routing number.",
		},
		"initiating_payment_method_details.us_bank_account.account_holder_name": {
			Type:        "string",
			Description: "The bank account holder's name.",
		},
		"initiating_payment_method_details.us_bank_account.account_number": {
			Type:        "string",
			Description: "The bank account number.",
		},
		"network": {
			Type:        "string",
			Description: "Specifies the network rails to be used. If not set, will default to the PaymentMethod's preferred network. See the [docs](https://docs.stripe.com/treasury/money-movement/timelines) to learn more about money movement timelines for each network type.",
			Required:    true,
			Enum: []resource.EnumSpec{
				{Value: "ach"},
				{Value: "us_domestic_wire"},
			},
		},
		"currency": {
			Type:        "string",
			Description: "Three-letter [ISO currency code](https://www.iso.org/iso-4217-currency-codes.html), in lowercase. Must be a [supported currency](https://stripe.com/docs/currencies).",
			Required:    true,
			Format:      "currency",
		},
		"description": {
			Type:        "string",
			Description: "An arbitrary string attached to the object. Often useful for displaying to users.",
		},
	},
}
