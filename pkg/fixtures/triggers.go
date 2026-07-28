package fixtures

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"
	"sync"

	"github.com/spf13/afero"

	"github.com/stripe/stripe-cli/pkg/stripe"
)

//go:embed triggers/*
var triggers embed.FS

var (
	events     map[string]string
	eventsOnce sync.Once
)

// v1TriggerableEvents is the set of snapshot event names that can also be
// triggered using the "v1." prefix. For example, if "customer.updated" is in
// this set, both of the following will run the same fixture:
//
//	stripe trigger customer.updated
//	stripe trigger v1.customer.updated
//
// Add an entry here when a new thin event becomes available for a snapshot
// event. The key must exactly match the snapshot event name (i.e. the filename
// in the triggers/ directory without the .json extension).
var v1TriggerableEvents = map[string]struct{}{
	"account.external_account.created":                     {},
	"account.external_account.deleted":                     {},
	"account.external_account.updated":                     {},
	"account.updated":                                      {},
	"balance.available":                                    {},
	"billing.alert.triggered":                              {},
	"billing_portal.configuration.created":                 {},
	"billing_portal.configuration.updated":                 {},
	"billing_portal.session.created":                       {},
	"capability.updated":                                   {},
	"cash_balance.funds_available":                         {},
	"charge.captured":                                      {},
	"charge.dispute.closed":                                {},
	"charge.dispute.created":                               {},
	"charge.dispute.funds_reinstated":                      {},
	"charge.dispute.funds_withdrawn":                       {},
	"charge.dispute.updated":                               {},
	"charge.expired":                                       {},
	"charge.failed":                                        {},
	"charge.pending":                                       {},
	"charge.refund.updated":                                {},
	"charge.refunded":                                      {},
	"charge.succeeded":                                     {},
	"charge.updated":                                       {},
	"checkout.session.async_payment_failed":                {},
	"checkout.session.async_payment_succeeded":             {},
	"checkout.session.completed":                           {},
	"checkout.session.expired":                             {},
	"climate.order.canceled":                               {},
	"climate.order.created":                                {},
	"climate.order.delayed":                                {},
	"climate.order.delivered":                              {},
	"climate.order.product_substituted":                    {},
	"climate.product.created":                              {},
	"climate.product.pricing_updated":                      {},
	"coupon.created":                                       {},
	"coupon.deleted":                                       {},
	"coupon.updated":                                       {},
	"credit_note.created":                                  {},
	"credit_note.updated":                                  {},
	"credit_note.voided":                                   {},
	"customer.created":                                     {},
	"customer.deleted":                                     {},
	"customer.discount.created":                            {},
	"customer.discount.deleted":                            {},
	"customer.discount.updated":                            {},
	"customer.subscription.created":                        {},
	"customer.subscription.deleted":                        {},
	"customer.subscription.paused":                         {},
	"customer.subscription.pending_update_applied":         {},
	"customer.subscription.pending_update_expired":         {},
	"customer.subscription.resumed":                        {},
	"customer.subscription.trial_will_end":                 {},
	"customer.subscription.updated":                        {},
	"customer.tax_id.created":                              {},
	"customer.tax_id.deleted":                              {},
	"customer.tax_id.updated":                              {},
	"customer.updated":                                     {},
	"customer_cash_balance_transaction.created":            {},
	"entitlements.active_entitlement_summary.updated":      {},
	"file.created":                                         {},
	"financial_connections.account.created":                {},
	"financial_connections.account.deactivated":            {},
	"financial_connections.account.disconnected":           {},
	"financial_connections.account.reactivated":            {},
	"financial_connections.account.refreshed_balance":      {},
	"financial_connections.account.refreshed_ownership":    {},
	"financial_connections.account.refreshed_transactions": {},
	"identity.verification_session.canceled":               {},
	"identity.verification_session.created":                {},
	"identity.verification_session.processing":             {},
	"identity.verification_session.redacted":               {},
	"identity.verification_session.requires_input":         {},
	"identity.verification_session.verified":               {},
	"invoice.created":                                      {},
	"invoice.deleted":                                      {},
	"invoice.finalization_failed":                          {},
	"invoice.finalized":                                    {},
	"invoice.marked_uncollectible":                         {},
	"invoice.overdue":                                      {},
	"invoice.overpaid":                                     {},
	"invoice.paid":                                         {},
	"invoice.payment_action_required":                      {},
	"invoice.payment_failed":                               {},
	"invoice.payment_succeeded":                            {},
	"invoice.sent":                                         {},
	"invoice.upcoming":                                     {},
	"invoice.updated":                                      {},
	"invoice.voided":                                       {},
	"invoice.will_be_due":                                  {},
	"invoice_payment.paid":                                 {},
	"invoiceitem.created":                                  {},
	"invoiceitem.deleted":                                  {},
	"issuing_authorization.created":                        {},
	"issuing_authorization.request":                        {},
	"issuing_authorization.updated":                        {},
	"issuing_card.created":                                 {},
	"issuing_card.updated":                                 {},
	"issuing_cardholder.created":                           {},
	"issuing_cardholder.updated":                           {},
	"issuing_dispute.closed":                               {},
	"issuing_dispute.created":                              {},
	"issuing_dispute.funds_reinstated":                     {},
	"issuing_dispute.funds_rescinded":                      {},
	"issuing_dispute.submitted":                            {},
	"issuing_dispute.updated":                              {},
	"issuing_personalization_design.activated":             {},
	"issuing_personalization_design.deactivated":           {},
	"issuing_personalization_design.rejected":              {},
	"issuing_personalization_design.updated":               {},
	"issuing_token.created":                                {},
	"issuing_token.updated":                                {},
	"issuing_transaction.created":                          {},
	"issuing_transaction.purchase_details_receipt_updated": {},
	"issuing_transaction.updated":                          {},
	"mandate.updated":                                      {},
	"payment_intent.amount_capturable_updated":             {},
	"payment_intent.canceled":                              {},
	"payment_intent.created":                               {},
	"payment_intent.partially_funded":                      {},
	"payment_intent.payment_failed":                        {},
	"payment_intent.processing":                            {},
	"payment_intent.requires_action":                       {},
	"payment_intent.succeeded":                             {},
	"payment_link.created":                                 {},
	"payment_link.updated":                                 {},
	"payment_method.attached":                              {},
	"payment_method.automatically_updated":                 {},
	"payment_method.detached":                              {},
	"payment_method.updated":                               {},
	"payout.canceled":                                      {},
	"payout.created":                                       {},
	"payout.failed":                                        {},
	"payout.paid":                                          {},
	"payout.reconciliation_completed":                      {},
	"payout.updated":                                       {},
	"person.created":                                       {},
	"person.deleted":                                       {},
	"person.updated":                                       {},
	"plan.created":                                         {},
	"plan.deleted":                                         {},
	"plan.updated":                                         {},
	"price.created":                                        {},
	"price.deleted":                                        {},
	"price.updated":                                        {},
	"product.created":                                      {},
	"product.deleted":                                      {},
	"product.updated":                                      {},
	"promotion_code.created":                               {},
	"promotion_code.updated":                               {},
	"quote.accepted":                                       {},
	"quote.canceled":                                       {},
	"quote.created":                                        {},
	"quote.finalized":                                      {},
	"radar.early_fraud_warning.created":                    {},
	"radar.early_fraud_warning.updated":                    {},
	"refund.created":                                       {},
	"refund.failed":                                        {},
	"refund.updated":                                       {},
	"review.closed":                                        {},
	"review.opened":                                        {},
	"setup_intent.canceled":                                {},
	"setup_intent.created":                                 {},
	"setup_intent.requires_action":                         {},
	"setup_intent.setup_failed":                            {},
	"setup_intent.succeeded":                               {},
	"sigma.scheduled_query_run.created":                    {},
	"source.canceled":                                      {},
	"source.chargeable":                                    {},
	"source.failed":                                        {},
	"source.refund_attributes_required":                    {},
	"subscription_schedule.aborted":                        {},
	"subscription_schedule.canceled":                       {},
	"subscription_schedule.completed":                      {},
	"subscription_schedule.created":                        {},
	"subscription_schedule.expiring":                       {},
	"subscription_schedule.released":                       {},
	"subscription_schedule.updated":                        {},
	"tax.settings.updated":                                 {},
	"tax_rate.created":                                     {},
	"tax_rate.updated":                                     {},
	"terminal.reader.action_failed":                        {},
	"terminal.reader.action_succeeded":                     {},
	"terminal.reader.action_updated":                       {},
	"test_helpers.test_clock.advancing":                    {},
	"test_helpers.test_clock.created":                      {},
	"test_helpers.test_clock.deleted":                      {},
	"test_helpers.test_clock.internal_failure":             {},
	"test_helpers.test_clock.ready":                        {},
	"topup.canceled":                                       {},
	"topup.created":                                        {},
	"topup.failed":                                         {},
	"topup.reversed":                                       {},
	"topup.succeeded":                                      {},
	"transfer.created":                                     {},
	"transfer.reversed":                                    {},
	"transfer.updated":                                     {},
}

