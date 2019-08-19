// This file is generated; DO NOT EDIT.

package cmd

import (
	"net/http"

	"github.com/spf13/cobra"

	"github.com/stripe/stripe-cli/pkg/cmd/resource"
)

func addAllResourcesCmds(rootCmd *cobra.Command) {
	// Namespace commands
	_ = resource.NewNamespaceCmd(rootCmd, "")
	nsCheckoutCmd := resource.NewNamespaceCmd(rootCmd, "checkout")
	nsIssuingCmd := resource.NewNamespaceCmd(rootCmd, "issuing")
	nsRadarCmd := resource.NewNamespaceCmd(rootCmd, "radar")
	nsReportingCmd := resource.NewNamespaceCmd(rootCmd, "reporting")
	nsTerminalCmd := resource.NewNamespaceCmd(rootCmd, "terminal")

	// Resource commands
	r3dSecureCmd := resource.NewResourceCmd(rootCmd, "3d_secure")
	rAccountLinksCmd := resource.NewResourceCmd(rootCmd, "account_links")
	rAccountsCmd := resource.NewResourceCmd(rootCmd, "accounts")
	rApplePayDomainsCmd := resource.NewResourceCmd(rootCmd, "apple_pay_domains")
	rApplicationFeesCmd := resource.NewResourceCmd(rootCmd, "application_fees")
	rBalanceCmd := resource.NewResourceCmd(rootCmd, "balance")
	rBalanceTransactionsCmd := resource.NewResourceCmd(rootCmd, "balance_transactions")
	rBankAccountsCmd := resource.NewResourceCmd(rootCmd, "bank_accounts")
	rBitcoinReceiversCmd := resource.NewResourceCmd(rootCmd, "bitcoin_receivers")
	rBitcoinTransactionsCmd := resource.NewResourceCmd(rootCmd, "bitcoin_transactions")
	rCapabilitiesCmd := resource.NewResourceCmd(rootCmd, "capabilities")
	rCardsCmd := resource.NewResourceCmd(rootCmd, "cards")
	rChargesCmd := resource.NewResourceCmd(rootCmd, "charges")
	rCountrySpecsCmd := resource.NewResourceCmd(rootCmd, "country_specs")
	rCouponsCmd := resource.NewResourceCmd(rootCmd, "coupons")
	rCreditNotesCmd := resource.NewResourceCmd(rootCmd, "credit_notes")
	rCustomerBalanceTransactionsCmd := resource.NewResourceCmd(rootCmd, "customer_balance_transactions")
	rCustomersCmd := resource.NewResourceCmd(rootCmd, "customers")
	rDisputesCmd := resource.NewResourceCmd(rootCmd, "disputes")
	rEphemeralKeysCmd := resource.NewResourceCmd(rootCmd, "ephemeral_keys")
	rEventsCmd := resource.NewResourceCmd(rootCmd, "events")
	rExchangeRatesCmd := resource.NewResourceCmd(rootCmd, "exchange_rates")
	rExternalAccountsCmd := resource.NewResourceCmd(rootCmd, "external_accounts")
	rFeeRefundsCmd := resource.NewResourceCmd(rootCmd, "fee_refunds")
	rFileLinksCmd := resource.NewResourceCmd(rootCmd, "file_links")
	rFilesCmd := resource.NewResourceCmd(rootCmd, "files")
	rInvoiceitemsCmd := resource.NewResourceCmd(rootCmd, "invoiceitems")
	rInvoicesCmd := resource.NewResourceCmd(rootCmd, "invoices")
	rIssuerFraudRecordsCmd := resource.NewResourceCmd(rootCmd, "issuer_fraud_records")
	rLineItemsCmd := resource.NewResourceCmd(rootCmd, "line_items")
	rLoginLinksCmd := resource.NewResourceCmd(rootCmd, "login_links")
	rOrderReturnsCmd := resource.NewResourceCmd(rootCmd, "order_returns")
	rOrdersCmd := resource.NewResourceCmd(rootCmd, "orders")
	rPaymentIntentsCmd := resource.NewResourceCmd(rootCmd, "payment_intents")
	rPaymentMethodsCmd := resource.NewResourceCmd(rootCmd, "payment_methods")
	rPaymentSourcesCmd := resource.NewResourceCmd(rootCmd, "payment_sources")
	rPayoutsCmd := resource.NewResourceCmd(rootCmd, "payouts")
	rPersonsCmd := resource.NewResourceCmd(rootCmd, "persons")
	rPlansCmd := resource.NewResourceCmd(rootCmd, "plans")
	rProductsCmd := resource.NewResourceCmd(rootCmd, "products")
	rRecipientsCmd := resource.NewResourceCmd(rootCmd, "recipients")
	rRefundsCmd := resource.NewResourceCmd(rootCmd, "refunds")
	rReviewsCmd := resource.NewResourceCmd(rootCmd, "reviews")
	rScheduledQueryRunsCmd := resource.NewResourceCmd(rootCmd, "scheduled_query_runs")
	rSetupIntentsCmd := resource.NewResourceCmd(rootCmd, "setup_intents")
	rSkusCmd := resource.NewResourceCmd(rootCmd, "skus")
	rSourcesCmd := resource.NewResourceCmd(rootCmd, "sources")
	rSubscriptionItemsCmd := resource.NewResourceCmd(rootCmd, "subscription_items")
	rSubscriptionSchedulesCmd := resource.NewResourceCmd(rootCmd, "subscription_schedules")
	rSubscriptionsCmd := resource.NewResourceCmd(rootCmd, "subscriptions")
	rTaxIdsCmd := resource.NewResourceCmd(rootCmd, "tax_ids")
	rTaxRatesCmd := resource.NewResourceCmd(rootCmd, "tax_rates")
	rTokensCmd := resource.NewResourceCmd(rootCmd, "tokens")
	rTopupsCmd := resource.NewResourceCmd(rootCmd, "topups")
	rTransferReversalsCmd := resource.NewResourceCmd(rootCmd, "transfer_reversals")
	rTransfersCmd := resource.NewResourceCmd(rootCmd, "transfers")
	rUsageRecordsCmd := resource.NewResourceCmd(rootCmd, "usage_records")
	rWebhookEndpointsCmd := resource.NewResourceCmd(rootCmd, "webhook_endpoints")

	rCheckoutSessionsCmd := resource.NewResourceCmd(nsCheckoutCmd.Cmd, "sessions")

	rIssuingAuthorizationsCmd := resource.NewResourceCmd(nsIssuingCmd.Cmd, "authorizations")
	rIssuingCardholdersCmd := resource.NewResourceCmd(nsIssuingCmd.Cmd, "cardholders")
	rIssuingCardsCmd := resource.NewResourceCmd(nsIssuingCmd.Cmd, "cards")
	rIssuingDisputesCmd := resource.NewResourceCmd(nsIssuingCmd.Cmd, "disputes")
	rIssuingTransactionsCmd := resource.NewResourceCmd(nsIssuingCmd.Cmd, "transactions")

	rRadarEarlyFraudWarningsCmd := resource.NewResourceCmd(nsRadarCmd.Cmd, "early_fraud_warnings")
	rRadarValueListItemsCmd := resource.NewResourceCmd(nsRadarCmd.Cmd, "value_list_items")
	rRadarValueListsCmd := resource.NewResourceCmd(nsRadarCmd.Cmd, "value_lists")

	rReportingReportRunsCmd := resource.NewResourceCmd(nsReportingCmd.Cmd, "report_runs")
	rReportingReportTypesCmd := resource.NewResourceCmd(nsReportingCmd.Cmd, "report_types")

	rTerminalConnectionTokensCmd := resource.NewResourceCmd(nsTerminalCmd.Cmd, "connection_tokens")
	rTerminalLocationsCmd := resource.NewResourceCmd(nsTerminalCmd.Cmd, "locations")
	rTerminalReadersCmd := resource.NewResourceCmd(nsTerminalCmd.Cmd, "readers")

	// Operation commands
	resource.NewOperationCmd(r3dSecureCmd.Cmd, "create", "/v1/3d_secure", http.MethodPost, &Config)
	resource.NewOperationCmd(r3dSecureCmd.Cmd, "retrieve", "/v1/3d_secure/{three_d_secure}", http.MethodGet, &Config)

	resource.NewOperationCmd(rAccountLinksCmd.Cmd, "create", "/v1/account_links", http.MethodPost, &Config)

	resource.NewOperationCmd(rAccountsCmd.Cmd, "capabilities", "/v1/accounts/{account}/capabilities", http.MethodGet, &Config)
	resource.NewOperationCmd(rAccountsCmd.Cmd, "create", "/v1/accounts", http.MethodPost, &Config)
	resource.NewOperationCmd(rAccountsCmd.Cmd, "delete", "/v1/accounts/{account}", http.MethodDelete, &Config)
	resource.NewOperationCmd(rAccountsCmd.Cmd, "list", "/v1/accounts", http.MethodGet, &Config)
	resource.NewOperationCmd(rAccountsCmd.Cmd, "reject", "/v1/accounts/{account}/reject", http.MethodPost, &Config)
	resource.NewOperationCmd(rAccountsCmd.Cmd, "retrieve", "/v1/account", http.MethodGet, &Config)
	resource.NewOperationCmd(rAccountsCmd.Cmd, "update", "/v1/accounts/{account}", http.MethodPost, &Config)

	resource.NewOperationCmd(rApplePayDomainsCmd.Cmd, "create", "/v1/apple_pay/domains", http.MethodPost, &Config)
	resource.NewOperationCmd(rApplePayDomainsCmd.Cmd, "delete", "/v1/apple_pay/domains/{domain}", http.MethodDelete, &Config)
	resource.NewOperationCmd(rApplePayDomainsCmd.Cmd, "list", "/v1/apple_pay/domains", http.MethodGet, &Config)
	resource.NewOperationCmd(rApplePayDomainsCmd.Cmd, "retrieve", "/v1/apple_pay/domains/{domain}", http.MethodGet, &Config)

	resource.NewOperationCmd(rApplicationFeesCmd.Cmd, "list", "/v1/application_fees", http.MethodGet, &Config)
	resource.NewOperationCmd(rApplicationFeesCmd.Cmd, "retrieve", "/v1/application_fees/{id}", http.MethodGet, &Config)

	resource.NewOperationCmd(rBalanceCmd.Cmd, "retrieve", "/v1/balance", http.MethodGet, &Config)

	resource.NewOperationCmd(rBalanceTransactionsCmd.Cmd, "list", "/v1/balance_transactions", http.MethodGet, &Config)
	resource.NewOperationCmd(rBalanceTransactionsCmd.Cmd, "retrieve", "/v1/balance_transactions/{id}", http.MethodGet, &Config)

	resource.NewOperationCmd(rBankAccountsCmd.Cmd, "delete", "/v1/customers/{customer}/sources/{id}", http.MethodDelete, &Config)
	resource.NewOperationCmd(rBankAccountsCmd.Cmd, "update", "/v1/customers/{customer}/sources/{id}", http.MethodPost, &Config)
	resource.NewOperationCmd(rBankAccountsCmd.Cmd, "verify", "/v1/customers/{customer}/sources/{id}/verify", http.MethodPost, &Config)

	resource.NewOperationCmd(rBitcoinReceiversCmd.Cmd, "list", "/v1/bitcoin/receivers", http.MethodGet, &Config)
	resource.NewOperationCmd(rBitcoinReceiversCmd.Cmd, "retrieve", "/v1/bitcoin/receivers/{id}", http.MethodGet, &Config)

	resource.NewOperationCmd(rBitcoinTransactionsCmd.Cmd, "list", "/v1/bitcoin/receivers/{receiver}/transactions", http.MethodGet, &Config)

	resource.NewOperationCmd(rCapabilitiesCmd.Cmd, "list", "/v1/accounts/{account}/capabilities", http.MethodGet, &Config)
	resource.NewOperationCmd(rCapabilitiesCmd.Cmd, "retrieve", "/v1/accounts/{account}/capabilities/{capability}", http.MethodGet, &Config)
	resource.NewOperationCmd(rCapabilitiesCmd.Cmd, "update", "/v1/accounts/{account}/capabilities/{capability}", http.MethodPost, &Config)

	resource.NewOperationCmd(rCardsCmd.Cmd, "delete", "/v1/customers/{customer}/sources/{id}", http.MethodDelete, &Config)
	resource.NewOperationCmd(rCardsCmd.Cmd, "update", "/v1/customers/{customer}/sources/{id}", http.MethodPost, &Config)

	resource.NewOperationCmd(rChargesCmd.Cmd, "capture", "/v1/charges/{charge}/capture", http.MethodPost, &Config)
	resource.NewOperationCmd(rChargesCmd.Cmd, "create", "/v1/charges", http.MethodPost, &Config)
	resource.NewOperationCmd(rChargesCmd.Cmd, "list", "/v1/charges", http.MethodGet, &Config)
	resource.NewOperationCmd(rChargesCmd.Cmd, "retrieve", "/v1/charges/{charge}", http.MethodGet, &Config)
	resource.NewOperationCmd(rChargesCmd.Cmd, "update", "/v1/charges/{charge}", http.MethodPost, &Config)

	resource.NewOperationCmd(rCountrySpecsCmd.Cmd, "list", "/v1/country_specs", http.MethodGet, &Config)
	resource.NewOperationCmd(rCountrySpecsCmd.Cmd, "retrieve", "/v1/country_specs/{country}", http.MethodGet, &Config)

	resource.NewOperationCmd(rCouponsCmd.Cmd, "create", "/v1/coupons", http.MethodPost, &Config)
	resource.NewOperationCmd(rCouponsCmd.Cmd, "delete", "/v1/coupons/{coupon}", http.MethodDelete, &Config)
	resource.NewOperationCmd(rCouponsCmd.Cmd, "list", "/v1/coupons", http.MethodGet, &Config)
	resource.NewOperationCmd(rCouponsCmd.Cmd, "retrieve", "/v1/coupons/{coupon}", http.MethodGet, &Config)
	resource.NewOperationCmd(rCouponsCmd.Cmd, "update", "/v1/coupons/{coupon}", http.MethodPost, &Config)

	resource.NewOperationCmd(rCreditNotesCmd.Cmd, "create", "/v1/credit_notes", http.MethodPost, &Config)
	resource.NewOperationCmd(rCreditNotesCmd.Cmd, "list", "/v1/credit_notes", http.MethodGet, &Config)
	resource.NewOperationCmd(rCreditNotesCmd.Cmd, "retrieve", "/v1/credit_notes/{id}", http.MethodGet, &Config)
	resource.NewOperationCmd(rCreditNotesCmd.Cmd, "update", "/v1/credit_notes/{id}", http.MethodPost, &Config)
	resource.NewOperationCmd(rCreditNotesCmd.Cmd, "void_credit_note", "/v1/credit_notes/{id}/void", http.MethodPost, &Config)

	resource.NewOperationCmd(rCustomerBalanceTransactionsCmd.Cmd, "create", "/v1/customers/{customer}/balance_transactions", http.MethodPost, &Config)
	resource.NewOperationCmd(rCustomerBalanceTransactionsCmd.Cmd, "list", "/v1/customers/{customer}/balance_transactions", http.MethodGet, &Config)
	resource.NewOperationCmd(rCustomerBalanceTransactionsCmd.Cmd, "retrieve", "/v1/customers/{customer}/balance_transactions/{transaction}", http.MethodGet, &Config)
	resource.NewOperationCmd(rCustomerBalanceTransactionsCmd.Cmd, "update", "/v1/customers/{customer}/balance_transactions/{transaction}", http.MethodPost, &Config)

	resource.NewOperationCmd(rCustomersCmd.Cmd, "create", "/v1/customers", http.MethodPost, &Config)
	resource.NewOperationCmd(rCustomersCmd.Cmd, "delete", "/v1/customers/{customer}", http.MethodDelete, &Config)
	resource.NewOperationCmd(rCustomersCmd.Cmd, "delete_discount", "/v1/customers/{customer}/discount", http.MethodDelete, &Config)
	resource.NewOperationCmd(rCustomersCmd.Cmd, "list", "/v1/customers", http.MethodGet, &Config)
	resource.NewOperationCmd(rCustomersCmd.Cmd, "retrieve", "/v1/customers/{customer}", http.MethodGet, &Config)
	resource.NewOperationCmd(rCustomersCmd.Cmd, "update", "/v1/customers/{customer}", http.MethodPost, &Config)

	resource.NewOperationCmd(rDisputesCmd.Cmd, "close", "/v1/disputes/{dispute}/close", http.MethodPost, &Config)
	resource.NewOperationCmd(rDisputesCmd.Cmd, "list", "/v1/disputes", http.MethodGet, &Config)
	resource.NewOperationCmd(rDisputesCmd.Cmd, "retrieve", "/v1/disputes/{dispute}", http.MethodGet, &Config)
	resource.NewOperationCmd(rDisputesCmd.Cmd, "update", "/v1/disputes/{dispute}", http.MethodPost, &Config)

	resource.NewOperationCmd(rEphemeralKeysCmd.Cmd, "create", "/v1/ephemeral_keys", http.MethodPost, &Config)
	resource.NewOperationCmd(rEphemeralKeysCmd.Cmd, "delete", "/v1/ephemeral_keys/{key}", http.MethodDelete, &Config)

	resource.NewOperationCmd(rEventsCmd.Cmd, "list", "/v1/events", http.MethodGet, &Config)
	resource.NewOperationCmd(rEventsCmd.Cmd, "retrieve", "/v1/events/{id}", http.MethodGet, &Config)

	resource.NewOperationCmd(rExchangeRatesCmd.Cmd, "list", "/v1/exchange_rates", http.MethodGet, &Config)
	resource.NewOperationCmd(rExchangeRatesCmd.Cmd, "retrieve", "/v1/exchange_rates/{currency}", http.MethodGet, &Config)

	resource.NewOperationCmd(rExternalAccountsCmd.Cmd, "create", "/v1/accounts/{account}/external_accounts", http.MethodPost, &Config)
	resource.NewOperationCmd(rExternalAccountsCmd.Cmd, "delete", "/v1/accounts/{account}/external_accounts/{id}", http.MethodDelete, &Config)
	resource.NewOperationCmd(rExternalAccountsCmd.Cmd, "list", "/v1/accounts/{account}/external_accounts", http.MethodGet, &Config)
	resource.NewOperationCmd(rExternalAccountsCmd.Cmd, "retrieve", "/v1/accounts/{account}/external_accounts/{id}", http.MethodGet, &Config)
	resource.NewOperationCmd(rExternalAccountsCmd.Cmd, "update", "/v1/accounts/{account}/external_accounts/{id}", http.MethodPost, &Config)

	resource.NewOperationCmd(rFeeRefundsCmd.Cmd, "create", "/v1/application_fees/{id}/refunds", http.MethodPost, &Config)
	resource.NewOperationCmd(rFeeRefundsCmd.Cmd, "list", "/v1/application_fees/{id}/refunds", http.MethodGet, &Config)
	resource.NewOperationCmd(rFeeRefundsCmd.Cmd, "retrieve", "/v1/application_fees/{fee}/refunds/{id}", http.MethodGet, &Config)
	resource.NewOperationCmd(rFeeRefundsCmd.Cmd, "update", "/v1/application_fees/{fee}/refunds/{id}", http.MethodPost, &Config)

	resource.NewOperationCmd(rFileLinksCmd.Cmd, "create", "/v1/file_links", http.MethodPost, &Config)
	resource.NewOperationCmd(rFileLinksCmd.Cmd, "list", "/v1/file_links", http.MethodGet, &Config)
	resource.NewOperationCmd(rFileLinksCmd.Cmd, "retrieve", "/v1/file_links/{link}", http.MethodGet, &Config)
	resource.NewOperationCmd(rFileLinksCmd.Cmd, "update", "/v1/file_links/{link}", http.MethodPost, &Config)

	resource.NewOperationCmd(rFilesCmd.Cmd, "create", "/v1/files", http.MethodPost, &Config)
	resource.NewOperationCmd(rFilesCmd.Cmd, "list", "/v1/files", http.MethodGet, &Config)
	resource.NewOperationCmd(rFilesCmd.Cmd, "retrieve", "/v1/files/{file}", http.MethodGet, &Config)

	resource.NewOperationCmd(rInvoiceitemsCmd.Cmd, "create", "/v1/invoiceitems", http.MethodPost, &Config)
	resource.NewOperationCmd(rInvoiceitemsCmd.Cmd, "delete", "/v1/invoiceitems/{invoiceitem}", http.MethodDelete, &Config)
	resource.NewOperationCmd(rInvoiceitemsCmd.Cmd, "list", "/v1/invoiceitems", http.MethodGet, &Config)
	resource.NewOperationCmd(rInvoiceitemsCmd.Cmd, "retrieve", "/v1/invoiceitems/{invoiceitem}", http.MethodGet, &Config)
	resource.NewOperationCmd(rInvoiceitemsCmd.Cmd, "update", "/v1/invoiceitems/{invoiceitem}", http.MethodPost, &Config)

	resource.NewOperationCmd(rInvoicesCmd.Cmd, "create", "/v1/invoices", http.MethodPost, &Config)
	resource.NewOperationCmd(rInvoicesCmd.Cmd, "delete", "/v1/invoices/{invoice}", http.MethodDelete, &Config)
	resource.NewOperationCmd(rInvoicesCmd.Cmd, "finalize_invoice", "/v1/invoices/{invoice}/finalize", http.MethodPost, &Config)
	resource.NewOperationCmd(rInvoicesCmd.Cmd, "list", "/v1/invoices", http.MethodGet, &Config)
	resource.NewOperationCmd(rInvoicesCmd.Cmd, "mark_uncollectible", "/v1/invoices/{invoice}/mark_uncollectible", http.MethodPost, &Config)
	resource.NewOperationCmd(rInvoicesCmd.Cmd, "pay", "/v1/invoices/{invoice}/pay", http.MethodPost, &Config)
	resource.NewOperationCmd(rInvoicesCmd.Cmd, "retrieve", "/v1/invoices/{invoice}", http.MethodGet, &Config)
	resource.NewOperationCmd(rInvoicesCmd.Cmd, "send_invoice", "/v1/invoices/{invoice}/send", http.MethodPost, &Config)
	resource.NewOperationCmd(rInvoicesCmd.Cmd, "upcoming", "/v1/invoices/upcoming", http.MethodGet, &Config)
	resource.NewOperationCmd(rInvoicesCmd.Cmd, "update", "/v1/invoices/{invoice}", http.MethodPost, &Config)
	resource.NewOperationCmd(rInvoicesCmd.Cmd, "void_invoice", "/v1/invoices/{invoice}/void", http.MethodPost, &Config)

	resource.NewOperationCmd(rIssuerFraudRecordsCmd.Cmd, "list", "/v1/issuer_fraud_records", http.MethodGet, &Config)
	resource.NewOperationCmd(rIssuerFraudRecordsCmd.Cmd, "retrieve", "/v1/issuer_fraud_records/{issuer_fraud_record}", http.MethodGet, &Config)

	resource.NewOperationCmd(rLineItemsCmd.Cmd, "list", "/v1/invoices/{invoice}/lines", http.MethodGet, &Config)

	resource.NewOperationCmd(rLoginLinksCmd.Cmd, "create", "/v1/accounts/{account}/login_links", http.MethodPost, &Config)

	resource.NewOperationCmd(rOrderReturnsCmd.Cmd, "list", "/v1/order_returns", http.MethodGet, &Config)
	resource.NewOperationCmd(rOrderReturnsCmd.Cmd, "retrieve", "/v1/order_returns/{id}", http.MethodGet, &Config)

	resource.NewOperationCmd(rOrdersCmd.Cmd, "create", "/v1/orders", http.MethodPost, &Config)
	resource.NewOperationCmd(rOrdersCmd.Cmd, "list", "/v1/orders", http.MethodGet, &Config)
	resource.NewOperationCmd(rOrdersCmd.Cmd, "pay", "/v1/orders/{id}/pay", http.MethodPost, &Config)
	resource.NewOperationCmd(rOrdersCmd.Cmd, "retrieve", "/v1/orders/{id}", http.MethodGet, &Config)
	resource.NewOperationCmd(rOrdersCmd.Cmd, "return_order", "/v1/orders/{id}/returns", http.MethodPost, &Config)
	resource.NewOperationCmd(rOrdersCmd.Cmd, "update", "/v1/orders/{id}", http.MethodPost, &Config)

	resource.NewOperationCmd(rPaymentIntentsCmd.Cmd, "cancel", "/v1/payment_intents/{intent}/cancel", http.MethodPost, &Config)
	resource.NewOperationCmd(rPaymentIntentsCmd.Cmd, "capture", "/v1/payment_intents/{intent}/capture", http.MethodPost, &Config)
	resource.NewOperationCmd(rPaymentIntentsCmd.Cmd, "confirm", "/v1/payment_intents/{intent}/confirm", http.MethodPost, &Config)
	resource.NewOperationCmd(rPaymentIntentsCmd.Cmd, "create", "/v1/payment_intents", http.MethodPost, &Config)
	resource.NewOperationCmd(rPaymentIntentsCmd.Cmd, "list", "/v1/payment_intents", http.MethodGet, &Config)
	resource.NewOperationCmd(rPaymentIntentsCmd.Cmd, "retrieve", "/v1/payment_intents/{intent}", http.MethodGet, &Config)
	resource.NewOperationCmd(rPaymentIntentsCmd.Cmd, "update", "/v1/payment_intents/{intent}", http.MethodPost, &Config)

	resource.NewOperationCmd(rPaymentMethodsCmd.Cmd, "attach", "/v1/payment_methods/{payment_method}/attach", http.MethodPost, &Config)
	resource.NewOperationCmd(rPaymentMethodsCmd.Cmd, "create", "/v1/payment_methods", http.MethodPost, &Config)
	resource.NewOperationCmd(rPaymentMethodsCmd.Cmd, "detach", "/v1/payment_methods/{payment_method}/detach", http.MethodPost, &Config)
	resource.NewOperationCmd(rPaymentMethodsCmd.Cmd, "list", "/v1/payment_methods", http.MethodGet, &Config)
	resource.NewOperationCmd(rPaymentMethodsCmd.Cmd, "retrieve", "/v1/payment_methods/{payment_method}", http.MethodGet, &Config)
	resource.NewOperationCmd(rPaymentMethodsCmd.Cmd, "update", "/v1/payment_methods/{payment_method}", http.MethodPost, &Config)

	resource.NewOperationCmd(rPaymentSourcesCmd.Cmd, "create", "/v1/customers/{customer}/sources", http.MethodPost, &Config)
	resource.NewOperationCmd(rPaymentSourcesCmd.Cmd, "list", "/v1/customers/{customer}/sources", http.MethodGet, &Config)
	resource.NewOperationCmd(rPaymentSourcesCmd.Cmd, "retrieve", "/v1/customers/{customer}/sources/{id}", http.MethodGet, &Config)

	resource.NewOperationCmd(rPayoutsCmd.Cmd, "cancel", "/v1/payouts/{payout}/cancel", http.MethodPost, &Config)
	resource.NewOperationCmd(rPayoutsCmd.Cmd, "create", "/v1/payouts", http.MethodPost, &Config)
	resource.NewOperationCmd(rPayoutsCmd.Cmd, "list", "/v1/payouts", http.MethodGet, &Config)
	resource.NewOperationCmd(rPayoutsCmd.Cmd, "retrieve", "/v1/payouts/{payout}", http.MethodGet, &Config)
	resource.NewOperationCmd(rPayoutsCmd.Cmd, "update", "/v1/payouts/{payout}", http.MethodPost, &Config)

	resource.NewOperationCmd(rPersonsCmd.Cmd, "create", "/v1/accounts/{account}/persons", http.MethodPost, &Config)
	resource.NewOperationCmd(rPersonsCmd.Cmd, "delete", "/v1/accounts/{account}/persons/{person}", http.MethodDelete, &Config)
	resource.NewOperationCmd(rPersonsCmd.Cmd, "list", "/v1/accounts/{account}/persons", http.MethodGet, &Config)
	resource.NewOperationCmd(rPersonsCmd.Cmd, "retrieve", "/v1/accounts/{account}/persons/{person}", http.MethodGet, &Config)
	resource.NewOperationCmd(rPersonsCmd.Cmd, "update", "/v1/accounts/{account}/persons/{person}", http.MethodPost, &Config)

	resource.NewOperationCmd(rPlansCmd.Cmd, "create", "/v1/plans", http.MethodPost, &Config)
	resource.NewOperationCmd(rPlansCmd.Cmd, "delete", "/v1/plans/{plan}", http.MethodDelete, &Config)
	resource.NewOperationCmd(rPlansCmd.Cmd, "list", "/v1/plans", http.MethodGet, &Config)
	resource.NewOperationCmd(rPlansCmd.Cmd, "retrieve", "/v1/plans/{plan}", http.MethodGet, &Config)
	resource.NewOperationCmd(rPlansCmd.Cmd, "update", "/v1/plans/{plan}", http.MethodPost, &Config)

	resource.NewOperationCmd(rProductsCmd.Cmd, "create", "/v1/products", http.MethodPost, &Config)
	resource.NewOperationCmd(rProductsCmd.Cmd, "delete", "/v1/products/{id}", http.MethodDelete, &Config)
	resource.NewOperationCmd(rProductsCmd.Cmd, "list", "/v1/products", http.MethodGet, &Config)
	resource.NewOperationCmd(rProductsCmd.Cmd, "retrieve", "/v1/products/{id}", http.MethodGet, &Config)
	resource.NewOperationCmd(rProductsCmd.Cmd, "update", "/v1/products/{id}", http.MethodPost, &Config)

	resource.NewOperationCmd(rRecipientsCmd.Cmd, "create", "/v1/recipients", http.MethodPost, &Config)
	resource.NewOperationCmd(rRecipientsCmd.Cmd, "delete", "/v1/recipients/{id}", http.MethodDelete, &Config)
	resource.NewOperationCmd(rRecipientsCmd.Cmd, "list", "/v1/recipients", http.MethodGet, &Config)
	resource.NewOperationCmd(rRecipientsCmd.Cmd, "retrieve", "/v1/recipients/{id}", http.MethodGet, &Config)
	resource.NewOperationCmd(rRecipientsCmd.Cmd, "update", "/v1/recipients/{id}", http.MethodPost, &Config)

	resource.NewOperationCmd(rRefundsCmd.Cmd, "create", "/v1/refunds", http.MethodPost, &Config)
	resource.NewOperationCmd(rRefundsCmd.Cmd, "list", "/v1/refunds", http.MethodGet, &Config)
	resource.NewOperationCmd(rRefundsCmd.Cmd, "retrieve", "/v1/refunds/{refund}", http.MethodGet, &Config)
	resource.NewOperationCmd(rRefundsCmd.Cmd, "update", "/v1/refunds/{refund}", http.MethodPost, &Config)

	resource.NewOperationCmd(rReviewsCmd.Cmd, "approve", "/v1/reviews/{review}/approve", http.MethodPost, &Config)
	resource.NewOperationCmd(rReviewsCmd.Cmd, "list", "/v1/reviews", http.MethodGet, &Config)
	resource.NewOperationCmd(rReviewsCmd.Cmd, "retrieve", "/v1/reviews/{review}", http.MethodGet, &Config)

	resource.NewOperationCmd(rScheduledQueryRunsCmd.Cmd, "list", "/v1/sigma/scheduled_query_runs", http.MethodGet, &Config)
	resource.NewOperationCmd(rScheduledQueryRunsCmd.Cmd, "retrieve", "/v1/sigma/scheduled_query_runs/{scheduled_query_run}", http.MethodGet, &Config)

	resource.NewOperationCmd(rSetupIntentsCmd.Cmd, "cancel", "/v1/setup_intents/{intent}/cancel", http.MethodPost, &Config)
	resource.NewOperationCmd(rSetupIntentsCmd.Cmd, "confirm", "/v1/setup_intents/{intent}/confirm", http.MethodPost, &Config)
	resource.NewOperationCmd(rSetupIntentsCmd.Cmd, "create", "/v1/setup_intents", http.MethodPost, &Config)
	resource.NewOperationCmd(rSetupIntentsCmd.Cmd, "list", "/v1/setup_intents", http.MethodGet, &Config)
	resource.NewOperationCmd(rSetupIntentsCmd.Cmd, "retrieve", "/v1/setup_intents/{intent}", http.MethodGet, &Config)
	resource.NewOperationCmd(rSetupIntentsCmd.Cmd, "update", "/v1/setup_intents/{intent}", http.MethodPost, &Config)

	resource.NewOperationCmd(rSkusCmd.Cmd, "create", "/v1/skus", http.MethodPost, &Config)
	resource.NewOperationCmd(rSkusCmd.Cmd, "delete", "/v1/skus/{id}", http.MethodDelete, &Config)
	resource.NewOperationCmd(rSkusCmd.Cmd, "list", "/v1/skus", http.MethodGet, &Config)
	resource.NewOperationCmd(rSkusCmd.Cmd, "retrieve", "/v1/skus/{id}", http.MethodGet, &Config)
	resource.NewOperationCmd(rSkusCmd.Cmd, "update", "/v1/skus/{id}", http.MethodPost, &Config)

	resource.NewOperationCmd(rSourcesCmd.Cmd, "create", "/v1/sources", http.MethodPost, &Config)
	resource.NewOperationCmd(rSourcesCmd.Cmd, "detach", "/v1/customers/{customer}/sources/{id}", http.MethodDelete, &Config)
	resource.NewOperationCmd(rSourcesCmd.Cmd, "retrieve", "/v1/sources/{source}", http.MethodGet, &Config)
	resource.NewOperationCmd(rSourcesCmd.Cmd, "source_transactions", "/v1/sources/{source}/source_transactions", http.MethodGet, &Config)
	resource.NewOperationCmd(rSourcesCmd.Cmd, "update", "/v1/sources/{source}", http.MethodPost, &Config)
	resource.NewOperationCmd(rSourcesCmd.Cmd, "verify", "/v1/sources/{source}/verify", http.MethodPost, &Config)

	resource.NewOperationCmd(rSubscriptionItemsCmd.Cmd, "create", "/v1/subscription_items", http.MethodPost, &Config)
	resource.NewOperationCmd(rSubscriptionItemsCmd.Cmd, "delete", "/v1/subscription_items/{item}", http.MethodDelete, &Config)
	resource.NewOperationCmd(rSubscriptionItemsCmd.Cmd, "list", "/v1/subscription_items", http.MethodGet, &Config)
	resource.NewOperationCmd(rSubscriptionItemsCmd.Cmd, "retrieve", "/v1/subscription_items/{item}", http.MethodGet, &Config)
	resource.NewOperationCmd(rSubscriptionItemsCmd.Cmd, "update", "/v1/subscription_items/{item}", http.MethodPost, &Config)
	resource.NewOperationCmd(rSubscriptionItemsCmd.Cmd, "usage_record_summaries", "/v1/subscription_items/{subscription_item}/usage_record_summaries", http.MethodGet, &Config)

	resource.NewOperationCmd(rSubscriptionSchedulesCmd.Cmd, "cancel", "/v1/subscription_schedules/{schedule}/cancel", http.MethodPost, &Config)
	resource.NewOperationCmd(rSubscriptionSchedulesCmd.Cmd, "create", "/v1/subscription_schedules", http.MethodPost, &Config)
	resource.NewOperationCmd(rSubscriptionSchedulesCmd.Cmd, "list", "/v1/subscription_schedules", http.MethodGet, &Config)
	resource.NewOperationCmd(rSubscriptionSchedulesCmd.Cmd, "release", "/v1/subscription_schedules/{schedule}/release", http.MethodPost, &Config)
	resource.NewOperationCmd(rSubscriptionSchedulesCmd.Cmd, "retrieve", "/v1/subscription_schedules/{schedule}", http.MethodGet, &Config)
	resource.NewOperationCmd(rSubscriptionSchedulesCmd.Cmd, "update", "/v1/subscription_schedules/{schedule}", http.MethodPost, &Config)

	resource.NewOperationCmd(rSubscriptionsCmd.Cmd, "cancel", "/v1/subscriptions/{subscription_exposed_id}", http.MethodDelete, &Config)
	resource.NewOperationCmd(rSubscriptionsCmd.Cmd, "create", "/v1/subscriptions", http.MethodPost, &Config)
	resource.NewOperationCmd(rSubscriptionsCmd.Cmd, "delete_discount", "/v1/subscriptions/{subscription_exposed_id}/discount", http.MethodDelete, &Config)
	resource.NewOperationCmd(rSubscriptionsCmd.Cmd, "list", "/v1/subscriptions", http.MethodGet, &Config)
	resource.NewOperationCmd(rSubscriptionsCmd.Cmd, "retrieve", "/v1/subscriptions/{subscription_exposed_id}", http.MethodGet, &Config)
	resource.NewOperationCmd(rSubscriptionsCmd.Cmd, "update", "/v1/subscriptions/{subscription_exposed_id}", http.MethodPost, &Config)

	resource.NewOperationCmd(rTaxIdsCmd.Cmd, "create", "/v1/customers/{customer}/tax_ids", http.MethodPost, &Config)
	resource.NewOperationCmd(rTaxIdsCmd.Cmd, "delete", "/v1/customers/{customer}/tax_ids/{id}", http.MethodDelete, &Config)
	resource.NewOperationCmd(rTaxIdsCmd.Cmd, "list", "/v1/customers/{customer}/tax_ids", http.MethodGet, &Config)
	resource.NewOperationCmd(rTaxIdsCmd.Cmd, "retrieve", "/v1/customers/{customer}/tax_ids/{id}", http.MethodGet, &Config)

	resource.NewOperationCmd(rTaxRatesCmd.Cmd, "create", "/v1/tax_rates", http.MethodPost, &Config)
	resource.NewOperationCmd(rTaxRatesCmd.Cmd, "list", "/v1/tax_rates", http.MethodGet, &Config)
	resource.NewOperationCmd(rTaxRatesCmd.Cmd, "retrieve", "/v1/tax_rates/{tax_rate}", http.MethodGet, &Config)
	resource.NewOperationCmd(rTaxRatesCmd.Cmd, "update", "/v1/tax_rates/{tax_rate}", http.MethodPost, &Config)

	resource.NewOperationCmd(rTokensCmd.Cmd, "create", "/v1/tokens", http.MethodPost, &Config)
	resource.NewOperationCmd(rTokensCmd.Cmd, "retrieve", "/v1/tokens/{token}", http.MethodGet, &Config)

	resource.NewOperationCmd(rTopupsCmd.Cmd, "cancel", "/v1/topups/{topup}/cancel", http.MethodPost, &Config)
	resource.NewOperationCmd(rTopupsCmd.Cmd, "create", "/v1/topups", http.MethodPost, &Config)
	resource.NewOperationCmd(rTopupsCmd.Cmd, "list", "/v1/topups", http.MethodGet, &Config)
	resource.NewOperationCmd(rTopupsCmd.Cmd, "retrieve", "/v1/topups/{topup}", http.MethodGet, &Config)
	resource.NewOperationCmd(rTopupsCmd.Cmd, "update", "/v1/topups/{topup}", http.MethodPost, &Config)

	resource.NewOperationCmd(rTransferReversalsCmd.Cmd, "create", "/v1/transfers/{id}/reversals", http.MethodPost, &Config)
	resource.NewOperationCmd(rTransferReversalsCmd.Cmd, "list", "/v1/transfers/{id}/reversals", http.MethodGet, &Config)
	resource.NewOperationCmd(rTransferReversalsCmd.Cmd, "retrieve", "/v1/transfers/{transfer}/reversals/{id}", http.MethodGet, &Config)
	resource.NewOperationCmd(rTransferReversalsCmd.Cmd, "update", "/v1/transfers/{transfer}/reversals/{id}", http.MethodPost, &Config)

	resource.NewOperationCmd(rTransfersCmd.Cmd, "create", "/v1/transfers", http.MethodPost, &Config)
	resource.NewOperationCmd(rTransfersCmd.Cmd, "list", "/v1/transfers", http.MethodGet, &Config)
	resource.NewOperationCmd(rTransfersCmd.Cmd, "retrieve", "/v1/transfers/{transfer}", http.MethodGet, &Config)
	resource.NewOperationCmd(rTransfersCmd.Cmd, "update", "/v1/transfers/{transfer}", http.MethodPost, &Config)

	resource.NewOperationCmd(rUsageRecordsCmd.Cmd, "create", "/v1/subscription_items/{subscription_item}/usage_records", http.MethodPost, &Config)

	resource.NewOperationCmd(rWebhookEndpointsCmd.Cmd, "create", "/v1/webhook_endpoints", http.MethodPost, &Config)
	resource.NewOperationCmd(rWebhookEndpointsCmd.Cmd, "delete", "/v1/webhook_endpoints/{webhook_endpoint}", http.MethodDelete, &Config)
	resource.NewOperationCmd(rWebhookEndpointsCmd.Cmd, "list", "/v1/webhook_endpoints", http.MethodGet, &Config)
	resource.NewOperationCmd(rWebhookEndpointsCmd.Cmd, "retrieve", "/v1/webhook_endpoints/{webhook_endpoint}", http.MethodGet, &Config)
	resource.NewOperationCmd(rWebhookEndpointsCmd.Cmd, "update", "/v1/webhook_endpoints/{webhook_endpoint}", http.MethodPost, &Config)

	resource.NewOperationCmd(rCheckoutSessionsCmd.Cmd, "create", "/v1/checkout/sessions", http.MethodPost, &Config)
	resource.NewOperationCmd(rCheckoutSessionsCmd.Cmd, "retrieve", "/v1/checkout/sessions/{session}", http.MethodGet, &Config)

	resource.NewOperationCmd(rIssuingAuthorizationsCmd.Cmd, "approve", "/v1/issuing/authorizations/{authorization}/approve", http.MethodPost, &Config)
	resource.NewOperationCmd(rIssuingAuthorizationsCmd.Cmd, "decline", "/v1/issuing/authorizations/{authorization}/decline", http.MethodPost, &Config)
	resource.NewOperationCmd(rIssuingAuthorizationsCmd.Cmd, "list", "/v1/issuing/authorizations", http.MethodGet, &Config)
	resource.NewOperationCmd(rIssuingAuthorizationsCmd.Cmd, "retrieve", "/v1/issuing/authorizations/{authorization}", http.MethodGet, &Config)
	resource.NewOperationCmd(rIssuingAuthorizationsCmd.Cmd, "update", "/v1/issuing/authorizations/{authorization}", http.MethodPost, &Config)

	resource.NewOperationCmd(rIssuingCardholdersCmd.Cmd, "create", "/v1/issuing/cardholders", http.MethodPost, &Config)
	resource.NewOperationCmd(rIssuingCardholdersCmd.Cmd, "list", "/v1/issuing/cardholders", http.MethodGet, &Config)
	resource.NewOperationCmd(rIssuingCardholdersCmd.Cmd, "retrieve", "/v1/issuing/cardholders/{cardholder}", http.MethodGet, &Config)
	resource.NewOperationCmd(rIssuingCardholdersCmd.Cmd, "update", "/v1/issuing/cardholders/{cardholder}", http.MethodPost, &Config)

	resource.NewOperationCmd(rIssuingCardsCmd.Cmd, "create", "/v1/issuing/cards", http.MethodPost, &Config)
	resource.NewOperationCmd(rIssuingCardsCmd.Cmd, "details", "/v1/issuing/cards/{card}/details", http.MethodGet, &Config)
	resource.NewOperationCmd(rIssuingCardsCmd.Cmd, "list", "/v1/issuing/cards", http.MethodGet, &Config)
	resource.NewOperationCmd(rIssuingCardsCmd.Cmd, "retrieve", "/v1/issuing/cards/{card}", http.MethodGet, &Config)
	resource.NewOperationCmd(rIssuingCardsCmd.Cmd, "update", "/v1/issuing/cards/{card}", http.MethodPost, &Config)

	resource.NewOperationCmd(rIssuingDisputesCmd.Cmd, "create", "/v1/issuing/disputes", http.MethodPost, &Config)
	resource.NewOperationCmd(rIssuingDisputesCmd.Cmd, "list", "/v1/issuing/disputes", http.MethodGet, &Config)
	resource.NewOperationCmd(rIssuingDisputesCmd.Cmd, "retrieve", "/v1/issuing/disputes/{dispute}", http.MethodGet, &Config)
	resource.NewOperationCmd(rIssuingDisputesCmd.Cmd, "update", "/v1/issuing/disputes/{dispute}", http.MethodPost, &Config)

	resource.NewOperationCmd(rIssuingTransactionsCmd.Cmd, "list", "/v1/issuing/transactions", http.MethodGet, &Config)
	resource.NewOperationCmd(rIssuingTransactionsCmd.Cmd, "retrieve", "/v1/issuing/transactions/{transaction}", http.MethodGet, &Config)
	resource.NewOperationCmd(rIssuingTransactionsCmd.Cmd, "update", "/v1/issuing/transactions/{transaction}", http.MethodPost, &Config)

	resource.NewOperationCmd(rRadarEarlyFraudWarningsCmd.Cmd, "list", "/v1/radar/early_fraud_warnings", http.MethodGet, &Config)
	resource.NewOperationCmd(rRadarEarlyFraudWarningsCmd.Cmd, "retrieve", "/v1/radar/early_fraud_warnings/{early_fraud_warning}", http.MethodGet, &Config)

	resource.NewOperationCmd(rRadarValueListItemsCmd.Cmd, "create", "/v1/radar/value_list_items", http.MethodPost, &Config)
	resource.NewOperationCmd(rRadarValueListItemsCmd.Cmd, "delete", "/v1/radar/value_list_items/{item}", http.MethodDelete, &Config)
	resource.NewOperationCmd(rRadarValueListItemsCmd.Cmd, "list", "/v1/radar/value_list_items", http.MethodGet, &Config)
	resource.NewOperationCmd(rRadarValueListItemsCmd.Cmd, "retrieve", "/v1/radar/value_list_items/{item}", http.MethodGet, &Config)

	resource.NewOperationCmd(rRadarValueListsCmd.Cmd, "create", "/v1/radar/value_lists", http.MethodPost, &Config)
	resource.NewOperationCmd(rRadarValueListsCmd.Cmd, "delete", "/v1/radar/value_lists/{value_list}", http.MethodDelete, &Config)
	resource.NewOperationCmd(rRadarValueListsCmd.Cmd, "list", "/v1/radar/value_lists", http.MethodGet, &Config)
	resource.NewOperationCmd(rRadarValueListsCmd.Cmd, "retrieve", "/v1/radar/value_lists/{value_list}", http.MethodGet, &Config)
	resource.NewOperationCmd(rRadarValueListsCmd.Cmd, "update", "/v1/radar/value_lists/{value_list}", http.MethodPost, &Config)

	resource.NewOperationCmd(rReportingReportRunsCmd.Cmd, "create", "/v1/reporting/report_runs", http.MethodPost, &Config)
	resource.NewOperationCmd(rReportingReportRunsCmd.Cmd, "list", "/v1/reporting/report_runs", http.MethodGet, &Config)
	resource.NewOperationCmd(rReportingReportRunsCmd.Cmd, "retrieve", "/v1/reporting/report_runs/{report_run}", http.MethodGet, &Config)

	resource.NewOperationCmd(rReportingReportTypesCmd.Cmd, "list", "/v1/reporting/report_types", http.MethodGet, &Config)
	resource.NewOperationCmd(rReportingReportTypesCmd.Cmd, "retrieve", "/v1/reporting/report_types/{report_type}", http.MethodGet, &Config)

	resource.NewOperationCmd(rTerminalConnectionTokensCmd.Cmd, "create", "/v1/terminal/connection_tokens", http.MethodPost, &Config)

	resource.NewOperationCmd(rTerminalLocationsCmd.Cmd, "create", "/v1/terminal/locations", http.MethodPost, &Config)
	resource.NewOperationCmd(rTerminalLocationsCmd.Cmd, "delete", "/v1/terminal/locations/{location}", http.MethodDelete, &Config)
	resource.NewOperationCmd(rTerminalLocationsCmd.Cmd, "list", "/v1/terminal/locations", http.MethodGet, &Config)
	resource.NewOperationCmd(rTerminalLocationsCmd.Cmd, "retrieve", "/v1/terminal/locations/{location}", http.MethodGet, &Config)
	resource.NewOperationCmd(rTerminalLocationsCmd.Cmd, "update", "/v1/terminal/locations/{location}", http.MethodPost, &Config)

	resource.NewOperationCmd(rTerminalReadersCmd.Cmd, "create", "/v1/terminal/readers", http.MethodPost, &Config)
	resource.NewOperationCmd(rTerminalReadersCmd.Cmd, "delete", "/v1/terminal/readers/{reader}", http.MethodDelete, &Config)
	resource.NewOperationCmd(rTerminalReadersCmd.Cmd, "list", "/v1/terminal/readers", http.MethodGet, &Config)
	resource.NewOperationCmd(rTerminalReadersCmd.Cmd, "retrieve", "/v1/terminal/readers/{reader}", http.MethodGet, &Config)
	resource.NewOperationCmd(rTerminalReadersCmd.Cmd, "update", "/v1/terminal/readers/{reader}", http.MethodPost, &Config)
}
