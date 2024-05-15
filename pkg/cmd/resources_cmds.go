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
	nsAppsCmd := resource.NewNamespaceCmd(rootCmd, "apps")
	nsBillingCmd := resource.NewNamespaceCmd(rootCmd, "billing")
	nsBillingPortalCmd := resource.NewNamespaceCmd(rootCmd, "billing_portal")
	nsCheckoutCmd := resource.NewNamespaceCmd(rootCmd, "checkout")
	nsClimateCmd := resource.NewNamespaceCmd(rootCmd, "climate")
	nsEntitlementsCmd := resource.NewNamespaceCmd(rootCmd, "entitlements")
	nsFinancialConnectionsCmd := resource.NewNamespaceCmd(rootCmd, "financial_connections")
	nsForwardingCmd := resource.NewNamespaceCmd(rootCmd, "forwarding")
	nsIdentityCmd := resource.NewNamespaceCmd(rootCmd, "identity")
	nsIssuingCmd := resource.NewNamespaceCmd(rootCmd, "issuing")
	nsRadarCmd := resource.NewNamespaceCmd(rootCmd, "radar")
	nsReportingCmd := resource.NewNamespaceCmd(rootCmd, "reporting")
	nsTaxCmd := resource.NewNamespaceCmd(rootCmd, "tax")
	nsTerminalCmd := resource.NewNamespaceCmd(rootCmd, "terminal")
	nsTestHelpersCmd := resource.NewNamespaceCmd(rootCmd, "test_helpers")
	nsTreasuryCmd := resource.NewNamespaceCmd(rootCmd, "treasury")

	// Resource commands
	rAccountLinksCmd := resource.NewResourceCmd(rootCmd, "account_links")
	rAccountSessionsCmd := resource.NewResourceCmd(rootCmd, "account_sessions")
	rAccountsCmd := resource.NewResourceCmd(rootCmd, "accounts")
	rApplePayDomainsCmd := resource.NewResourceCmd(rootCmd, "apple_pay_domains")
	rApplicationFeesCmd := resource.NewResourceCmd(rootCmd, "application_fees")
	rBalanceCmd := resource.NewResourceCmd(rootCmd, "balance")
	rBalanceTransactionsCmd := resource.NewResourceCmd(rootCmd, "balance_transactions")
	rBankAccountsCmd := resource.NewResourceCmd(rootCmd, "bank_accounts")
	rCapabilitiesCmd := resource.NewResourceCmd(rootCmd, "capabilities")
	rCardsCmd := resource.NewResourceCmd(rootCmd, "cards")
	rCashBalancesCmd := resource.NewResourceCmd(rootCmd, "cash_balances")
	rChargesCmd := resource.NewResourceCmd(rootCmd, "charges")
	rConfirmationTokensCmd := resource.NewResourceCmd(rootCmd, "confirmation_tokens")
	rConfirmationTokensTestHelpersCmd := resource.NewResourceCmd(rConfirmationTokensCmd.Cmd, "test_helpers")
	rCountrySpecsCmd := resource.NewResourceCmd(rootCmd, "country_specs")
	rCouponsCmd := resource.NewResourceCmd(rootCmd, "coupons")
	rCreditNoteLineItemsCmd := resource.NewResourceCmd(rootCmd, "credit_note_line_items")
	rCreditNotesCmd := resource.NewResourceCmd(rootCmd, "credit_notes")
	rCustomerBalanceTransactionsCmd := resource.NewResourceCmd(rootCmd, "customer_balance_transactions")
	rCustomerCashBalanceTransactionsCmd := resource.NewResourceCmd(rootCmd, "customer_cash_balance_transactions")
	rCustomerSessionsCmd := resource.NewResourceCmd(rootCmd, "customer_sessions")
	rCustomersCmd := resource.NewResourceCmd(rootCmd, "customers")
	rCustomersTestHelpersCmd := resource.NewResourceCmd(rCustomersCmd.Cmd, "test_helpers")
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
	rLineItemsCmd := resource.NewResourceCmd(rootCmd, "line_items")
	rLoginLinksCmd := resource.NewResourceCmd(rootCmd, "login_links")
	rMandatesCmd := resource.NewResourceCmd(rootCmd, "mandates")
	rPaymentIntentsCmd := resource.NewResourceCmd(rootCmd, "payment_intents")
	rPaymentLinksCmd := resource.NewResourceCmd(rootCmd, "payment_links")
	rPaymentMethodConfigurationsCmd := resource.NewResourceCmd(rootCmd, "payment_method_configurations")
	rPaymentMethodDomainsCmd := resource.NewResourceCmd(rootCmd, "payment_method_domains")
	rPaymentMethodsCmd := resource.NewResourceCmd(rootCmd, "payment_methods")
	rPaymentSourcesCmd := resource.NewResourceCmd(rootCmd, "payment_sources")
	rPayoutsCmd := resource.NewResourceCmd(rootCmd, "payouts")
	rPersonsCmd := resource.NewResourceCmd(rootCmd, "persons")
	rPlansCmd := resource.NewResourceCmd(rootCmd, "plans")
	rPricesCmd := resource.NewResourceCmd(rootCmd, "prices")
	rProductFeaturesCmd := resource.NewResourceCmd(rootCmd, "product_features")
	rProductsCmd := resource.NewResourceCmd(rootCmd, "products")
	rPromotionCodesCmd := resource.NewResourceCmd(rootCmd, "promotion_codes")
	rQuotesCmd := resource.NewResourceCmd(rootCmd, "quotes")
	rRefundsCmd := resource.NewResourceCmd(rootCmd, "refunds")
	rRefundsTestHelpersCmd := resource.NewResourceCmd(rRefundsCmd.Cmd, "test_helpers")
	rReviewsCmd := resource.NewResourceCmd(rootCmd, "reviews")
	rScheduledQueryRunsCmd := resource.NewResourceCmd(rootCmd, "scheduled_query_runs")
	rSetupAttemptsCmd := resource.NewResourceCmd(rootCmd, "setup_attempts")
	rSetupIntentsCmd := resource.NewResourceCmd(rootCmd, "setup_intents")
	rShippingRatesCmd := resource.NewResourceCmd(rootCmd, "shipping_rates")
	rSourcesCmd := resource.NewResourceCmd(rootCmd, "sources")
	rSubscriptionItemsCmd := resource.NewResourceCmd(rootCmd, "subscription_items")
	rSubscriptionSchedulesCmd := resource.NewResourceCmd(rootCmd, "subscription_schedules")
	rSubscriptionsCmd := resource.NewResourceCmd(rootCmd, "subscriptions")
	rTaxCodesCmd := resource.NewResourceCmd(rootCmd, "tax_codes")
	rTaxIdsCmd := resource.NewResourceCmd(rootCmd, "tax_ids")
	rTaxRatesCmd := resource.NewResourceCmd(rootCmd, "tax_rates")
	rTokensCmd := resource.NewResourceCmd(rootCmd, "tokens")
	rTopupsCmd := resource.NewResourceCmd(rootCmd, "topups")
	rTransferReversalsCmd := resource.NewResourceCmd(rootCmd, "transfer_reversals")
	rTransfersCmd := resource.NewResourceCmd(rootCmd, "transfers")
	rUsageRecordSummariesCmd := resource.NewResourceCmd(rootCmd, "usage_record_summaries")
	rUsageRecordsCmd := resource.NewResourceCmd(rootCmd, "usage_records")
	rWebhookEndpointsCmd := resource.NewResourceCmd(rootCmd, "webhook_endpoints")
	rAppsSecretsCmd := resource.NewResourceCmd(nsAppsCmd.Cmd, "secrets")
	rBillingMeterEventAdjustmentsCmd := resource.NewResourceCmd(nsBillingCmd.Cmd, "meter_event_adjustments")
	rBillingMeterEventSummarysCmd := resource.NewResourceCmd(nsBillingCmd.Cmd, "meter_event_summarys")
	rBillingMeterEventsCmd := resource.NewResourceCmd(nsBillingCmd.Cmd, "meter_events")
	rBillingMetersCmd := resource.NewResourceCmd(nsBillingCmd.Cmd, "meters")
	rBillingPortalConfigurationsCmd := resource.NewResourceCmd(nsBillingPortalCmd.Cmd, "configurations")
	rBillingPortalSessionsCmd := resource.NewResourceCmd(nsBillingPortalCmd.Cmd, "sessions")
	rCheckoutSessionsCmd := resource.NewResourceCmd(nsCheckoutCmd.Cmd, "sessions")
	rClimateOrdersCmd := resource.NewResourceCmd(nsClimateCmd.Cmd, "orders")
	rClimateProductsCmd := resource.NewResourceCmd(nsClimateCmd.Cmd, "products")
	rClimateSuppliersCmd := resource.NewResourceCmd(nsClimateCmd.Cmd, "suppliers")
	rEntitlementsActiveEntitlementsCmd := resource.NewResourceCmd(nsEntitlementsCmd.Cmd, "active_entitlements")
	rEntitlementsFeaturesCmd := resource.NewResourceCmd(nsEntitlementsCmd.Cmd, "features")
	rFinancialConnectionsAccountsCmd := resource.NewResourceCmd(nsFinancialConnectionsCmd.Cmd, "accounts")
	rFinancialConnectionsSessionsCmd := resource.NewResourceCmd(nsFinancialConnectionsCmd.Cmd, "sessions")
	rFinancialConnectionsTransactionsCmd := resource.NewResourceCmd(nsFinancialConnectionsCmd.Cmd, "transactions")
	rForwardingRequestsCmd := resource.NewResourceCmd(nsForwardingCmd.Cmd, "requests")
	rIdentityVerificationReportsCmd := resource.NewResourceCmd(nsIdentityCmd.Cmd, "verification_reports")
	rIdentityVerificationSessionsCmd := resource.NewResourceCmd(nsIdentityCmd.Cmd, "verification_sessions")
	rIssuingAuthorizationsCmd := resource.NewResourceCmd(nsIssuingCmd.Cmd, "authorizations")
	rIssuingAuthorizationsTestHelpersCmd := resource.NewResourceCmd(rIssuingAuthorizationsCmd.Cmd, "test_helpers")
	rIssuingCardholdersCmd := resource.NewResourceCmd(nsIssuingCmd.Cmd, "cardholders")
	rIssuingCardsCmd := resource.NewResourceCmd(nsIssuingCmd.Cmd, "cards")
	rIssuingCardsTestHelpersCmd := resource.NewResourceCmd(rIssuingCardsCmd.Cmd, "test_helpers")
	rIssuingDisputesCmd := resource.NewResourceCmd(nsIssuingCmd.Cmd, "disputes")
	rIssuingPersonalizationDesignsCmd := resource.NewResourceCmd(nsIssuingCmd.Cmd, "personalization_designs")
	rIssuingPersonalizationDesignsTestHelpersCmd := resource.NewResourceCmd(rIssuingPersonalizationDesignsCmd.Cmd, "test_helpers")
	rIssuingPhysicalBundlesCmd := resource.NewResourceCmd(nsIssuingCmd.Cmd, "physical_bundles")
	rIssuingTokensCmd := resource.NewResourceCmd(nsIssuingCmd.Cmd, "tokens")
	rIssuingTransactionsCmd := resource.NewResourceCmd(nsIssuingCmd.Cmd, "transactions")
	rIssuingTransactionsTestHelpersCmd := resource.NewResourceCmd(rIssuingTransactionsCmd.Cmd, "test_helpers")
	rRadarEarlyFraudWarningsCmd := resource.NewResourceCmd(nsRadarCmd.Cmd, "early_fraud_warnings")
	rRadarValueListItemsCmd := resource.NewResourceCmd(nsRadarCmd.Cmd, "value_list_items")
	rRadarValueListsCmd := resource.NewResourceCmd(nsRadarCmd.Cmd, "value_lists")
	rReportingReportRunsCmd := resource.NewResourceCmd(nsReportingCmd.Cmd, "report_runs")
	rReportingReportTypesCmd := resource.NewResourceCmd(nsReportingCmd.Cmd, "report_types")
	rTaxCalculationsCmd := resource.NewResourceCmd(nsTaxCmd.Cmd, "calculations")
	rTaxRegistrationsCmd := resource.NewResourceCmd(nsTaxCmd.Cmd, "registrations")
	rTaxSettingsCmd := resource.NewResourceCmd(nsTaxCmd.Cmd, "settings")
	rTaxTransactionsCmd := resource.NewResourceCmd(nsTaxCmd.Cmd, "transactions")
	rTerminalConfigurationsCmd := resource.NewResourceCmd(nsTerminalCmd.Cmd, "configurations")
	rTerminalConnectionTokensCmd := resource.NewResourceCmd(nsTerminalCmd.Cmd, "connection_tokens")
	rTerminalLocationsCmd := resource.NewResourceCmd(nsTerminalCmd.Cmd, "locations")
	rTerminalReadersCmd := resource.NewResourceCmd(nsTerminalCmd.Cmd, "readers")
	rTerminalReadersTestHelpersCmd := resource.NewResourceCmd(rTerminalReadersCmd.Cmd, "test_helpers")
	rTestHelpersConfirmationTokensCmd := resource.NewResourceCmd(nsTestHelpersCmd.Cmd, "confirmation_tokens")
	rTestHelpersCustomersCmd := resource.NewResourceCmd(nsTestHelpersCmd.Cmd, "customers")
	rTestHelpersIssuingCmd := resource.NewResourceCmd(nsTestHelpersCmd.Cmd, "issuing")
	rTestHelpersIssuingAuthorizationsCmd := resource.NewResourceCmd(rTestHelpersIssuingCmd.Cmd, "authorizations")
	rTestHelpersIssuingCardsCmd := resource.NewResourceCmd(rTestHelpersIssuingCmd.Cmd, "cards")
	rTestHelpersIssuingPersonalizationDesignsCmd := resource.NewResourceCmd(rTestHelpersIssuingCmd.Cmd, "personalization_designs")
	rTestHelpersIssuingTransactionsCmd := resource.NewResourceCmd(rTestHelpersIssuingCmd.Cmd, "transactions")
	rTestHelpersRefundsCmd := resource.NewResourceCmd(nsTestHelpersCmd.Cmd, "refunds")
	rTestHelpersTerminalCmd := resource.NewResourceCmd(nsTestHelpersCmd.Cmd, "terminal")
	rTestHelpersTerminalReadersCmd := resource.NewResourceCmd(rTestHelpersTerminalCmd.Cmd, "readers")
	rTestHelpersTestClocksCmd := resource.NewResourceCmd(nsTestHelpersCmd.Cmd, "test_clocks")
	rTestHelpersTreasuryCmd := resource.NewResourceCmd(nsTestHelpersCmd.Cmd, "treasury")
	rTestHelpersTreasuryInboundTransfersCmd := resource.NewResourceCmd(rTestHelpersTreasuryCmd.Cmd, "inbound_transfers")
	rTestHelpersTreasuryOutboundPaymentsCmd := resource.NewResourceCmd(rTestHelpersTreasuryCmd.Cmd, "outbound_payments")
	rTestHelpersTreasuryOutboundTransfersCmd := resource.NewResourceCmd(rTestHelpersTreasuryCmd.Cmd, "outbound_transfers")
	rTestHelpersTreasuryReceivedCreditsCmd := resource.NewResourceCmd(rTestHelpersTreasuryCmd.Cmd, "received_credits")
	rTestHelpersTreasuryReceivedDebitsCmd := resource.NewResourceCmd(rTestHelpersTreasuryCmd.Cmd, "received_debits")
	rTreasuryCreditReversalsCmd := resource.NewResourceCmd(nsTreasuryCmd.Cmd, "credit_reversals")
	rTreasuryDebitReversalsCmd := resource.NewResourceCmd(nsTreasuryCmd.Cmd, "debit_reversals")
	rTreasuryFinancialAccountsCmd := resource.NewResourceCmd(nsTreasuryCmd.Cmd, "financial_accounts")
	rTreasuryInboundTransfersCmd := resource.NewResourceCmd(nsTreasuryCmd.Cmd, "inbound_transfers")
	rTreasuryInboundTransfersTestHelpersCmd := resource.NewResourceCmd(rTreasuryInboundTransfersCmd.Cmd, "test_helpers")
	rTreasuryOutboundPaymentsCmd := resource.NewResourceCmd(nsTreasuryCmd.Cmd, "outbound_payments")
	rTreasuryOutboundPaymentsTestHelpersCmd := resource.NewResourceCmd(rTreasuryOutboundPaymentsCmd.Cmd, "test_helpers")
	rTreasuryOutboundTransfersCmd := resource.NewResourceCmd(nsTreasuryCmd.Cmd, "outbound_transfers")
	rTreasuryOutboundTransfersTestHelpersCmd := resource.NewResourceCmd(rTreasuryOutboundTransfersCmd.Cmd, "test_helpers")
	rTreasuryReceivedCreditsCmd := resource.NewResourceCmd(nsTreasuryCmd.Cmd, "received_credits")
	rTreasuryReceivedCreditsTestHelpersCmd := resource.NewResourceCmd(rTreasuryReceivedCreditsCmd.Cmd, "test_helpers")
	rTreasuryReceivedDebitsCmd := resource.NewResourceCmd(nsTreasuryCmd.Cmd, "received_debits")
	rTreasuryReceivedDebitsTestHelpersCmd := resource.NewResourceCmd(rTreasuryReceivedDebitsCmd.Cmd, "test_helpers")
	rTreasuryTransactionEntrysCmd := resource.NewResourceCmd(nsTreasuryCmd.Cmd, "transaction_entrys")
	rTreasuryTransactionsCmd := resource.NewResourceCmd(nsTreasuryCmd.Cmd, "transactions")

	// Operation commands
	resource.NewOperationCmd(rAccountLinksCmd.Cmd, "create", "/v1/account_links", http.MethodPost, map[string]string{
		"account":                                "string",
		"collect":                                "string",
		"collection_options.fields":              "string",
		"collection_options.future_requirements": "string",
		"refresh_url":                            "string",
		"return_url":                             "string",
		"type":                                   "string",
	}, &Config)
	resource.NewOperationCmd(rAccountSessionsCmd.Cmd, "create", "/v1/account_sessions", http.MethodPost, map[string]string{
		"account":                               "string",
		"components.account_management.enabled": "boolean",
		"components.account_management.features.external_account_collection":             "boolean",
		"components.account_onboarding.enabled":                                          "boolean",
		"components.account_onboarding.features.external_account_collection":             "boolean",
		"components.balances.enabled":                                                    "boolean",
		"components.balances.features.edit_payout_schedule":                              "boolean",
		"components.balances.features.instant_payouts":                                   "boolean",
		"components.balances.features.standard_payouts":                                  "boolean",
		"components.documents.enabled":                                                   "boolean",
		"components.notification_banner.enabled":                                         "boolean",
		"components.notification_banner.features.external_account_collection":            "boolean",
		"components.payment_details.enabled":                                             "boolean",
		"components.payment_details.features.capture_payments":                           "boolean",
		"components.payment_details.features.destination_on_behalf_of_charge_management": "boolean",
		"components.payment_details.features.dispute_management":                         "boolean",
		"components.payment_details.features.refund_management":                          "boolean",
		"components.payments.enabled":                                                    "boolean",
		"components.payments.features.capture_payments":                                  "boolean",
		"components.payments.features.destination_on_behalf_of_charge_management":        "boolean",
		"components.payments.features.dispute_management":                                "boolean",
		"components.payments.features.refund_management":                                 "boolean",
		"components.payouts.enabled":                                                     "boolean",
		"components.payouts.features.edit_payout_schedule":                               "boolean",
		"components.payouts.features.instant_payouts":                                    "boolean",
		"components.payouts.features.standard_payouts":                                   "boolean",
		"components.payouts_list.enabled":                                                "boolean",
	}, &Config)
	resource.NewOperationCmd(rAccountsCmd.Cmd, "capabilities", "/v1/accounts/{account}/capabilities", http.MethodGet, map[string]string{}, &Config)
	resource.NewOperationCmd(rAccountsCmd.Cmd, "create", "/v1/accounts", http.MethodPost, map[string]string{
		"account_token":                                            "string",
		"business_profile.annual_revenue.amount":                   "integer",
		"business_profile.annual_revenue.currency":                 "string",
		"business_profile.annual_revenue.fiscal_year_end":          "string",
		"business_profile.estimated_worker_count":                  "integer",
		"business_profile.mcc":                                     "string",
		"business_profile.monthly_estimated_revenue.amount":        "integer",
		"business_profile.monthly_estimated_revenue.currency":      "string",
		"business_profile.name":                                    "string",
		"business_profile.product_description":                     "string",
		"business_profile.support_address.city":                    "string",
		"business_profile.support_address.country":                 "string",
		"business_profile.support_address.line1":                   "string",
		"business_profile.support_address.line2":                   "string",
		"business_profile.support_address.postal_code":             "string",
		"business_profile.support_address.state":                   "string",
		"business_profile.support_email":                           "string",
		"business_profile.support_phone":                           "string",
		"business_profile.support_url":                             "string",
		"business_profile.url":                                     "string",
		"business_type":                                            "string",
		"capabilities.acss_debit_payments.requested":               "boolean",
		"capabilities.affirm_payments.requested":                   "boolean",
		"capabilities.afterpay_clearpay_payments.requested":        "boolean",
		"capabilities.amazon_pay_payments.requested":               "boolean",
		"capabilities.au_becs_debit_payments.requested":            "boolean",
		"capabilities.bacs_debit_payments.requested":               "boolean",
		"capabilities.bancontact_payments.requested":               "boolean",
		"capabilities.bank_transfer_payments.requested":            "boolean",
		"capabilities.blik_payments.requested":                     "boolean",
		"capabilities.boleto_payments.requested":                   "boolean",
		"capabilities.card_issuing.requested":                      "boolean",
		"capabilities.card_payments.requested":                     "boolean",
		"capabilities.cartes_bancaires_payments.requested":         "boolean",
		"capabilities.cashapp_payments.requested":                  "boolean",
		"capabilities.eps_payments.requested":                      "boolean",
		"capabilities.fpx_payments.requested":                      "boolean",
		"capabilities.giropay_payments.requested":                  "boolean",
		"capabilities.grabpay_payments.requested":                  "boolean",
		"capabilities.ideal_payments.requested":                    "boolean",
		"capabilities.india_international_payments.requested":      "boolean",
		"capabilities.jcb_payments.requested":                      "boolean",
		"capabilities.klarna_payments.requested":                   "boolean",
		"capabilities.konbini_payments.requested":                  "boolean",
		"capabilities.legacy_payments.requested":                   "boolean",
		"capabilities.link_payments.requested":                     "boolean",
		"capabilities.mobilepay_payments.requested":                "boolean",
		"capabilities.oxxo_payments.requested":                     "boolean",
		"capabilities.p24_payments.requested":                      "boolean",
		"capabilities.paynow_payments.requested":                   "boolean",
		"capabilities.promptpay_payments.requested":                "boolean",
		"capabilities.revolut_pay_payments.requested":              "boolean",
		"capabilities.sepa_debit_payments.requested":               "boolean",
		"capabilities.sofort_payments.requested":                   "boolean",
		"capabilities.swish_payments.requested":                    "boolean",
		"capabilities.tax_reporting_us_1099_k.requested":           "boolean",
		"capabilities.tax_reporting_us_1099_misc.requested":        "boolean",
		"capabilities.transfers.requested":                         "boolean",
		"capabilities.treasury.requested":                          "boolean",
		"capabilities.us_bank_account_ach_payments.requested":      "boolean",
		"capabilities.zip_payments.requested":                      "boolean",
		"company.address.city":                                     "string",
		"company.address.country":                                  "string",
		"company.address.line1":                                    "string",
		"company.address.line2":                                    "string",
		"company.address.postal_code":                              "string",
		"company.address.state":                                    "string",
		"company.address_kana.city":                                "string",
		"company.address_kana.country":                             "string",
		"company.address_kana.line1":                               "string",
		"company.address_kana.line2":                               "string",
		"company.address_kana.postal_code":                         "string",
		"company.address_kana.state":                               "string",
		"company.address_kana.town":                                "string",
		"company.address_kanji.city":                               "string",
		"company.address_kanji.country":                            "string",
		"company.address_kanji.line1":                              "string",
		"company.address_kanji.line2":                              "string",
		"company.address_kanji.postal_code":                        "string",
		"company.address_kanji.state":                              "string",
		"company.address_kanji.town":                               "string",
		"company.directors_provided":                               "boolean",
		"company.executives_provided":                              "boolean",
		"company.export_license_id":                                "string",
		"company.export_purpose_code":                              "string",
		"company.name":                                             "string",
		"company.name_kana":                                        "string",
		"company.name_kanji":                                       "string",
		"company.owners_provided":                                  "boolean",
		"company.ownership_declaration.date":                       "integer",
		"company.ownership_declaration.ip":                         "string",
		"company.ownership_declaration.user_agent":                 "string",
		"company.phone":                                            "string",
		"company.registration_number":                              "string",
		"company.structure":                                        "string",
		"company.tax_id":                                           "string",
		"company.tax_id_registrar":                                 "string",
		"company.vat_id":                                           "string",
		"company.verification.document.back":                       "string",
		"company.verification.document.front":                      "string",
		"controller.fees.payer":                                    "string",
		"controller.losses.payments":                               "string",
		"controller.requirement_collection":                        "string",
		"controller.stripe_dashboard.type":                         "string",
		"country":                                                  "string",
		"default_currency":                                         "string",
		"documents.bank_account_ownership_verification.files":      "array",
		"documents.company_license.files":                          "array",
		"documents.company_memorandum_of_association.files":        "array",
		"documents.company_ministerial_decree.files":               "array",
		"documents.company_registration_verification.files":        "array",
		"documents.company_tax_id_verification.files":              "array",
		"documents.proof_of_registration.files":                    "array",
		"email":                                                    "string",
		"external_account":                                         "string",
		"individual.address.city":                                  "string",
		"individual.address.country":                               "string",
		"individual.address.line1":                                 "string",
		"individual.address.line2":                                 "string",
		"individual.address.postal_code":                           "string",
		"individual.address.state":                                 "string",
		"individual.address_kana.city":                             "string",
		"individual.address_kana.country":                          "string",
		"individual.address_kana.line1":                            "string",
		"individual.address_kana.line2":                            "string",
		"individual.address_kana.postal_code":                      "string",
		"individual.address_kana.state":                            "string",
		"individual.address_kana.town":                             "string",
		"individual.address_kanji.city":                            "string",
		"individual.address_kanji.country":                         "string",
		"individual.address_kanji.line1":                           "string",
		"individual.address_kanji.line2":                           "string",
		"individual.address_kanji.postal_code":                     "string",
		"individual.address_kanji.state":                           "string",
		"individual.address_kanji.town":                            "string",
		"individual.email":                                         "string",
		"individual.first_name":                                    "string",
		"individual.first_name_kana":                               "string",
		"individual.first_name_kanji":                              "string",
		"individual.full_name_aliases":                             "array",
		"individual.gender":                                        "string",
		"individual.id_number":                                     "string",
		"individual.id_number_secondary":                           "string",
		"individual.last_name":                                     "string",
		"individual.last_name_kana":                                "string",
		"individual.last_name_kanji":                               "string",
		"individual.maiden_name":                                   "string",
		"individual.phone":                                         "string",
		"individual.political_exposure":                            "string",
		"individual.registered_address.city":                       "string",
		"individual.registered_address.country":                    "string",
		"individual.registered_address.line1":                      "string",
		"individual.registered_address.line2":                      "string",
		"individual.registered_address.postal_code":                "string",
		"individual.registered_address.state":                      "string",
		"individual.relationship.director":                         "boolean",
		"individual.relationship.executive":                        "boolean",
		"individual.relationship.owner":                            "boolean",
		"individual.relationship.percent_ownership":                "number",
		"individual.relationship.title":                            "string",
		"individual.ssn_last_4":                                    "string",
		"individual.verification.additional_document.back":         "string",
		"individual.verification.additional_document.front":        "string",
		"individual.verification.document.back":                    "string",
		"individual.verification.document.front":                   "string",
		"settings.bacs_debit_payments.display_name":                "string",
		"settings.branding.icon":                                   "string",
		"settings.branding.logo":                                   "string",
		"settings.branding.primary_color":                          "string",
		"settings.branding.secondary_color":                        "string",
		"settings.card_issuing.tos_acceptance.date":                "integer",
		"settings.card_issuing.tos_acceptance.ip":                  "string",
		"settings.card_issuing.tos_acceptance.user_agent":          "string",
		"settings.card_payments.decline_on.avs_failure":            "boolean",
		"settings.card_payments.decline_on.cvc_failure":            "boolean",
		"settings.card_payments.statement_descriptor_prefix":       "string",
		"settings.card_payments.statement_descriptor_prefix_kana":  "string",
		"settings.card_payments.statement_descriptor_prefix_kanji": "string",
		"settings.payments.statement_descriptor":                   "string",
		"settings.payments.statement_descriptor_kana":              "string",
		"settings.payments.statement_descriptor_kanji":             "string",
		"settings.payouts.debit_negative_balances":                 "boolean",
		"settings.payouts.schedule.delay_days":                     "string",
		"settings.payouts.schedule.interval":                       "string",
		"settings.payouts.schedule.monthly_anchor":                 "integer",
		"settings.payouts.schedule.weekly_anchor":                  "string",
		"settings.payouts.statement_descriptor":                    "string",
		"settings.treasury.tos_acceptance.date":                    "integer",
		"settings.treasury.tos_acceptance.ip":                      "string",
		"settings.treasury.tos_acceptance.user_agent":              "string",
		"tos_acceptance.date":                                      "integer",
		"tos_acceptance.ip":                                        "string",
		"tos_acceptance.service_agreement":                         "string",
		"tos_acceptance.user_agent":                                "string",
		"type":                                                     "string",
	}, &Config)
	resource.NewOperationCmd(rAccountsCmd.Cmd, "delete", "/v1/accounts/{account}", http.MethodDelete, map[string]string{}, &Config)
	resource.NewOperationCmd(rAccountsCmd.Cmd, "list", "/v1/accounts", http.MethodGet, map[string]string{
		"created":        "integer",
		"ending_before":  "string",
		"limit":          "integer",
		"starting_after": "string",
	}, &Config)
	resource.NewOperationCmd(rAccountsCmd.Cmd, "persons", "/v1/accounts/{account}/persons", http.MethodGet, map[string]string{
		"ending_before":  "string",
		"limit":          "integer",
		"starting_after": "string",
	}, &Config)
	resource.NewOperationCmd(rAccountsCmd.Cmd, "reject", "/v1/accounts/{account}/reject", http.MethodPost, map[string]string{
		"reason": "string",
	}, &Config)
	resource.NewOperationCmd(rAccountsCmd.Cmd, "retrieve", "/v1/account", http.MethodGet, map[string]string{}, &Config)
	resource.NewOperationCmd(rAccountsCmd.Cmd, "update", "/v1/accounts/{account}", http.MethodPost, map[string]string{
		"account_token":                                            "string",
		"business_profile.annual_revenue.amount":                   "integer",
		"business_profile.annual_revenue.currency":                 "string",
		"business_profile.annual_revenue.fiscal_year_end":          "string",
		"business_profile.estimated_worker_count":                  "integer",
		"business_profile.mcc":                                     "string",
		"business_profile.monthly_estimated_revenue.amount":        "integer",
		"business_profile.monthly_estimated_revenue.currency":      "string",
		"business_profile.name":                                    "string",
		"business_profile.product_description":                     "string",
		"business_profile.support_address.city":                    "string",
		"business_profile.support_address.country":                 "string",
		"business_profile.support_address.line1":                   "string",
		"business_profile.support_address.line2":                   "string",
		"business_profile.support_address.postal_code":             "string",
		"business_profile.support_address.state":                   "string",
		"business_profile.support_email":                           "string",
		"business_profile.support_phone":                           "string",
		"business_profile.support_url":                             "string",
		"business_profile.url":                                     "string",
		"business_type":                                            "string",
		"capabilities.acss_debit_payments.requested":               "boolean",
		"capabilities.affirm_payments.requested":                   "boolean",
		"capabilities.afterpay_clearpay_payments.requested":        "boolean",
		"capabilities.amazon_pay_payments.requested":               "boolean",
		"capabilities.au_becs_debit_payments.requested":            "boolean",
		"capabilities.bacs_debit_payments.requested":               "boolean",
		"capabilities.bancontact_payments.requested":               "boolean",
		"capabilities.bank_transfer_payments.requested":            "boolean",
		"capabilities.blik_payments.requested":                     "boolean",
		"capabilities.boleto_payments.requested":                   "boolean",
		"capabilities.card_issuing.requested":                      "boolean",
		"capabilities.card_payments.requested":                     "boolean",
		"capabilities.cartes_bancaires_payments.requested":         "boolean",
		"capabilities.cashapp_payments.requested":                  "boolean",
		"capabilities.eps_payments.requested":                      "boolean",
		"capabilities.fpx_payments.requested":                      "boolean",
		"capabilities.giropay_payments.requested":                  "boolean",
		"capabilities.grabpay_payments.requested":                  "boolean",
		"capabilities.ideal_payments.requested":                    "boolean",
		"capabilities.india_international_payments.requested":      "boolean",
		"capabilities.jcb_payments.requested":                      "boolean",
		"capabilities.klarna_payments.requested":                   "boolean",
		"capabilities.konbini_payments.requested":                  "boolean",
		"capabilities.legacy_payments.requested":                   "boolean",
		"capabilities.link_payments.requested":                     "boolean",
		"capabilities.mobilepay_payments.requested":                "boolean",
		"capabilities.oxxo_payments.requested":                     "boolean",
		"capabilities.p24_payments.requested":                      "boolean",
		"capabilities.paynow_payments.requested":                   "boolean",
		"capabilities.promptpay_payments.requested":                "boolean",
		"capabilities.revolut_pay_payments.requested":              "boolean",
		"capabilities.sepa_debit_payments.requested":               "boolean",
		"capabilities.sofort_payments.requested":                   "boolean",
		"capabilities.swish_payments.requested":                    "boolean",
		"capabilities.tax_reporting_us_1099_k.requested":           "boolean",
		"capabilities.tax_reporting_us_1099_misc.requested":        "boolean",
		"capabilities.transfers.requested":                         "boolean",
		"capabilities.treasury.requested":                          "boolean",
		"capabilities.us_bank_account_ach_payments.requested":      "boolean",
		"capabilities.zip_payments.requested":                      "boolean",
		"company.address.city":                                     "string",
		"company.address.country":                                  "string",
		"company.address.line1":                                    "string",
		"company.address.line2":                                    "string",
		"company.address.postal_code":                              "string",
		"company.address.state":                                    "string",
		"company.address_kana.city":                                "string",
		"company.address_kana.country":                             "string",
		"company.address_kana.line1":                               "string",
		"company.address_kana.line2":                               "string",
		"company.address_kana.postal_code":                         "string",
		"company.address_kana.state":                               "string",
		"company.address_kana.town":                                "string",
		"company.address_kanji.city":                               "string",
		"company.address_kanji.country":                            "string",
		"company.address_kanji.line1":                              "string",
		"company.address_kanji.line2":                              "string",
		"company.address_kanji.postal_code":                        "string",
		"company.address_kanji.state":                              "string",
		"company.address_kanji.town":                               "string",
		"company.directors_provided":                               "boolean",
		"company.executives_provided":                              "boolean",
		"company.export_license_id":                                "string",
		"company.export_purpose_code":                              "string",
		"company.name":                                             "string",
		"company.name_kana":                                        "string",
		"company.name_kanji":                                       "string",
		"company.owners_provided":                                  "boolean",
		"company.ownership_declaration.date":                       "integer",
		"company.ownership_declaration.ip":                         "string",
		"company.ownership_declaration.user_agent":                 "string",
		"company.phone":                                            "string",
		"company.registration_number":                              "string",
		"company.structure":                                        "string",
		"company.tax_id":                                           "string",
		"company.tax_id_registrar":                                 "string",
		"company.vat_id":                                           "string",
		"company.verification.document.back":                       "string",
		"company.verification.document.front":                      "string",
		"default_currency":                                         "string",
		"documents.bank_account_ownership_verification.files":      "array",
		"documents.company_license.files":                          "array",
		"documents.company_memorandum_of_association.files":        "array",
		"documents.company_ministerial_decree.files":               "array",
		"documents.company_registration_verification.files":        "array",
		"documents.company_tax_id_verification.files":              "array",
		"documents.proof_of_registration.files":                    "array",
		"email":                                                    "string",
		"external_account":                                         "string",
		"individual.address.city":                                  "string",
		"individual.address.country":                               "string",
		"individual.address.line1":                                 "string",
		"individual.address.line2":                                 "string",
		"individual.address.postal_code":                           "string",
		"individual.address.state":                                 "string",
		"individual.address_kana.city":                             "string",
		"individual.address_kana.country":                          "string",
		"individual.address_kana.line1":                            "string",
		"individual.address_kana.line2":                            "string",
		"individual.address_kana.postal_code":                      "string",
		"individual.address_kana.state":                            "string",
		"individual.address_kana.town":                             "string",
		"individual.address_kanji.city":                            "string",
		"individual.address_kanji.country":                         "string",
		"individual.address_kanji.line1":                           "string",
		"individual.address_kanji.line2":                           "string",
		"individual.address_kanji.postal_code":                     "string",
		"individual.address_kanji.state":                           "string",
		"individual.address_kanji.town":                            "string",
		"individual.email":                                         "string",
		"individual.first_name":                                    "string",
		"individual.first_name_kana":                               "string",
		"individual.first_name_kanji":                              "string",
		"individual.full_name_aliases":                             "array",
		"individual.gender":                                        "string",
		"individual.id_number":                                     "string",
		"individual.id_number_secondary":                           "string",
		"individual.last_name":                                     "string",
		"individual.last_name_kana":                                "string",
		"individual.last_name_kanji":                               "string",
		"individual.maiden_name":                                   "string",
		"individual.phone":                                         "string",
		"individual.political_exposure":                            "string",
		"individual.registered_address.city":                       "string",
		"individual.registered_address.country":                    "string",
		"individual.registered_address.line1":                      "string",
		"individual.registered_address.line2":                      "string",
		"individual.registered_address.postal_code":                "string",
		"individual.registered_address.state":                      "string",
		"individual.relationship.director":                         "boolean",
		"individual.relationship.executive":                        "boolean",
		"individual.relationship.owner":                            "boolean",
		"individual.relationship.percent_ownership":                "number",
		"individual.relationship.title":                            "string",
		"individual.ssn_last_4":                                    "string",
		"individual.verification.additional_document.back":         "string",
		"individual.verification.additional_document.front":        "string",
		"individual.verification.document.back":                    "string",
		"individual.verification.document.front":                   "string",
		"settings.bacs_debit_payments.display_name":                "string",
		"settings.branding.icon":                                   "string",
		"settings.branding.logo":                                   "string",
		"settings.branding.primary_color":                          "string",
		"settings.branding.secondary_color":                        "string",
		"settings.card_issuing.tos_acceptance.date":                "integer",
		"settings.card_issuing.tos_acceptance.ip":                  "string",
		"settings.card_issuing.tos_acceptance.user_agent":          "string",
		"settings.card_payments.decline_on.avs_failure":            "boolean",
		"settings.card_payments.decline_on.cvc_failure":            "boolean",
		"settings.card_payments.statement_descriptor_prefix":       "string",
		"settings.card_payments.statement_descriptor_prefix_kana":  "string",
		"settings.card_payments.statement_descriptor_prefix_kanji": "string",
		"settings.invoices.default_account_tax_ids":                "array",
		"settings.payments.statement_descriptor":                   "string",
		"settings.payments.statement_descriptor_kana":              "string",
		"settings.payments.statement_descriptor_kanji":             "string",
		"settings.payouts.debit_negative_balances":                 "boolean",
		"settings.payouts.schedule.delay_days":                     "string",
		"settings.payouts.schedule.interval":                       "string",
		"settings.payouts.schedule.monthly_anchor":                 "integer",
		"settings.payouts.schedule.weekly_anchor":                  "string",
		"settings.payouts.statement_descriptor":                    "string",
		"settings.treasury.tos_acceptance.date":                    "integer",
		"settings.treasury.tos_acceptance.ip":                      "string",
		"settings.treasury.tos_acceptance.user_agent":              "string",
		"tos_acceptance.date":                                      "integer",
		"tos_acceptance.ip":                                        "string",
		"tos_acceptance.service_agreement":                         "string",
		"tos_acceptance.user_agent":                                "string",
	}, &Config)
	resource.NewOperationCmd(rApplePayDomainsCmd.Cmd, "create", "/v1/apple_pay/domains", http.MethodPost, map[string]string{
		"domain_name": "string",
	}, &Config)
	resource.NewOperationCmd(rApplePayDomainsCmd.Cmd, "delete", "/v1/apple_pay/domains/{domain}", http.MethodDelete, map[string]string{}, &Config)
	resource.NewOperationCmd(rApplePayDomainsCmd.Cmd, "list", "/v1/apple_pay/domains", http.MethodGet, map[string]string{
		"domain_name":    "string",
		"ending_before":  "string",
		"limit":          "integer",
		"starting_after": "string",
	}, &Config)
	resource.NewOperationCmd(rApplePayDomainsCmd.Cmd, "retrieve", "/v1/apple_pay/domains/{domain}", http.MethodGet, map[string]string{}, &Config)
	resource.NewOperationCmd(rApplicationFeesCmd.Cmd, "list", "/v1/application_fees", http.MethodGet, map[string]string{
		"charge":         "string",
		"created":        "integer",
		"ending_before":  "string",
		"limit":          "integer",
		"starting_after": "string",
	}, &Config)
	resource.NewOperationCmd(rApplicationFeesCmd.Cmd, "retrieve", "/v1/application_fees/{id}", http.MethodGet, map[string]string{}, &Config)
	resource.NewOperationCmd(rBalanceCmd.Cmd, "retrieve", "/v1/balance", http.MethodGet, map[string]string{}, &Config)
	resource.NewOperationCmd(rBalanceTransactionsCmd.Cmd, "list", "/v1/balance_transactions", http.MethodGet, map[string]string{
		"created":        "integer",
		"currency":       "string",
		"ending_before":  "string",
		"limit":          "integer",
		"payout":         "string",
		"source":         "string",
		"starting_after": "string",
		"type":           "string",
	}, &Config)
	resource.NewOperationCmd(rBalanceTransactionsCmd.Cmd, "retrieve", "/v1/balance_transactions/{id}", http.MethodGet, map[string]string{}, &Config)
	resource.NewOperationCmd(rBankAccountsCmd.Cmd, "delete", "/v1/accounts/{account}/external_accounts/{id}", http.MethodDelete, map[string]string{}, &Config)
	resource.NewOperationCmd(rBankAccountsCmd.Cmd, "update", "/v1/accounts/{account}/external_accounts/{id}", http.MethodPost, map[string]string{
		"account_holder_name":  "string",
		"account_holder_type":  "string",
		"account_type":         "string",
		"address_city":         "string",
		"address_country":      "string",
		"address_line1":        "string",
		"address_line2":        "string",
		"address_state":        "string",
		"address_zip":          "string",
		"default_for_currency": "boolean",
		"documents.bank_account_ownership_verification.files": "array",
		"exp_month": "string",
		"exp_year":  "string",
		"name":      "string",
	}, &Config)
	resource.NewOperationCmd(rBankAccountsCmd.Cmd, "verify", "/v1/customers/{customer}/sources/{id}/verify", http.MethodPost, map[string]string{
		"amounts": "array",
	}, &Config)
	resource.NewOperationCmd(rCapabilitiesCmd.Cmd, "list", "/v1/accounts/{account}/capabilities", http.MethodGet, map[string]string{}, &Config)
	resource.NewOperationCmd(rCapabilitiesCmd.Cmd, "retrieve", "/v1/accounts/{account}/capabilities/{capability}", http.MethodGet, map[string]string{}, &Config)
	resource.NewOperationCmd(rCapabilitiesCmd.Cmd, "update", "/v1/accounts/{account}/capabilities/{capability}", http.MethodPost, map[string]string{
		"requested": "boolean",
	}, &Config)
	resource.NewOperationCmd(rCardsCmd.Cmd, "delete", "/v1/accounts/{account}/external_accounts/{id}", http.MethodDelete, map[string]string{}, &Config)
	resource.NewOperationCmd(rCardsCmd.Cmd, "update", "/v1/accounts/{account}/external_accounts/{id}", http.MethodPost, map[string]string{
		"account_holder_name":  "string",
		"account_holder_type":  "string",
		"account_type":         "string",
		"address_city":         "string",
		"address_country":      "string",
		"address_line1":        "string",
		"address_line2":        "string",
		"address_state":        "string",
		"address_zip":          "string",
		"default_for_currency": "boolean",
		"documents.bank_account_ownership_verification.files": "array",
		"exp_month": "string",
		"exp_year":  "string",
		"name":      "string",
	}, &Config)
	resource.NewOperationCmd(rCashBalancesCmd.Cmd, "retrieve", "/v1/customers/{customer}/cash_balance", http.MethodGet, map[string]string{}, &Config)
	resource.NewOperationCmd(rCashBalancesCmd.Cmd, "update", "/v1/customers/{customer}/cash_balance", http.MethodPost, map[string]string{
		"settings.reconciliation_mode": "string",
	}, &Config)
	resource.NewOperationCmd(rChargesCmd.Cmd, "capture", "/v1/charges/{charge}/capture", http.MethodPost, map[string]string{
		"amount":                      "integer",
		"application_fee":             "integer",
		"application_fee_amount":      "integer",
		"receipt_email":               "string",
		"statement_descriptor":        "string",
		"statement_descriptor_suffix": "string",
		"transfer_data.amount":        "integer",
		"transfer_group":              "string",
	}, &Config)
	resource.NewOperationCmd(rChargesCmd.Cmd, "create", "/v1/charges", http.MethodPost, map[string]string{
		"amount":                       "integer",
		"application_fee":              "integer",
		"application_fee_amount":       "integer",
		"capture":                      "boolean",
		"currency":                     "string",
		"customer":                     "string",
		"description":                  "string",
		"destination.account":          "string",
		"destination.amount":           "integer",
		"on_behalf_of":                 "string",
		"radar_options.session":        "string",
		"receipt_email":                "string",
		"shipping.address.city":        "string",
		"shipping.address.country":     "string",
		"shipping.address.line1":       "string",
		"shipping.address.line2":       "string",
		"shipping.address.postal_code": "string",
		"shipping.address.state":       "string",
		"shipping.carrier":             "string",
		"shipping.name":                "string",
		"shipping.phone":               "string",
		"shipping.tracking_number":     "string",
		"source":                       "string",
		"statement_descriptor":         "string",
		"statement_descriptor_suffix":  "string",
		"transfer_data.amount":         "integer",
		"transfer_data.destination":    "string",
		"transfer_group":               "string",
	}, &Config)
	resource.NewOperationCmd(rChargesCmd.Cmd, "list", "/v1/charges", http.MethodGet, map[string]string{
		"created":        "integer",
		"customer":       "string",
		"ending_before":  "string",
		"limit":          "integer",
		"payment_intent": "string",
		"starting_after": "string",
		"transfer_group": "string",
	}, &Config)
	resource.NewOperationCmd(rChargesCmd.Cmd, "retrieve", "/v1/charges/{charge}", http.MethodGet, map[string]string{}, &Config)
	resource.NewOperationCmd(rChargesCmd.Cmd, "search", "/v1/charges/search", http.MethodGet, map[string]string{
		"limit": "integer",
		"page":  "string",
		"query": "string",
	}, &Config)
	resource.NewOperationCmd(rChargesCmd.Cmd, "update", "/v1/charges/{charge}", http.MethodPost, map[string]string{
		"customer":                     "string",
		"description":                  "string",
		"fraud_details.user_report":    "string",
		"receipt_email":                "string",
		"shipping.address.city":        "string",
		"shipping.address.country":     "string",
		"shipping.address.line1":       "string",
		"shipping.address.line2":       "string",
		"shipping.address.postal_code": "string",
		"shipping.address.state":       "string",
		"shipping.carrier":             "string",
		"shipping.name":                "string",
		"shipping.phone":               "string",
		"shipping.tracking_number":     "string",
		"transfer_group":               "string",
	}, &Config)
	resource.NewOperationCmd(rConfirmationTokensCmd.Cmd, "retrieve", "/v1/confirmation_tokens/{confirmation_token}", http.MethodGet, map[string]string{}, &Config)
	resource.NewOperationCmd(rConfirmationTokensTestHelpersCmd.Cmd, "create", "/v1/test_helpers/confirmation_tokens", http.MethodPost, map[string]string{
		"payment_method": "string",
		"payment_method_data.acss_debit.account_number":                     "string",
		"payment_method_data.acss_debit.institution_number":                 "string",
		"payment_method_data.acss_debit.transit_number":                     "string",
		"payment_method_data.allow_redisplay":                               "string",
		"payment_method_data.au_becs_debit.account_number":                  "string",
		"payment_method_data.au_becs_debit.bsb_number":                      "string",
		"payment_method_data.bacs_debit.account_number":                     "string",
		"payment_method_data.bacs_debit.sort_code":                          "string",
		"payment_method_data.billing_details.email":                         "string",
		"payment_method_data.billing_details.name":                          "string",
		"payment_method_data.billing_details.phone":                         "string",
		"payment_method_data.boleto.tax_id":                                 "string",
		"payment_method_data.eps.bank":                                      "string",
		"payment_method_data.fpx.account_holder_type":                       "string",
		"payment_method_data.fpx.bank":                                      "string",
		"payment_method_data.ideal.bank":                                    "string",
		"payment_method_data.klarna.dob.day":                                "integer",
		"payment_method_data.klarna.dob.month":                              "integer",
		"payment_method_data.klarna.dob.year":                               "integer",
		"payment_method_data.p24.bank":                                      "string",
		"payment_method_data.radar_options.session":                         "string",
		"payment_method_data.sepa_debit.iban":                               "string",
		"payment_method_data.sofort.country":                                "string",
		"payment_method_data.type":                                          "string",
		"payment_method_data.us_bank_account.account_holder_type":           "string",
		"payment_method_data.us_bank_account.account_number":                "string",
		"payment_method_data.us_bank_account.account_type":                  "string",
		"payment_method_data.us_bank_account.financial_connections_account": "string",
		"payment_method_data.us_bank_account.routing_number":                "string",
		"return_url":                   "string",
		"setup_future_usage":           "string",
		"shipping.address.city":        "string",
		"shipping.address.country":     "string",
		"shipping.address.line1":       "string",
		"shipping.address.line2":       "string",
		"shipping.address.postal_code": "string",
		"shipping.address.state":       "string",
		"shipping.name":                "string",
		"shipping.phone":               "string",
	}, &Config)
	resource.NewOperationCmd(rCountrySpecsCmd.Cmd, "list", "/v1/country_specs", http.MethodGet, map[string]string{
		"ending_before":  "string",
		"limit":          "integer",
		"starting_after": "string",
	}, &Config)
	resource.NewOperationCmd(rCountrySpecsCmd.Cmd, "retrieve", "/v1/country_specs/{country}", http.MethodGet, map[string]string{}, &Config)
	resource.NewOperationCmd(rCouponsCmd.Cmd, "create", "/v1/coupons", http.MethodPost, map[string]string{
		"amount_off":          "integer",
		"applies_to.products": "array",
		"currency":            "string",
		"duration":            "string",
		"duration_in_months":  "integer",
		"id":                  "string",
		"max_redemptions":     "integer",
		"name":                "string",
		"percent_off":         "number",
		"redeem_by":           "integer",
	}, &Config)
	resource.NewOperationCmd(rCouponsCmd.Cmd, "delete", "/v1/coupons/{coupon}", http.MethodDelete, map[string]string{}, &Config)
	resource.NewOperationCmd(rCouponsCmd.Cmd, "list", "/v1/coupons", http.MethodGet, map[string]string{
		"created":        "integer",
		"ending_before":  "string",
		"limit":          "integer",
		"starting_after": "string",
	}, &Config)
	resource.NewOperationCmd(rCouponsCmd.Cmd, "retrieve", "/v1/coupons/{coupon}", http.MethodGet, map[string]string{}, &Config)
	resource.NewOperationCmd(rCouponsCmd.Cmd, "update", "/v1/coupons/{coupon}", http.MethodPost, map[string]string{
		"name": "string",
	}, &Config)
	resource.NewOperationCmd(rCreditNoteLineItemsCmd.Cmd, "list", "/v1/credit_notes/{credit_note}/lines", http.MethodGet, map[string]string{
		"ending_before":  "string",
		"limit":          "integer",
		"starting_after": "string",
	}, &Config)
	resource.NewOperationCmd(rCreditNotesCmd.Cmd, "create", "/v1/credit_notes", http.MethodPost, map[string]string{
		"amount":                      "integer",
		"credit_amount":               "integer",
		"effective_at":                "integer",
		"invoice":                     "string",
		"memo":                        "string",
		"out_of_band_amount":          "integer",
		"reason":                      "string",
		"refund":                      "string",
		"refund_amount":               "integer",
		"shipping_cost.shipping_rate": "string",
	}, &Config)
	resource.NewOperationCmd(rCreditNotesCmd.Cmd, "list", "/v1/credit_notes", http.MethodGet, map[string]string{
		"created":        "integer",
		"customer":       "string",
		"ending_before":  "string",
		"invoice":        "string",
		"limit":          "integer",
		"starting_after": "string",
	}, &Config)
	resource.NewOperationCmd(rCreditNotesCmd.Cmd, "preview", "/v1/credit_notes/preview", http.MethodGet, map[string]string{
		"amount":             "integer",
		"credit_amount":      "integer",
		"effective_at":       "integer",
		"invoice":            "string",
		"memo":               "string",
		"out_of_band_amount": "integer",
		"reason":             "string",
		"refund":             "string",
		"refund_amount":      "integer",
	}, &Config)
	resource.NewOperationCmd(rCreditNotesCmd.Cmd, "preview_lines", "/v1/credit_notes/preview/lines", http.MethodGet, map[string]string{
		"amount":             "integer",
		"credit_amount":      "integer",
		"effective_at":       "integer",
		"ending_before":      "string",
		"invoice":            "string",
		"limit":              "integer",
		"memo":               "string",
		"out_of_band_amount": "integer",
		"reason":             "string",
		"refund":             "string",
		"refund_amount":      "integer",
		"starting_after":     "string",
	}, &Config)
	resource.NewOperationCmd(rCreditNotesCmd.Cmd, "retrieve", "/v1/credit_notes/{id}", http.MethodGet, map[string]string{}, &Config)
	resource.NewOperationCmd(rCreditNotesCmd.Cmd, "update", "/v1/credit_notes/{id}", http.MethodPost, map[string]string{
		"memo": "string",
	}, &Config)
	resource.NewOperationCmd(rCreditNotesCmd.Cmd, "void_credit_note", "/v1/credit_notes/{id}/void", http.MethodPost, map[string]string{}, &Config)
	resource.NewOperationCmd(rCustomerBalanceTransactionsCmd.Cmd, "create", "/v1/customers/{customer}/balance_transactions", http.MethodPost, map[string]string{
		"amount":      "integer",
		"currency":    "string",
		"description": "string",
	}, &Config)
	resource.NewOperationCmd(rCustomerBalanceTransactionsCmd.Cmd, "list", "/v1/customers/{customer}/balance_transactions", http.MethodGet, map[string]string{
		"ending_before":  "string",
		"limit":          "integer",
		"starting_after": "string",
	}, &Config)
	resource.NewOperationCmd(rCustomerBalanceTransactionsCmd.Cmd, "retrieve", "/v1/customers/{customer}/balance_transactions/{transaction}", http.MethodGet, map[string]string{}, &Config)
	resource.NewOperationCmd(rCustomerBalanceTransactionsCmd.Cmd, "update", "/v1/customers/{customer}/balance_transactions/{transaction}", http.MethodPost, map[string]string{
		"description": "string",
	}, &Config)
	resource.NewOperationCmd(rCustomerCashBalanceTransactionsCmd.Cmd, "list", "/v1/customers/{customer}/cash_balance_transactions", http.MethodGet, map[string]string{
		"ending_before":  "string",
		"limit":          "integer",
		"starting_after": "string",
	}, &Config)
	resource.NewOperationCmd(rCustomerCashBalanceTransactionsCmd.Cmd, "retrieve", "/v1/customers/{customer}/cash_balance_transactions/{transaction}", http.MethodGet, map[string]string{}, &Config)
	resource.NewOperationCmd(rCustomerSessionsCmd.Cmd, "create", "/v1/customer_sessions", http.MethodPost, map[string]string{
		"components.buy_button.enabled":    "boolean",
		"components.pricing_table.enabled": "boolean",
		"customer":                         "string",
	}, &Config)
	resource.NewOperationCmd(rCustomersCmd.Cmd, "balance_transactions", "/v1/customers/{customer}/balance_transactions", http.MethodGet, map[string]string{
		"ending_before":  "string",
		"limit":          "integer",
		"starting_after": "string",
	}, &Config)
	resource.NewOperationCmd(rCustomersCmd.Cmd, "create", "/v1/customers", http.MethodPost, map[string]string{
		"balance": "integer",
		"cash_balance.settings.reconciliation_mode": "string",
		"coupon":         "string",
		"description":    "string",
		"email":          "string",
		"invoice_prefix": "string",
		"invoice_settings.default_payment_method": "string",
		"invoice_settings.footer":                 "string",
		"name":                                    "string",
		"next_invoice_sequence":                   "integer",
		"payment_method":                          "string",
		"phone":                                   "string",
		"preferred_locales":                       "array",
		"promotion_code":                          "string",
		"source":                                  "string",
		"tax.ip_address":                          "string",
		"tax.validate_location":                   "string",
		"tax_exempt":                              "string",
		"test_clock":                              "string",
		"validate":                                "boolean",
	}, &Config)
	resource.NewOperationCmd(rCustomersCmd.Cmd, "create_funding_instructions", "/v1/customers/{customer}/funding_instructions", http.MethodPost, map[string]string{
		"bank_transfer.eu_bank_transfer.country": "string",
		"bank_transfer.requested_address_types":  "array",
		"bank_transfer.type":                     "string",
		"currency":                               "string",
		"funding_type":                           "string",
	}, &Config)
	resource.NewOperationCmd(rCustomersCmd.Cmd, "delete", "/v1/customers/{customer}", http.MethodDelete, map[string]string{}, &Config)
	resource.NewOperationCmd(rCustomersCmd.Cmd, "delete_discount", "/v1/customers/{customer}/discount", http.MethodDelete, map[string]string{}, &Config)
	resource.NewOperationCmd(rCustomersCmd.Cmd, "list", "/v1/customers", http.MethodGet, map[string]string{
		"created":        "integer",
		"email":          "string",
		"ending_before":  "string",
		"limit":          "integer",
		"starting_after": "string",
		"test_clock":     "string",
	}, &Config)
	resource.NewOperationCmd(rCustomersCmd.Cmd, "list_payment_methods", "/v1/customers/{customer}/payment_methods", http.MethodGet, map[string]string{
		"allow_redisplay": "string",
		"ending_before":   "string",
		"limit":           "integer",
		"starting_after":  "string",
		"type":            "string",
	}, &Config)
	resource.NewOperationCmd(rCustomersCmd.Cmd, "retrieve", "/v1/customers/{customer}", http.MethodGet, map[string]string{}, &Config)
	resource.NewOperationCmd(rCustomersCmd.Cmd, "retrieve_payment_method", "/v1/customers/{customer}/payment_methods/{payment_method}", http.MethodGet, map[string]string{}, &Config)
	resource.NewOperationCmd(rCustomersCmd.Cmd, "search", "/v1/customers/search", http.MethodGet, map[string]string{
		"limit": "integer",
		"page":  "string",
		"query": "string",
	}, &Config)
	resource.NewOperationCmd(rCustomersCmd.Cmd, "update", "/v1/customers/{customer}", http.MethodPost, map[string]string{
		"balance": "integer",
		"cash_balance.settings.reconciliation_mode": "string",
		"coupon":         "string",
		"default_source": "string",
		"description":    "string",
		"email":          "string",
		"invoice_prefix": "string",
		"invoice_settings.default_payment_method": "string",
		"invoice_settings.footer":                 "string",
		"name":                                    "string",
		"next_invoice_sequence":                   "integer",
		"phone":                                   "string",
		"preferred_locales":                       "array",
		"promotion_code":                          "string",
		"source":                                  "string",
		"tax.ip_address":                          "string",
		"tax.validate_location":                   "string",
		"tax_exempt":                              "string",
		"validate":                                "boolean",
	}, &Config)
	resource.NewOperationCmd(rCustomersTestHelpersCmd.Cmd, "fund_cash_balance", "/v1/test_helpers/customers/{customer}/fund_cash_balance", http.MethodPost, map[string]string{
		"amount":    "integer",
		"currency":  "string",
		"reference": "string",
	}, &Config)
	resource.NewOperationCmd(rDisputesCmd.Cmd, "close", "/v1/disputes/{dispute}/close", http.MethodPost, map[string]string{}, &Config)
	resource.NewOperationCmd(rDisputesCmd.Cmd, "list", "/v1/disputes", http.MethodGet, map[string]string{
		"charge":         "string",
		"created":        "integer",
		"ending_before":  "string",
		"limit":          "integer",
		"payment_intent": "string",
		"starting_after": "string",
	}, &Config)
	resource.NewOperationCmd(rDisputesCmd.Cmd, "retrieve", "/v1/disputes/{dispute}", http.MethodGet, map[string]string{}, &Config)
	resource.NewOperationCmd(rDisputesCmd.Cmd, "update", "/v1/disputes/{dispute}", http.MethodPost, map[string]string{
		"evidence.access_activity_log":            "string",
		"evidence.billing_address":                "string",
		"evidence.cancellation_policy":            "string",
		"evidence.cancellation_policy_disclosure": "string",
		"evidence.cancellation_rebuttal":          "string",
		"evidence.customer_communication":         "string",
		"evidence.customer_email_address":         "string",
		"evidence.customer_name":                  "string",
		"evidence.customer_purchase_ip":           "string",
		"evidence.customer_signature":             "string",
		"evidence.duplicate_charge_documentation": "string",
		"evidence.duplicate_charge_explanation":   "string",
		"evidence.duplicate_charge_id":            "string",
		"evidence.product_description":            "string",
		"evidence.receipt":                        "string",
		"evidence.refund_policy":                  "string",
		"evidence.refund_policy_disclosure":       "string",
		"evidence.refund_refusal_explanation":     "string",
		"evidence.service_date":                   "string",
		"evidence.service_documentation":          "string",
		"evidence.shipping_address":               "string",
		"evidence.shipping_carrier":               "string",
		"evidence.shipping_date":                  "string",
		"evidence.shipping_documentation":         "string",
		"evidence.shipping_tracking_number":       "string",
		"evidence.uncategorized_file":             "string",
		"evidence.uncategorized_text":             "string",
		"submit":                                  "boolean",
	}, &Config)
	resource.NewOperationCmd(rEphemeralKeysCmd.Cmd, "create", "/v1/ephemeral_keys", http.MethodPost, map[string]string{
		"customer":             "string",
		"issuing_card":         "string",
		"nonce":                "string",
		"verification_session": "string",
	}, &Config)
	resource.NewOperationCmd(rEphemeralKeysCmd.Cmd, "delete", "/v1/ephemeral_keys/{key}", http.MethodDelete, map[string]string{}, &Config)
	resource.NewOperationCmd(rEventsCmd.Cmd, "list", "/v1/events", http.MethodGet, map[string]string{
		"created":          "integer",
		"delivery_success": "boolean",
		"ending_before":    "string",
		"limit":            "integer",
		"starting_after":   "string",
		"type":             "string",
		"types":            "array",
	}, &Config)
	resource.NewOperationCmd(rEventsCmd.Cmd, "retrieve", "/v1/events/{id}", http.MethodGet, map[string]string{}, &Config)
	resource.NewOperationCmd(rExchangeRatesCmd.Cmd, "list", "/v1/exchange_rates", http.MethodGet, map[string]string{
		"ending_before":  "string",
		"limit":          "integer",
		"starting_after": "string",
	}, &Config)
	resource.NewOperationCmd(rExchangeRatesCmd.Cmd, "retrieve", "/v1/exchange_rates/{rate_id}", http.MethodGet, map[string]string{}, &Config)
	resource.NewOperationCmd(rExternalAccountsCmd.Cmd, "create", "/v1/accounts/{account}/external_accounts", http.MethodPost, map[string]string{
		"default_for_currency": "boolean",
		"external_account":     "string",
	}, &Config)
	resource.NewOperationCmd(rExternalAccountsCmd.Cmd, "delete", "/v1/accounts/{account}/external_accounts/{id}", http.MethodDelete, map[string]string{}, &Config)
	resource.NewOperationCmd(rExternalAccountsCmd.Cmd, "list", "/v1/accounts/{account}/external_accounts", http.MethodGet, map[string]string{
		"ending_before":  "string",
		"limit":          "integer",
		"object":         "string",
		"starting_after": "string",
	}, &Config)
	resource.NewOperationCmd(rExternalAccountsCmd.Cmd, "retrieve", "/v1/accounts/{account}/external_accounts/{id}", http.MethodGet, map[string]string{}, &Config)
	resource.NewOperationCmd(rExternalAccountsCmd.Cmd, "update", "/v1/accounts/{account}/external_accounts/{id}", http.MethodPost, map[string]string{
		"account_holder_name":  "string",
		"account_holder_type":  "string",
		"account_type":         "string",
		"address_city":         "string",
		"address_country":      "string",
		"address_line1":        "string",
		"address_line2":        "string",
		"address_state":        "string",
		"address_zip":          "string",
		"default_for_currency": "boolean",
		"documents.bank_account_ownership_verification.files": "array",
		"exp_month": "string",
		"exp_year":  "string",
		"name":      "string",
	}, &Config)
	resource.NewOperationCmd(rFeeRefundsCmd.Cmd, "create", "/v1/application_fees/{id}/refunds", http.MethodPost, map[string]string{
		"amount": "integer",
	}, &Config)
	resource.NewOperationCmd(rFeeRefundsCmd.Cmd, "list", "/v1/application_fees/{id}/refunds", http.MethodGet, map[string]string{
		"ending_before":  "string",
		"limit":          "integer",
		"starting_after": "string",
	}, &Config)
	resource.NewOperationCmd(rFeeRefundsCmd.Cmd, "retrieve", "/v1/application_fees/{fee}/refunds/{id}", http.MethodGet, map[string]string{}, &Config)
	resource.NewOperationCmd(rFeeRefundsCmd.Cmd, "update", "/v1/application_fees/{fee}/refunds/{id}", http.MethodPost, map[string]string{}, &Config)
	resource.NewOperationCmd(rFileLinksCmd.Cmd, "create", "/v1/file_links", http.MethodPost, map[string]string{
		"expires_at": "integer",
		"file":       "string",
	}, &Config)
	resource.NewOperationCmd(rFileLinksCmd.Cmd, "list", "/v1/file_links", http.MethodGet, map[string]string{
		"created":        "integer",
		"ending_before":  "string",
		"expired":        "boolean",
		"file":           "string",
		"limit":          "integer",
		"starting_after": "string",
	}, &Config)
	resource.NewOperationCmd(rFileLinksCmd.Cmd, "retrieve", "/v1/file_links/{link}", http.MethodGet, map[string]string{}, &Config)
	resource.NewOperationCmd(rFileLinksCmd.Cmd, "update", "/v1/file_links/{link}", http.MethodPost, map[string]string{
		"expires_at": "string",
	}, &Config)
	resource.NewOperationCmd(rFilesCmd.Cmd, "create", "/v1/files", http.MethodPost, map[string]string{}, &Config)
	resource.NewOperationCmd(rFilesCmd.Cmd, "list", "/v1/files", http.MethodGet, map[string]string{
		"created":        "integer",
		"ending_before":  "string",
		"limit":          "integer",
		"purpose":        "string",
		"starting_after": "string",
	}, &Config)
	resource.NewOperationCmd(rFilesCmd.Cmd, "retrieve", "/v1/files/{file}", http.MethodGet, map[string]string{}, &Config)
	resource.NewOperationCmd(rInvoiceitemsCmd.Cmd, "create", "/v1/invoiceitems", http.MethodPost, map[string]string{
		"amount":                         "integer",
		"currency":                       "string",
		"customer":                       "string",
		"description":                    "string",
		"discountable":                   "boolean",
		"invoice":                        "string",
		"period.end":                     "integer",
		"period.start":                   "integer",
		"price":                          "string",
		"price_data.currency":            "string",
		"price_data.product":             "string",
		"price_data.tax_behavior":        "string",
		"price_data.unit_amount":         "integer",
		"price_data.unit_amount_decimal": "string",
		"quantity":                       "integer",
		"subscription":                   "string",
		"tax_behavior":                   "string",
		"tax_code":                       "string",
		"tax_rates":                      "array",
		"unit_amount":                    "integer",
		"unit_amount_decimal":            "string",
	}, &Config)
	resource.NewOperationCmd(rInvoiceitemsCmd.Cmd, "delete", "/v1/invoiceitems/{invoiceitem}", http.MethodDelete, map[string]string{}, &Config)
	resource.NewOperationCmd(rInvoiceitemsCmd.Cmd, "list", "/v1/invoiceitems", http.MethodGet, map[string]string{
		"created":        "integer",
		"customer":       "string",
		"ending_before":  "string",
		"invoice":        "string",
		"limit":          "integer",
		"pending":        "boolean",
		"starting_after": "string",
	}, &Config)
	resource.NewOperationCmd(rInvoiceitemsCmd.Cmd, "retrieve", "/v1/invoiceitems/{invoiceitem}", http.MethodGet, map[string]string{}, &Config)
	resource.NewOperationCmd(rInvoiceitemsCmd.Cmd, "update", "/v1/invoiceitems/{invoiceitem}", http.MethodPost, map[string]string{
		"amount":                         "integer",
		"description":                    "string",
		"discountable":                   "boolean",
		"period.end":                     "integer",
		"period.start":                   "integer",
		"price":                          "string",
		"price_data.currency":            "string",
		"price_data.product":             "string",
		"price_data.tax_behavior":        "string",
		"price_data.unit_amount":         "integer",
		"price_data.unit_amount_decimal": "string",
		"quantity":                       "integer",
		"tax_behavior":                   "string",
		"tax_code":                       "string",
		"tax_rates":                      "array",
		"unit_amount":                    "integer",
		"unit_amount_decimal":            "string",
	}, &Config)
	resource.NewOperationCmd(rInvoicesCmd.Cmd, "create", "/v1/invoices", http.MethodPost, map[string]string{
		"account_tax_ids":                       "array",
		"application_fee_amount":                "integer",
		"auto_advance":                          "boolean",
		"automatic_tax.enabled":                 "boolean",
		"automatic_tax.liability.account":       "string",
		"automatic_tax.liability.type":          "string",
		"collection_method":                     "string",
		"currency":                              "string",
		"customer":                              "string",
		"days_until_due":                        "integer",
		"default_payment_method":                "string",
		"default_source":                        "string",
		"default_tax_rates":                     "array",
		"description":                           "string",
		"due_date":                              "integer",
		"effective_at":                          "integer",
		"footer":                                "string",
		"from_invoice.action":                   "string",
		"from_invoice.invoice":                  "string",
		"issuer.account":                        "string",
		"issuer.type":                           "string",
		"number":                                "string",
		"on_behalf_of":                          "string",
		"payment_settings.default_mandate":      "string",
		"payment_settings.payment_method_types": "array",
		"pending_invoice_items_behavior":        "string",
		"rendering.amount_tax_display":          "string",
		"rendering.pdf.page_size":               "string",
		"shipping_cost.shipping_rate":           "string",
		"shipping_cost.shipping_rate_data.delivery_estimate.maximum.unit":  "string",
		"shipping_cost.shipping_rate_data.delivery_estimate.maximum.value": "integer",
		"shipping_cost.shipping_rate_data.delivery_estimate.minimum.unit":  "string",
		"shipping_cost.shipping_rate_data.delivery_estimate.minimum.value": "integer",
		"shipping_cost.shipping_rate_data.display_name":                    "string",
		"shipping_cost.shipping_rate_data.fixed_amount.amount":             "integer",
		"shipping_cost.shipping_rate_data.fixed_amount.currency":           "string",
		"shipping_cost.shipping_rate_data.tax_behavior":                    "string",
		"shipping_cost.shipping_rate_data.tax_code":                        "string",
		"shipping_cost.shipping_rate_data.type":                            "string",
		"shipping_details.address.city":                                    "string",
		"shipping_details.address.country":                                 "string",
		"shipping_details.address.line1":                                   "string",
		"shipping_details.address.line2":                                   "string",
		"shipping_details.address.postal_code":                             "string",
		"shipping_details.address.state":                                   "string",
		"shipping_details.name":                                            "string",
		"shipping_details.phone":                                           "string",
		"statement_descriptor":                                             "string",
		"subscription":                                                     "string",
		"transfer_data.amount":                                             "integer",
		"transfer_data.destination":                                        "string",
	}, &Config)
	resource.NewOperationCmd(rInvoicesCmd.Cmd, "create_preview", "/v1/invoices/create_preview", http.MethodPost, map[string]string{
		"automatic_tax.enabled":                     "boolean",
		"automatic_tax.liability.account":           "string",
		"automatic_tax.liability.type":              "string",
		"coupon":                                    "string",
		"currency":                                  "string",
		"customer":                                  "string",
		"customer_details.tax.ip_address":           "string",
		"customer_details.tax_exempt":               "string",
		"issuer.account":                            "string",
		"issuer.type":                               "string",
		"on_behalf_of":                              "string",
		"preview_mode":                              "string",
		"schedule":                                  "string",
		"schedule_details.end_behavior":             "string",
		"schedule_details.proration_behavior":       "string",
		"subscription":                              "string",
		"subscription_details.billing_cycle_anchor": "string",
		"subscription_details.cancel_at":            "integer",
		"subscription_details.cancel_at_period_end": "boolean",
		"subscription_details.cancel_now":           "boolean",
		"subscription_details.default_tax_rates":    "array",
		"subscription_details.proration_behavior":   "string",
		"subscription_details.proration_date":       "integer",
		"subscription_details.resume_at":            "string",
		"subscription_details.start_date":           "integer",
		"subscription_details.trial_end":            "string",
	}, &Config)
	resource.NewOperationCmd(rInvoicesCmd.Cmd, "delete", "/v1/invoices/{invoice}", http.MethodDelete, map[string]string{}, &Config)
	resource.NewOperationCmd(rInvoicesCmd.Cmd, "finalize_invoice", "/v1/invoices/{invoice}/finalize", http.MethodPost, map[string]string{
		"auto_advance": "boolean",
	}, &Config)
	resource.NewOperationCmd(rInvoicesCmd.Cmd, "list", "/v1/invoices", http.MethodGet, map[string]string{
		"collection_method": "string",
		"created":           "integer",
		"customer":          "string",
		"due_date":          "integer",
		"ending_before":     "string",
		"limit":             "integer",
		"starting_after":    "string",
		"status":            "string",
		"subscription":      "string",
	}, &Config)
	resource.NewOperationCmd(rInvoicesCmd.Cmd, "mark_uncollectible", "/v1/invoices/{invoice}/mark_uncollectible", http.MethodPost, map[string]string{}, &Config)
	resource.NewOperationCmd(rInvoicesCmd.Cmd, "pay", "/v1/invoices/{invoice}/pay", http.MethodPost, map[string]string{
		"forgive":          "boolean",
		"mandate":          "string",
		"off_session":      "boolean",
		"paid_out_of_band": "boolean",
		"payment_method":   "string",
		"source":           "string",
	}, &Config)
	resource.NewOperationCmd(rInvoicesCmd.Cmd, "retrieve", "/v1/invoices/{invoice}", http.MethodGet, map[string]string{}, &Config)
	resource.NewOperationCmd(rInvoicesCmd.Cmd, "search", "/v1/invoices/search", http.MethodGet, map[string]string{
		"limit": "integer",
		"page":  "string",
		"query": "string",
	}, &Config)
	resource.NewOperationCmd(rInvoicesCmd.Cmd, "send_invoice", "/v1/invoices/{invoice}/send", http.MethodPost, map[string]string{}, &Config)
	resource.NewOperationCmd(rInvoicesCmd.Cmd, "upcoming", "/v1/invoices/upcoming", http.MethodGet, map[string]string{
		"coupon":                            "string",
		"currency":                          "string",
		"customer":                          "string",
		"on_behalf_of":                      "string",
		"preview_mode":                      "string",
		"schedule":                          "string",
		"subscription":                      "string",
		"subscription_billing_cycle_anchor": "string",
		"subscription_cancel_at":            "integer",
		"subscription_cancel_at_period_end": "boolean",
		"subscription_cancel_now":           "boolean",
		"subscription_default_tax_rates":    "array",
		"subscription_proration_behavior":   "string",
		"subscription_proration_date":       "integer",
		"subscription_resume_at":            "string",
		"subscription_start_date":           "integer",
		"subscription_trial_end":            "string",
		"subscription_trial_from_plan":      "boolean",
	}, &Config)
	resource.NewOperationCmd(rInvoicesCmd.Cmd, "upcomingLines", "/v1/invoices/upcoming/lines", http.MethodGet, map[string]string{
		"coupon":                            "string",
		"currency":                          "string",
		"customer":                          "string",
		"ending_before":                     "string",
		"limit":                             "integer",
		"on_behalf_of":                      "string",
		"preview_mode":                      "string",
		"schedule":                          "string",
		"starting_after":                    "string",
		"subscription":                      "string",
		"subscription_billing_cycle_anchor": "string",
		"subscription_cancel_at":            "integer",
		"subscription_cancel_at_period_end": "boolean",
		"subscription_cancel_now":           "boolean",
		"subscription_default_tax_rates":    "array",
		"subscription_proration_behavior":   "string",
		"subscription_proration_date":       "integer",
		"subscription_resume_at":            "string",
		"subscription_start_date":           "integer",
		"subscription_trial_end":            "string",
		"subscription_trial_from_plan":      "boolean",
	}, &Config)
	resource.NewOperationCmd(rInvoicesCmd.Cmd, "update", "/v1/invoices/{invoice}", http.MethodPost, map[string]string{
		"account_tax_ids":                       "array",
		"application_fee_amount":                "integer",
		"auto_advance":                          "boolean",
		"automatic_tax.enabled":                 "boolean",
		"automatic_tax.liability.account":       "string",
		"automatic_tax.liability.type":          "string",
		"collection_method":                     "string",
		"days_until_due":                        "integer",
		"default_payment_method":                "string",
		"default_source":                        "string",
		"default_tax_rates":                     "array",
		"description":                           "string",
		"due_date":                              "integer",
		"effective_at":                          "integer",
		"footer":                                "string",
		"issuer.account":                        "string",
		"issuer.type":                           "string",
		"number":                                "string",
		"on_behalf_of":                          "string",
		"payment_settings.default_mandate":      "string",
		"payment_settings.payment_method_types": "array",
		"rendering.amount_tax_display":          "string",
		"rendering.pdf.page_size":               "string",
		"statement_descriptor":                  "string",
	}, &Config)
	resource.NewOperationCmd(rInvoicesCmd.Cmd, "void_invoice", "/v1/invoices/{invoice}/void", http.MethodPost, map[string]string{}, &Config)
	resource.NewOperationCmd(rLineItemsCmd.Cmd, "list", "/v1/invoices/{invoice}/lines", http.MethodGet, map[string]string{
		"ending_before":  "string",
		"limit":          "integer",
		"starting_after": "string",
	}, &Config)
	resource.NewOperationCmd(rLineItemsCmd.Cmd, "update", "/v1/invoices/{invoice}/lines/{line_item_id}", http.MethodPost, map[string]string{
		"amount":                              "integer",
		"description":                         "string",
		"discountable":                        "boolean",
		"period.end":                          "integer",
		"period.start":                        "integer",
		"price":                               "string",
		"price_data.currency":                 "string",
		"price_data.product":                  "string",
		"price_data.product_data.description": "string",
		"price_data.product_data.images":      "array",
		"price_data.product_data.name":        "string",
		"price_data.product_data.tax_code":    "string",
		"price_data.tax_behavior":             "string",
		"price_data.unit_amount":              "integer",
		"price_data.unit_amount_decimal":      "string",
		"quantity":                            "integer",
		"tax_rates":                           "array",
	}, &Config)
	resource.NewOperationCmd(rLoginLinksCmd.Cmd, "create", "/v1/accounts/{account}/login_links", http.MethodPost, map[string]string{}, &Config)
	resource.NewOperationCmd(rMandatesCmd.Cmd, "retrieve", "/v1/mandates/{mandate}", http.MethodGet, map[string]string{}, &Config)
	resource.NewOperationCmd(rPaymentIntentsCmd.Cmd, "apply_customer_balance", "/v1/payment_intents/{intent}/apply_customer_balance", http.MethodPost, map[string]string{
		"amount":   "integer",
		"currency": "string",
	}, &Config)
	resource.NewOperationCmd(rPaymentIntentsCmd.Cmd, "cancel", "/v1/payment_intents/{intent}/cancel", http.MethodPost, map[string]string{
		"cancellation_reason": "string",
	}, &Config)
	resource.NewOperationCmd(rPaymentIntentsCmd.Cmd, "capture", "/v1/payment_intents/{intent}/capture", http.MethodPost, map[string]string{
		"amount_to_capture":           "integer",
		"application_fee_amount":      "integer",
		"final_capture":               "boolean",
		"statement_descriptor":        "string",
		"statement_descriptor_suffix": "string",
		"transfer_data.amount":        "integer",
	}, &Config)
	resource.NewOperationCmd(rPaymentIntentsCmd.Cmd, "confirm", "/v1/payment_intents/{intent}/confirm", http.MethodPost, map[string]string{
		"capture_method":           "string",
		"confirmation_token":       "string",
		"error_on_requires_action": "boolean",
		"mandate":                  "string",
		"off_session":              "boolean",
		"payment_method":           "string",
		"payment_method_data.acss_debit.account_number":                     "string",
		"payment_method_data.acss_debit.institution_number":                 "string",
		"payment_method_data.acss_debit.transit_number":                     "string",
		"payment_method_data.allow_redisplay":                               "string",
		"payment_method_data.au_becs_debit.account_number":                  "string",
		"payment_method_data.au_becs_debit.bsb_number":                      "string",
		"payment_method_data.bacs_debit.account_number":                     "string",
		"payment_method_data.bacs_debit.sort_code":                          "string",
		"payment_method_data.billing_details.email":                         "string",
		"payment_method_data.billing_details.name":                          "string",
		"payment_method_data.billing_details.phone":                         "string",
		"payment_method_data.boleto.tax_id":                                 "string",
		"payment_method_data.eps.bank":                                      "string",
		"payment_method_data.fpx.account_holder_type":                       "string",
		"payment_method_data.fpx.bank":                                      "string",
		"payment_method_data.ideal.bank":                                    "string",
		"payment_method_data.klarna.dob.day":                                "integer",
		"payment_method_data.klarna.dob.month":                              "integer",
		"payment_method_data.klarna.dob.year":                               "integer",
		"payment_method_data.p24.bank":                                      "string",
		"payment_method_data.radar_options.session":                         "string",
		"payment_method_data.sepa_debit.iban":                               "string",
		"payment_method_data.sofort.country":                                "string",
		"payment_method_data.type":                                          "string",
		"payment_method_data.us_bank_account.account_holder_type":           "string",
		"payment_method_data.us_bank_account.account_number":                "string",
		"payment_method_data.us_bank_account.account_type":                  "string",
		"payment_method_data.us_bank_account.financial_connections_account": "string",
		"payment_method_data.us_bank_account.routing_number":                "string",
		"payment_method_types":                                              "array",
		"radar_options.session":                                             "string",
		"receipt_email":                                                     "string",
		"return_url":                                                        "string",
		"setup_future_usage":                                                "string",
		"use_stripe_sdk":                                                    "boolean",
	}, &Config)
	resource.NewOperationCmd(rPaymentIntentsCmd.Cmd, "create", "/v1/payment_intents", http.MethodPost, map[string]string{
		"amount":                 "integer",
		"application_fee_amount": "integer",
		"automatic_payment_methods.allow_redirects": "string",
		"automatic_payment_methods.enabled":         "boolean",
		"capture_method":                            "string",
		"confirm":                                   "boolean",
		"confirmation_method":                       "string",
		"confirmation_token":                        "string",
		"currency":                                  "string",
		"customer":                                  "string",
		"description":                               "string",
		"error_on_requires_action":                  "boolean",
		"mandate":                                   "string",
		"off_session":                               "boolean",
		"on_behalf_of":                              "string",
		"payment_method":                            "string",
		"payment_method_configuration":              "string",
		"payment_method_data.acss_debit.account_number":                     "string",
		"payment_method_data.acss_debit.institution_number":                 "string",
		"payment_method_data.acss_debit.transit_number":                     "string",
		"payment_method_data.allow_redisplay":                               "string",
		"payment_method_data.au_becs_debit.account_number":                  "string",
		"payment_method_data.au_becs_debit.bsb_number":                      "string",
		"payment_method_data.bacs_debit.account_number":                     "string",
		"payment_method_data.bacs_debit.sort_code":                          "string",
		"payment_method_data.billing_details.email":                         "string",
		"payment_method_data.billing_details.name":                          "string",
		"payment_method_data.billing_details.phone":                         "string",
		"payment_method_data.boleto.tax_id":                                 "string",
		"payment_method_data.eps.bank":                                      "string",
		"payment_method_data.fpx.account_holder_type":                       "string",
		"payment_method_data.fpx.bank":                                      "string",
		"payment_method_data.ideal.bank":                                    "string",
		"payment_method_data.klarna.dob.day":                                "integer",
		"payment_method_data.klarna.dob.month":                              "integer",
		"payment_method_data.klarna.dob.year":                               "integer",
		"payment_method_data.p24.bank":                                      "string",
		"payment_method_data.radar_options.session":                         "string",
		"payment_method_data.sepa_debit.iban":                               "string",
		"payment_method_data.sofort.country":                                "string",
		"payment_method_data.type":                                          "string",
		"payment_method_data.us_bank_account.account_holder_type":           "string",
		"payment_method_data.us_bank_account.account_number":                "string",
		"payment_method_data.us_bank_account.account_type":                  "string",
		"payment_method_data.us_bank_account.financial_connections_account": "string",
		"payment_method_data.us_bank_account.routing_number":                "string",
		"payment_method_types":                                              "array",
		"radar_options.session":                                             "string",
		"receipt_email":                                                     "string",
		"return_url":                                                        "string",
		"setup_future_usage":                                                "string",
		"shipping.address.city":                                             "string",
		"shipping.address.country":                                          "string",
		"shipping.address.line1":                                            "string",
		"shipping.address.line2":                                            "string",
		"shipping.address.postal_code":                                      "string",
		"shipping.address.state":                                            "string",
		"shipping.carrier":                                                  "string",
		"shipping.name":                                                     "string",
		"shipping.phone":                                                    "string",
		"shipping.tracking_number":                                          "string",
		"statement_descriptor":                                              "string",
		"statement_descriptor_suffix":                                       "string",
		"transfer_data.amount":                                              "integer",
		"transfer_data.destination":                                         "string",
		"transfer_group":                                                    "string",
		"use_stripe_sdk":                                                    "boolean",
	}, &Config)
	resource.NewOperationCmd(rPaymentIntentsCmd.Cmd, "increment_authorization", "/v1/payment_intents/{intent}/increment_authorization", http.MethodPost, map[string]string{
		"amount":                 "integer",
		"application_fee_amount": "integer",
		"description":            "string",
		"statement_descriptor":   "string",
		"transfer_data.amount":   "integer",
	}, &Config)
	resource.NewOperationCmd(rPaymentIntentsCmd.Cmd, "list", "/v1/payment_intents", http.MethodGet, map[string]string{
		"created":        "integer",
		"customer":       "string",
		"ending_before":  "string",
		"limit":          "integer",
		"starting_after": "string",
	}, &Config)
	resource.NewOperationCmd(rPaymentIntentsCmd.Cmd, "retrieve", "/v1/payment_intents/{intent}", http.MethodGet, map[string]string{
		"client_secret": "string",
	}, &Config)
	resource.NewOperationCmd(rPaymentIntentsCmd.Cmd, "search", "/v1/payment_intents/search", http.MethodGet, map[string]string{
		"limit": "integer",
		"page":  "string",
		"query": "string",
	}, &Config)
	resource.NewOperationCmd(rPaymentIntentsCmd.Cmd, "update", "/v1/payment_intents/{intent}", http.MethodPost, map[string]string{
		"amount":                       "integer",
		"application_fee_amount":       "integer",
		"capture_method":               "string",
		"currency":                     "string",
		"customer":                     "string",
		"description":                  "string",
		"payment_method":               "string",
		"payment_method_configuration": "string",
		"payment_method_data.acss_debit.account_number":                     "string",
		"payment_method_data.acss_debit.institution_number":                 "string",
		"payment_method_data.acss_debit.transit_number":                     "string",
		"payment_method_data.allow_redisplay":                               "string",
		"payment_method_data.au_becs_debit.account_number":                  "string",
		"payment_method_data.au_becs_debit.bsb_number":                      "string",
		"payment_method_data.bacs_debit.account_number":                     "string",
		"payment_method_data.bacs_debit.sort_code":                          "string",
		"payment_method_data.billing_details.email":                         "string",
		"payment_method_data.billing_details.name":                          "string",
		"payment_method_data.billing_details.phone":                         "string",
		"payment_method_data.boleto.tax_id":                                 "string",
		"payment_method_data.eps.bank":                                      "string",
		"payment_method_data.fpx.account_holder_type":                       "string",
		"payment_method_data.fpx.bank":                                      "string",
		"payment_method_data.ideal.bank":                                    "string",
		"payment_method_data.klarna.dob.day":                                "integer",
		"payment_method_data.klarna.dob.month":                              "integer",
		"payment_method_data.klarna.dob.year":                               "integer",
		"payment_method_data.p24.bank":                                      "string",
		"payment_method_data.radar_options.session":                         "string",
		"payment_method_data.sepa_debit.iban":                               "string",
		"payment_method_data.sofort.country":                                "string",
		"payment_method_data.type":                                          "string",
		"payment_method_data.us_bank_account.account_holder_type":           "string",
		"payment_method_data.us_bank_account.account_number":                "string",
		"payment_method_data.us_bank_account.account_type":                  "string",
		"payment_method_data.us_bank_account.financial_connections_account": "string",
		"payment_method_data.us_bank_account.routing_number":                "string",
		"payment_method_types":                                              "array",
		"receipt_email":                                                     "string",
		"setup_future_usage":                                                "string",
		"statement_descriptor":                                              "string",
		"statement_descriptor_suffix":                                       "string",
		"transfer_data.amount":                                              "integer",
		"transfer_group":                                                    "string",
	}, &Config)
	resource.NewOperationCmd(rPaymentIntentsCmd.Cmd, "verify_microdeposits", "/v1/payment_intents/{intent}/verify_microdeposits", http.MethodPost, map[string]string{
		"amounts":         "array",
		"descriptor_code": "string",
	}, &Config)
	resource.NewOperationCmd(rPaymentLinksCmd.Cmd, "create", "/v1/payment_links", http.MethodPost, map[string]string{
		"after_completion.hosted_confirmation.custom_message":                  "string",
		"after_completion.redirect.url":                                        "string",
		"after_completion.type":                                                "string",
		"allow_promotion_codes":                                                "boolean",
		"application_fee_amount":                                               "integer",
		"application_fee_percent":                                              "number",
		"automatic_tax.enabled":                                                "boolean",
		"automatic_tax.liability.account":                                      "string",
		"automatic_tax.liability.type":                                         "string",
		"billing_address_collection":                                           "string",
		"consent_collection.payment_method_reuse_agreement.position":           "string",
		"consent_collection.promotions":                                        "string",
		"consent_collection.terms_of_service":                                  "string",
		"currency":                                                             "string",
		"customer_creation":                                                    "string",
		"inactive_message":                                                     "string",
		"invoice_creation.enabled":                                             "boolean",
		"invoice_creation.invoice_data.account_tax_ids":                        "array",
		"invoice_creation.invoice_data.description":                            "string",
		"invoice_creation.invoice_data.footer":                                 "string",
		"invoice_creation.invoice_data.issuer.account":                         "string",
		"invoice_creation.invoice_data.issuer.type":                            "string",
		"on_behalf_of":                                                         "string",
		"payment_intent_data.capture_method":                                   "string",
		"payment_intent_data.description":                                      "string",
		"payment_intent_data.setup_future_usage":                               "string",
		"payment_intent_data.statement_descriptor":                             "string",
		"payment_intent_data.statement_descriptor_suffix":                      "string",
		"payment_intent_data.transfer_group":                                   "string",
		"payment_method_collection":                                            "string",
		"payment_method_types":                                                 "array",
		"phone_number_collection.enabled":                                      "boolean",
		"restrictions.completed_sessions.limit":                                "integer",
		"shipping_address_collection.allowed_countries":                        "array",
		"submit_type":                                                          "string",
		"subscription_data.description":                                        "string",
		"subscription_data.invoice_settings.issuer.account":                    "string",
		"subscription_data.invoice_settings.issuer.type":                       "string",
		"subscription_data.trial_period_days":                                  "integer",
		"subscription_data.trial_settings.end_behavior.missing_payment_method": "string",
		"tax_id_collection.enabled":                                            "boolean",
		"transfer_data.amount":                                                 "integer",
		"transfer_data.destination":                                            "string",
	}, &Config)
	resource.NewOperationCmd(rPaymentLinksCmd.Cmd, "list", "/v1/payment_links", http.MethodGet, map[string]string{
		"active":         "boolean",
		"ending_before":  "string",
		"limit":          "integer",
		"starting_after": "string",
	}, &Config)
	resource.NewOperationCmd(rPaymentLinksCmd.Cmd, "list_line_items", "/v1/payment_links/{payment_link}/line_items", http.MethodGet, map[string]string{
		"ending_before":  "string",
		"limit":          "integer",
		"starting_after": "string",
	}, &Config)
	resource.NewOperationCmd(rPaymentLinksCmd.Cmd, "retrieve", "/v1/payment_links/{payment_link}", http.MethodGet, map[string]string{}, &Config)
	resource.NewOperationCmd(rPaymentLinksCmd.Cmd, "update", "/v1/payment_links/{payment_link}", http.MethodPost, map[string]string{
		"active": "boolean",
		"after_completion.hosted_confirmation.custom_message": "string",
		"after_completion.redirect.url":                       "string",
		"after_completion.type":                               "string",
		"allow_promotion_codes":                               "boolean",
		"automatic_tax.enabled":                               "boolean",
		"automatic_tax.liability.account":                     "string",
		"automatic_tax.liability.type":                        "string",
		"billing_address_collection":                          "string",
		"customer_creation":                                   "string",
		"inactive_message":                                    "string",
		"invoice_creation.enabled":                            "boolean",
		"invoice_creation.invoice_data.account_tax_ids":       "array",
		"invoice_creation.invoice_data.description":           "string",
		"invoice_creation.invoice_data.footer":                "string",
		"invoice_creation.invoice_data.issuer.account":        "string",
		"invoice_creation.invoice_data.issuer.type":           "string",
		"payment_intent_data.description":                     "string",
		"payment_intent_data.statement_descriptor":            "string",
		"payment_intent_data.statement_descriptor_suffix":     "string",
		"payment_intent_data.transfer_group":                  "string",
		"payment_method_collection":                           "string",
		"payment_method_types":                                "array",
		"subscription_data.invoice_settings.issuer.account":   "string",
		"subscription_data.invoice_settings.issuer.type":      "string",
	}, &Config)
	resource.NewOperationCmd(rPaymentMethodConfigurationsCmd.Cmd, "create", "/v1/payment_method_configurations", http.MethodPost, map[string]string{
		"acss_debit.display_preference.preference":        "string",
		"affirm.display_preference.preference":            "string",
		"afterpay_clearpay.display_preference.preference": "string",
		"alipay.display_preference.preference":            "string",
		"amazon_pay.display_preference.preference":        "string",
		"apple_pay.display_preference.preference":         "string",
		"apple_pay_later.display_preference.preference":   "string",
		"au_becs_debit.display_preference.preference":     "string",
		"bacs_debit.display_preference.preference":        "string",
		"bancontact.display_preference.preference":        "string",
		"blik.display_preference.preference":              "string",
		"boleto.display_preference.preference":            "string",
		"card.display_preference.preference":              "string",
		"cartes_bancaires.display_preference.preference":  "string",
		"cashapp.display_preference.preference":           "string",
		"customer_balance.display_preference.preference":  "string",
		"eps.display_preference.preference":               "string",
		"fpx.display_preference.preference":               "string",
		"giropay.display_preference.preference":           "string",
		"google_pay.display_preference.preference":        "string",
		"grabpay.display_preference.preference":           "string",
		"ideal.display_preference.preference":             "string",
		"jcb.display_preference.preference":               "string",
		"klarna.display_preference.preference":            "string",
		"konbini.display_preference.preference":           "string",
		"link.display_preference.preference":              "string",
		"mobilepay.display_preference.preference":         "string",
		"name":                                          "string",
		"oxxo.display_preference.preference":            "string",
		"p24.display_preference.preference":             "string",
		"parent":                                        "string",
		"paynow.display_preference.preference":          "string",
		"paypal.display_preference.preference":          "string",
		"promptpay.display_preference.preference":       "string",
		"revolut_pay.display_preference.preference":     "string",
		"sepa_debit.display_preference.preference":      "string",
		"sofort.display_preference.preference":          "string",
		"swish.display_preference.preference":           "string",
		"us_bank_account.display_preference.preference": "string",
		"wechat_pay.display_preference.preference":      "string",
		"zip.display_preference.preference":             "string",
	}, &Config)
	resource.NewOperationCmd(rPaymentMethodConfigurationsCmd.Cmd, "list", "/v1/payment_method_configurations", http.MethodGet, map[string]string{
		"application":    "string",
		"ending_before":  "string",
		"limit":          "integer",
		"starting_after": "string",
	}, &Config)
	resource.NewOperationCmd(rPaymentMethodConfigurationsCmd.Cmd, "retrieve", "/v1/payment_method_configurations/{configuration}", http.MethodGet, map[string]string{}, &Config)
	resource.NewOperationCmd(rPaymentMethodConfigurationsCmd.Cmd, "update", "/v1/payment_method_configurations/{configuration}", http.MethodPost, map[string]string{
		"acss_debit.display_preference.preference": "string",
		"active":                               "boolean",
		"affirm.display_preference.preference": "string",
		"afterpay_clearpay.display_preference.preference": "string",
		"alipay.display_preference.preference":            "string",
		"amazon_pay.display_preference.preference":        "string",
		"apple_pay.display_preference.preference":         "string",
		"apple_pay_later.display_preference.preference":   "string",
		"au_becs_debit.display_preference.preference":     "string",
		"bacs_debit.display_preference.preference":        "string",
		"bancontact.display_preference.preference":        "string",
		"blik.display_preference.preference":              "string",
		"boleto.display_preference.preference":            "string",
		"card.display_preference.preference":              "string",
		"cartes_bancaires.display_preference.preference":  "string",
		"cashapp.display_preference.preference":           "string",
		"customer_balance.display_preference.preference":  "string",
		"eps.display_preference.preference":               "string",
		"fpx.display_preference.preference":               "string",
		"giropay.display_preference.preference":           "string",
		"google_pay.display_preference.preference":        "string",
		"grabpay.display_preference.preference":           "string",
		"ideal.display_preference.preference":             "string",
		"jcb.display_preference.preference":               "string",
		"klarna.display_preference.preference":            "string",
		"konbini.display_preference.preference":           "string",
		"link.display_preference.preference":              "string",
		"mobilepay.display_preference.preference":         "string",
		"name":                                          "string",
		"oxxo.display_preference.preference":            "string",
		"p24.display_preference.preference":             "string",
		"paynow.display_preference.preference":          "string",
		"paypal.display_preference.preference":          "string",
		"promptpay.display_preference.preference":       "string",
		"revolut_pay.display_preference.preference":     "string",
		"sepa_debit.display_preference.preference":      "string",
		"sofort.display_preference.preference":          "string",
		"swish.display_preference.preference":           "string",
		"us_bank_account.display_preference.preference": "string",
		"wechat_pay.display_preference.preference":      "string",
		"zip.display_preference.preference":             "string",
	}, &Config)
	resource.NewOperationCmd(rPaymentMethodDomainsCmd.Cmd, "create", "/v1/payment_method_domains", http.MethodPost, map[string]string{
		"domain_name": "string",
		"enabled":     "boolean",
	}, &Config)
	resource.NewOperationCmd(rPaymentMethodDomainsCmd.Cmd, "list", "/v1/payment_method_domains", http.MethodGet, map[string]string{
		"domain_name":    "string",
		"enabled":        "boolean",
		"ending_before":  "string",
		"limit":          "integer",
		"starting_after": "string",
	}, &Config)
	resource.NewOperationCmd(rPaymentMethodDomainsCmd.Cmd, "retrieve", "/v1/payment_method_domains/{payment_method_domain}", http.MethodGet, map[string]string{}, &Config)
	resource.NewOperationCmd(rPaymentMethodDomainsCmd.Cmd, "update", "/v1/payment_method_domains/{payment_method_domain}", http.MethodPost, map[string]string{
		"enabled": "boolean",
	}, &Config)
	resource.NewOperationCmd(rPaymentMethodDomainsCmd.Cmd, "validate", "/v1/payment_method_domains/{payment_method_domain}/validate", http.MethodPost, map[string]string{}, &Config)
	resource.NewOperationCmd(rPaymentMethodsCmd.Cmd, "attach", "/v1/payment_methods/{payment_method}/attach", http.MethodPost, map[string]string{
		"customer": "string",
	}, &Config)
	resource.NewOperationCmd(rPaymentMethodsCmd.Cmd, "create", "/v1/payment_methods", http.MethodPost, map[string]string{
		"acss_debit.account_number":                     "string",
		"acss_debit.institution_number":                 "string",
		"acss_debit.transit_number":                     "string",
		"allow_redisplay":                               "string",
		"au_becs_debit.account_number":                  "string",
		"au_becs_debit.bsb_number":                      "string",
		"bacs_debit.account_number":                     "string",
		"bacs_debit.sort_code":                          "string",
		"billing_details.email":                         "string",
		"billing_details.name":                          "string",
		"billing_details.phone":                         "string",
		"boleto.tax_id":                                 "string",
		"customer":                                      "string",
		"eps.bank":                                      "string",
		"fpx.account_holder_type":                       "string",
		"fpx.bank":                                      "string",
		"ideal.bank":                                    "string",
		"klarna.dob.day":                                "integer",
		"klarna.dob.month":                              "integer",
		"klarna.dob.year":                               "integer",
		"p24.bank":                                      "string",
		"payment_method":                                "string",
		"radar_options.session":                         "string",
		"sepa_debit.iban":                               "string",
		"sofort.country":                                "string",
		"type":                                          "string",
		"us_bank_account.account_holder_type":           "string",
		"us_bank_account.account_number":                "string",
		"us_bank_account.account_type":                  "string",
		"us_bank_account.financial_connections_account": "string",
		"us_bank_account.routing_number":                "string",
	}, &Config)
	resource.NewOperationCmd(rPaymentMethodsCmd.Cmd, "detach", "/v1/payment_methods/{payment_method}/detach", http.MethodPost, map[string]string{}, &Config)
	resource.NewOperationCmd(rPaymentMethodsCmd.Cmd, "list", "/v1/payment_methods", http.MethodGet, map[string]string{
		"customer":       "string",
		"ending_before":  "string",
		"limit":          "integer",
		"starting_after": "string",
		"type":           "string",
	}, &Config)
	resource.NewOperationCmd(rPaymentMethodsCmd.Cmd, "retrieve", "/v1/payment_methods/{payment_method}", http.MethodGet, map[string]string{}, &Config)
	resource.NewOperationCmd(rPaymentMethodsCmd.Cmd, "update", "/v1/payment_methods/{payment_method}", http.MethodPost, map[string]string{
		"allow_redisplay":                     "string",
		"billing_details.email":               "string",
		"billing_details.name":                "string",
		"billing_details.phone":               "string",
		"card.exp_month":                      "integer",
		"card.exp_year":                       "integer",
		"card.networks.preferred":             "string",
		"us_bank_account.account_holder_type": "string",
		"us_bank_account.account_type":        "string",
	}, &Config)
	resource.NewOperationCmd(rPaymentSourcesCmd.Cmd, "create", "/v1/customers/{customer}/sources", http.MethodPost, map[string]string{
		"source":   "string",
		"validate": "boolean",
	}, &Config)
	resource.NewOperationCmd(rPaymentSourcesCmd.Cmd, "list", "/v1/customers/{customer}/sources", http.MethodGet, map[string]string{
		"ending_before":  "string",
		"limit":          "integer",
		"object":         "string",
		"starting_after": "string",
	}, &Config)
	resource.NewOperationCmd(rPaymentSourcesCmd.Cmd, "retrieve", "/v1/customers/{customer}/sources/{id}", http.MethodGet, map[string]string{}, &Config)
	resource.NewOperationCmd(rPayoutsCmd.Cmd, "cancel", "/v1/payouts/{payout}/cancel", http.MethodPost, map[string]string{}, &Config)
	resource.NewOperationCmd(rPayoutsCmd.Cmd, "create", "/v1/payouts", http.MethodPost, map[string]string{
		"amount":               "integer",
		"currency":             "string",
		"description":          "string",
		"destination":          "string",
		"method":               "string",
		"source_type":          "string",
		"statement_descriptor": "string",
	}, &Config)
	resource.NewOperationCmd(rPayoutsCmd.Cmd, "list", "/v1/payouts", http.MethodGet, map[string]string{
		"arrival_date":   "integer",
		"created":        "integer",
		"destination":    "string",
		"ending_before":  "string",
		"limit":          "integer",
		"starting_after": "string",
		"status":         "string",
	}, &Config)
	resource.NewOperationCmd(rPayoutsCmd.Cmd, "retrieve", "/v1/payouts/{payout}", http.MethodGet, map[string]string{}, &Config)
	resource.NewOperationCmd(rPayoutsCmd.Cmd, "reverse", "/v1/payouts/{payout}/reverse", http.MethodPost, map[string]string{}, &Config)
	resource.NewOperationCmd(rPayoutsCmd.Cmd, "update", "/v1/payouts/{payout}", http.MethodPost, map[string]string{}, &Config)
	resource.NewOperationCmd(rPersonsCmd.Cmd, "create", "/v1/accounts/{account}/persons", http.MethodPost, map[string]string{
		"additional_tos_acceptances.account.date":       "integer",
		"additional_tos_acceptances.account.ip":         "string",
		"additional_tos_acceptances.account.user_agent": "string",
		"address.city":                           "string",
		"address.country":                        "string",
		"address.line1":                          "string",
		"address.line2":                          "string",
		"address.postal_code":                    "string",
		"address.state":                          "string",
		"address_kana.city":                      "string",
		"address_kana.country":                   "string",
		"address_kana.line1":                     "string",
		"address_kana.line2":                     "string",
		"address_kana.postal_code":               "string",
		"address_kana.state":                     "string",
		"address_kana.town":                      "string",
		"address_kanji.city":                     "string",
		"address_kanji.country":                  "string",
		"address_kanji.line1":                    "string",
		"address_kanji.line2":                    "string",
		"address_kanji.postal_code":              "string",
		"address_kanji.state":                    "string",
		"address_kanji.town":                     "string",
		"documents.company_authorization.files":  "array",
		"documents.passport.files":               "array",
		"documents.visa.files":                   "array",
		"email":                                  "string",
		"first_name":                             "string",
		"first_name_kana":                        "string",
		"first_name_kanji":                       "string",
		"full_name_aliases":                      "array",
		"gender":                                 "string",
		"id_number":                              "string",
		"id_number_secondary":                    "string",
		"last_name":                              "string",
		"last_name_kana":                         "string",
		"last_name_kanji":                        "string",
		"maiden_name":                            "string",
		"nationality":                            "string",
		"person_token":                           "string",
		"phone":                                  "string",
		"political_exposure":                     "string",
		"registered_address.city":                "string",
		"registered_address.country":             "string",
		"registered_address.line1":               "string",
		"registered_address.line2":               "string",
		"registered_address.postal_code":         "string",
		"registered_address.state":               "string",
		"relationship.director":                  "boolean",
		"relationship.executive":                 "boolean",
		"relationship.legal_guardian":            "boolean",
		"relationship.owner":                     "boolean",
		"relationship.percent_ownership":         "number",
		"relationship.representative":            "boolean",
		"relationship.title":                     "string",
		"ssn_last_4":                             "string",
		"verification.additional_document.back":  "string",
		"verification.additional_document.front": "string",
		"verification.document.back":             "string",
		"verification.document.front":            "string",
	}, &Config)
	resource.NewOperationCmd(rPersonsCmd.Cmd, "delete", "/v1/accounts/{account}/persons/{person}", http.MethodDelete, map[string]string{}, &Config)
	resource.NewOperationCmd(rPersonsCmd.Cmd, "list", "/v1/accounts/{account}/persons", http.MethodGet, map[string]string{
		"ending_before":  "string",
		"limit":          "integer",
		"starting_after": "string",
	}, &Config)
	resource.NewOperationCmd(rPersonsCmd.Cmd, "retrieve", "/v1/accounts/{account}/persons/{person}", http.MethodGet, map[string]string{}, &Config)
	resource.NewOperationCmd(rPersonsCmd.Cmd, "update", "/v1/accounts/{account}/persons/{person}", http.MethodPost, map[string]string{
		"additional_tos_acceptances.account.date":       "integer",
		"additional_tos_acceptances.account.ip":         "string",
		"additional_tos_acceptances.account.user_agent": "string",
		"address.city":                           "string",
		"address.country":                        "string",
		"address.line1":                          "string",
		"address.line2":                          "string",
		"address.postal_code":                    "string",
		"address.state":                          "string",
		"address_kana.city":                      "string",
		"address_kana.country":                   "string",
		"address_kana.line1":                     "string",
		"address_kana.line2":                     "string",
		"address_kana.postal_code":               "string",
		"address_kana.state":                     "string",
		"address_kana.town":                      "string",
		"address_kanji.city":                     "string",
		"address_kanji.country":                  "string",
		"address_kanji.line1":                    "string",
		"address_kanji.line2":                    "string",
		"address_kanji.postal_code":              "string",
		"address_kanji.state":                    "string",
		"address_kanji.town":                     "string",
		"documents.company_authorization.files":  "array",
		"documents.passport.files":               "array",
		"documents.visa.files":                   "array",
		"email":                                  "string",
		"first_name":                             "string",
		"first_name_kana":                        "string",
		"first_name_kanji":                       "string",
		"full_name_aliases":                      "array",
		"gender":                                 "string",
		"id_number":                              "string",
		"id_number_secondary":                    "string",
		"last_name":                              "string",
		"last_name_kana":                         "string",
		"last_name_kanji":                        "string",
		"maiden_name":                            "string",
		"nationality":                            "string",
		"person_token":                           "string",
		"phone":                                  "string",
		"political_exposure":                     "string",
		"registered_address.city":                "string",
		"registered_address.country":             "string",
		"registered_address.line1":               "string",
		"registered_address.line2":               "string",
		"registered_address.postal_code":         "string",
		"registered_address.state":               "string",
		"relationship.director":                  "boolean",
		"relationship.executive":                 "boolean",
		"relationship.legal_guardian":            "boolean",
		"relationship.owner":                     "boolean",
		"relationship.percent_ownership":         "number",
		"relationship.representative":            "boolean",
		"relationship.title":                     "string",
		"ssn_last_4":                             "string",
		"verification.additional_document.back":  "string",
		"verification.additional_document.front": "string",
		"verification.document.back":             "string",
		"verification.document.front":            "string",
	}, &Config)
	resource.NewOperationCmd(rPlansCmd.Cmd, "create", "/v1/plans", http.MethodPost, map[string]string{
		"active":                    "boolean",
		"aggregate_usage":           "string",
		"amount":                    "integer",
		"amount_decimal":            "string",
		"billing_scheme":            "string",
		"currency":                  "string",
		"id":                        "string",
		"interval":                  "string",
		"interval_count":            "integer",
		"meter":                     "string",
		"nickname":                  "string",
		"product":                   "string",
		"tiers_mode":                "string",
		"transform_usage.divide_by": "integer",
		"transform_usage.round":     "string",
		"trial_period_days":         "integer",
		"usage_type":                "string",
	}, &Config)
	resource.NewOperationCmd(rPlansCmd.Cmd, "delete", "/v1/plans/{plan}", http.MethodDelete, map[string]string{}, &Config)
	resource.NewOperationCmd(rPlansCmd.Cmd, "list", "/v1/plans", http.MethodGet, map[string]string{
		"active":         "boolean",
		"created":        "integer",
		"ending_before":  "string",
		"limit":          "integer",
		"product":        "string",
		"starting_after": "string",
	}, &Config)
	resource.NewOperationCmd(rPlansCmd.Cmd, "retrieve", "/v1/plans/{plan}", http.MethodGet, map[string]string{}, &Config)
	resource.NewOperationCmd(rPlansCmd.Cmd, "update", "/v1/plans/{plan}", http.MethodPost, map[string]string{
		"active":            "boolean",
		"nickname":          "string",
		"product":           "string",
		"trial_period_days": "integer",
	}, &Config)
	resource.NewOperationCmd(rPricesCmd.Cmd, "create", "/v1/prices", http.MethodPost, map[string]string{
		"active":                            "boolean",
		"billing_scheme":                    "string",
		"currency":                          "string",
		"custom_unit_amount.enabled":        "boolean",
		"custom_unit_amount.maximum":        "integer",
		"custom_unit_amount.minimum":        "integer",
		"custom_unit_amount.preset":         "integer",
		"lookup_key":                        "string",
		"nickname":                          "string",
		"product":                           "string",
		"product_data.active":               "boolean",
		"product_data.id":                   "string",
		"product_data.name":                 "string",
		"product_data.statement_descriptor": "string",
		"product_data.tax_code":             "string",
		"product_data.unit_label":           "string",
		"recurring.aggregate_usage":         "string",
		"recurring.interval":                "string",
		"recurring.interval_count":          "integer",
		"recurring.meter":                   "string",
		"recurring.trial_period_days":       "integer",
		"recurring.usage_type":              "string",
		"tax_behavior":                      "string",
		"tiers_mode":                        "string",
		"transfer_lookup_key":               "boolean",
		"transform_quantity.divide_by":      "integer",
		"transform_quantity.round":          "string",
		"unit_amount":                       "integer",
		"unit_amount_decimal":               "string",
	}, &Config)
	resource.NewOperationCmd(rPricesCmd.Cmd, "list", "/v1/prices", http.MethodGet, map[string]string{
		"active":         "boolean",
		"created":        "integer",
		"currency":       "string",
		"ending_before":  "string",
		"limit":          "integer",
		"lookup_keys":    "array",
		"product":        "string",
		"starting_after": "string",
		"type":           "string",
	}, &Config)
	resource.NewOperationCmd(rPricesCmd.Cmd, "retrieve", "/v1/prices/{price}", http.MethodGet, map[string]string{}, &Config)
	resource.NewOperationCmd(rPricesCmd.Cmd, "search", "/v1/prices/search", http.MethodGet, map[string]string{
		"limit": "integer",
		"page":  "string",
		"query": "string",
	}, &Config)
	resource.NewOperationCmd(rPricesCmd.Cmd, "update", "/v1/prices/{price}", http.MethodPost, map[string]string{
		"active":              "boolean",
		"lookup_key":          "string",
		"nickname":            "string",
		"tax_behavior":        "string",
		"transfer_lookup_key": "boolean",
	}, &Config)
	resource.NewOperationCmd(rProductFeaturesCmd.Cmd, "create", "/v1/products/{product}/features", http.MethodPost, map[string]string{
		"entitlement_feature": "string",
	}, &Config)
	resource.NewOperationCmd(rProductFeaturesCmd.Cmd, "delete", "/v1/products/{product}/features/{id}", http.MethodDelete, map[string]string{}, &Config)
	resource.NewOperationCmd(rProductFeaturesCmd.Cmd, "list", "/v1/products/{product}/features", http.MethodGet, map[string]string{
		"ending_before":  "string",
		"limit":          "integer",
		"starting_after": "string",
	}, &Config)
	resource.NewOperationCmd(rProductFeaturesCmd.Cmd, "retrieve", "/v1/products/{product}/features/{id}", http.MethodGet, map[string]string{}, &Config)
	resource.NewOperationCmd(rProductsCmd.Cmd, "create", "/v1/products", http.MethodPost, map[string]string{
		"active":                                      "boolean",
		"default_price_data.currency":                 "string",
		"default_price_data.recurring.interval":       "string",
		"default_price_data.recurring.interval_count": "integer",
		"default_price_data.tax_behavior":             "string",
		"default_price_data.unit_amount":              "integer",
		"default_price_data.unit_amount_decimal":      "string",
		"description":                                 "string",
		"id":                                          "string",
		"images":                                      "array",
		"name":                                        "string",
		"package_dimensions.height":                   "number",
		"package_dimensions.length":                   "number",
		"package_dimensions.weight":                   "number",
		"package_dimensions.width":                    "number",
		"shippable":                                   "boolean",
		"statement_descriptor":                        "string",
		"tax_code":                                    "string",
		"type":                                        "string",
		"unit_label":                                  "string",
		"url":                                         "string",
	}, &Config)
	resource.NewOperationCmd(rProductsCmd.Cmd, "delete", "/v1/products/{id}", http.MethodDelete, map[string]string{}, &Config)
	resource.NewOperationCmd(rProductsCmd.Cmd, "list", "/v1/products", http.MethodGet, map[string]string{
		"active":         "boolean",
		"created":        "integer",
		"ending_before":  "string",
		"ids":            "array",
		"limit":          "integer",
		"shippable":      "boolean",
		"starting_after": "string",
		"type":           "string",
		"url":            "string",
	}, &Config)
	resource.NewOperationCmd(rProductsCmd.Cmd, "retrieve", "/v1/products/{id}", http.MethodGet, map[string]string{}, &Config)
	resource.NewOperationCmd(rProductsCmd.Cmd, "search", "/v1/products/search", http.MethodGet, map[string]string{
		"limit": "integer",
		"page":  "string",
		"query": "string",
	}, &Config)
	resource.NewOperationCmd(rProductsCmd.Cmd, "update", "/v1/products/{id}", http.MethodPost, map[string]string{
		"active":               "boolean",
		"default_price":        "string",
		"description":          "string",
		"images":               "array",
		"name":                 "string",
		"shippable":            "boolean",
		"statement_descriptor": "string",
		"tax_code":             "string",
		"unit_label":           "string",
		"url":                  "string",
	}, &Config)
	resource.NewOperationCmd(rPromotionCodesCmd.Cmd, "create", "/v1/promotion_codes", http.MethodPost, map[string]string{
		"active":                               "boolean",
		"code":                                 "string",
		"coupon":                               "string",
		"customer":                             "string",
		"expires_at":                           "integer",
		"max_redemptions":                      "integer",
		"restrictions.first_time_transaction":  "boolean",
		"restrictions.minimum_amount":          "integer",
		"restrictions.minimum_amount_currency": "string",
	}, &Config)
	resource.NewOperationCmd(rPromotionCodesCmd.Cmd, "list", "/v1/promotion_codes", http.MethodGet, map[string]string{
		"active":         "boolean",
		"code":           "string",
		"coupon":         "string",
		"created":        "integer",
		"customer":       "string",
		"ending_before":  "string",
		"limit":          "integer",
		"starting_after": "string",
	}, &Config)
	resource.NewOperationCmd(rPromotionCodesCmd.Cmd, "retrieve", "/v1/promotion_codes/{promotion_code}", http.MethodGet, map[string]string{}, &Config)
	resource.NewOperationCmd(rPromotionCodesCmd.Cmd, "update", "/v1/promotion_codes/{promotion_code}", http.MethodPost, map[string]string{
		"active": "boolean",
	}, &Config)
	resource.NewOperationCmd(rQuotesCmd.Cmd, "accept", "/v1/quotes/{quote}/accept", http.MethodPost, map[string]string{}, &Config)
	resource.NewOperationCmd(rQuotesCmd.Cmd, "cancel", "/v1/quotes/{quote}/cancel", http.MethodPost, map[string]string{}, &Config)
	resource.NewOperationCmd(rQuotesCmd.Cmd, "create", "/v1/quotes", http.MethodPost, map[string]string{
		"application_fee_amount":              "integer",
		"application_fee_percent":             "number",
		"automatic_tax.enabled":               "boolean",
		"automatic_tax.liability.account":     "string",
		"automatic_tax.liability.type":        "string",
		"collection_method":                   "string",
		"customer":                            "string",
		"default_tax_rates":                   "array",
		"description":                         "string",
		"expires_at":                          "integer",
		"footer":                              "string",
		"from_quote.is_revision":              "boolean",
		"from_quote.quote":                    "string",
		"header":                              "string",
		"invoice_settings.days_until_due":     "integer",
		"invoice_settings.issuer.account":     "string",
		"invoice_settings.issuer.type":        "string",
		"on_behalf_of":                        "string",
		"subscription_data.description":       "string",
		"subscription_data.effective_date":    "string",
		"subscription_data.trial_period_days": "integer",
		"test_clock":                          "string",
	}, &Config)
	resource.NewOperationCmd(rQuotesCmd.Cmd, "finalize_quote", "/v1/quotes/{quote}/finalize", http.MethodPost, map[string]string{
		"expires_at": "integer",
	}, &Config)
	resource.NewOperationCmd(rQuotesCmd.Cmd, "list", "/v1/quotes", http.MethodGet, map[string]string{
		"customer":       "string",
		"ending_before":  "string",
		"limit":          "integer",
		"starting_after": "string",
		"status":         "string",
		"test_clock":     "string",
	}, &Config)
	resource.NewOperationCmd(rQuotesCmd.Cmd, "list_computed_upfront_line_items", "/v1/quotes/{quote}/computed_upfront_line_items", http.MethodGet, map[string]string{
		"ending_before":  "string",
		"limit":          "integer",
		"starting_after": "string",
	}, &Config)
	resource.NewOperationCmd(rQuotesCmd.Cmd, "list_line_items", "/v1/quotes/{quote}/line_items", http.MethodGet, map[string]string{
		"ending_before":  "string",
		"limit":          "integer",
		"starting_after": "string",
	}, &Config)
	resource.NewOperationCmd(rQuotesCmd.Cmd, "pdf", "/v1/quotes/{quote}/pdf", http.MethodGet, map[string]string{}, &Config)
	resource.NewOperationCmd(rQuotesCmd.Cmd, "retrieve", "/v1/quotes/{quote}", http.MethodGet, map[string]string{}, &Config)
	resource.NewOperationCmd(rQuotesCmd.Cmd, "update", "/v1/quotes/{quote}", http.MethodPost, map[string]string{
		"application_fee_amount":              "integer",
		"application_fee_percent":             "number",
		"automatic_tax.enabled":               "boolean",
		"automatic_tax.liability.account":     "string",
		"automatic_tax.liability.type":        "string",
		"collection_method":                   "string",
		"customer":                            "string",
		"default_tax_rates":                   "array",
		"description":                         "string",
		"expires_at":                          "integer",
		"footer":                              "string",
		"header":                              "string",
		"invoice_settings.days_until_due":     "integer",
		"invoice_settings.issuer.account":     "string",
		"invoice_settings.issuer.type":        "string",
		"on_behalf_of":                        "string",
		"subscription_data.description":       "string",
		"subscription_data.effective_date":    "string",
		"subscription_data.trial_period_days": "integer",
	}, &Config)
	resource.NewOperationCmd(rRefundsCmd.Cmd, "cancel", "/v1/refunds/{refund}/cancel", http.MethodPost, map[string]string{}, &Config)
	resource.NewOperationCmd(rRefundsCmd.Cmd, "create", "/v1/refunds", http.MethodPost, map[string]string{
		"amount":                 "integer",
		"charge":                 "string",
		"currency":               "string",
		"customer":               "string",
		"instructions_email":     "string",
		"origin":                 "string",
		"payment_intent":         "string",
		"reason":                 "string",
		"refund_application_fee": "boolean",
		"reverse_transfer":       "boolean",
	}, &Config)
	resource.NewOperationCmd(rRefundsCmd.Cmd, "list", "/v1/refunds", http.MethodGet, map[string]string{
		"charge":         "string",
		"created":        "integer",
		"ending_before":  "string",
		"limit":          "integer",
		"payment_intent": "string",
		"starting_after": "string",
	}, &Config)
	resource.NewOperationCmd(rRefundsCmd.Cmd, "retrieve", "/v1/refunds/{refund}", http.MethodGet, map[string]string{}, &Config)
	resource.NewOperationCmd(rRefundsCmd.Cmd, "update", "/v1/refunds/{refund}", http.MethodPost, map[string]string{}, &Config)
	resource.NewOperationCmd(rRefundsTestHelpersCmd.Cmd, "expire", "/v1/test_helpers/refunds/{refund}/expire", http.MethodPost, map[string]string{}, &Config)
	resource.NewOperationCmd(rReviewsCmd.Cmd, "approve", "/v1/reviews/{review}/approve", http.MethodPost, map[string]string{}, &Config)
	resource.NewOperationCmd(rReviewsCmd.Cmd, "list", "/v1/reviews", http.MethodGet, map[string]string{
		"created":        "integer",
		"ending_before":  "string",
		"limit":          "integer",
		"starting_after": "string",
	}, &Config)
	resource.NewOperationCmd(rReviewsCmd.Cmd, "retrieve", "/v1/reviews/{review}", http.MethodGet, map[string]string{}, &Config)
	resource.NewOperationCmd(rScheduledQueryRunsCmd.Cmd, "list", "/v1/sigma/scheduled_query_runs", http.MethodGet, map[string]string{
		"ending_before":  "string",
		"limit":          "integer",
		"starting_after": "string",
	}, &Config)
	resource.NewOperationCmd(rScheduledQueryRunsCmd.Cmd, "retrieve", "/v1/sigma/scheduled_query_runs/{scheduled_query_run}", http.MethodGet, map[string]string{}, &Config)
	resource.NewOperationCmd(rSetupAttemptsCmd.Cmd, "list", "/v1/setup_attempts", http.MethodGet, map[string]string{
		"created":        "integer",
		"ending_before":  "string",
		"limit":          "integer",
		"setup_intent":   "string",
		"starting_after": "string",
	}, &Config)
	resource.NewOperationCmd(rSetupIntentsCmd.Cmd, "cancel", "/v1/setup_intents/{intent}/cancel", http.MethodPost, map[string]string{
		"cancellation_reason": "string",
	}, &Config)
	resource.NewOperationCmd(rSetupIntentsCmd.Cmd, "confirm", "/v1/setup_intents/{intent}/confirm", http.MethodPost, map[string]string{
		"confirmation_token": "string",
		"payment_method":     "string",
		"payment_method_data.acss_debit.account_number":                                            "string",
		"payment_method_data.acss_debit.institution_number":                                        "string",
		"payment_method_data.acss_debit.transit_number":                                            "string",
		"payment_method_data.allow_redisplay":                                                      "string",
		"payment_method_data.au_becs_debit.account_number":                                         "string",
		"payment_method_data.au_becs_debit.bsb_number":                                             "string",
		"payment_method_data.bacs_debit.account_number":                                            "string",
		"payment_method_data.bacs_debit.sort_code":                                                 "string",
		"payment_method_data.billing_details.email":                                                "string",
		"payment_method_data.billing_details.name":                                                 "string",
		"payment_method_data.billing_details.phone":                                                "string",
		"payment_method_data.boleto.tax_id":                                                        "string",
		"payment_method_data.eps.bank":                                                             "string",
		"payment_method_data.fpx.account_holder_type":                                              "string",
		"payment_method_data.fpx.bank":                                                             "string",
		"payment_method_data.ideal.bank":                                                           "string",
		"payment_method_data.klarna.dob.day":                                                       "integer",
		"payment_method_data.klarna.dob.month":                                                     "integer",
		"payment_method_data.klarna.dob.year":                                                      "integer",
		"payment_method_data.p24.bank":                                                             "string",
		"payment_method_data.radar_options.session":                                                "string",
		"payment_method_data.sepa_debit.iban":                                                      "string",
		"payment_method_data.sofort.country":                                                       "string",
		"payment_method_data.type":                                                                 "string",
		"payment_method_data.us_bank_account.account_holder_type":                                  "string",
		"payment_method_data.us_bank_account.account_number":                                       "string",
		"payment_method_data.us_bank_account.account_type":                                         "string",
		"payment_method_data.us_bank_account.financial_connections_account":                        "string",
		"payment_method_data.us_bank_account.routing_number":                                       "string",
		"payment_method_options.acss_debit.currency":                                               "string",
		"payment_method_options.acss_debit.mandate_options.custom_mandate_url":                     "string",
		"payment_method_options.acss_debit.mandate_options.default_for":                            "array",
		"payment_method_options.acss_debit.mandate_options.interval_description":                   "string",
		"payment_method_options.acss_debit.mandate_options.payment_schedule":                       "string",
		"payment_method_options.acss_debit.mandate_options.transaction_type":                       "string",
		"payment_method_options.acss_debit.verification_method":                                    "string",
		"payment_method_options.card.mandate_options.amount":                                       "integer",
		"payment_method_options.card.mandate_options.amount_type":                                  "string",
		"payment_method_options.card.mandate_options.currency":                                     "string",
		"payment_method_options.card.mandate_options.description":                                  "string",
		"payment_method_options.card.mandate_options.end_date":                                     "integer",
		"payment_method_options.card.mandate_options.interval":                                     "string",
		"payment_method_options.card.mandate_options.interval_count":                               "integer",
		"payment_method_options.card.mandate_options.reference":                                    "string",
		"payment_method_options.card.mandate_options.start_date":                                   "integer",
		"payment_method_options.card.mandate_options.supported_types":                              "array",
		"payment_method_options.card.moto":                                                         "boolean",
		"payment_method_options.card.network":                                                      "string",
		"payment_method_options.card.request_three_d_secure":                                       "string",
		"payment_method_options.card.three_d_secure.ares_trans_status":                             "string",
		"payment_method_options.card.three_d_secure.cryptogram":                                    "string",
		"payment_method_options.card.three_d_secure.electronic_commerce_indicator":                 "string",
		"payment_method_options.card.three_d_secure.network_options.cartes_bancaires.cb_avalgo":    "string",
		"payment_method_options.card.three_d_secure.network_options.cartes_bancaires.cb_exemption": "string",
		"payment_method_options.card.three_d_secure.network_options.cartes_bancaires.cb_score":     "integer",
		"payment_method_options.card.three_d_secure.requestor_challenge_indicator":                 "string",
		"payment_method_options.card.three_d_secure.transaction_id":                                "string",
		"payment_method_options.card.three_d_secure.version":                                       "string",
		"payment_method_options.link.persistent_token":                                             "string",
		"payment_method_options.paypal.billing_agreement_id":                                       "string",
		"payment_method_options.us_bank_account.financial_connections.permissions":                 "array",
		"payment_method_options.us_bank_account.financial_connections.prefetch":                    "array",
		"payment_method_options.us_bank_account.financial_connections.return_url":                  "string",
		"payment_method_options.us_bank_account.mandate_options.collection_method":                 "string",
		"payment_method_options.us_bank_account.networks.requested":                                "array",
		"payment_method_options.us_bank_account.verification_method":                               "string",
		"return_url":     "string",
		"use_stripe_sdk": "boolean",
	}, &Config)
	resource.NewOperationCmd(rSetupIntentsCmd.Cmd, "create", "/v1/setup_intents", http.MethodPost, map[string]string{
		"attach_to_self": "boolean",
		"automatic_payment_methods.allow_redirects": "string",
		"automatic_payment_methods.enabled":         "boolean",
		"confirm":                                   "boolean",
		"confirmation_token":                        "string",
		"customer":                                  "string",
		"description":                               "string",
		"flow_directions":                           "array",
		"on_behalf_of":                              "string",
		"payment_method":                            "string",
		"payment_method_configuration":              "string",
		"payment_method_data.acss_debit.account_number":                                            "string",
		"payment_method_data.acss_debit.institution_number":                                        "string",
		"payment_method_data.acss_debit.transit_number":                                            "string",
		"payment_method_data.allow_redisplay":                                                      "string",
		"payment_method_data.au_becs_debit.account_number":                                         "string",
		"payment_method_data.au_becs_debit.bsb_number":                                             "string",
		"payment_method_data.bacs_debit.account_number":                                            "string",
		"payment_method_data.bacs_debit.sort_code":                                                 "string",
		"payment_method_data.billing_details.email":                                                "string",
		"payment_method_data.billing_details.name":                                                 "string",
		"payment_method_data.billing_details.phone":                                                "string",
		"payment_method_data.boleto.tax_id":                                                        "string",
		"payment_method_data.eps.bank":                                                             "string",
		"payment_method_data.fpx.account_holder_type":                                              "string",
		"payment_method_data.fpx.bank":                                                             "string",
		"payment_method_data.ideal.bank":                                                           "string",
		"payment_method_data.klarna.dob.day":                                                       "integer",
		"payment_method_data.klarna.dob.month":                                                     "integer",
		"payment_method_data.klarna.dob.year":                                                      "integer",
		"payment_method_data.p24.bank":                                                             "string",
		"payment_method_data.radar_options.session":                                                "string",
		"payment_method_data.sepa_debit.iban":                                                      "string",
		"payment_method_data.sofort.country":                                                       "string",
		"payment_method_data.type":                                                                 "string",
		"payment_method_data.us_bank_account.account_holder_type":                                  "string",
		"payment_method_data.us_bank_account.account_number":                                       "string",
		"payment_method_data.us_bank_account.account_type":                                         "string",
		"payment_method_data.us_bank_account.financial_connections_account":                        "string",
		"payment_method_data.us_bank_account.routing_number":                                       "string",
		"payment_method_options.acss_debit.currency":                                               "string",
		"payment_method_options.acss_debit.mandate_options.custom_mandate_url":                     "string",
		"payment_method_options.acss_debit.mandate_options.default_for":                            "array",
		"payment_method_options.acss_debit.mandate_options.interval_description":                   "string",
		"payment_method_options.acss_debit.mandate_options.payment_schedule":                       "string",
		"payment_method_options.acss_debit.mandate_options.transaction_type":                       "string",
		"payment_method_options.acss_debit.verification_method":                                    "string",
		"payment_method_options.card.mandate_options.amount":                                       "integer",
		"payment_method_options.card.mandate_options.amount_type":                                  "string",
		"payment_method_options.card.mandate_options.currency":                                     "string",
		"payment_method_options.card.mandate_options.description":                                  "string",
		"payment_method_options.card.mandate_options.end_date":                                     "integer",
		"payment_method_options.card.mandate_options.interval":                                     "string",
		"payment_method_options.card.mandate_options.interval_count":                               "integer",
		"payment_method_options.card.mandate_options.reference":                                    "string",
		"payment_method_options.card.mandate_options.start_date":                                   "integer",
		"payment_method_options.card.mandate_options.supported_types":                              "array",
		"payment_method_options.card.moto":                                                         "boolean",
		"payment_method_options.card.network":                                                      "string",
		"payment_method_options.card.request_three_d_secure":                                       "string",
		"payment_method_options.card.three_d_secure.ares_trans_status":                             "string",
		"payment_method_options.card.three_d_secure.cryptogram":                                    "string",
		"payment_method_options.card.three_d_secure.electronic_commerce_indicator":                 "string",
		"payment_method_options.card.three_d_secure.network_options.cartes_bancaires.cb_avalgo":    "string",
		"payment_method_options.card.three_d_secure.network_options.cartes_bancaires.cb_exemption": "string",
		"payment_method_options.card.three_d_secure.network_options.cartes_bancaires.cb_score":     "integer",
		"payment_method_options.card.three_d_secure.requestor_challenge_indicator":                 "string",
		"payment_method_options.card.three_d_secure.transaction_id":                                "string",
		"payment_method_options.card.three_d_secure.version":                                       "string",
		"payment_method_options.link.persistent_token":                                             "string",
		"payment_method_options.paypal.billing_agreement_id":                                       "string",
		"payment_method_options.us_bank_account.financial_connections.permissions":                 "array",
		"payment_method_options.us_bank_account.financial_connections.prefetch":                    "array",
		"payment_method_options.us_bank_account.financial_connections.return_url":                  "string",
		"payment_method_options.us_bank_account.mandate_options.collection_method":                 "string",
		"payment_method_options.us_bank_account.networks.requested":                                "array",
		"payment_method_options.us_bank_account.verification_method":                               "string",
		"payment_method_types":                                                                     "array",
		"return_url":                                                                               "string",
		"single_use.amount":                                                                        "integer",
		"single_use.currency":                                                                      "string",
		"usage":                                                                                    "string",
		"use_stripe_sdk":                                                                           "boolean",
	}, &Config)
	resource.NewOperationCmd(rSetupIntentsCmd.Cmd, "list", "/v1/setup_intents", http.MethodGet, map[string]string{
		"attach_to_self": "boolean",
		"created":        "integer",
		"customer":       "string",
		"ending_before":  "string",
		"limit":          "integer",
		"payment_method": "string",
		"starting_after": "string",
	}, &Config)
	resource.NewOperationCmd(rSetupIntentsCmd.Cmd, "retrieve", "/v1/setup_intents/{intent}", http.MethodGet, map[string]string{
		"client_secret": "string",
	}, &Config)
	resource.NewOperationCmd(rSetupIntentsCmd.Cmd, "update", "/v1/setup_intents/{intent}", http.MethodPost, map[string]string{
		"attach_to_self":               "boolean",
		"customer":                     "string",
		"description":                  "string",
		"flow_directions":              "array",
		"payment_method":               "string",
		"payment_method_configuration": "string",
		"payment_method_data.acss_debit.account_number":                                            "string",
		"payment_method_data.acss_debit.institution_number":                                        "string",
		"payment_method_data.acss_debit.transit_number":                                            "string",
		"payment_method_data.allow_redisplay":                                                      "string",
		"payment_method_data.au_becs_debit.account_number":                                         "string",
		"payment_method_data.au_becs_debit.bsb_number":                                             "string",
		"payment_method_data.bacs_debit.account_number":                                            "string",
		"payment_method_data.bacs_debit.sort_code":                                                 "string",
		"payment_method_data.billing_details.email":                                                "string",
		"payment_method_data.billing_details.name":                                                 "string",
		"payment_method_data.billing_details.phone":                                                "string",
		"payment_method_data.boleto.tax_id":                                                        "string",
		"payment_method_data.eps.bank":                                                             "string",
		"payment_method_data.fpx.account_holder_type":                                              "string",
		"payment_method_data.fpx.bank":                                                             "string",
		"payment_method_data.ideal.bank":                                                           "string",
		"payment_method_data.klarna.dob.day":                                                       "integer",
		"payment_method_data.klarna.dob.month":                                                     "integer",
		"payment_method_data.klarna.dob.year":                                                      "integer",
		"payment_method_data.p24.bank":                                                             "string",
		"payment_method_data.radar_options.session":                                                "string",
		"payment_method_data.sepa_debit.iban":                                                      "string",
		"payment_method_data.sofort.country":                                                       "string",
		"payment_method_data.type":                                                                 "string",
		"payment_method_data.us_bank_account.account_holder_type":                                  "string",
		"payment_method_data.us_bank_account.account_number":                                       "string",
		"payment_method_data.us_bank_account.account_type":                                         "string",
		"payment_method_data.us_bank_account.financial_connections_account":                        "string",
		"payment_method_data.us_bank_account.routing_number":                                       "string",
		"payment_method_options.acss_debit.currency":                                               "string",
		"payment_method_options.acss_debit.mandate_options.custom_mandate_url":                     "string",
		"payment_method_options.acss_debit.mandate_options.default_for":                            "array",
		"payment_method_options.acss_debit.mandate_options.interval_description":                   "string",
		"payment_method_options.acss_debit.mandate_options.payment_schedule":                       "string",
		"payment_method_options.acss_debit.mandate_options.transaction_type":                       "string",
		"payment_method_options.acss_debit.verification_method":                                    "string",
		"payment_method_options.card.mandate_options.amount":                                       "integer",
		"payment_method_options.card.mandate_options.amount_type":                                  "string",
		"payment_method_options.card.mandate_options.currency":                                     "string",
		"payment_method_options.card.mandate_options.description":                                  "string",
		"payment_method_options.card.mandate_options.end_date":                                     "integer",
		"payment_method_options.card.mandate_options.interval":                                     "string",
		"payment_method_options.card.mandate_options.interval_count":                               "integer",
		"payment_method_options.card.mandate_options.reference":                                    "string",
		"payment_method_options.card.mandate_options.start_date":                                   "integer",
		"payment_method_options.card.mandate_options.supported_types":                              "array",
		"payment_method_options.card.moto":                                                         "boolean",
		"payment_method_options.card.network":                                                      "string",
		"payment_method_options.card.request_three_d_secure":                                       "string",
		"payment_method_options.card.three_d_secure.ares_trans_status":                             "string",
		"payment_method_options.card.three_d_secure.cryptogram":                                    "string",
		"payment_method_options.card.three_d_secure.electronic_commerce_indicator":                 "string",
		"payment_method_options.card.three_d_secure.network_options.cartes_bancaires.cb_avalgo":    "string",
		"payment_method_options.card.three_d_secure.network_options.cartes_bancaires.cb_exemption": "string",
		"payment_method_options.card.three_d_secure.network_options.cartes_bancaires.cb_score":     "integer",
		"payment_method_options.card.three_d_secure.requestor_challenge_indicator":                 "string",
		"payment_method_options.card.three_d_secure.transaction_id":                                "string",
		"payment_method_options.card.three_d_secure.version":                                       "string",
		"payment_method_options.link.persistent_token":                                             "string",
		"payment_method_options.paypal.billing_agreement_id":                                       "string",
		"payment_method_options.us_bank_account.financial_connections.permissions":                 "array",
		"payment_method_options.us_bank_account.financial_connections.prefetch":                    "array",
		"payment_method_options.us_bank_account.financial_connections.return_url":                  "string",
		"payment_method_options.us_bank_account.mandate_options.collection_method":                 "string",
		"payment_method_options.us_bank_account.networks.requested":                                "array",
		"payment_method_options.us_bank_account.verification_method":                               "string",
		"payment_method_types":                                                                     "array",
	}, &Config)
	resource.NewOperationCmd(rSetupIntentsCmd.Cmd, "verify_microdeposits", "/v1/setup_intents/{intent}/verify_microdeposits", http.MethodPost, map[string]string{
		"amounts":         "array",
		"descriptor_code": "string",
	}, &Config)
	resource.NewOperationCmd(rShippingRatesCmd.Cmd, "create", "/v1/shipping_rates", http.MethodPost, map[string]string{
		"delivery_estimate.maximum.unit":  "string",
		"delivery_estimate.maximum.value": "integer",
		"delivery_estimate.minimum.unit":  "string",
		"delivery_estimate.minimum.value": "integer",
		"display_name":                    "string",
		"fixed_amount.amount":             "integer",
		"fixed_amount.currency":           "string",
		"tax_behavior":                    "string",
		"tax_code":                        "string",
		"type":                            "string",
	}, &Config)
	resource.NewOperationCmd(rShippingRatesCmd.Cmd, "list", "/v1/shipping_rates", http.MethodGet, map[string]string{
		"active":         "boolean",
		"created":        "integer",
		"currency":       "string",
		"ending_before":  "string",
		"limit":          "integer",
		"starting_after": "string",
	}, &Config)
	resource.NewOperationCmd(rShippingRatesCmd.Cmd, "retrieve", "/v1/shipping_rates/{shipping_rate_token}", http.MethodGet, map[string]string{}, &Config)
	resource.NewOperationCmd(rShippingRatesCmd.Cmd, "update", "/v1/shipping_rates/{shipping_rate_token}", http.MethodPost, map[string]string{
		"active":       "boolean",
		"tax_behavior": "string",
	}, &Config)
	resource.NewOperationCmd(rSourcesCmd.Cmd, "create", "/v1/sources", http.MethodPost, map[string]string{
		"amount":                  "integer",
		"currency":                "string",
		"customer":                "string",
		"flow":                    "string",
		"mandate.acceptance.date": "integer",
		"mandate.acceptance.ip":   "string",
		"mandate.acceptance.offline.contact_email":  "string",
		"mandate.acceptance.online.date":            "integer",
		"mandate.acceptance.online.ip":              "string",
		"mandate.acceptance.online.user_agent":      "string",
		"mandate.acceptance.status":                 "string",
		"mandate.acceptance.type":                   "string",
		"mandate.acceptance.user_agent":             "string",
		"mandate.amount":                            "integer",
		"mandate.currency":                          "string",
		"mandate.interval":                          "string",
		"mandate.notification_method":               "string",
		"original_source":                           "string",
		"owner.address.city":                        "string",
		"owner.address.country":                     "string",
		"owner.address.line1":                       "string",
		"owner.address.line2":                       "string",
		"owner.address.postal_code":                 "string",
		"owner.address.state":                       "string",
		"owner.email":                               "string",
		"owner.name":                                "string",
		"owner.phone":                               "string",
		"receiver.refund_attributes_method":         "string",
		"redirect.return_url":                       "string",
		"source_order.shipping.address.city":        "string",
		"source_order.shipping.address.country":     "string",
		"source_order.shipping.address.line1":       "string",
		"source_order.shipping.address.line2":       "string",
		"source_order.shipping.address.postal_code": "string",
		"source_order.shipping.address.state":       "string",
		"source_order.shipping.carrier":             "string",
		"source_order.shipping.name":                "string",
		"source_order.shipping.phone":               "string",
		"source_order.shipping.tracking_number":     "string",
		"statement_descriptor":                      "string",
		"token":                                     "string",
		"type":                                      "string",
		"usage":                                     "string",
	}, &Config)
	resource.NewOperationCmd(rSourcesCmd.Cmd, "detach", "/v1/customers/{customer}/sources/{id}", http.MethodDelete, map[string]string{}, &Config)
	resource.NewOperationCmd(rSourcesCmd.Cmd, "retrieve", "/v1/sources/{source}", http.MethodGet, map[string]string{
		"client_secret": "string",
	}, &Config)
	resource.NewOperationCmd(rSourcesCmd.Cmd, "source_transactions", "/v1/sources/{source}/source_transactions", http.MethodGet, map[string]string{
		"ending_before":  "string",
		"limit":          "integer",
		"starting_after": "string",
	}, &Config)
	resource.NewOperationCmd(rSourcesCmd.Cmd, "update", "/v1/sources/{source}", http.MethodPost, map[string]string{
		"amount":                  "integer",
		"mandate.acceptance.date": "integer",
		"mandate.acceptance.ip":   "string",
		"mandate.acceptance.offline.contact_email":  "string",
		"mandate.acceptance.online.date":            "integer",
		"mandate.acceptance.online.ip":              "string",
		"mandate.acceptance.online.user_agent":      "string",
		"mandate.acceptance.status":                 "string",
		"mandate.acceptance.type":                   "string",
		"mandate.acceptance.user_agent":             "string",
		"mandate.amount":                            "integer",
		"mandate.currency":                          "string",
		"mandate.interval":                          "string",
		"mandate.notification_method":               "string",
		"owner.address.city":                        "string",
		"owner.address.country":                     "string",
		"owner.address.line1":                       "string",
		"owner.address.line2":                       "string",
		"owner.address.postal_code":                 "string",
		"owner.address.state":                       "string",
		"owner.email":                               "string",
		"owner.name":                                "string",
		"owner.phone":                               "string",
		"source_order.shipping.address.city":        "string",
		"source_order.shipping.address.country":     "string",
		"source_order.shipping.address.line1":       "string",
		"source_order.shipping.address.line2":       "string",
		"source_order.shipping.address.postal_code": "string",
		"source_order.shipping.address.state":       "string",
		"source_order.shipping.carrier":             "string",
		"source_order.shipping.name":                "string",
		"source_order.shipping.phone":               "string",
		"source_order.shipping.tracking_number":     "string",
	}, &Config)
	resource.NewOperationCmd(rSourcesCmd.Cmd, "verify", "/v1/sources/{source}/verify", http.MethodPost, map[string]string{
		"values": "array",
	}, &Config)
	resource.NewOperationCmd(rSubscriptionItemsCmd.Cmd, "create", "/v1/subscription_items", http.MethodPost, map[string]string{
		"payment_behavior":                    "string",
		"plan":                                "string",
		"price":                               "string",
		"price_data.currency":                 "string",
		"price_data.product":                  "string",
		"price_data.recurring.interval":       "string",
		"price_data.recurring.interval_count": "integer",
		"price_data.tax_behavior":             "string",
		"price_data.unit_amount":              "integer",
		"price_data.unit_amount_decimal":      "string",
		"proration_behavior":                  "string",
		"proration_date":                      "integer",
		"quantity":                            "integer",
		"subscription":                        "string",
		"tax_rates":                           "array",
	}, &Config)
	resource.NewOperationCmd(rSubscriptionItemsCmd.Cmd, "delete", "/v1/subscription_items/{item}", http.MethodDelete, map[string]string{}, &Config)
	resource.NewOperationCmd(rSubscriptionItemsCmd.Cmd, "list", "/v1/subscription_items", http.MethodGet, map[string]string{
		"ending_before":  "string",
		"limit":          "integer",
		"starting_after": "string",
		"subscription":   "string",
	}, &Config)
	resource.NewOperationCmd(rSubscriptionItemsCmd.Cmd, "retrieve", "/v1/subscription_items/{item}", http.MethodGet, map[string]string{}, &Config)
	resource.NewOperationCmd(rSubscriptionItemsCmd.Cmd, "update", "/v1/subscription_items/{item}", http.MethodPost, map[string]string{
		"off_session":                         "boolean",
		"payment_behavior":                    "string",
		"plan":                                "string",
		"price":                               "string",
		"price_data.currency":                 "string",
		"price_data.product":                  "string",
		"price_data.recurring.interval":       "string",
		"price_data.recurring.interval_count": "integer",
		"price_data.tax_behavior":             "string",
		"price_data.unit_amount":              "integer",
		"price_data.unit_amount_decimal":      "string",
		"proration_behavior":                  "string",
		"proration_date":                      "integer",
		"quantity":                            "integer",
		"tax_rates":                           "array",
	}, &Config)
	resource.NewOperationCmd(rSubscriptionItemsCmd.Cmd, "usage_record_summaries", "/v1/subscription_items/{subscription_item}/usage_record_summaries", http.MethodGet, map[string]string{
		"ending_before":  "string",
		"limit":          "integer",
		"starting_after": "string",
	}, &Config)
	resource.NewOperationCmd(rSubscriptionSchedulesCmd.Cmd, "cancel", "/v1/subscription_schedules/{schedule}/cancel", http.MethodPost, map[string]string{
		"invoice_now": "boolean",
		"prorate":     "boolean",
	}, &Config)
	resource.NewOperationCmd(rSubscriptionSchedulesCmd.Cmd, "create", "/v1/subscription_schedules", http.MethodPost, map[string]string{
		"customer": "string",
		"default_settings.application_fee_percent":          "number",
		"default_settings.automatic_tax.enabled":            "boolean",
		"default_settings.automatic_tax.liability.account":  "string",
		"default_settings.automatic_tax.liability.type":     "string",
		"default_settings.billing_cycle_anchor":             "string",
		"default_settings.collection_method":                "string",
		"default_settings.default_payment_method":           "string",
		"default_settings.description":                      "string",
		"default_settings.invoice_settings.account_tax_ids": "array",
		"default_settings.invoice_settings.days_until_due":  "integer",
		"default_settings.invoice_settings.issuer.account":  "string",
		"default_settings.invoice_settings.issuer.type":     "string",
		"default_settings.on_behalf_of":                     "string",
		"end_behavior":                                      "string",
		"from_subscription":                                 "string",
		"start_date":                                        "integer",
	}, &Config)
	resource.NewOperationCmd(rSubscriptionSchedulesCmd.Cmd, "list", "/v1/subscription_schedules", http.MethodGet, map[string]string{
		"canceled_at":    "integer",
		"completed_at":   "integer",
		"created":        "integer",
		"customer":       "string",
		"ending_before":  "string",
		"limit":          "integer",
		"released_at":    "integer",
		"scheduled":      "boolean",
		"starting_after": "string",
	}, &Config)
	resource.NewOperationCmd(rSubscriptionSchedulesCmd.Cmd, "release", "/v1/subscription_schedules/{schedule}/release", http.MethodPost, map[string]string{
		"preserve_cancel_date": "boolean",
	}, &Config)
	resource.NewOperationCmd(rSubscriptionSchedulesCmd.Cmd, "retrieve", "/v1/subscription_schedules/{schedule}", http.MethodGet, map[string]string{}, &Config)
	resource.NewOperationCmd(rSubscriptionSchedulesCmd.Cmd, "update", "/v1/subscription_schedules/{schedule}", http.MethodPost, map[string]string{
		"default_settings.application_fee_percent":          "number",
		"default_settings.automatic_tax.enabled":            "boolean",
		"default_settings.automatic_tax.liability.account":  "string",
		"default_settings.automatic_tax.liability.type":     "string",
		"default_settings.billing_cycle_anchor":             "string",
		"default_settings.collection_method":                "string",
		"default_settings.default_payment_method":           "string",
		"default_settings.description":                      "string",
		"default_settings.invoice_settings.account_tax_ids": "array",
		"default_settings.invoice_settings.days_until_due":  "integer",
		"default_settings.invoice_settings.issuer.account":  "string",
		"default_settings.invoice_settings.issuer.type":     "string",
		"default_settings.on_behalf_of":                     "string",
		"end_behavior":                                      "string",
		"proration_behavior":                                "string",
	}, &Config)
	resource.NewOperationCmd(rSubscriptionsCmd.Cmd, "cancel", "/v1/subscriptions/{subscription_exposed_id}", http.MethodDelete, map[string]string{}, &Config)
	resource.NewOperationCmd(rSubscriptionsCmd.Cmd, "create", "/v1/subscriptions", http.MethodPost, map[string]string{
		"application_fee_percent":                            "number",
		"automatic_tax.enabled":                              "boolean",
		"automatic_tax.liability.account":                    "string",
		"automatic_tax.liability.type":                       "string",
		"backdate_start_date":                                "integer",
		"billing_cycle_anchor":                               "integer",
		"billing_cycle_anchor_config.day_of_month":           "integer",
		"billing_cycle_anchor_config.hour":                   "integer",
		"billing_cycle_anchor_config.minute":                 "integer",
		"billing_cycle_anchor_config.month":                  "integer",
		"billing_cycle_anchor_config.second":                 "integer",
		"cancel_at":                                          "integer",
		"cancel_at_period_end":                               "boolean",
		"collection_method":                                  "string",
		"coupon":                                             "string",
		"currency":                                           "string",
		"customer":                                           "string",
		"days_until_due":                                     "integer",
		"default_payment_method":                             "string",
		"default_source":                                     "string",
		"default_tax_rates":                                  "array",
		"description":                                        "string",
		"invoice_settings.account_tax_ids":                   "array",
		"invoice_settings.issuer.account":                    "string",
		"invoice_settings.issuer.type":                       "string",
		"off_session":                                        "boolean",
		"on_behalf_of":                                       "string",
		"payment_behavior":                                   "string",
		"payment_settings.payment_method_types":              "array",
		"payment_settings.save_default_payment_method":       "string",
		"promotion_code":                                     "string",
		"proration_behavior":                                 "string",
		"transfer_data.amount_percent":                       "number",
		"transfer_data.destination":                          "string",
		"trial_end":                                          "string",
		"trial_from_plan":                                    "boolean",
		"trial_period_days":                                  "integer",
		"trial_settings.end_behavior.missing_payment_method": "string",
	}, &Config)
	resource.NewOperationCmd(rSubscriptionsCmd.Cmd, "delete_discount", "/v1/subscriptions/{subscription_exposed_id}/discount", http.MethodDelete, map[string]string{}, &Config)
	resource.NewOperationCmd(rSubscriptionsCmd.Cmd, "list", "/v1/subscriptions", http.MethodGet, map[string]string{
		"collection_method":    "string",
		"created":              "integer",
		"current_period_end":   "integer",
		"current_period_start": "integer",
		"customer":             "string",
		"ending_before":        "string",
		"limit":                "integer",
		"plan":                 "string",
		"price":                "string",
		"starting_after":       "string",
		"status":               "string",
		"test_clock":           "string",
	}, &Config)
	resource.NewOperationCmd(rSubscriptionsCmd.Cmd, "resume", "/v1/subscriptions/{subscription}/resume", http.MethodPost, map[string]string{
		"billing_cycle_anchor": "string",
		"proration_behavior":   "string",
		"proration_date":       "integer",
	}, &Config)
	resource.NewOperationCmd(rSubscriptionsCmd.Cmd, "retrieve", "/v1/subscriptions/{subscription_exposed_id}", http.MethodGet, map[string]string{}, &Config)
	resource.NewOperationCmd(rSubscriptionsCmd.Cmd, "search", "/v1/subscriptions/search", http.MethodGet, map[string]string{
		"limit": "integer",
		"page":  "string",
		"query": "string",
	}, &Config)
	resource.NewOperationCmd(rSubscriptionsCmd.Cmd, "update", "/v1/subscriptions/{subscription_exposed_id}", http.MethodPost, map[string]string{
		"application_fee_percent":                      "number",
		"automatic_tax.enabled":                        "boolean",
		"automatic_tax.liability.account":              "string",
		"automatic_tax.liability.type":                 "string",
		"billing_cycle_anchor":                         "string",
		"cancel_at":                                    "integer",
		"cancel_at_period_end":                         "boolean",
		"cancellation_details.comment":                 "string",
		"cancellation_details.feedback":                "string",
		"collection_method":                            "string",
		"coupon":                                       "string",
		"days_until_due":                               "integer",
		"default_payment_method":                       "string",
		"default_source":                               "string",
		"default_tax_rates":                            "array",
		"description":                                  "string",
		"invoice_settings.account_tax_ids":             "array",
		"invoice_settings.issuer.account":              "string",
		"invoice_settings.issuer.type":                 "string",
		"off_session":                                  "boolean",
		"on_behalf_of":                                 "string",
		"payment_behavior":                             "string",
		"payment_settings.payment_method_types":        "array",
		"payment_settings.save_default_payment_method": "string",
		"promotion_code":                               "string",
		"proration_behavior":                           "string",
		"proration_date":                               "integer",
		"trial_end":                                    "string",
		"trial_from_plan":                              "boolean",
		"trial_settings.end_behavior.missing_payment_method": "string",
	}, &Config)
	resource.NewOperationCmd(rTaxCodesCmd.Cmd, "list", "/v1/tax_codes", http.MethodGet, map[string]string{
		"ending_before":  "string",
		"limit":          "integer",
		"starting_after": "string",
	}, &Config)
	resource.NewOperationCmd(rTaxCodesCmd.Cmd, "retrieve", "/v1/tax_codes/{id}", http.MethodGet, map[string]string{}, &Config)
	resource.NewOperationCmd(rTaxIdsCmd.Cmd, "create", "/v1/customers/{customer}/tax_ids", http.MethodPost, map[string]string{
		"type":  "string",
		"value": "string",
	}, &Config)
	resource.NewOperationCmd(rTaxIdsCmd.Cmd, "delete", "/v1/customers/{customer}/tax_ids/{id}", http.MethodDelete, map[string]string{}, &Config)
	resource.NewOperationCmd(rTaxIdsCmd.Cmd, "list", "/v1/customers/{customer}/tax_ids", http.MethodGet, map[string]string{
		"ending_before":  "string",
		"limit":          "integer",
		"starting_after": "string",
	}, &Config)
	resource.NewOperationCmd(rTaxIdsCmd.Cmd, "retrieve", "/v1/customers/{customer}/tax_ids/{id}", http.MethodGet, map[string]string{}, &Config)
	resource.NewOperationCmd(rTaxRatesCmd.Cmd, "create", "/v1/tax_rates", http.MethodPost, map[string]string{
		"active":       "boolean",
		"country":      "string",
		"description":  "string",
		"display_name": "string",
		"inclusive":    "boolean",
		"jurisdiction": "string",
		"percentage":   "number",
		"state":        "string",
		"tax_type":     "string",
	}, &Config)
	resource.NewOperationCmd(rTaxRatesCmd.Cmd, "list", "/v1/tax_rates", http.MethodGet, map[string]string{
		"active":         "boolean",
		"created":        "integer",
		"ending_before":  "string",
		"inclusive":      "boolean",
		"limit":          "integer",
		"starting_after": "string",
	}, &Config)
	resource.NewOperationCmd(rTaxRatesCmd.Cmd, "retrieve", "/v1/tax_rates/{tax_rate}", http.MethodGet, map[string]string{}, &Config)
	resource.NewOperationCmd(rTaxRatesCmd.Cmd, "update", "/v1/tax_rates/{tax_rate}", http.MethodPost, map[string]string{
		"active":       "boolean",
		"country":      "string",
		"description":  "string",
		"display_name": "string",
		"jurisdiction": "string",
		"state":        "string",
		"tax_type":     "string",
	}, &Config)
	resource.NewOperationCmd(rTokensCmd.Cmd, "create", "/v1/tokens", http.MethodPost, map[string]string{
		"account.business_type":                                     "string",
		"account.company.address.city":                              "string",
		"account.company.address.country":                           "string",
		"account.company.address.line1":                             "string",
		"account.company.address.line2":                             "string",
		"account.company.address.postal_code":                       "string",
		"account.company.address.state":                             "string",
		"account.company.address_kana.city":                         "string",
		"account.company.address_kana.country":                      "string",
		"account.company.address_kana.line1":                        "string",
		"account.company.address_kana.line2":                        "string",
		"account.company.address_kana.postal_code":                  "string",
		"account.company.address_kana.state":                        "string",
		"account.company.address_kana.town":                         "string",
		"account.company.address_kanji.city":                        "string",
		"account.company.address_kanji.country":                     "string",
		"account.company.address_kanji.line1":                       "string",
		"account.company.address_kanji.line2":                       "string",
		"account.company.address_kanji.postal_code":                 "string",
		"account.company.address_kanji.state":                       "string",
		"account.company.address_kanji.town":                        "string",
		"account.company.directors_provided":                        "boolean",
		"account.company.executives_provided":                       "boolean",
		"account.company.export_license_id":                         "string",
		"account.company.export_purpose_code":                       "string",
		"account.company.name":                                      "string",
		"account.company.name_kana":                                 "string",
		"account.company.name_kanji":                                "string",
		"account.company.owners_provided":                           "boolean",
		"account.company.ownership_declaration.date":                "integer",
		"account.company.ownership_declaration.ip":                  "string",
		"account.company.ownership_declaration.user_agent":          "string",
		"account.company.ownership_declaration_shown_and_signed":    "boolean",
		"account.company.phone":                                     "string",
		"account.company.registration_number":                       "string",
		"account.company.structure":                                 "string",
		"account.company.tax_id":                                    "string",
		"account.company.tax_id_registrar":                          "string",
		"account.company.vat_id":                                    "string",
		"account.company.verification.document.back":                "string",
		"account.company.verification.document.front":               "string",
		"account.individual.address.city":                           "string",
		"account.individual.address.country":                        "string",
		"account.individual.address.line1":                          "string",
		"account.individual.address.line2":                          "string",
		"account.individual.address.postal_code":                    "string",
		"account.individual.address.state":                          "string",
		"account.individual.address_kana.city":                      "string",
		"account.individual.address_kana.country":                   "string",
		"account.individual.address_kana.line1":                     "string",
		"account.individual.address_kana.line2":                     "string",
		"account.individual.address_kana.postal_code":               "string",
		"account.individual.address_kana.state":                     "string",
		"account.individual.address_kana.town":                      "string",
		"account.individual.address_kanji.city":                     "string",
		"account.individual.address_kanji.country":                  "string",
		"account.individual.address_kanji.line1":                    "string",
		"account.individual.address_kanji.line2":                    "string",
		"account.individual.address_kanji.postal_code":              "string",
		"account.individual.address_kanji.state":                    "string",
		"account.individual.address_kanji.town":                     "string",
		"account.individual.email":                                  "string",
		"account.individual.first_name":                             "string",
		"account.individual.first_name_kana":                        "string",
		"account.individual.first_name_kanji":                       "string",
		"account.individual.full_name_aliases":                      "array",
		"account.individual.gender":                                 "string",
		"account.individual.id_number":                              "string",
		"account.individual.id_number_secondary":                    "string",
		"account.individual.last_name":                              "string",
		"account.individual.last_name_kana":                         "string",
		"account.individual.last_name_kanji":                        "string",
		"account.individual.maiden_name":                            "string",
		"account.individual.phone":                                  "string",
		"account.individual.political_exposure":                     "string",
		"account.individual.registered_address.city":                "string",
		"account.individual.registered_address.country":             "string",
		"account.individual.registered_address.line1":               "string",
		"account.individual.registered_address.line2":               "string",
		"account.individual.registered_address.postal_code":         "string",
		"account.individual.registered_address.state":               "string",
		"account.individual.relationship.director":                  "boolean",
		"account.individual.relationship.executive":                 "boolean",
		"account.individual.relationship.owner":                     "boolean",
		"account.individual.relationship.percent_ownership":         "number",
		"account.individual.relationship.title":                     "string",
		"account.individual.ssn_last_4":                             "string",
		"account.individual.verification.additional_document.back":  "string",
		"account.individual.verification.additional_document.front": "string",
		"account.individual.verification.document.back":             "string",
		"account.individual.verification.document.front":            "string",
		"account.tos_shown_and_accepted":                            "boolean",
		"bank_account.account_holder_name":                          "string",
		"bank_account.account_holder_type":                          "string",
		"bank_account.account_number":                               "string",
		"bank_account.account_type":                                 "string",
		"bank_account.country":                                      "string",
		"bank_account.currency":                                     "string",
		"bank_account.payment_method":                               "string",
		"bank_account.routing_number":                               "string",
		"card":                                                      "string",
		"customer":                                                  "string",
		"cvc_update.cvc":                                            "string",
		"person.additional_tos_acceptances.account.date":            "integer",
		"person.additional_tos_acceptances.account.ip":              "string",
		"person.additional_tos_acceptances.account.user_agent":      "string",
		"person.address.city":                                       "string",
		"person.address.country":                                    "string",
		"person.address.line1":                                      "string",
		"person.address.line2":                                      "string",
		"person.address.postal_code":                                "string",
		"person.address.state":                                      "string",
		"person.address_kana.city":                                  "string",
		"person.address_kana.country":                               "string",
		"person.address_kana.line1":                                 "string",
		"person.address_kana.line2":                                 "string",
		"person.address_kana.postal_code":                           "string",
		"person.address_kana.state":                                 "string",
		"person.address_kana.town":                                  "string",
		"person.address_kanji.city":                                 "string",
		"person.address_kanji.country":                              "string",
		"person.address_kanji.line1":                                "string",
		"person.address_kanji.line2":                                "string",
		"person.address_kanji.postal_code":                          "string",
		"person.address_kanji.state":                                "string",
		"person.address_kanji.town":                                 "string",
		"person.documents.company_authorization.files":              "array",
		"person.documents.passport.files":                           "array",
		"person.documents.visa.files":                               "array",
		"person.email":                                              "string",
		"person.first_name":                                         "string",
		"person.first_name_kana":                                    "string",
		"person.first_name_kanji":                                   "string",
		"person.full_name_aliases":                                  "array",
		"person.gender":                                             "string",
		"person.id_number":                                          "string",
		"person.id_number_secondary":                                "string",
		"person.last_name":                                          "string",
		"person.last_name_kana":                                     "string",
		"person.last_name_kanji":                                    "string",
		"person.maiden_name":                                        "string",
		"person.nationality":                                        "string",
		"person.phone":                                              "string",
		"person.political_exposure":                                 "string",
		"person.registered_address.city":                            "string",
		"person.registered_address.country":                         "string",
		"person.registered_address.line1":                           "string",
		"person.registered_address.line2":                           "string",
		"person.registered_address.postal_code":                     "string",
		"person.registered_address.state":                           "string",
		"person.relationship.director":                              "boolean",
		"person.relationship.executive":                             "boolean",
		"person.relationship.legal_guardian":                        "boolean",
		"person.relationship.owner":                                 "boolean",
		"person.relationship.percent_ownership":                     "number",
		"person.relationship.representative":                        "boolean",
		"person.relationship.title":                                 "string",
		"person.ssn_last_4":                                         "string",
		"person.verification.additional_document.back":              "string",
		"person.verification.additional_document.front":             "string",
		"person.verification.document.back":                         "string",
		"person.verification.document.front":                        "string",
		"pii.id_number":                                             "string",
	}, &Config)
	resource.NewOperationCmd(rTokensCmd.Cmd, "retrieve", "/v1/tokens/{token}", http.MethodGet, map[string]string{}, &Config)
	resource.NewOperationCmd(rTopupsCmd.Cmd, "cancel", "/v1/topups/{topup}/cancel", http.MethodPost, map[string]string{}, &Config)
	resource.NewOperationCmd(rTopupsCmd.Cmd, "create", "/v1/topups", http.MethodPost, map[string]string{
		"amount":               "integer",
		"currency":             "string",
		"description":          "string",
		"source":               "string",
		"statement_descriptor": "string",
		"transfer_group":       "string",
	}, &Config)
	resource.NewOperationCmd(rTopupsCmd.Cmd, "list", "/v1/topups", http.MethodGet, map[string]string{
		"amount":         "integer",
		"created":        "integer",
		"ending_before":  "string",
		"limit":          "integer",
		"starting_after": "string",
		"status":         "string",
	}, &Config)
	resource.NewOperationCmd(rTopupsCmd.Cmd, "retrieve", "/v1/topups/{topup}", http.MethodGet, map[string]string{}, &Config)
	resource.NewOperationCmd(rTopupsCmd.Cmd, "update", "/v1/topups/{topup}", http.MethodPost, map[string]string{
		"description": "string",
	}, &Config)
	resource.NewOperationCmd(rTransferReversalsCmd.Cmd, "create", "/v1/transfers/{id}/reversals", http.MethodPost, map[string]string{
		"amount":                 "integer",
		"description":            "string",
		"refund_application_fee": "boolean",
	}, &Config)
	resource.NewOperationCmd(rTransferReversalsCmd.Cmd, "list", "/v1/transfers/{id}/reversals", http.MethodGet, map[string]string{
		"ending_before":  "string",
		"limit":          "integer",
		"starting_after": "string",
	}, &Config)
	resource.NewOperationCmd(rTransferReversalsCmd.Cmd, "retrieve", "/v1/transfers/{transfer}/reversals/{id}", http.MethodGet, map[string]string{}, &Config)
	resource.NewOperationCmd(rTransferReversalsCmd.Cmd, "update", "/v1/transfers/{transfer}/reversals/{id}", http.MethodPost, map[string]string{}, &Config)
	resource.NewOperationCmd(rTransfersCmd.Cmd, "create", "/v1/transfers", http.MethodPost, map[string]string{
		"amount":             "integer",
		"currency":           "string",
		"description":        "string",
		"destination":        "string",
		"source_transaction": "string",
		"source_type":        "string",
		"transfer_group":     "string",
	}, &Config)
	resource.NewOperationCmd(rTransfersCmd.Cmd, "list", "/v1/transfers", http.MethodGet, map[string]string{
		"created":        "integer",
		"destination":    "string",
		"ending_before":  "string",
		"limit":          "integer",
		"starting_after": "string",
		"transfer_group": "string",
	}, &Config)
	resource.NewOperationCmd(rTransfersCmd.Cmd, "retrieve", "/v1/transfers/{transfer}", http.MethodGet, map[string]string{}, &Config)
	resource.NewOperationCmd(rTransfersCmd.Cmd, "update", "/v1/transfers/{transfer}", http.MethodPost, map[string]string{
		"description": "string",
	}, &Config)
	resource.NewOperationCmd(rUsageRecordSummariesCmd.Cmd, "list", "/v1/subscription_items/{subscription_item}/usage_record_summaries", http.MethodGet, map[string]string{
		"ending_before":  "string",
		"limit":          "integer",
		"starting_after": "string",
	}, &Config)
	resource.NewOperationCmd(rUsageRecordsCmd.Cmd, "create", "/v1/subscription_items/{subscription_item}/usage_records", http.MethodPost, map[string]string{
		"action":    "string",
		"quantity":  "integer",
		"timestamp": "string",
	}, &Config)
	resource.NewOperationCmd(rWebhookEndpointsCmd.Cmd, "create", "/v1/webhook_endpoints", http.MethodPost, map[string]string{
		"api_version":    "string",
		"connect":        "boolean",
		"description":    "string",
		"enabled_events": "array",
		"url":            "string",
	}, &Config)
	resource.NewOperationCmd(rWebhookEndpointsCmd.Cmd, "delete", "/v1/webhook_endpoints/{webhook_endpoint}", http.MethodDelete, map[string]string{}, &Config)
	resource.NewOperationCmd(rWebhookEndpointsCmd.Cmd, "list", "/v1/webhook_endpoints", http.MethodGet, map[string]string{
		"ending_before":  "string",
		"limit":          "integer",
		"starting_after": "string",
	}, &Config)
	resource.NewOperationCmd(rWebhookEndpointsCmd.Cmd, "retrieve", "/v1/webhook_endpoints/{webhook_endpoint}", http.MethodGet, map[string]string{}, &Config)
	resource.NewOperationCmd(rWebhookEndpointsCmd.Cmd, "update", "/v1/webhook_endpoints/{webhook_endpoint}", http.MethodPost, map[string]string{
		"description":    "string",
		"disabled":       "boolean",
		"enabled_events": "array",
		"url":            "string",
	}, &Config)
	resource.NewOperationCmd(rAppsSecretsCmd.Cmd, "create", "/v1/apps/secrets", http.MethodPost, map[string]string{
		"expires_at": "integer",
		"name":       "string",
		"payload":    "string",
		"scope.type": "string",
		"scope.user": "string",
	}, &Config)
	resource.NewOperationCmd(rAppsSecretsCmd.Cmd, "delete_where", "/v1/apps/secrets/delete", http.MethodPost, map[string]string{
		"name":       "string",
		"scope.type": "string",
		"scope.user": "string",
	}, &Config)
	resource.NewOperationCmd(rAppsSecretsCmd.Cmd, "find", "/v1/apps/secrets/find", http.MethodGet, map[string]string{
		"name": "string",
	}, &Config)
	resource.NewOperationCmd(rAppsSecretsCmd.Cmd, "list", "/v1/apps/secrets", http.MethodGet, map[string]string{
		"ending_before":  "string",
		"limit":          "integer",
		"starting_after": "string",
	}, &Config)
	resource.NewOperationCmd(rBillingMeterEventAdjustmentsCmd.Cmd, "create", "/v1/billing/meter_event_adjustments", http.MethodPost, map[string]string{
		"cancel.identifier": "string",
		"event_name":        "string",
		"type":              "string",
	}, &Config)
	resource.NewOperationCmd(rBillingMeterEventSummarysCmd.Cmd, "list", "/v1/billing/meters/{id}/event_summaries", http.MethodGet, map[string]string{
		"customer":              "string",
		"end_time":              "integer",
		"ending_before":         "string",
		"limit":                 "integer",
		"start_time":            "integer",
		"starting_after":        "string",
		"value_grouping_window": "string",
	}, &Config)
	resource.NewOperationCmd(rBillingMeterEventsCmd.Cmd, "create", "/v1/billing/meter_events", http.MethodPost, map[string]string{
		"event_name": "string",
		"identifier": "string",
		"timestamp":  "integer",
	}, &Config)
	resource.NewOperationCmd(rBillingMetersCmd.Cmd, "create", "/v1/billing/meters", http.MethodPost, map[string]string{
		"customer_mapping.event_payload_key": "string",
		"customer_mapping.type":              "string",
		"default_aggregation.formula":        "string",
		"display_name":                       "string",
		"event_name":                         "string",
		"event_time_window":                  "string",
		"value_settings.event_payload_key":   "string",
	}, &Config)
	resource.NewOperationCmd(rBillingMetersCmd.Cmd, "deactivate", "/v1/billing/meters/{id}/deactivate", http.MethodPost, map[string]string{}, &Config)
	resource.NewOperationCmd(rBillingMetersCmd.Cmd, "list", "/v1/billing/meters", http.MethodGet, map[string]string{
		"ending_before":  "string",
		"limit":          "integer",
		"starting_after": "string",
		"status":         "string",
	}, &Config)
	resource.NewOperationCmd(rBillingMetersCmd.Cmd, "reactivate", "/v1/billing/meters/{id}/reactivate", http.MethodPost, map[string]string{}, &Config)
	resource.NewOperationCmd(rBillingMetersCmd.Cmd, "retrieve", "/v1/billing/meters/{id}", http.MethodGet, map[string]string{}, &Config)
	resource.NewOperationCmd(rBillingMetersCmd.Cmd, "update", "/v1/billing/meters/{id}", http.MethodPost, map[string]string{
		"display_name": "string",
	}, &Config)
	resource.NewOperationCmd(rBillingPortalConfigurationsCmd.Cmd, "create", "/v1/billing_portal/configurations", http.MethodPost, map[string]string{
		"business_profile.headline":                                "string",
		"business_profile.privacy_policy_url":                      "string",
		"business_profile.terms_of_service_url":                    "string",
		"default_return_url":                                       "string",
		"features.customer_update.allowed_updates":                 "array",
		"features.customer_update.enabled":                         "boolean",
		"features.invoice_history.enabled":                         "boolean",
		"features.payment_method_update.enabled":                   "boolean",
		"features.subscription_cancel.cancellation_reason.enabled": "boolean",
		"features.subscription_cancel.cancellation_reason.options": "array",
		"features.subscription_cancel.enabled":                     "boolean",
		"features.subscription_cancel.mode":                        "string",
		"features.subscription_cancel.proration_behavior":          "string",
		"features.subscription_update.default_allowed_updates":     "array",
		"features.subscription_update.enabled":                     "boolean",
		"features.subscription_update.proration_behavior":          "string",
		"login_page.enabled":                                       "boolean",
	}, &Config)
	resource.NewOperationCmd(rBillingPortalConfigurationsCmd.Cmd, "list", "/v1/billing_portal/configurations", http.MethodGet, map[string]string{
		"active":         "boolean",
		"ending_before":  "string",
		"is_default":     "boolean",
		"limit":          "integer",
		"starting_after": "string",
	}, &Config)
	resource.NewOperationCmd(rBillingPortalConfigurationsCmd.Cmd, "retrieve", "/v1/billing_portal/configurations/{configuration}", http.MethodGet, map[string]string{}, &Config)
	resource.NewOperationCmd(rBillingPortalConfigurationsCmd.Cmd, "update", "/v1/billing_portal/configurations/{configuration}", http.MethodPost, map[string]string{
		"active":                                                   "boolean",
		"business_profile.headline":                                "string",
		"business_profile.privacy_policy_url":                      "string",
		"business_profile.terms_of_service_url":                    "string",
		"default_return_url":                                       "string",
		"features.customer_update.allowed_updates":                 "array",
		"features.customer_update.enabled":                         "boolean",
		"features.invoice_history.enabled":                         "boolean",
		"features.payment_method_update.enabled":                   "boolean",
		"features.subscription_cancel.cancellation_reason.enabled": "boolean",
		"features.subscription_cancel.cancellation_reason.options": "array",
		"features.subscription_cancel.enabled":                     "boolean",
		"features.subscription_cancel.mode":                        "string",
		"features.subscription_cancel.proration_behavior":          "string",
		"features.subscription_update.default_allowed_updates":     "array",
		"features.subscription_update.enabled":                     "boolean",
		"features.subscription_update.proration_behavior":          "string",
		"login_page.enabled":                                       "boolean",
	}, &Config)
	resource.NewOperationCmd(rBillingPortalSessionsCmd.Cmd, "create", "/v1/billing_portal/sessions", http.MethodPost, map[string]string{
		"configuration": "string",
		"customer":      "string",
		"flow_data.after_completion.hosted_confirmation.custom_message": "string",
		"flow_data.after_completion.redirect.return_url":                "string",
		"flow_data.after_completion.type":                               "string",
		"flow_data.subscription_cancel.retention.coupon_offer.coupon":   "string",
		"flow_data.subscription_cancel.retention.type":                  "string",
		"flow_data.subscription_cancel.subscription":                    "string",
		"flow_data.subscription_update.subscription":                    "string",
		"flow_data.subscription_update_confirm.subscription":            "string",
		"flow_data.type": "string",
		"locale":         "string",
		"on_behalf_of":   "string",
		"return_url":     "string",
	}, &Config)
	resource.NewOperationCmd(rCheckoutSessionsCmd.Cmd, "create", "/v1/checkout/sessions", http.MethodPost, map[string]string{
		"after_expiration.recovery.allow_promotion_codes":            "boolean",
		"after_expiration.recovery.enabled":                          "boolean",
		"allow_promotion_codes":                                      "boolean",
		"automatic_tax.enabled":                                      "boolean",
		"automatic_tax.liability.account":                            "string",
		"automatic_tax.liability.type":                               "string",
		"billing_address_collection":                                 "string",
		"cancel_url":                                                 "string",
		"client_reference_id":                                        "string",
		"consent_collection.payment_method_reuse_agreement.position": "string",
		"consent_collection.promotions":                              "string",
		"consent_collection.terms_of_service":                        "string",
		"currency":                                                   "string",
		"customer":                                                   "string",
		"customer_creation":                                          "string",
		"customer_email":                                             "string",
		"customer_update.address":                                    "string",
		"customer_update.name":                                       "string",
		"customer_update.shipping":                                   "string",
		"expires_at":                                                 "integer",
		"invoice_creation.enabled":                                   "boolean",
		"invoice_creation.invoice_data.account_tax_ids":              "array",
		"invoice_creation.invoice_data.description":                  "string",
		"invoice_creation.invoice_data.footer":                       "string",
		"invoice_creation.invoice_data.issuer.account":               "string",
		"invoice_creation.invoice_data.issuer.type":                  "string",
		"locale": "string",
		"mode":   "string",
		"payment_intent_data.application_fee_amount":                                     "integer",
		"payment_intent_data.capture_method":                                             "string",
		"payment_intent_data.description":                                                "string",
		"payment_intent_data.on_behalf_of":                                               "string",
		"payment_intent_data.receipt_email":                                              "string",
		"payment_intent_data.setup_future_usage":                                         "string",
		"payment_intent_data.shipping.address.city":                                      "string",
		"payment_intent_data.shipping.address.country":                                   "string",
		"payment_intent_data.shipping.address.line1":                                     "string",
		"payment_intent_data.shipping.address.line2":                                     "string",
		"payment_intent_data.shipping.address.postal_code":                               "string",
		"payment_intent_data.shipping.address.state":                                     "string",
		"payment_intent_data.shipping.carrier":                                           "string",
		"payment_intent_data.shipping.name":                                              "string",
		"payment_intent_data.shipping.phone":                                             "string",
		"payment_intent_data.shipping.tracking_number":                                   "string",
		"payment_intent_data.statement_descriptor":                                       "string",
		"payment_intent_data.statement_descriptor_suffix":                                "string",
		"payment_intent_data.transfer_data.amount":                                       "integer",
		"payment_intent_data.transfer_data.destination":                                  "string",
		"payment_intent_data.transfer_group":                                             "string",
		"payment_method_collection":                                                      "string",
		"payment_method_configuration":                                                   "string",
		"payment_method_data.allow_redisplay":                                            "string",
		"payment_method_options.acss_debit.currency":                                     "string",
		"payment_method_options.acss_debit.mandate_options.custom_mandate_url":           "string",
		"payment_method_options.acss_debit.mandate_options.default_for":                  "array",
		"payment_method_options.acss_debit.mandate_options.interval_description":         "string",
		"payment_method_options.acss_debit.mandate_options.payment_schedule":             "string",
		"payment_method_options.acss_debit.mandate_options.transaction_type":             "string",
		"payment_method_options.acss_debit.setup_future_usage":                           "string",
		"payment_method_options.acss_debit.verification_method":                          "string",
		"payment_method_options.affirm.setup_future_usage":                               "string",
		"payment_method_options.afterpay_clearpay.setup_future_usage":                    "string",
		"payment_method_options.alipay.setup_future_usage":                               "string",
		"payment_method_options.amazon_pay.setup_future_usage":                           "string",
		"payment_method_options.au_becs_debit.setup_future_usage":                        "string",
		"payment_method_options.bacs_debit.setup_future_usage":                           "string",
		"payment_method_options.bancontact.setup_future_usage":                           "string",
		"payment_method_options.boleto.expires_after_days":                               "integer",
		"payment_method_options.boleto.setup_future_usage":                               "string",
		"payment_method_options.card.installments.enabled":                               "boolean",
		"payment_method_options.card.request_three_d_secure":                             "string",
		"payment_method_options.card.setup_future_usage":                                 "string",
		"payment_method_options.card.statement_descriptor_suffix_kana":                   "string",
		"payment_method_options.card.statement_descriptor_suffix_kanji":                  "string",
		"payment_method_options.cashapp.setup_future_usage":                              "string",
		"payment_method_options.customer_balance.bank_transfer.eu_bank_transfer.country": "string",
		"payment_method_options.customer_balance.bank_transfer.requested_address_types":  "array",
		"payment_method_options.customer_balance.bank_transfer.type":                     "string",
		"payment_method_options.customer_balance.funding_type":                           "string",
		"payment_method_options.customer_balance.setup_future_usage":                     "string",
		"payment_method_options.eps.setup_future_usage":                                  "string",
		"payment_method_options.fpx.setup_future_usage":                                  "string",
		"payment_method_options.giropay.setup_future_usage":                              "string",
		"payment_method_options.grabpay.setup_future_usage":                              "string",
		"payment_method_options.ideal.setup_future_usage":                                "string",
		"payment_method_options.klarna.setup_future_usage":                               "string",
		"payment_method_options.konbini.expires_after_days":                              "integer",
		"payment_method_options.konbini.setup_future_usage":                              "string",
		"payment_method_options.link.setup_future_usage":                                 "string",
		"payment_method_options.mobilepay.setup_future_usage":                            "string",
		"payment_method_options.oxxo.expires_after_days":                                 "integer",
		"payment_method_options.oxxo.setup_future_usage":                                 "string",
		"payment_method_options.p24.setup_future_usage":                                  "string",
		"payment_method_options.p24.tos_shown_and_accepted":                              "boolean",
		"payment_method_options.paynow.setup_future_usage":                               "string",
		"payment_method_options.paypal.capture_method":                                   "string",
		"payment_method_options.paypal.preferred_locale":                                 "string",
		"payment_method_options.paypal.reference":                                        "string",
		"payment_method_options.paypal.risk_correlation_id":                              "string",
		"payment_method_options.paypal.setup_future_usage":                               "string",
		"payment_method_options.pix.expires_after_seconds":                               "integer",
		"payment_method_options.revolut_pay.setup_future_usage":                          "string",
		"payment_method_options.sepa_debit.setup_future_usage":                           "string",
		"payment_method_options.sofort.setup_future_usage":                               "string",
		"payment_method_options.swish.reference":                                         "string",
		"payment_method_options.us_bank_account.financial_connections.permissions":       "array",
		"payment_method_options.us_bank_account.financial_connections.prefetch":          "array",
		"payment_method_options.us_bank_account.setup_future_usage":                      "string",
		"payment_method_options.us_bank_account.verification_method":                     "string",
		"payment_method_options.wechat_pay.app_id":                                       "string",
		"payment_method_options.wechat_pay.client":                                       "string",
		"payment_method_options.wechat_pay.setup_future_usage":                           "string",
		"payment_method_types":                                                           "array",
		"phone_number_collection.enabled":                                                "boolean",
		"redirect_on_completion":                                                         "string",
		"return_url":                                                                     "string",
		"saved_payment_method_options.allow_redisplay_filters":                           "array",
		"saved_payment_method_options.payment_method_save":                               "string",
		"setup_intent_data.description":                                                  "string",
		"setup_intent_data.on_behalf_of":                                                 "string",
		"shipping_address_collection.allowed_countries":                                  "array",
		"submit_type": "string",
		"subscription_data.application_fee_percent":                            "number",
		"subscription_data.billing_cycle_anchor":                               "integer",
		"subscription_data.default_tax_rates":                                  "array",
		"subscription_data.description":                                        "string",
		"subscription_data.invoice_settings.issuer.account":                    "string",
		"subscription_data.invoice_settings.issuer.type":                       "string",
		"subscription_data.on_behalf_of":                                       "string",
		"subscription_data.proration_behavior":                                 "string",
		"subscription_data.transfer_data.amount_percent":                       "number",
		"subscription_data.transfer_data.destination":                          "string",
		"subscription_data.trial_end":                                          "integer",
		"subscription_data.trial_period_days":                                  "integer",
		"subscription_data.trial_settings.end_behavior.missing_payment_method": "string",
		"success_url":               "string",
		"tax_id_collection.enabled": "boolean",
		"ui_mode":                   "string",
	}, &Config)
	resource.NewOperationCmd(rCheckoutSessionsCmd.Cmd, "expire", "/v1/checkout/sessions/{session}/expire", http.MethodPost, map[string]string{}, &Config)
	resource.NewOperationCmd(rCheckoutSessionsCmd.Cmd, "list", "/v1/checkout/sessions", http.MethodGet, map[string]string{
		"created":        "integer",
		"customer":       "string",
		"ending_before":  "string",
		"limit":          "integer",
		"payment_intent": "string",
		"payment_link":   "string",
		"starting_after": "string",
		"status":         "string",
		"subscription":   "string",
	}, &Config)
	resource.NewOperationCmd(rCheckoutSessionsCmd.Cmd, "list_line_items", "/v1/checkout/sessions/{session}/line_items", http.MethodGet, map[string]string{
		"ending_before":  "string",
		"limit":          "integer",
		"starting_after": "string",
	}, &Config)
	resource.NewOperationCmd(rCheckoutSessionsCmd.Cmd, "retrieve", "/v1/checkout/sessions/{session}", http.MethodGet, map[string]string{}, &Config)
	resource.NewOperationCmd(rClimateOrdersCmd.Cmd, "cancel", "/v1/climate/orders/{order}/cancel", http.MethodPost, map[string]string{}, &Config)
	resource.NewOperationCmd(rClimateOrdersCmd.Cmd, "create", "/v1/climate/orders", http.MethodPost, map[string]string{
		"amount":                  "integer",
		"beneficiary.public_name": "string",
		"currency":                "string",
		"metric_tons":             "string",
		"product":                 "string",
	}, &Config)
	resource.NewOperationCmd(rClimateOrdersCmd.Cmd, "list", "/v1/climate/orders", http.MethodGet, map[string]string{
		"ending_before":  "string",
		"limit":          "integer",
		"starting_after": "string",
	}, &Config)
	resource.NewOperationCmd(rClimateOrdersCmd.Cmd, "retrieve", "/v1/climate/orders/{order}", http.MethodGet, map[string]string{}, &Config)
	resource.NewOperationCmd(rClimateOrdersCmd.Cmd, "update", "/v1/climate/orders/{order}", http.MethodPost, map[string]string{}, &Config)
	resource.NewOperationCmd(rClimateProductsCmd.Cmd, "list", "/v1/climate/products", http.MethodGet, map[string]string{
		"ending_before":  "string",
		"limit":          "integer",
		"starting_after": "string",
	}, &Config)
	resource.NewOperationCmd(rClimateProductsCmd.Cmd, "retrieve", "/v1/climate/products/{product}", http.MethodGet, map[string]string{}, &Config)
	resource.NewOperationCmd(rClimateSuppliersCmd.Cmd, "list", "/v1/climate/suppliers", http.MethodGet, map[string]string{
		"ending_before":  "string",
		"limit":          "integer",
		"starting_after": "string",
	}, &Config)
	resource.NewOperationCmd(rClimateSuppliersCmd.Cmd, "retrieve", "/v1/climate/suppliers/{supplier}", http.MethodGet, map[string]string{}, &Config)
	resource.NewOperationCmd(rEntitlementsActiveEntitlementsCmd.Cmd, "list", "/v1/entitlements/active_entitlements", http.MethodGet, map[string]string{
		"customer":       "string",
		"ending_before":  "string",
		"limit":          "integer",
		"starting_after": "string",
	}, &Config)
	resource.NewOperationCmd(rEntitlementsActiveEntitlementsCmd.Cmd, "retrieve", "/v1/entitlements/active_entitlements/{id}", http.MethodGet, map[string]string{}, &Config)
	resource.NewOperationCmd(rEntitlementsFeaturesCmd.Cmd, "create", "/v1/entitlements/features", http.MethodPost, map[string]string{
		"lookup_key": "string",
		"name":       "string",
	}, &Config)
	resource.NewOperationCmd(rEntitlementsFeaturesCmd.Cmd, "list", "/v1/entitlements/features", http.MethodGet, map[string]string{
		"archived":       "boolean",
		"ending_before":  "string",
		"limit":          "integer",
		"lookup_key":     "string",
		"starting_after": "string",
	}, &Config)
	resource.NewOperationCmd(rEntitlementsFeaturesCmd.Cmd, "retrieve", "/v1/entitlements/features/{id}", http.MethodGet, map[string]string{}, &Config)
	resource.NewOperationCmd(rEntitlementsFeaturesCmd.Cmd, "update", "/v1/entitlements/features/{id}", http.MethodPost, map[string]string{
		"active": "boolean",
		"name":   "string",
	}, &Config)
	resource.NewOperationCmd(rFinancialConnectionsAccountsCmd.Cmd, "disconnect", "/v1/financial_connections/accounts/{account}/disconnect", http.MethodPost, map[string]string{}, &Config)
	resource.NewOperationCmd(rFinancialConnectionsAccountsCmd.Cmd, "list", "/v1/financial_connections/accounts", http.MethodGet, map[string]string{
		"ending_before":  "string",
		"limit":          "integer",
		"session":        "string",
		"starting_after": "string",
	}, &Config)
	resource.NewOperationCmd(rFinancialConnectionsAccountsCmd.Cmd, "list_owners", "/v1/financial_connections/accounts/{account}/owners", http.MethodGet, map[string]string{
		"ending_before":  "string",
		"limit":          "integer",
		"ownership":      "string",
		"starting_after": "string",
	}, &Config)
	resource.NewOperationCmd(rFinancialConnectionsAccountsCmd.Cmd, "refresh", "/v1/financial_connections/accounts/{account}/refresh", http.MethodPost, map[string]string{
		"features": "array",
	}, &Config)
	resource.NewOperationCmd(rFinancialConnectionsAccountsCmd.Cmd, "retrieve", "/v1/financial_connections/accounts/{account}", http.MethodGet, map[string]string{}, &Config)
	resource.NewOperationCmd(rFinancialConnectionsAccountsCmd.Cmd, "subscribe", "/v1/financial_connections/accounts/{account}/subscribe", http.MethodPost, map[string]string{
		"features": "array",
	}, &Config)
	resource.NewOperationCmd(rFinancialConnectionsAccountsCmd.Cmd, "unsubscribe", "/v1/financial_connections/accounts/{account}/unsubscribe", http.MethodPost, map[string]string{
		"features": "array",
	}, &Config)
	resource.NewOperationCmd(rFinancialConnectionsSessionsCmd.Cmd, "create", "/v1/financial_connections/sessions", http.MethodPost, map[string]string{
		"account_holder.account":  "string",
		"account_holder.customer": "string",
		"account_holder.type":     "string",
		"filters.countries":       "array",
		"permissions":             "array",
		"prefetch":                "array",
		"return_url":              "string",
	}, &Config)
	resource.NewOperationCmd(rFinancialConnectionsSessionsCmd.Cmd, "retrieve", "/v1/financial_connections/sessions/{session}", http.MethodGet, map[string]string{}, &Config)
	resource.NewOperationCmd(rFinancialConnectionsTransactionsCmd.Cmd, "list", "/v1/financial_connections/transactions", http.MethodGet, map[string]string{
		"account":        "string",
		"ending_before":  "string",
		"limit":          "integer",
		"starting_after": "string",
		"transacted_at":  "integer",
	}, &Config)
	resource.NewOperationCmd(rFinancialConnectionsTransactionsCmd.Cmd, "retrieve", "/v1/financial_connections/transactions/{transaction}", http.MethodGet, map[string]string{}, &Config)
	resource.NewOperationCmd(rForwardingRequestsCmd.Cmd, "create", "/v1/forwarding/requests", http.MethodPost, map[string]string{
		"payment_method": "string",
		"replacements":   "array",
		"request.body":   "string",
		"url":            "string",
	}, &Config)
	resource.NewOperationCmd(rForwardingRequestsCmd.Cmd, "list", "/v1/forwarding/requests", http.MethodGet, map[string]string{
		"ending_before":  "string",
		"limit":          "integer",
		"starting_after": "string",
	}, &Config)
	resource.NewOperationCmd(rForwardingRequestsCmd.Cmd, "retrieve", "/v1/forwarding/requests/{id}", http.MethodGet, map[string]string{}, &Config)
	resource.NewOperationCmd(rIdentityVerificationReportsCmd.Cmd, "list", "/v1/identity/verification_reports", http.MethodGet, map[string]string{
		"client_reference_id":  "string",
		"created":              "integer",
		"ending_before":        "string",
		"limit":                "integer",
		"starting_after":       "string",
		"type":                 "string",
		"verification_session": "string",
	}, &Config)
	resource.NewOperationCmd(rIdentityVerificationReportsCmd.Cmd, "retrieve", "/v1/identity/verification_reports/{report}", http.MethodGet, map[string]string{}, &Config)
	resource.NewOperationCmd(rIdentityVerificationSessionsCmd.Cmd, "cancel", "/v1/identity/verification_sessions/{session}/cancel", http.MethodPost, map[string]string{}, &Config)
	resource.NewOperationCmd(rIdentityVerificationSessionsCmd.Cmd, "create", "/v1/identity/verification_sessions", http.MethodPost, map[string]string{
		"client_reference_id":    "string",
		"provided_details.email": "string",
		"provided_details.phone": "string",
		"return_url":             "string",
		"type":                   "string",
		"verification_flow":      "string",
	}, &Config)
	resource.NewOperationCmd(rIdentityVerificationSessionsCmd.Cmd, "list", "/v1/identity/verification_sessions", http.MethodGet, map[string]string{
		"client_reference_id": "string",
		"created":             "integer",
		"ending_before":       "string",
		"limit":               "integer",
		"starting_after":      "string",
		"status":              "string",
	}, &Config)
	resource.NewOperationCmd(rIdentityVerificationSessionsCmd.Cmd, "redact", "/v1/identity/verification_sessions/{session}/redact", http.MethodPost, map[string]string{}, &Config)
	resource.NewOperationCmd(rIdentityVerificationSessionsCmd.Cmd, "retrieve", "/v1/identity/verification_sessions/{session}", http.MethodGet, map[string]string{}, &Config)
	resource.NewOperationCmd(rIdentityVerificationSessionsCmd.Cmd, "update", "/v1/identity/verification_sessions/{session}", http.MethodPost, map[string]string{
		"provided_details.email": "string",
		"provided_details.phone": "string",
		"type":                   "string",
	}, &Config)
	resource.NewOperationCmd(rIssuingAuthorizationsCmd.Cmd, "list", "/v1/issuing/authorizations", http.MethodGet, map[string]string{
		"card":           "string",
		"cardholder":     "string",
		"created":        "integer",
		"ending_before":  "string",
		"limit":          "integer",
		"starting_after": "string",
		"status":         "string",
	}, &Config)
	resource.NewOperationCmd(rIssuingAuthorizationsCmd.Cmd, "retrieve", "/v1/issuing/authorizations/{authorization}", http.MethodGet, map[string]string{}, &Config)
	resource.NewOperationCmd(rIssuingAuthorizationsCmd.Cmd, "update", "/v1/issuing/authorizations/{authorization}", http.MethodPost, map[string]string{}, &Config)
	resource.NewOperationCmd(rIssuingAuthorizationsTestHelpersCmd.Cmd, "capture", "/v1/test_helpers/issuing/authorizations/{authorization}/capture", http.MethodPost, map[string]string{
		"capture_amount":                          "integer",
		"close_authorization":                     "boolean",
		"purchase_details.flight.departure_at":    "integer",
		"purchase_details.flight.passenger_name":  "string",
		"purchase_details.flight.refundable":      "boolean",
		"purchase_details.flight.travel_agency":   "string",
		"purchase_details.fuel.type":              "string",
		"purchase_details.fuel.unit":              "string",
		"purchase_details.fuel.unit_cost_decimal": "string",
		"purchase_details.fuel.volume_decimal":    "string",
		"purchase_details.lodging.check_in_at":    "integer",
		"purchase_details.lodging.nights":         "integer",
		"purchase_details.reference":              "string",
	}, &Config)
	resource.NewOperationCmd(rIssuingAuthorizationsTestHelpersCmd.Cmd, "create", "/v1/test_helpers/issuing/authorizations", http.MethodPost, map[string]string{
		"amount":                                      "integer",
		"amount_details.atm_fee":                      "integer",
		"amount_details.cashback_amount":              "integer",
		"authorization_method":                        "string",
		"card":                                        "string",
		"currency":                                    "string",
		"is_amount_controllable":                      "boolean",
		"merchant_data.category":                      "string",
		"merchant_data.city":                          "string",
		"merchant_data.country":                       "string",
		"merchant_data.name":                          "string",
		"merchant_data.network_id":                    "string",
		"merchant_data.postal_code":                   "string",
		"merchant_data.state":                         "string",
		"merchant_data.terminal_id":                   "string",
		"merchant_data.url":                           "string",
		"network_data.acquiring_institution_id":       "string",
		"verification_data.address_line1_check":       "string",
		"verification_data.address_postal_code_check": "string",
		"verification_data.authentication_exemption.claimed_by": "string",
		"verification_data.authentication_exemption.type":       "string",
		"verification_data.cvc_check":                           "string",
		"verification_data.expiry_check":                        "string",
		"verification_data.three_d_secure.result":               "string",
		"wallet": "string",
	}, &Config)
	resource.NewOperationCmd(rIssuingAuthorizationsTestHelpersCmd.Cmd, "expire", "/v1/test_helpers/issuing/authorizations/{authorization}/expire", http.MethodPost, map[string]string{}, &Config)
	resource.NewOperationCmd(rIssuingAuthorizationsTestHelpersCmd.Cmd, "increment", "/v1/test_helpers/issuing/authorizations/{authorization}/increment", http.MethodPost, map[string]string{
		"increment_amount":       "integer",
		"is_amount_controllable": "boolean",
	}, &Config)
	resource.NewOperationCmd(rIssuingAuthorizationsTestHelpersCmd.Cmd, "reverse", "/v1/test_helpers/issuing/authorizations/{authorization}/reverse", http.MethodPost, map[string]string{
		"reverse_amount": "integer",
	}, &Config)
	resource.NewOperationCmd(rIssuingCardholdersCmd.Cmd, "create", "/v1/issuing/cardholders", http.MethodPost, map[string]string{
		"billing.address.city":        "string",
		"billing.address.country":     "string",
		"billing.address.line1":       "string",
		"billing.address.line2":       "string",
		"billing.address.postal_code": "string",
		"billing.address.state":       "string",
		"company.tax_id":              "string",
		"email":                       "string",
		"individual.card_issuing.user_terms_acceptance.date":       "integer",
		"individual.card_issuing.user_terms_acceptance.ip":         "string",
		"individual.card_issuing.user_terms_acceptance.user_agent": "string",
		"individual.dob.day":                           "integer",
		"individual.dob.month":                         "integer",
		"individual.dob.year":                          "integer",
		"individual.first_name":                        "string",
		"individual.last_name":                         "string",
		"individual.verification.document.back":        "string",
		"individual.verification.document.front":       "string",
		"name":                                         "string",
		"phone_number":                                 "string",
		"preferred_locales":                            "array",
		"spending_controls.allowed_categories":         "array",
		"spending_controls.allowed_merchant_countries": "array",
		"spending_controls.blocked_categories":         "array",
		"spending_controls.blocked_merchant_countries": "array",
		"spending_controls.spending_limits_currency":   "string",
		"status": "string",
		"type":   "string",
	}, &Config)
	resource.NewOperationCmd(rIssuingCardholdersCmd.Cmd, "list", "/v1/issuing/cardholders", http.MethodGet, map[string]string{
		"created":        "integer",
		"email":          "string",
		"ending_before":  "string",
		"limit":          "integer",
		"phone_number":   "string",
		"starting_after": "string",
		"status":         "string",
		"type":           "string",
	}, &Config)
	resource.NewOperationCmd(rIssuingCardholdersCmd.Cmd, "retrieve", "/v1/issuing/cardholders/{cardholder}", http.MethodGet, map[string]string{}, &Config)
	resource.NewOperationCmd(rIssuingCardholdersCmd.Cmd, "update", "/v1/issuing/cardholders/{cardholder}", http.MethodPost, map[string]string{
		"billing.address.city":        "string",
		"billing.address.country":     "string",
		"billing.address.line1":       "string",
		"billing.address.line2":       "string",
		"billing.address.postal_code": "string",
		"billing.address.state":       "string",
		"company.tax_id":              "string",
		"email":                       "string",
		"individual.card_issuing.user_terms_acceptance.date":       "integer",
		"individual.card_issuing.user_terms_acceptance.ip":         "string",
		"individual.card_issuing.user_terms_acceptance.user_agent": "string",
		"individual.dob.day":                           "integer",
		"individual.dob.month":                         "integer",
		"individual.dob.year":                          "integer",
		"individual.first_name":                        "string",
		"individual.last_name":                         "string",
		"individual.verification.document.back":        "string",
		"individual.verification.document.front":       "string",
		"phone_number":                                 "string",
		"preferred_locales":                            "array",
		"spending_controls.allowed_categories":         "array",
		"spending_controls.allowed_merchant_countries": "array",
		"spending_controls.blocked_categories":         "array",
		"spending_controls.blocked_merchant_countries": "array",
		"spending_controls.spending_limits_currency":   "string",
		"status": "string",
	}, &Config)
	resource.NewOperationCmd(rIssuingCardsCmd.Cmd, "create", "/v1/issuing/cards", http.MethodPost, map[string]string{
		"cardholder":                                   "string",
		"currency":                                     "string",
		"financial_account":                            "string",
		"personalization_design":                       "string",
		"pin.encrypted_number":                         "string",
		"replacement_for":                              "string",
		"replacement_reason":                           "string",
		"second_line":                                  "string",
		"shipping.address.city":                        "string",
		"shipping.address.country":                     "string",
		"shipping.address.line1":                       "string",
		"shipping.address.line2":                       "string",
		"shipping.address.postal_code":                 "string",
		"shipping.address.state":                       "string",
		"shipping.customs.eori_number":                 "string",
		"shipping.name":                                "string",
		"shipping.phone_number":                        "string",
		"shipping.require_signature":                   "boolean",
		"shipping.service":                             "string",
		"shipping.type":                                "string",
		"spending_controls.allowed_categories":         "array",
		"spending_controls.allowed_merchant_countries": "array",
		"spending_controls.blocked_categories":         "array",
		"spending_controls.blocked_merchant_countries": "array",
		"status": "string",
		"type":   "string",
	}, &Config)
	resource.NewOperationCmd(rIssuingCardsCmd.Cmd, "list", "/v1/issuing/cards", http.MethodGet, map[string]string{
		"cardholder":             "string",
		"created":                "integer",
		"ending_before":          "string",
		"exp_month":              "integer",
		"exp_year":               "integer",
		"last4":                  "string",
		"limit":                  "integer",
		"personalization_design": "string",
		"starting_after":         "string",
		"status":                 "string",
		"type":                   "string",
	}, &Config)
	resource.NewOperationCmd(rIssuingCardsCmd.Cmd, "retrieve", "/v1/issuing/cards/{card}", http.MethodGet, map[string]string{}, &Config)
	resource.NewOperationCmd(rIssuingCardsCmd.Cmd, "update", "/v1/issuing/cards/{card}", http.MethodPost, map[string]string{
		"cancellation_reason":                          "string",
		"personalization_design":                       "string",
		"pin.encrypted_number":                         "string",
		"spending_controls.allowed_categories":         "array",
		"spending_controls.allowed_merchant_countries": "array",
		"spending_controls.blocked_categories":         "array",
		"spending_controls.blocked_merchant_countries": "array",
		"status": "string",
	}, &Config)
	resource.NewOperationCmd(rIssuingCardsTestHelpersCmd.Cmd, "deliver_card", "/v1/test_helpers/issuing/cards/{card}/shipping/deliver", http.MethodPost, map[string]string{}, &Config)
	resource.NewOperationCmd(rIssuingCardsTestHelpersCmd.Cmd, "fail_card", "/v1/test_helpers/issuing/cards/{card}/shipping/fail", http.MethodPost, map[string]string{}, &Config)
	resource.NewOperationCmd(rIssuingCardsTestHelpersCmd.Cmd, "return_card", "/v1/test_helpers/issuing/cards/{card}/shipping/return", http.MethodPost, map[string]string{}, &Config)
	resource.NewOperationCmd(rIssuingCardsTestHelpersCmd.Cmd, "ship_card", "/v1/test_helpers/issuing/cards/{card}/shipping/ship", http.MethodPost, map[string]string{}, &Config)
	resource.NewOperationCmd(rIssuingDisputesCmd.Cmd, "create", "/v1/issuing/disputes", http.MethodPost, map[string]string{
		"amount":                  "integer",
		"evidence.reason":         "string",
		"transaction":             "string",
		"treasury.received_debit": "string",
	}, &Config)
	resource.NewOperationCmd(rIssuingDisputesCmd.Cmd, "list", "/v1/issuing/disputes", http.MethodGet, map[string]string{
		"created":        "integer",
		"ending_before":  "string",
		"limit":          "integer",
		"starting_after": "string",
		"status":         "string",
		"transaction":    "string",
	}, &Config)
	resource.NewOperationCmd(rIssuingDisputesCmd.Cmd, "retrieve", "/v1/issuing/disputes/{dispute}", http.MethodGet, map[string]string{}, &Config)
	resource.NewOperationCmd(rIssuingDisputesCmd.Cmd, "submit", "/v1/issuing/disputes/{dispute}/submit", http.MethodPost, map[string]string{}, &Config)
	resource.NewOperationCmd(rIssuingDisputesCmd.Cmd, "update", "/v1/issuing/disputes/{dispute}", http.MethodPost, map[string]string{
		"amount":          "integer",
		"evidence.reason": "string",
	}, &Config)
	resource.NewOperationCmd(rIssuingPersonalizationDesignsCmd.Cmd, "create", "/v1/issuing/personalization_designs", http.MethodPost, map[string]string{
		"card_logo":                 "string",
		"carrier_text.footer_body":  "string",
		"carrier_text.footer_title": "string",
		"carrier_text.header_body":  "string",
		"carrier_text.header_title": "string",
		"lookup_key":                "string",
		"name":                      "string",
		"physical_bundle":           "string",
		"preferences.is_default":    "boolean",
		"transfer_lookup_key":       "boolean",
	}, &Config)
	resource.NewOperationCmd(rIssuingPersonalizationDesignsCmd.Cmd, "list", "/v1/issuing/personalization_designs", http.MethodGet, map[string]string{
		"ending_before":  "string",
		"limit":          "integer",
		"lookup_keys":    "array",
		"starting_after": "string",
		"status":         "string",
	}, &Config)
	resource.NewOperationCmd(rIssuingPersonalizationDesignsCmd.Cmd, "retrieve", "/v1/issuing/personalization_designs/{personalization_design}", http.MethodGet, map[string]string{}, &Config)
	resource.NewOperationCmd(rIssuingPersonalizationDesignsCmd.Cmd, "update", "/v1/issuing/personalization_designs/{personalization_design}", http.MethodPost, map[string]string{
		"card_logo":              "string",
		"lookup_key":             "string",
		"name":                   "string",
		"physical_bundle":        "string",
		"preferences.is_default": "boolean",
		"transfer_lookup_key":    "boolean",
	}, &Config)
	resource.NewOperationCmd(rIssuingPersonalizationDesignsTestHelpersCmd.Cmd, "activate", "/v1/test_helpers/issuing/personalization_designs/{personalization_design}/activate", http.MethodPost, map[string]string{}, &Config)
	resource.NewOperationCmd(rIssuingPersonalizationDesignsTestHelpersCmd.Cmd, "deactivate", "/v1/test_helpers/issuing/personalization_designs/{personalization_design}/deactivate", http.MethodPost, map[string]string{}, &Config)
	resource.NewOperationCmd(rIssuingPersonalizationDesignsTestHelpersCmd.Cmd, "reject", "/v1/test_helpers/issuing/personalization_designs/{personalization_design}/reject", http.MethodPost, map[string]string{
		"rejection_reasons.card_logo":    "array",
		"rejection_reasons.carrier_text": "array",
	}, &Config)
	resource.NewOperationCmd(rIssuingPhysicalBundlesCmd.Cmd, "list", "/v1/issuing/physical_bundles", http.MethodGet, map[string]string{
		"ending_before":  "string",
		"limit":          "integer",
		"starting_after": "string",
		"status":         "string",
		"type":           "string",
	}, &Config)
	resource.NewOperationCmd(rIssuingPhysicalBundlesCmd.Cmd, "retrieve", "/v1/issuing/physical_bundles/{physical_bundle}", http.MethodGet, map[string]string{}, &Config)
	resource.NewOperationCmd(rIssuingTokensCmd.Cmd, "list", "/v1/issuing/tokens", http.MethodGet, map[string]string{
		"card":           "string",
		"created":        "integer",
		"ending_before":  "string",
		"limit":          "integer",
		"starting_after": "string",
		"status":         "string",
	}, &Config)
	resource.NewOperationCmd(rIssuingTokensCmd.Cmd, "retrieve", "/v1/issuing/tokens/{token}", http.MethodGet, map[string]string{}, &Config)
	resource.NewOperationCmd(rIssuingTokensCmd.Cmd, "update", "/v1/issuing/tokens/{token}", http.MethodPost, map[string]string{
		"status": "string",
	}, &Config)
	resource.NewOperationCmd(rIssuingTransactionsCmd.Cmd, "list", "/v1/issuing/transactions", http.MethodGet, map[string]string{
		"card":           "string",
		"cardholder":     "string",
		"created":        "integer",
		"ending_before":  "string",
		"limit":          "integer",
		"starting_after": "string",
		"type":           "string",
	}, &Config)
	resource.NewOperationCmd(rIssuingTransactionsCmd.Cmd, "retrieve", "/v1/issuing/transactions/{transaction}", http.MethodGet, map[string]string{}, &Config)
	resource.NewOperationCmd(rIssuingTransactionsCmd.Cmd, "update", "/v1/issuing/transactions/{transaction}", http.MethodPost, map[string]string{}, &Config)
	resource.NewOperationCmd(rIssuingTransactionsTestHelpersCmd.Cmd, "create_force_capture", "/v1/test_helpers/issuing/transactions/create_force_capture", http.MethodPost, map[string]string{
		"amount":                                  "integer",
		"card":                                    "string",
		"currency":                                "string",
		"merchant_data.category":                  "string",
		"merchant_data.city":                      "string",
		"merchant_data.country":                   "string",
		"merchant_data.name":                      "string",
		"merchant_data.network_id":                "string",
		"merchant_data.postal_code":               "string",
		"merchant_data.state":                     "string",
		"merchant_data.terminal_id":               "string",
		"merchant_data.url":                       "string",
		"purchase_details.flight.departure_at":    "integer",
		"purchase_details.flight.passenger_name":  "string",
		"purchase_details.flight.refundable":      "boolean",
		"purchase_details.flight.travel_agency":   "string",
		"purchase_details.fuel.type":              "string",
		"purchase_details.fuel.unit":              "string",
		"purchase_details.fuel.unit_cost_decimal": "string",
		"purchase_details.fuel.volume_decimal":    "string",
		"purchase_details.lodging.check_in_at":    "integer",
		"purchase_details.lodging.nights":         "integer",
		"purchase_details.reference":              "string",
	}, &Config)
	resource.NewOperationCmd(rIssuingTransactionsTestHelpersCmd.Cmd, "create_unlinked_refund", "/v1/test_helpers/issuing/transactions/create_unlinked_refund", http.MethodPost, map[string]string{
		"amount":                                  "integer",
		"card":                                    "string",
		"currency":                                "string",
		"merchant_data.category":                  "string",
		"merchant_data.city":                      "string",
		"merchant_data.country":                   "string",
		"merchant_data.name":                      "string",
		"merchant_data.network_id":                "string",
		"merchant_data.postal_code":               "string",
		"merchant_data.state":                     "string",
		"merchant_data.terminal_id":               "string",
		"merchant_data.url":                       "string",
		"purchase_details.flight.departure_at":    "integer",
		"purchase_details.flight.passenger_name":  "string",
		"purchase_details.flight.refundable":      "boolean",
		"purchase_details.flight.travel_agency":   "string",
		"purchase_details.fuel.type":              "string",
		"purchase_details.fuel.unit":              "string",
		"purchase_details.fuel.unit_cost_decimal": "string",
		"purchase_details.fuel.volume_decimal":    "string",
		"purchase_details.lodging.check_in_at":    "integer",
		"purchase_details.lodging.nights":         "integer",
		"purchase_details.reference":              "string",
	}, &Config)
	resource.NewOperationCmd(rIssuingTransactionsTestHelpersCmd.Cmd, "refund", "/v1/test_helpers/issuing/transactions/{transaction}/refund", http.MethodPost, map[string]string{
		"refund_amount": "integer",
	}, &Config)
	resource.NewOperationCmd(rRadarEarlyFraudWarningsCmd.Cmd, "list", "/v1/radar/early_fraud_warnings", http.MethodGet, map[string]string{
		"charge":         "string",
		"created":        "integer",
		"ending_before":  "string",
		"limit":          "integer",
		"payment_intent": "string",
		"starting_after": "string",
	}, &Config)
	resource.NewOperationCmd(rRadarEarlyFraudWarningsCmd.Cmd, "retrieve", "/v1/radar/early_fraud_warnings/{early_fraud_warning}", http.MethodGet, map[string]string{}, &Config)
	resource.NewOperationCmd(rRadarValueListItemsCmd.Cmd, "create", "/v1/radar/value_list_items", http.MethodPost, map[string]string{
		"value":      "string",
		"value_list": "string",
	}, &Config)
	resource.NewOperationCmd(rRadarValueListItemsCmd.Cmd, "delete", "/v1/radar/value_list_items/{item}", http.MethodDelete, map[string]string{}, &Config)
	resource.NewOperationCmd(rRadarValueListItemsCmd.Cmd, "list", "/v1/radar/value_list_items", http.MethodGet, map[string]string{
		"created":        "integer",
		"ending_before":  "string",
		"limit":          "integer",
		"starting_after": "string",
		"value":          "string",
		"value_list":     "string",
	}, &Config)
	resource.NewOperationCmd(rRadarValueListItemsCmd.Cmd, "retrieve", "/v1/radar/value_list_items/{item}", http.MethodGet, map[string]string{}, &Config)
	resource.NewOperationCmd(rRadarValueListsCmd.Cmd, "create", "/v1/radar/value_lists", http.MethodPost, map[string]string{
		"alias":     "string",
		"item_type": "string",
		"name":      "string",
	}, &Config)
	resource.NewOperationCmd(rRadarValueListsCmd.Cmd, "delete", "/v1/radar/value_lists/{value_list}", http.MethodDelete, map[string]string{}, &Config)
	resource.NewOperationCmd(rRadarValueListsCmd.Cmd, "list", "/v1/radar/value_lists", http.MethodGet, map[string]string{
		"alias":          "string",
		"contains":       "string",
		"created":        "integer",
		"ending_before":  "string",
		"limit":          "integer",
		"starting_after": "string",
	}, &Config)
	resource.NewOperationCmd(rRadarValueListsCmd.Cmd, "retrieve", "/v1/radar/value_lists/{value_list}", http.MethodGet, map[string]string{}, &Config)
	resource.NewOperationCmd(rRadarValueListsCmd.Cmd, "update", "/v1/radar/value_lists/{value_list}", http.MethodPost, map[string]string{
		"alias": "string",
		"name":  "string",
	}, &Config)
	resource.NewOperationCmd(rReportingReportRunsCmd.Cmd, "create", "/v1/reporting/report_runs", http.MethodPost, map[string]string{
		"parameters.columns":            "array",
		"parameters.connected_account":  "string",
		"parameters.currency":           "string",
		"parameters.interval_end":       "integer",
		"parameters.interval_start":     "integer",
		"parameters.payout":             "string",
		"parameters.reporting_category": "string",
		"parameters.timezone":           "string",
		"report_type":                   "string",
	}, &Config)
	resource.NewOperationCmd(rReportingReportRunsCmd.Cmd, "list", "/v1/reporting/report_runs", http.MethodGet, map[string]string{
		"created":        "integer",
		"ending_before":  "string",
		"limit":          "integer",
		"starting_after": "string",
	}, &Config)
	resource.NewOperationCmd(rReportingReportRunsCmd.Cmd, "retrieve", "/v1/reporting/report_runs/{report_run}", http.MethodGet, map[string]string{}, &Config)
	resource.NewOperationCmd(rReportingReportTypesCmd.Cmd, "list", "/v1/reporting/report_types", http.MethodGet, map[string]string{}, &Config)
	resource.NewOperationCmd(rReportingReportTypesCmd.Cmd, "retrieve", "/v1/reporting/report_types/{report_type}", http.MethodGet, map[string]string{}, &Config)
	resource.NewOperationCmd(rTaxCalculationsCmd.Cmd, "create", "/v1/tax/calculations", http.MethodPost, map[string]string{
		"currency":                              "string",
		"customer":                              "string",
		"customer_details.address.city":         "string",
		"customer_details.address.country":      "string",
		"customer_details.address.line1":        "string",
		"customer_details.address.line2":        "string",
		"customer_details.address.postal_code":  "string",
		"customer_details.address.state":        "string",
		"customer_details.address_source":       "string",
		"customer_details.ip_address":           "string",
		"customer_details.taxability_override":  "string",
		"ship_from_details.address.city":        "string",
		"ship_from_details.address.country":     "string",
		"ship_from_details.address.line1":       "string",
		"ship_from_details.address.line2":       "string",
		"ship_from_details.address.postal_code": "string",
		"ship_from_details.address.state":       "string",
		"shipping_cost.amount":                  "integer",
		"shipping_cost.shipping_rate":           "string",
		"shipping_cost.tax_behavior":            "string",
		"shipping_cost.tax_code":                "string",
		"tax_date":                              "integer",
	}, &Config)
	resource.NewOperationCmd(rTaxCalculationsCmd.Cmd, "list_line_items", "/v1/tax/calculations/{calculation}/line_items", http.MethodGet, map[string]string{
		"ending_before":  "string",
		"limit":          "integer",
		"starting_after": "string",
	}, &Config)
	resource.NewOperationCmd(rTaxRegistrationsCmd.Cmd, "create", "/v1/tax/registrations", http.MethodPost, map[string]string{
		"active_from":             "string",
		"country":                 "string",
		"country_options.ae.type": "string",
		"country_options.at.standard.place_of_supply_scheme":  "string",
		"country_options.at.type":                             "string",
		"country_options.au.type":                             "string",
		"country_options.be.standard.place_of_supply_scheme":  "string",
		"country_options.be.type":                             "string",
		"country_options.bg.standard.place_of_supply_scheme":  "string",
		"country_options.bg.type":                             "string",
		"country_options.bh.type":                             "string",
		"country_options.ca.province_standard.province":       "string",
		"country_options.ca.type":                             "string",
		"country_options.ch.type":                             "string",
		"country_options.cl.type":                             "string",
		"country_options.co.type":                             "string",
		"country_options.cy.standard.place_of_supply_scheme":  "string",
		"country_options.cy.type":                             "string",
		"country_options.cz.standard.place_of_supply_scheme":  "string",
		"country_options.cz.type":                             "string",
		"country_options.de.standard.place_of_supply_scheme":  "string",
		"country_options.de.type":                             "string",
		"country_options.dk.standard.place_of_supply_scheme":  "string",
		"country_options.dk.type":                             "string",
		"country_options.ee.standard.place_of_supply_scheme":  "string",
		"country_options.ee.type":                             "string",
		"country_options.eg.type":                             "string",
		"country_options.es.standard.place_of_supply_scheme":  "string",
		"country_options.es.type":                             "string",
		"country_options.fi.standard.place_of_supply_scheme":  "string",
		"country_options.fi.type":                             "string",
		"country_options.fr.standard.place_of_supply_scheme":  "string",
		"country_options.fr.type":                             "string",
		"country_options.gb.type":                             "string",
		"country_options.ge.type":                             "string",
		"country_options.gr.standard.place_of_supply_scheme":  "string",
		"country_options.gr.type":                             "string",
		"country_options.hr.standard.place_of_supply_scheme":  "string",
		"country_options.hr.type":                             "string",
		"country_options.hu.standard.place_of_supply_scheme":  "string",
		"country_options.hu.type":                             "string",
		"country_options.id.type":                             "string",
		"country_options.ie.standard.place_of_supply_scheme":  "string",
		"country_options.ie.type":                             "string",
		"country_options.is.type":                             "string",
		"country_options.it.standard.place_of_supply_scheme":  "string",
		"country_options.it.type":                             "string",
		"country_options.jp.type":                             "string",
		"country_options.ke.type":                             "string",
		"country_options.kr.type":                             "string",
		"country_options.kz.type":                             "string",
		"country_options.lt.standard.place_of_supply_scheme":  "string",
		"country_options.lt.type":                             "string",
		"country_options.lu.standard.place_of_supply_scheme":  "string",
		"country_options.lu.type":                             "string",
		"country_options.lv.standard.place_of_supply_scheme":  "string",
		"country_options.lv.type":                             "string",
		"country_options.mt.standard.place_of_supply_scheme":  "string",
		"country_options.mt.type":                             "string",
		"country_options.mx.type":                             "string",
		"country_options.my.type":                             "string",
		"country_options.ng.type":                             "string",
		"country_options.nl.standard.place_of_supply_scheme":  "string",
		"country_options.nl.type":                             "string",
		"country_options.no.type":                             "string",
		"country_options.nz.type":                             "string",
		"country_options.om.type":                             "string",
		"country_options.pl.standard.place_of_supply_scheme":  "string",
		"country_options.pl.type":                             "string",
		"country_options.pt.standard.place_of_supply_scheme":  "string",
		"country_options.pt.type":                             "string",
		"country_options.ro.standard.place_of_supply_scheme":  "string",
		"country_options.ro.type":                             "string",
		"country_options.sa.type":                             "string",
		"country_options.se.standard.place_of_supply_scheme":  "string",
		"country_options.se.type":                             "string",
		"country_options.sg.type":                             "string",
		"country_options.si.standard.place_of_supply_scheme":  "string",
		"country_options.si.type":                             "string",
		"country_options.sk.standard.place_of_supply_scheme":  "string",
		"country_options.sk.type":                             "string",
		"country_options.th.type":                             "string",
		"country_options.tr.type":                             "string",
		"country_options.us.local_amusement_tax.jurisdiction": "string",
		"country_options.us.local_lease_tax.jurisdiction":     "string",
		"country_options.us.state":                            "string",
		"country_options.us.type":                             "string",
		"country_options.vn.type":                             "string",
		"country_options.za.type":                             "string",
		"expires_at":                                          "integer",
	}, &Config)
	resource.NewOperationCmd(rTaxRegistrationsCmd.Cmd, "list", "/v1/tax/registrations", http.MethodGet, map[string]string{
		"ending_before":  "string",
		"limit":          "integer",
		"starting_after": "string",
		"status":         "string",
	}, &Config)
	resource.NewOperationCmd(rTaxRegistrationsCmd.Cmd, "retrieve", "/v1/tax/registrations/{id}", http.MethodGet, map[string]string{}, &Config)
	resource.NewOperationCmd(rTaxRegistrationsCmd.Cmd, "update", "/v1/tax/registrations/{id}", http.MethodPost, map[string]string{
		"active_from": "string",
		"expires_at":  "string",
	}, &Config)
	resource.NewOperationCmd(rTaxSettingsCmd.Cmd, "retrieve", "/v1/tax/settings", http.MethodGet, map[string]string{}, &Config)
	resource.NewOperationCmd(rTaxSettingsCmd.Cmd, "update", "/v1/tax/settings", http.MethodPost, map[string]string{
		"defaults.tax_behavior":           "string",
		"defaults.tax_code":               "string",
		"head_office.address.city":        "string",
		"head_office.address.country":     "string",
		"head_office.address.line1":       "string",
		"head_office.address.line2":       "string",
		"head_office.address.postal_code": "string",
		"head_office.address.state":       "string",
	}, &Config)
	resource.NewOperationCmd(rTaxTransactionsCmd.Cmd, "create_from_calculation", "/v1/tax/transactions/create_from_calculation", http.MethodPost, map[string]string{
		"calculation": "string",
		"reference":   "string",
	}, &Config)
	resource.NewOperationCmd(rTaxTransactionsCmd.Cmd, "create_reversal", "/v1/tax/transactions/create_reversal", http.MethodPost, map[string]string{
		"flat_amount":              "integer",
		"mode":                     "string",
		"original_transaction":     "string",
		"reference":                "string",
		"shipping_cost.amount":     "integer",
		"shipping_cost.amount_tax": "integer",
	}, &Config)
	resource.NewOperationCmd(rTaxTransactionsCmd.Cmd, "list_line_items", "/v1/tax/transactions/{transaction}/line_items", http.MethodGet, map[string]string{
		"ending_before":  "string",
		"limit":          "integer",
		"starting_after": "string",
	}, &Config)
	resource.NewOperationCmd(rTaxTransactionsCmd.Cmd, "retrieve", "/v1/tax/transactions/{transaction}", http.MethodGet, map[string]string{}, &Config)
	resource.NewOperationCmd(rTerminalConfigurationsCmd.Cmd, "create", "/v1/terminal/configurations", http.MethodPost, map[string]string{
		"bbpos_wisepos_e.splashscreen": "string",
		"name":                         "string",
		"verifone_p400.splashscreen":   "string",
	}, &Config)
	resource.NewOperationCmd(rTerminalConfigurationsCmd.Cmd, "delete", "/v1/terminal/configurations/{configuration}", http.MethodDelete, map[string]string{}, &Config)
	resource.NewOperationCmd(rTerminalConfigurationsCmd.Cmd, "list", "/v1/terminal/configurations", http.MethodGet, map[string]string{
		"ending_before":      "string",
		"is_account_default": "boolean",
		"limit":              "integer",
		"starting_after":     "string",
	}, &Config)
	resource.NewOperationCmd(rTerminalConfigurationsCmd.Cmd, "retrieve", "/v1/terminal/configurations/{configuration}", http.MethodGet, map[string]string{}, &Config)
	resource.NewOperationCmd(rTerminalConfigurationsCmd.Cmd, "update", "/v1/terminal/configurations/{configuration}", http.MethodPost, map[string]string{
		"name": "string",
	}, &Config)
	resource.NewOperationCmd(rTerminalConnectionTokensCmd.Cmd, "create", "/v1/terminal/connection_tokens", http.MethodPost, map[string]string{
		"location": "string",
	}, &Config)
	resource.NewOperationCmd(rTerminalLocationsCmd.Cmd, "create", "/v1/terminal/locations", http.MethodPost, map[string]string{
		"address.city":            "string",
		"address.country":         "string",
		"address.line1":           "string",
		"address.line2":           "string",
		"address.postal_code":     "string",
		"address.state":           "string",
		"configuration_overrides": "string",
		"display_name":            "string",
	}, &Config)
	resource.NewOperationCmd(rTerminalLocationsCmd.Cmd, "delete", "/v1/terminal/locations/{location}", http.MethodDelete, map[string]string{}, &Config)
	resource.NewOperationCmd(rTerminalLocationsCmd.Cmd, "list", "/v1/terminal/locations", http.MethodGet, map[string]string{
		"ending_before":  "string",
		"limit":          "integer",
		"starting_after": "string",
	}, &Config)
	resource.NewOperationCmd(rTerminalLocationsCmd.Cmd, "retrieve", "/v1/terminal/locations/{location}", http.MethodGet, map[string]string{}, &Config)
	resource.NewOperationCmd(rTerminalLocationsCmd.Cmd, "update", "/v1/terminal/locations/{location}", http.MethodPost, map[string]string{
		"address.city":            "string",
		"address.country":         "string",
		"address.line1":           "string",
		"address.line2":           "string",
		"address.postal_code":     "string",
		"address.state":           "string",
		"configuration_overrides": "string",
		"display_name":            "string",
	}, &Config)
	resource.NewOperationCmd(rTerminalReadersCmd.Cmd, "cancel_action", "/v1/terminal/readers/{reader}/cancel_action", http.MethodPost, map[string]string{}, &Config)
	resource.NewOperationCmd(rTerminalReadersCmd.Cmd, "create", "/v1/terminal/readers", http.MethodPost, map[string]string{
		"label":             "string",
		"location":          "string",
		"registration_code": "string",
	}, &Config)
	resource.NewOperationCmd(rTerminalReadersCmd.Cmd, "delete", "/v1/terminal/readers/{reader}", http.MethodDelete, map[string]string{}, &Config)
	resource.NewOperationCmd(rTerminalReadersCmd.Cmd, "list", "/v1/terminal/readers", http.MethodGet, map[string]string{
		"device_type":    "string",
		"ending_before":  "string",
		"limit":          "integer",
		"location":       "string",
		"serial_number":  "string",
		"starting_after": "string",
		"status":         "string",
	}, &Config)
	resource.NewOperationCmd(rTerminalReadersCmd.Cmd, "process_payment_intent", "/v1/terminal/readers/{reader}/process_payment_intent", http.MethodPost, map[string]string{
		"payment_intent": "string",
		"process_config.enable_customer_cancellation": "boolean",
		"process_config.skip_tipping":                 "boolean",
		"process_config.tipping.amount_eligible":      "integer",
	}, &Config)
	resource.NewOperationCmd(rTerminalReadersCmd.Cmd, "process_setup_intent", "/v1/terminal/readers/{reader}/process_setup_intent", http.MethodPost, map[string]string{
		"customer_consent_collected":                  "boolean",
		"process_config.enable_customer_cancellation": "boolean",
		"setup_intent": "string",
	}, &Config)
	resource.NewOperationCmd(rTerminalReadersCmd.Cmd, "refund_payment", "/v1/terminal/readers/{reader}/refund_payment", http.MethodPost, map[string]string{
		"amount":                 "integer",
		"charge":                 "string",
		"payment_intent":         "string",
		"refund_application_fee": "boolean",
		"refund_payment_config.enable_customer_cancellation": "boolean",
		"reverse_transfer": "boolean",
	}, &Config)
	resource.NewOperationCmd(rTerminalReadersCmd.Cmd, "retrieve", "/v1/terminal/readers/{reader}", http.MethodGet, map[string]string{}, &Config)
	resource.NewOperationCmd(rTerminalReadersCmd.Cmd, "set_reader_display", "/v1/terminal/readers/{reader}/set_reader_display", http.MethodPost, map[string]string{
		"cart.currency": "string",
		"cart.tax":      "integer",
		"cart.total":    "integer",
		"type":          "string",
	}, &Config)
	resource.NewOperationCmd(rTerminalReadersCmd.Cmd, "update", "/v1/terminal/readers/{reader}", http.MethodPost, map[string]string{
		"label": "string",
	}, &Config)
	resource.NewOperationCmd(rTerminalReadersTestHelpersCmd.Cmd, "present_payment_method", "/v1/test_helpers/terminal/readers/{reader}/present_payment_method", http.MethodPost, map[string]string{
		"amount_tip":             "integer",
		"card_present.number":    "string",
		"interac_present.number": "string",
		"type":                   "string",
	}, &Config)
	resource.NewOperationCmd(rTestHelpersConfirmationTokensCmd.Cmd, "create", "/v1/test_helpers/confirmation_tokens", http.MethodPost, map[string]string{
		"payment_method": "string",
		"payment_method_data.acss_debit.account_number":                     "string",
		"payment_method_data.acss_debit.institution_number":                 "string",
		"payment_method_data.acss_debit.transit_number":                     "string",
		"payment_method_data.allow_redisplay":                               "string",
		"payment_method_data.au_becs_debit.account_number":                  "string",
		"payment_method_data.au_becs_debit.bsb_number":                      "string",
		"payment_method_data.bacs_debit.account_number":                     "string",
		"payment_method_data.bacs_debit.sort_code":                          "string",
		"payment_method_data.billing_details.email":                         "string",
		"payment_method_data.billing_details.name":                          "string",
		"payment_method_data.billing_details.phone":                         "string",
		"payment_method_data.boleto.tax_id":                                 "string",
		"payment_method_data.eps.bank":                                      "string",
		"payment_method_data.fpx.account_holder_type":                       "string",
		"payment_method_data.fpx.bank":                                      "string",
		"payment_method_data.ideal.bank":                                    "string",
		"payment_method_data.klarna.dob.day":                                "integer",
		"payment_method_data.klarna.dob.month":                              "integer",
		"payment_method_data.klarna.dob.year":                               "integer",
		"payment_method_data.p24.bank":                                      "string",
		"payment_method_data.radar_options.session":                         "string",
		"payment_method_data.sepa_debit.iban":                               "string",
		"payment_method_data.sofort.country":                                "string",
		"payment_method_data.type":                                          "string",
		"payment_method_data.us_bank_account.account_holder_type":           "string",
		"payment_method_data.us_bank_account.account_number":                "string",
		"payment_method_data.us_bank_account.account_type":                  "string",
		"payment_method_data.us_bank_account.financial_connections_account": "string",
		"payment_method_data.us_bank_account.routing_number":                "string",
		"return_url":                   "string",
		"setup_future_usage":           "string",
		"shipping.address.city":        "string",
		"shipping.address.country":     "string",
		"shipping.address.line1":       "string",
		"shipping.address.line2":       "string",
		"shipping.address.postal_code": "string",
		"shipping.address.state":       "string",
		"shipping.name":                "string",
		"shipping.phone":               "string",
	}, &Config)
	resource.NewOperationCmd(rTestHelpersCustomersCmd.Cmd, "fund_cash_balance", "/v1/test_helpers/customers/{customer}/fund_cash_balance", http.MethodPost, map[string]string{
		"amount":    "integer",
		"currency":  "string",
		"reference": "string",
	}, &Config)
	resource.NewOperationCmd(rTestHelpersIssuingAuthorizationsCmd.Cmd, "capture", "/v1/test_helpers/issuing/authorizations/{authorization}/capture", http.MethodPost, map[string]string{
		"capture_amount":                          "integer",
		"close_authorization":                     "boolean",
		"purchase_details.flight.departure_at":    "integer",
		"purchase_details.flight.passenger_name":  "string",
		"purchase_details.flight.refundable":      "boolean",
		"purchase_details.flight.travel_agency":   "string",
		"purchase_details.fuel.type":              "string",
		"purchase_details.fuel.unit":              "string",
		"purchase_details.fuel.unit_cost_decimal": "string",
		"purchase_details.fuel.volume_decimal":    "string",
		"purchase_details.lodging.check_in_at":    "integer",
		"purchase_details.lodging.nights":         "integer",
		"purchase_details.reference":              "string",
	}, &Config)
	resource.NewOperationCmd(rTestHelpersIssuingAuthorizationsCmd.Cmd, "create", "/v1/test_helpers/issuing/authorizations", http.MethodPost, map[string]string{
		"amount":                                      "integer",
		"amount_details.atm_fee":                      "integer",
		"amount_details.cashback_amount":              "integer",
		"authorization_method":                        "string",
		"card":                                        "string",
		"currency":                                    "string",
		"is_amount_controllable":                      "boolean",
		"merchant_data.category":                      "string",
		"merchant_data.city":                          "string",
		"merchant_data.country":                       "string",
		"merchant_data.name":                          "string",
		"merchant_data.network_id":                    "string",
		"merchant_data.postal_code":                   "string",
		"merchant_data.state":                         "string",
		"merchant_data.terminal_id":                   "string",
		"merchant_data.url":                           "string",
		"network_data.acquiring_institution_id":       "string",
		"verification_data.address_line1_check":       "string",
		"verification_data.address_postal_code_check": "string",
		"verification_data.authentication_exemption.claimed_by": "string",
		"verification_data.authentication_exemption.type":       "string",
		"verification_data.cvc_check":                           "string",
		"verification_data.expiry_check":                        "string",
		"verification_data.three_d_secure.result":               "string",
		"wallet": "string",
	}, &Config)
	resource.NewOperationCmd(rTestHelpersIssuingAuthorizationsCmd.Cmd, "expire", "/v1/test_helpers/issuing/authorizations/{authorization}/expire", http.MethodPost, map[string]string{}, &Config)
	resource.NewOperationCmd(rTestHelpersIssuingAuthorizationsCmd.Cmd, "increment", "/v1/test_helpers/issuing/authorizations/{authorization}/increment", http.MethodPost, map[string]string{
		"increment_amount":       "integer",
		"is_amount_controllable": "boolean",
	}, &Config)
	resource.NewOperationCmd(rTestHelpersIssuingAuthorizationsCmd.Cmd, "reverse", "/v1/test_helpers/issuing/authorizations/{authorization}/reverse", http.MethodPost, map[string]string{
		"reverse_amount": "integer",
	}, &Config)
	resource.NewOperationCmd(rTestHelpersIssuingCardsCmd.Cmd, "deliver_card", "/v1/test_helpers/issuing/cards/{card}/shipping/deliver", http.MethodPost, map[string]string{}, &Config)
	resource.NewOperationCmd(rTestHelpersIssuingCardsCmd.Cmd, "fail_card", "/v1/test_helpers/issuing/cards/{card}/shipping/fail", http.MethodPost, map[string]string{}, &Config)
	resource.NewOperationCmd(rTestHelpersIssuingCardsCmd.Cmd, "return_card", "/v1/test_helpers/issuing/cards/{card}/shipping/return", http.MethodPost, map[string]string{}, &Config)
	resource.NewOperationCmd(rTestHelpersIssuingCardsCmd.Cmd, "ship_card", "/v1/test_helpers/issuing/cards/{card}/shipping/ship", http.MethodPost, map[string]string{}, &Config)
	resource.NewOperationCmd(rTestHelpersIssuingPersonalizationDesignsCmd.Cmd, "activate", "/v1/test_helpers/issuing/personalization_designs/{personalization_design}/activate", http.MethodPost, map[string]string{}, &Config)
	resource.NewOperationCmd(rTestHelpersIssuingPersonalizationDesignsCmd.Cmd, "deactivate", "/v1/test_helpers/issuing/personalization_designs/{personalization_design}/deactivate", http.MethodPost, map[string]string{}, &Config)
	resource.NewOperationCmd(rTestHelpersIssuingPersonalizationDesignsCmd.Cmd, "reject", "/v1/test_helpers/issuing/personalization_designs/{personalization_design}/reject", http.MethodPost, map[string]string{
		"rejection_reasons.card_logo":    "array",
		"rejection_reasons.carrier_text": "array",
	}, &Config)
	resource.NewOperationCmd(rTestHelpersIssuingTransactionsCmd.Cmd, "create_force_capture", "/v1/test_helpers/issuing/transactions/create_force_capture", http.MethodPost, map[string]string{
		"amount":                                  "integer",
		"card":                                    "string",
		"currency":                                "string",
		"merchant_data.category":                  "string",
		"merchant_data.city":                      "string",
		"merchant_data.country":                   "string",
		"merchant_data.name":                      "string",
		"merchant_data.network_id":                "string",
		"merchant_data.postal_code":               "string",
		"merchant_data.state":                     "string",
		"merchant_data.terminal_id":               "string",
		"merchant_data.url":                       "string",
		"purchase_details.flight.departure_at":    "integer",
		"purchase_details.flight.passenger_name":  "string",
		"purchase_details.flight.refundable":      "boolean",
		"purchase_details.flight.travel_agency":   "string",
		"purchase_details.fuel.type":              "string",
		"purchase_details.fuel.unit":              "string",
		"purchase_details.fuel.unit_cost_decimal": "string",
		"purchase_details.fuel.volume_decimal":    "string",
		"purchase_details.lodging.check_in_at":    "integer",
		"purchase_details.lodging.nights":         "integer",
		"purchase_details.reference":              "string",
	}, &Config)
	resource.NewOperationCmd(rTestHelpersIssuingTransactionsCmd.Cmd, "create_unlinked_refund", "/v1/test_helpers/issuing/transactions/create_unlinked_refund", http.MethodPost, map[string]string{
		"amount":                                  "integer",
		"card":                                    "string",
		"currency":                                "string",
		"merchant_data.category":                  "string",
		"merchant_data.city":                      "string",
		"merchant_data.country":                   "string",
		"merchant_data.name":                      "string",
		"merchant_data.network_id":                "string",
		"merchant_data.postal_code":               "string",
		"merchant_data.state":                     "string",
		"merchant_data.terminal_id":               "string",
		"merchant_data.url":                       "string",
		"purchase_details.flight.departure_at":    "integer",
		"purchase_details.flight.passenger_name":  "string",
		"purchase_details.flight.refundable":      "boolean",
		"purchase_details.flight.travel_agency":   "string",
		"purchase_details.fuel.type":              "string",
		"purchase_details.fuel.unit":              "string",
		"purchase_details.fuel.unit_cost_decimal": "string",
		"purchase_details.fuel.volume_decimal":    "string",
		"purchase_details.lodging.check_in_at":    "integer",
		"purchase_details.lodging.nights":         "integer",
		"purchase_details.reference":              "string",
	}, &Config)
	resource.NewOperationCmd(rTestHelpersIssuingTransactionsCmd.Cmd, "refund", "/v1/test_helpers/issuing/transactions/{transaction}/refund", http.MethodPost, map[string]string{
		"refund_amount": "integer",
	}, &Config)
	resource.NewOperationCmd(rTestHelpersRefundsCmd.Cmd, "expire", "/v1/test_helpers/refunds/{refund}/expire", http.MethodPost, map[string]string{}, &Config)
	resource.NewOperationCmd(rTestHelpersTerminalReadersCmd.Cmd, "present_payment_method", "/v1/test_helpers/terminal/readers/{reader}/present_payment_method", http.MethodPost, map[string]string{
		"amount_tip":             "integer",
		"card_present.number":    "string",
		"interac_present.number": "string",
		"type":                   "string",
	}, &Config)
	resource.NewOperationCmd(rTestHelpersTestClocksCmd.Cmd, "advance", "/v1/test_helpers/test_clocks/{test_clock}/advance", http.MethodPost, map[string]string{
		"frozen_time": "integer",
	}, &Config)
	resource.NewOperationCmd(rTestHelpersTestClocksCmd.Cmd, "create", "/v1/test_helpers/test_clocks", http.MethodPost, map[string]string{
		"frozen_time": "integer",
		"name":        "string",
	}, &Config)
	resource.NewOperationCmd(rTestHelpersTestClocksCmd.Cmd, "delete", "/v1/test_helpers/test_clocks/{test_clock}", http.MethodDelete, map[string]string{}, &Config)
	resource.NewOperationCmd(rTestHelpersTestClocksCmd.Cmd, "list", "/v1/test_helpers/test_clocks", http.MethodGet, map[string]string{
		"ending_before":  "string",
		"limit":          "integer",
		"starting_after": "string",
	}, &Config)
	resource.NewOperationCmd(rTestHelpersTestClocksCmd.Cmd, "retrieve", "/v1/test_helpers/test_clocks/{test_clock}", http.MethodGet, map[string]string{}, &Config)
	resource.NewOperationCmd(rTestHelpersTreasuryInboundTransfersCmd.Cmd, "fail", "/v1/test_helpers/treasury/inbound_transfers/{id}/fail", http.MethodPost, map[string]string{
		"failure_details.code": "string",
	}, &Config)
	resource.NewOperationCmd(rTestHelpersTreasuryInboundTransfersCmd.Cmd, "return_inbound_transfer", "/v1/test_helpers/treasury/inbound_transfers/{id}/return", http.MethodPost, map[string]string{}, &Config)
	resource.NewOperationCmd(rTestHelpersTreasuryInboundTransfersCmd.Cmd, "succeed", "/v1/test_helpers/treasury/inbound_transfers/{id}/succeed", http.MethodPost, map[string]string{}, &Config)
	resource.NewOperationCmd(rTestHelpersTreasuryOutboundPaymentsCmd.Cmd, "fail", "/v1/test_helpers/treasury/outbound_payments/{id}/fail", http.MethodPost, map[string]string{}, &Config)
	resource.NewOperationCmd(rTestHelpersTreasuryOutboundPaymentsCmd.Cmd, "post", "/v1/test_helpers/treasury/outbound_payments/{id}/post", http.MethodPost, map[string]string{}, &Config)
	resource.NewOperationCmd(rTestHelpersTreasuryOutboundPaymentsCmd.Cmd, "return_outbound_payment", "/v1/test_helpers/treasury/outbound_payments/{id}/return", http.MethodPost, map[string]string{
		"returned_details.code": "string",
	}, &Config)
	resource.NewOperationCmd(rTestHelpersTreasuryOutboundPaymentsCmd.Cmd, "update", "/v1/test_helpers/treasury/outbound_payments/{id}", http.MethodPost, map[string]string{
		"tracking_details.ach.trace_id":          "string",
		"tracking_details.type":                  "string",
		"tracking_details.us_domestic_wire.imad": "string",
		"tracking_details.us_domestic_wire.omad": "string",
	}, &Config)
	resource.NewOperationCmd(rTestHelpersTreasuryOutboundTransfersCmd.Cmd, "fail", "/v1/test_helpers/treasury/outbound_transfers/{outbound_transfer}/fail", http.MethodPost, map[string]string{}, &Config)
	resource.NewOperationCmd(rTestHelpersTreasuryOutboundTransfersCmd.Cmd, "post", "/v1/test_helpers/treasury/outbound_transfers/{outbound_transfer}/post", http.MethodPost, map[string]string{}, &Config)
	resource.NewOperationCmd(rTestHelpersTreasuryOutboundTransfersCmd.Cmd, "return_outbound_transfer", "/v1/test_helpers/treasury/outbound_transfers/{outbound_transfer}/return", http.MethodPost, map[string]string{
		"returned_details.code": "string",
	}, &Config)
	resource.NewOperationCmd(rTestHelpersTreasuryOutboundTransfersCmd.Cmd, "update", "/v1/test_helpers/treasury/outbound_transfers/{outbound_transfer}", http.MethodPost, map[string]string{
		"tracking_details.ach.trace_id":          "string",
		"tracking_details.type":                  "string",
		"tracking_details.us_domestic_wire.imad": "string",
		"tracking_details.us_domestic_wire.omad": "string",
	}, &Config)
	resource.NewOperationCmd(rTestHelpersTreasuryReceivedCreditsCmd.Cmd, "create", "/v1/test_helpers/treasury/received_credits", http.MethodPost, map[string]string{
		"amount":                                 "integer",
		"currency":                               "string",
		"description":                            "string",
		"financial_account":                      "string",
		"initiating_payment_method_details.type": "string",
		"initiating_payment_method_details.us_bank_account.account_holder_name": "string",
		"initiating_payment_method_details.us_bank_account.account_number":      "string",
		"initiating_payment_method_details.us_bank_account.routing_number":      "string",
		"network": "string",
	}, &Config)
	resource.NewOperationCmd(rTestHelpersTreasuryReceivedDebitsCmd.Cmd, "create", "/v1/test_helpers/treasury/received_debits", http.MethodPost, map[string]string{
		"amount":                                 "integer",
		"currency":                               "string",
		"description":                            "string",
		"financial_account":                      "string",
		"initiating_payment_method_details.type": "string",
		"initiating_payment_method_details.us_bank_account.account_holder_name": "string",
		"initiating_payment_method_details.us_bank_account.account_number":      "string",
		"initiating_payment_method_details.us_bank_account.routing_number":      "string",
		"network": "string",
	}, &Config)
	resource.NewOperationCmd(rTreasuryCreditReversalsCmd.Cmd, "create", "/v1/treasury/credit_reversals", http.MethodPost, map[string]string{
		"received_credit": "string",
	}, &Config)
	resource.NewOperationCmd(rTreasuryCreditReversalsCmd.Cmd, "list", "/v1/treasury/credit_reversals", http.MethodGet, map[string]string{
		"ending_before":     "string",
		"financial_account": "string",
		"limit":             "integer",
		"received_credit":   "string",
		"starting_after":    "string",
		"status":            "string",
	}, &Config)
	resource.NewOperationCmd(rTreasuryCreditReversalsCmd.Cmd, "retrieve", "/v1/treasury/credit_reversals/{credit_reversal}", http.MethodGet, map[string]string{}, &Config)
	resource.NewOperationCmd(rTreasuryDebitReversalsCmd.Cmd, "create", "/v1/treasury/debit_reversals", http.MethodPost, map[string]string{
		"received_debit": "string",
	}, &Config)
	resource.NewOperationCmd(rTreasuryDebitReversalsCmd.Cmd, "list", "/v1/treasury/debit_reversals", http.MethodGet, map[string]string{
		"ending_before":     "string",
		"financial_account": "string",
		"limit":             "integer",
		"received_debit":    "string",
		"resolution":        "string",
		"starting_after":    "string",
		"status":            "string",
	}, &Config)
	resource.NewOperationCmd(rTreasuryDebitReversalsCmd.Cmd, "retrieve", "/v1/treasury/debit_reversals/{debit_reversal}", http.MethodGet, map[string]string{}, &Config)
	resource.NewOperationCmd(rTreasuryFinancialAccountsCmd.Cmd, "create", "/v1/treasury/financial_accounts", http.MethodPost, map[string]string{
		"features.card_issuing.requested":                        "boolean",
		"features.deposit_insurance.requested":                   "boolean",
		"features.financial_addresses.aba.requested":             "boolean",
		"features.inbound_transfers.ach.requested":               "boolean",
		"features.intra_stripe_flows.requested":                  "boolean",
		"features.outbound_payments.ach.requested":               "boolean",
		"features.outbound_payments.us_domestic_wire.requested":  "boolean",
		"features.outbound_transfers.ach.requested":              "boolean",
		"features.outbound_transfers.us_domestic_wire.requested": "boolean",
		"platform_restrictions.inbound_flows":                    "string",
		"platform_restrictions.outbound_flows":                   "string",
		"supported_currencies":                                   "array",
	}, &Config)
	resource.NewOperationCmd(rTreasuryFinancialAccountsCmd.Cmd, "list", "/v1/treasury/financial_accounts", http.MethodGet, map[string]string{
		"created":        "integer",
		"ending_before":  "string",
		"limit":          "integer",
		"starting_after": "string",
	}, &Config)
	resource.NewOperationCmd(rTreasuryFinancialAccountsCmd.Cmd, "retrieve", "/v1/treasury/financial_accounts/{financial_account}", http.MethodGet, map[string]string{}, &Config)
	resource.NewOperationCmd(rTreasuryFinancialAccountsCmd.Cmd, "retrieve_features", "/v1/treasury/financial_accounts/{financial_account}/features", http.MethodGet, map[string]string{}, &Config)
	resource.NewOperationCmd(rTreasuryFinancialAccountsCmd.Cmd, "update", "/v1/treasury/financial_accounts/{financial_account}", http.MethodPost, map[string]string{
		"features.card_issuing.requested":                        "boolean",
		"features.deposit_insurance.requested":                   "boolean",
		"features.financial_addresses.aba.requested":             "boolean",
		"features.inbound_transfers.ach.requested":               "boolean",
		"features.intra_stripe_flows.requested":                  "boolean",
		"features.outbound_payments.ach.requested":               "boolean",
		"features.outbound_payments.us_domestic_wire.requested":  "boolean",
		"features.outbound_transfers.ach.requested":              "boolean",
		"features.outbound_transfers.us_domestic_wire.requested": "boolean",
		"platform_restrictions.inbound_flows":                    "string",
		"platform_restrictions.outbound_flows":                   "string",
	}, &Config)
	resource.NewOperationCmd(rTreasuryFinancialAccountsCmd.Cmd, "update_features", "/v1/treasury/financial_accounts/{financial_account}/features", http.MethodPost, map[string]string{
		"card_issuing.requested":                        "boolean",
		"deposit_insurance.requested":                   "boolean",
		"financial_addresses.aba.requested":             "boolean",
		"inbound_transfers.ach.requested":               "boolean",
		"intra_stripe_flows.requested":                  "boolean",
		"outbound_payments.ach.requested":               "boolean",
		"outbound_payments.us_domestic_wire.requested":  "boolean",
		"outbound_transfers.ach.requested":              "boolean",
		"outbound_transfers.us_domestic_wire.requested": "boolean",
	}, &Config)
	resource.NewOperationCmd(rTreasuryInboundTransfersCmd.Cmd, "cancel", "/v1/treasury/inbound_transfers/{inbound_transfer}/cancel", http.MethodPost, map[string]string{}, &Config)
	resource.NewOperationCmd(rTreasuryInboundTransfersCmd.Cmd, "create", "/v1/treasury/inbound_transfers", http.MethodPost, map[string]string{
		"amount":                "integer",
		"currency":              "string",
		"description":           "string",
		"financial_account":     "string",
		"origin_payment_method": "string",
		"statement_descriptor":  "string",
	}, &Config)
	resource.NewOperationCmd(rTreasuryInboundTransfersCmd.Cmd, "list", "/v1/treasury/inbound_transfers", http.MethodGet, map[string]string{
		"ending_before":     "string",
		"financial_account": "string",
		"limit":             "integer",
		"starting_after":    "string",
		"status":            "string",
	}, &Config)
	resource.NewOperationCmd(rTreasuryInboundTransfersCmd.Cmd, "retrieve", "/v1/treasury/inbound_transfers/{id}", http.MethodGet, map[string]string{}, &Config)
	resource.NewOperationCmd(rTreasuryInboundTransfersTestHelpersCmd.Cmd, "fail", "/v1/test_helpers/treasury/inbound_transfers/{id}/fail", http.MethodPost, map[string]string{
		"failure_details.code": "string",
	}, &Config)
	resource.NewOperationCmd(rTreasuryInboundTransfersTestHelpersCmd.Cmd, "return_inbound_transfer", "/v1/test_helpers/treasury/inbound_transfers/{id}/return", http.MethodPost, map[string]string{}, &Config)
	resource.NewOperationCmd(rTreasuryInboundTransfersTestHelpersCmd.Cmd, "succeed", "/v1/test_helpers/treasury/inbound_transfers/{id}/succeed", http.MethodPost, map[string]string{}, &Config)
	resource.NewOperationCmd(rTreasuryOutboundPaymentsCmd.Cmd, "cancel", "/v1/treasury/outbound_payments/{id}/cancel", http.MethodPost, map[string]string{}, &Config)
	resource.NewOperationCmd(rTreasuryOutboundPaymentsCmd.Cmd, "create", "/v1/treasury/outbound_payments", http.MethodPost, map[string]string{
		"amount":                     "integer",
		"currency":                   "string",
		"customer":                   "string",
		"description":                "string",
		"destination_payment_method": "string",
		"destination_payment_method_data.billing_details.email":                         "string",
		"destination_payment_method_data.billing_details.name":                          "string",
		"destination_payment_method_data.billing_details.phone":                         "string",
		"destination_payment_method_data.financial_account":                             "string",
		"destination_payment_method_data.type":                                          "string",
		"destination_payment_method_data.us_bank_account.account_holder_type":           "string",
		"destination_payment_method_data.us_bank_account.account_number":                "string",
		"destination_payment_method_data.us_bank_account.account_type":                  "string",
		"destination_payment_method_data.us_bank_account.financial_connections_account": "string",
		"destination_payment_method_data.us_bank_account.routing_number":                "string",
		"end_user_details.ip_address":                                                   "string",
		"end_user_details.present":                                                      "boolean",
		"financial_account":                                                             "string",
		"statement_descriptor":                                                          "string",
	}, &Config)
	resource.NewOperationCmd(rTreasuryOutboundPaymentsCmd.Cmd, "list", "/v1/treasury/outbound_payments", http.MethodGet, map[string]string{
		"created":           "integer",
		"customer":          "string",
		"ending_before":     "string",
		"financial_account": "string",
		"limit":             "integer",
		"starting_after":    "string",
		"status":            "string",
	}, &Config)
	resource.NewOperationCmd(rTreasuryOutboundPaymentsCmd.Cmd, "retrieve", "/v1/treasury/outbound_payments/{id}", http.MethodGet, map[string]string{}, &Config)
	resource.NewOperationCmd(rTreasuryOutboundPaymentsTestHelpersCmd.Cmd, "fail", "/v1/test_helpers/treasury/outbound_payments/{id}/fail", http.MethodPost, map[string]string{}, &Config)
	resource.NewOperationCmd(rTreasuryOutboundPaymentsTestHelpersCmd.Cmd, "post", "/v1/test_helpers/treasury/outbound_payments/{id}/post", http.MethodPost, map[string]string{}, &Config)
	resource.NewOperationCmd(rTreasuryOutboundPaymentsTestHelpersCmd.Cmd, "return_outbound_payment", "/v1/test_helpers/treasury/outbound_payments/{id}/return", http.MethodPost, map[string]string{
		"returned_details.code": "string",
	}, &Config)
	resource.NewOperationCmd(rTreasuryOutboundPaymentsTestHelpersCmd.Cmd, "update", "/v1/test_helpers/treasury/outbound_payments/{id}", http.MethodPost, map[string]string{
		"tracking_details.ach.trace_id":          "string",
		"tracking_details.type":                  "string",
		"tracking_details.us_domestic_wire.imad": "string",
		"tracking_details.us_domestic_wire.omad": "string",
	}, &Config)
	resource.NewOperationCmd(rTreasuryOutboundTransfersCmd.Cmd, "cancel", "/v1/treasury/outbound_transfers/{outbound_transfer}/cancel", http.MethodPost, map[string]string{}, &Config)
	resource.NewOperationCmd(rTreasuryOutboundTransfersCmd.Cmd, "create", "/v1/treasury/outbound_transfers", http.MethodPost, map[string]string{
		"amount":                     "integer",
		"currency":                   "string",
		"description":                "string",
		"destination_payment_method": "string",
		"financial_account":          "string",
		"statement_descriptor":       "string",
	}, &Config)
	resource.NewOperationCmd(rTreasuryOutboundTransfersCmd.Cmd, "list", "/v1/treasury/outbound_transfers", http.MethodGet, map[string]string{
		"ending_before":     "string",
		"financial_account": "string",
		"limit":             "integer",
		"starting_after":    "string",
		"status":            "string",
	}, &Config)
	resource.NewOperationCmd(rTreasuryOutboundTransfersCmd.Cmd, "retrieve", "/v1/treasury/outbound_transfers/{outbound_transfer}", http.MethodGet, map[string]string{}, &Config)
	resource.NewOperationCmd(rTreasuryOutboundTransfersTestHelpersCmd.Cmd, "fail", "/v1/test_helpers/treasury/outbound_transfers/{outbound_transfer}/fail", http.MethodPost, map[string]string{}, &Config)
	resource.NewOperationCmd(rTreasuryOutboundTransfersTestHelpersCmd.Cmd, "post", "/v1/test_helpers/treasury/outbound_transfers/{outbound_transfer}/post", http.MethodPost, map[string]string{}, &Config)
	resource.NewOperationCmd(rTreasuryOutboundTransfersTestHelpersCmd.Cmd, "return_outbound_transfer", "/v1/test_helpers/treasury/outbound_transfers/{outbound_transfer}/return", http.MethodPost, map[string]string{
		"returned_details.code": "string",
	}, &Config)
	resource.NewOperationCmd(rTreasuryOutboundTransfersTestHelpersCmd.Cmd, "update", "/v1/test_helpers/treasury/outbound_transfers/{outbound_transfer}", http.MethodPost, map[string]string{
		"tracking_details.ach.trace_id":          "string",
		"tracking_details.type":                  "string",
		"tracking_details.us_domestic_wire.imad": "string",
		"tracking_details.us_domestic_wire.omad": "string",
	}, &Config)
	resource.NewOperationCmd(rTreasuryReceivedCreditsCmd.Cmd, "list", "/v1/treasury/received_credits", http.MethodGet, map[string]string{
		"ending_before":     "string",
		"financial_account": "string",
		"limit":             "integer",
		"starting_after":    "string",
		"status":            "string",
	}, &Config)
	resource.NewOperationCmd(rTreasuryReceivedCreditsCmd.Cmd, "retrieve", "/v1/treasury/received_credits/{id}", http.MethodGet, map[string]string{}, &Config)
	resource.NewOperationCmd(rTreasuryReceivedCreditsTestHelpersCmd.Cmd, "create", "/v1/test_helpers/treasury/received_credits", http.MethodPost, map[string]string{
		"amount":                                 "integer",
		"currency":                               "string",
		"description":                            "string",
		"financial_account":                      "string",
		"initiating_payment_method_details.type": "string",
		"initiating_payment_method_details.us_bank_account.account_holder_name": "string",
		"initiating_payment_method_details.us_bank_account.account_number":      "string",
		"initiating_payment_method_details.us_bank_account.routing_number":      "string",
		"network": "string",
	}, &Config)
	resource.NewOperationCmd(rTreasuryReceivedDebitsCmd.Cmd, "list", "/v1/treasury/received_debits", http.MethodGet, map[string]string{
		"ending_before":     "string",
		"financial_account": "string",
		"limit":             "integer",
		"starting_after":    "string",
		"status":            "string",
	}, &Config)
	resource.NewOperationCmd(rTreasuryReceivedDebitsCmd.Cmd, "retrieve", "/v1/treasury/received_debits/{id}", http.MethodGet, map[string]string{}, &Config)
	resource.NewOperationCmd(rTreasuryReceivedDebitsTestHelpersCmd.Cmd, "create", "/v1/test_helpers/treasury/received_debits", http.MethodPost, map[string]string{
		"amount":                                 "integer",
		"currency":                               "string",
		"description":                            "string",
		"financial_account":                      "string",
		"initiating_payment_method_details.type": "string",
		"initiating_payment_method_details.us_bank_account.account_holder_name": "string",
		"initiating_payment_method_details.us_bank_account.account_number":      "string",
		"initiating_payment_method_details.us_bank_account.routing_number":      "string",
		"network": "string",
	}, &Config)
	resource.NewOperationCmd(rTreasuryTransactionEntrysCmd.Cmd, "list", "/v1/treasury/transaction_entries", http.MethodGet, map[string]string{
		"created":           "integer",
		"effective_at":      "integer",
		"ending_before":     "string",
		"financial_account": "string",
		"limit":             "integer",
		"order_by":          "string",
		"starting_after":    "string",
		"transaction":       "string",
	}, &Config)
	resource.NewOperationCmd(rTreasuryTransactionEntrysCmd.Cmd, "retrieve", "/v1/treasury/transaction_entries/{id}", http.MethodGet, map[string]string{}, &Config)
	resource.NewOperationCmd(rTreasuryTransactionsCmd.Cmd, "list", "/v1/treasury/transactions", http.MethodGet, map[string]string{
		"created":           "integer",
		"ending_before":     "string",
		"financial_account": "string",
		"limit":             "integer",
		"order_by":          "string",
		"starting_after":    "string",
		"status":            "string",
	}, &Config)
	resource.NewOperationCmd(rTreasuryTransactionsCmd.Cmd, "retrieve", "/v1/treasury/transactions/{id}", http.MethodGet, map[string]string{}, &Config)
}