// getEvents returns the lazily-initialized event→fixture-path map. The map is built
// once on first access by scanning the embedded triggers/ directory. Event names are
// derived from filenames (e.g. customer.created.json → customer.created); fixture files
// may declare additional names in _meta.aliases.
func getEvents() map[string]string {
	eventsOnce.Do(func() { events = buildEventsMap() })
	return events
}

func buildEventsMap() map[string]string {
	m := make(map[string]string)
	entries, err := triggers.ReadDir("triggers")
	if err != nil {
		panic(fmt.Sprintf("failed to read embedded triggers dir: %v", err))
	}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		path := "triggers/" + entry.Name()
		eventName := strings.TrimSuffix(entry.Name(), ".json")
		m[eventName] = path

		f, err := triggers.Open(path)
		if err != nil {
			continue
		}
		b, readErr := io.ReadAll(f)
		f.Close()
		if readErr != nil {
			continue
		}
		// Partial parse: only _meta.aliases is needed here; unmarshaling into
		// the full FixtureData struct would wastefully parse the fixtures array.
		var meta struct {
			Meta struct {
				Aliases []string `json:"aliases"`
			} `json:"_meta"`
		}
		if json.Unmarshal(b, &meta) == nil {
			for _, alias := range meta.Meta.Aliases {
				m[alias] = path
			}
		}
	}

	for name, path := range m {
		if _, ok := v1TriggerableEvents[name]; ok {
			v1Name := "v1." + name
			if _, exists := m[v1Name]; !exists {
				m[v1Name] = path
			}
		}
	}

	return m
}

