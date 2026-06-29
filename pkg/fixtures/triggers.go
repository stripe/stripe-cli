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
	return m
}

// interopEvents is the set of traditional webhook events that are eligible for delivery
// as "thin" v1-prefixed events via event destinations. When a user runs
// `stripe trigger v1.<event>`, the CLI resolves it to the unprefixed fixture if the
// event is in this allowlist and no dedicated v1.* fixture exists.
//
// This list is derived from the event_specs.yaml in the API event notification system.
// To regenerate, extract interop-eligible event names from that specification.
var interopEvents = map[string]bool{
	"file.created":                              true,
	"issuing_personalization_design.activated":   true,
	"issuing_personalization_design.deactivated": true,
	"issuing_personalization_design.rejected":    true,
	"issuing_personalization_design.updated":     true,
	"account.application.authorized":            true,
	"account.application.deauthorized":           true,
	"account.external_account.created":           true,
	"account.external_account.deleted":           true,
	"account.external_account.updated":           true,
	"account.updated":                            true,
	"application_fee.created":                    true,
	"application_fee.refund.updated":             true,
	"application_fee.refunded":                   true,
	"balance.available":                          true,
	"billing.alert.triggered":                    true,
	"billing.credit_grant.created":               true,
	"billing_portal.configuration.created":       true,
	"billing_portal.configuration.updated":       true,
	"billing_portal.session.created":             true,
	"capability.updated":                         true,
	"cash_balance.funds_available":               true,
	"charge.captured":                            true,
	"charge.dispute.closed":                      true,
	"charge.dispute.created":                     true,
	"charge.dispute.funds_reinstated":            true,
	"charge.dispute.funds_withdrawn":             true,
	"charge.dispute.updated":                     true,
	"charge.expired":                             true,
	"charge.failed":                              true,
	"charge.pending":                             true,
	"charge.refund.updated":                      true,
	"charge.refunded":                            true,
	"charge.succeeded":                           true,
	"charge.updated":                             true,
	"checkout.session.async_payment_failed":      true,
	"checkout.session.async_payment_succeeded":   true,
	"checkout.session.completed":                 true,
	"checkout.session.expired":                   true,
	"climate.order.canceled":                     true,
	"climate.order.created":                      true,
	"climate.order.delayed":                      true,
	"climate.order.delivered":                    true,
	"climate.order.product_substituted":          true,
	"climate.product.created":                    true,
	"climate.product.pricing_updated":            true,
	"coupon.created":                             true,
	"coupon.deleted":                             true,
	"coupon.updated":                             true,
	"credit_note.created":                        true,
	"credit_note.updated":                        true,
	"credit_note.voided":                         true,
	"customer_cash_balance_transaction.created":  true,
	"customer.created":                           true,
	"customer.deleted":                           true,
	"customer.discount.created":                  true,
	"customer.discount.deleted":                  true,
	"customer.discount.updated":                  true,
	"entitlements.active_entitlement_summary.updated": true,
	"customer.subscription.created":                   true,
	"customer.subscription.deleted":                   true,
	"customer.subscription.paused":                    true,
	"customer.subscription.pending_update_applied":    true,
	"customer.subscription.pending_update_expired":    true,
	"customer.subscription.resumed":                   true,
	"customer.subscription.trial_will_end":            true,
	"customer.subscription.updated":                   true,
	"customer.tax_id.created":                         true,
	"customer.tax_id.deleted":                         true,
	"customer.tax_id.updated":                         true,
	"customer.updated":                                true,
	"financial_connections.account.created":                    true,
	"financial_connections.account.deactivated":                true,
	"financial_connections.account.disconnected":               true,
	"financial_connections.account.account_numbers_updated":    true,
	"financial_connections.account.reactivated":                true,
	"financial_connections.account.refreshed_balance":          true,
	"financial_connections.account.refreshed_ownership":        true,
	"financial_connections.account.refreshed_transactions":     true,
	"financial_connections.account.upcoming_account_number_expiry": true,
	"identity.verification_session.canceled":       true,
	"identity.verification_session.created":        true,
	"identity.verification_session.processing":     true,
	"identity.verification_session.redacted":       true,
	"identity.verification_session.requires_input": true,
	"identity.verification_session.verified":       true,
	"invoice.created":                              true,
	"invoice.deleted":                              true,
	"invoice.finalization_failed":                  true,
	"invoice.finalized":                            true,
	"invoice.marked_uncollectible":                 true,
	"invoice.overdue":                              true,
	"invoice.overpaid":                             true,
	"invoice.paid":                                 true,
	"invoice.payment_action_required":              true,
	"invoice.payment_attempt_required":             true,
	"invoice.payment_failed":                       true,
	"invoice_payment.paid":                         true,
	"invoice.payment_succeeded":                    true,
	"invoice.sent":                                 true,
	"invoice.upcoming":                             true,
	"invoice.updated":                              true,
	"invoice.voided":                               true,
	"invoice.will_be_due":                          true,
	"invoiceitem.created":                          true,
	"invoiceitem.deleted":                          true,
	"issuing_authorization.created":                true,
	"issuing_authorization.request":                true,
	"issuing_authorization.updated":                true,
	"issuing_card.created":                         true,
	"issuing_card.updated":                         true,
	"issuing_cardholder.created":                   true,
	"issuing_cardholder.updated":                   true,
	"issuing_dispute.closed":                       true,
	"issuing_dispute.created":                      true,
	"issuing_dispute.funds_reinstated":             true,
	"issuing_dispute.funds_rescinded":              true,
	"issuing_dispute.submitted":                    true,
	"issuing_dispute.updated":                      true,
	"issuing_token.created":                        true,
	"issuing_token.updated":                        true,
	"issuing_transaction.created":                  true,
	"issuing_transaction.purchase_details_receipt_updated": true,
	"issuing_transaction.updated":                         true,
	"mandate.updated":                                     true,
	"payment_intent.amount_capturable_updated":            true,
	"payment_intent.canceled":                             true,
	"payment_intent.created":                              true,
	"payment_intent.partially_funded":                     true,
	"payment_intent.payment_failed":                       true,
	"payment_intent.processing":                           true,
	"payment_intent.requires_action":                      true,
	"payment_intent.succeeded":                            true,
	"payment_link.created":                                true,
	"payment_link.updated":                                true,
	"payment_method.attached":                             true,
	"payment_method.automatically_updated":                true,
	"payment_method.detached":                             true,
	"payment_method.updated":                              true,
	"payout.canceled":                                     true,
	"payout.created":                                      true,
	"payout.failed":                                       true,
	"payout.paid":                                         true,
	"payout.reconciliation_completed":                     true,
	"payout.updated":                                      true,
	"person.created":                                      true,
	"person.deleted":                                      true,
	"person.updated":                                      true,
	"plan.created":                                        true,
	"plan.deleted":                                        true,
	"plan.updated":                                        true,
	"price.created":                                       true,
	"price.deleted":                                       true,
	"price.updated":                                       true,
	"product.created":                                     true,
	"product.deleted":                                     true,
	"product.updated":                                     true,
	"promotion_code.created":                              true,
	"promotion_code.updated":                              true,
	"quote.accepted":                                      true,
	"quote.canceled":                                      true,
	"quote.created":                                       true,
	"quote.finalized":                                     true,
	"radar.early_fraud_warning.created":                   true,
	"radar.early_fraud_warning.updated":                   true,
	"refund.created":                                      true,
	"refund.failed":                                       true,
	"refund.updated":                                      true,
	"review.closed":                                       true,
	"review.opened":                                       true,
	"setup_intent.canceled":                               true,
	"setup_intent.created":                                true,
	"setup_intent.requires_action":                        true,
	"setup_intent.setup_failed":                           true,
	"setup_intent.succeeded":                              true,
	"sigma.scheduled_query_run.created":                   true,
	"source.canceled":                                     true,
	"source.chargeable":                                   true,
	"source.failed":                                       true,
	"source.refund_attributes_required":                   true,
	"subscription_schedule.aborted":                       true,
	"subscription_schedule.canceled":                      true,
	"subscription_schedule.completed":                     true,
	"subscription_schedule.created":                       true,
	"subscription_schedule.expiring":                      true,
	"subscription_schedule.released":                      true,
	"subscription_schedule.updated":                       true,
	"tax_rate.created":                                    true,
	"tax_rate.updated":                                    true,
	"tax.settings.updated":                                true,
	"terminal.reader.action_failed":                       true,
	"terminal.reader.action_succeeded":                    true,
	"terminal.reader.action_updated":                      true,
	"test_helpers.test_clock.advancing":                   true,
	"test_helpers.test_clock.created":                     true,
	"test_helpers.test_clock.deleted":                     true,
	"test_helpers.test_clock.internal_failure":            true,
	"test_helpers.test_clock.ready":                       true,
	"topup.canceled":                                      true,
	"topup.created":                                       true,
	"topup.failed":                                        true,
	"topup.reversed":                                      true,
	"topup.succeeded":                                     true,
	"transfer.created":                                    true,
	"transfer.reversed":                                   true,
	"transfer.updated":                                    true,
}

