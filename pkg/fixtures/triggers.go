package fixtures

import (
	"fmt"
	"sort"

	"github.com/spf13/afero"
)

// Events is a mapping of pre-built trigger events and the corresponding json file
var Events = map[string]string{
	"balance.available":                        "/balance.available.json",
	"charge.captured":                          "/charge.captured.json",
	"charge.dispute.created":                   "/charge.disputed.created.json",
	"charge.failed":                            "/charge.failed.json",
	"charge.refunded":                          "/charge.refunded.json",
	"charge.succeeded":                         "/charge.succeeded.json",
	"checkout.session.async_payment_failed":    "/checkout.session.async_payment_failed.json",
	"checkout.session.async_payment_succeeded": "/checkout.session.async_payment_succeeded.json",
	"checkout.session.completed":               "/checkout.session.completed.json",
	"customer.created":                         "/customer.created.json",
	"customer.deleted":                         "/customer.deleted.json",
	"customer.updated":                         "/customer.updated.json",
	"customer.source.created":                  "/customer.source.created.json",
	"customer.source.updated":                  "/customer.source.updated.json",
	"customer.subscription.created":            "/customer.subscription.created.json",
	"customer.subscription.deleted":            "/customer.subscription.deleted.json",
	"customer.subscription.updated":            "/customer.subscription.updated.json",
	"invoice.created":                          "/invoice.created.json",
	"invoice.finalized":                        "/invoice.finalized.json",
	"invoice.payment_failed":                   "/invoice.payment_failed.json",
	"invoice.payment_succeeded":                "/invoice.payment_succeeded.json",
	"invoice.updated":                          "/invoice.updated.json",
	"issuing_authorization.request":            "/issuing_authorization.request.json",
	"issuing_card.created":                     "/issuing_card.created.json",
	"issuing_cardholder.created":               "/issuing_cardholder.created.json",
	"payment_intent.amount_capturable_updated": "/payment_intent.amount_capturable_updated.json",
	"payment_intent.created":                   "/payment_intent.created.json",
	"payment_intent.payment_failed":            "/payment_intent.payment_failed.json",
	"payment_intent.succeeded":                 "/payment_intent.succeeded.json",
	"payment_intent.canceled":                  "/payment_intent.canceled.json",
	"payment_method.attached":                  "/payment_method.attached.json",
	"plan.created":                             "/plan.created.json",
	"plan.deleted":                             "/plan.deleted.json",
	"plan.updated":                             "/plan.updated.json",
	"product.created":                          "/product.created.json",
	"product.deleted":                          "/product.deleted.json",
	"product.updated":                          "/product.updated.json",
	"setup_intent.canceled":                    "/setup_intent.canceled.json",
	"setup_intent.created":                     "/setup_intent.created.json",
	"setup_intent.setup_failed":                "/setup_intent.setup_failed.json",
	"setup_intent.succeeded":                   "/setup_intent.succeeded.json",
	"subscription_schedule.canceled":           "/subscription_schedule.canceled.json",
	"subscription_schedule.created":            "/subscription_schedule.created.json",
	"subscription_schedule.released":           "/subscription_schedule.released.json",
	"subscription_schedule.updated":            "/subscription_schedule.updated.json",
}

// BuildFromFixture creates a new fixture struct for a file
func BuildFromFixture(fs afero.Fs, apiKey, stripeAccount, apiBaseURL, jsonFile string) (*Fixture, error) {
	fixture, err := NewFixture(
		fs,
		apiKey,
		stripeAccount,
		apiBaseURL,
		jsonFile,
	)
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
	for name := range Events {
		names = append(names, name)
	}

	sort.Strings(names)

	return names
}

func reverseMap() map[string]string {
	reversed := make(map[string]string)
	for name, file := range Events {
		reversed[file] = name
	}

	return reversed
}