// FixtureContents returns the JSON content of the embedded fixture for the given event
// name. The JSON is re-serialized from the parsed FixtureData struct, matching the
// format produced by GetFixtureFileContent.
func FixtureContents(eventName string) (string, error) {
	path, ok := getEvents()[eventName]
	if !ok {
		return "", fmt.Errorf("event %q is not supported", eventName)
	}
	f, err := NewFixtureFromFile(nil, "", "", "", path, nil, nil, nil, nil, false)
	if err != nil {
		return "", err
	}
	return f.GetFixtureFileContent(), nil
}

// BuildFromFixtureFile creates a new fixture struct for a file
func BuildFromFixtureFile(fs afero.Fs, apiKey, stripeAccount, apiBaseURL, jsonFile string, skip, override, add, remove []string, edit bool) (*Fixture, error) {
	fixture, err := NewFixtureFromFile(
		fs,
		apiKey,
		stripeAccount,
		apiBaseURL,
		jsonFile,
		skip,
		override,
		add,
		remove,
		edit,
	)
	if err != nil {
		return nil, err
	}

	return fixture, nil
}

// BuildFromFixtureString creates a new fixture from a string
func BuildFromFixtureString(fs afero.Fs, apiKey, stripeAccount, apiBaseURL, raw string) (*Fixture, error) {
	fixture, err := NewFixtureFromRawString(fs, apiKey, stripeAccount, apiBaseURL, raw)
	if err != nil {
		return nil, err
	}
	return fixture, nil
}

// EventList prints out a padded list of supported trigger events for printing the help file
func EventList() string {
	var eventList string
	for _, event := range EventNames() {
		eventList += fmt.Sprintf("  %s\n", event)
	}

	return eventList
}

// EventNames returns an array of all the event names
func EventNames() []string {
	names := []string{}
	for name, path := range getEvents() {
		// Hide auto-generated v1. aliases from the public event list.
		// Native v1. fixture files have a path starting with "triggers/v1."
		// and are always shown.
		if strings.HasPrefix(name, "v1.") && !strings.HasPrefix(path, "triggers/v1.") {
			continue
		}
		names = append(names, name)
	}

	sort.Strings(names)

	return names
}

// Trigger triggers a Stripe event.
func Trigger(ctx context.Context, event string, stripeAccount string, baseURL string, apiKey string, skip, override, add, remove []string, raw string, apiVersion string, edit bool) ([]string, error) {
	var fixture *Fixture
	var err error
	fs := afero.NewOsFs()

	// send event triggered
	telemetryClient := stripe.GetTelemetryClient(ctx)
	if telemetryClient != nil {
		go telemetryClient.SendEvent(ctx, "Triggered Event", event)
	}

	if len(raw) == 0 {
		if file, ok := getEvents()[event]; ok {
			fixture, err = BuildFromFixtureFile(fs, apiKey, stripeAccount, baseURL, file, skip, override, add, remove, edit)
			if err != nil {
				return nil, err
			}
		} else {
			exists, _ := afero.Exists(fs, event)
			if !exists {
				return nil, fmt.Errorf("%s", fmt.Sprintf("The event `%s` is not supported by Stripe CLI. To trigger unsupported events, use the Stripe API or Dashboard to perform actions that lead to the event you want to trigger (for example, create a Customer to generate a `customer.created` event). You can also create a custom fixture: https://docs.stripe.com/cli/fixtures", event))
			}

			fixture, err = BuildFromFixtureFile(fs, apiKey, stripeAccount, baseURL, event, skip, override, add, remove, edit)
			if err != nil {
				return nil, err
			}
		}
	} else {
		fixture, err = BuildFromFixtureString(fs, apiKey, stripeAccount, baseURL, raw)
		if err != nil {
			return nil, err
		}
	}

	requestNames, err := fixture.Execute(ctx, apiVersion)
	if err != nil {
		return nil, fmt.Errorf("%s", fmt.Sprintf("Trigger failed: %s\n", err))
	}

	return requestNames, nil
}

func reverseMap() map[string]string {
	reversed := make(map[string]string)
	for name, file := range getEvents() {
		reversed[file] = name
	}

	return reversed
}