// resolveInteropEvent checks if the given event name is a v1.-prefixed interop event
// and returns the fixture path for the corresponding unprefixed event if eligible.
func resolveInteropEvent(event string, evts map[string]string) (string, bool) {
	if !strings.HasPrefix(event, "v1.") {
		return "", false
	}
	unprefixed := strings.TrimPrefix(event, "v1.")
	if !interopEvents[unprefixed] {
		return "", false
	}
	file, ok := evts[unprefixed]
	return file, ok
}

// FixtureContents returns the JSON content of the embedded fixture for the given event
// name. The JSON is re-serialized from the parsed FixtureData struct, matching the
// format produced by GetFixtureFileContent.
func FixtureContents(eventName string) (string, error) {
	evts := getEvents()
	path, ok := evts[eventName]
	if !ok {
		path, ok = resolveInteropEvent(eventName, evts)
	}
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
	evts := getEvents()
	names := []string{}
	for name := range evts {
		names = append(names, name)
	}

	for interopEvent := range interopEvents {
		prefixed := "v1." + interopEvent
		if _, hasDedicated := evts[prefixed]; hasDedicated {
			continue // already included via the getEvents() loop above
		}
		if _, hasFixture := evts[interopEvent]; hasFixture {
			names = append(names, prefixed)
		}
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
		evts := getEvents()
		file, ok := evts[event]
		if !ok {
			file, ok = resolveInteropEvent(event, evts)
		}
		if ok {
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
