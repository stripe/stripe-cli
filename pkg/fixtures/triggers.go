package fixtures

import (
	"fmt"
	"sort"

	"github.com/spf13/afero"

	"github.com/stripe/stripe-cli/pkg/stripe"
)

// Events is a mapping of pre-built trigger events and the corresponding json file
var Events = map[string]string{
	"charge.captured":                          "/charge.captured.json",
	"charge.dispute.created":                   "/charge.disputed.created.json",
	"charge.failed":                            "/charge.failed.json",
	"charge.refunded":                          "/charge.refunded.json",
	"charge.succeeded":                         "/charge.succeeded.json",
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
	"payment_intent.amount_capturable_updated": "/payment_intent.amount_capturable_updated.json",
	"payment_intent.created":                   "/payment_intent.created.json",
	"payment_intent.payment_failed":            "/payment_intent.payment_failed.json",
	"payment_intent.succeeded":                 "/payment_intent.succeeded.json",
	"payment_intent.canceled":                  "/payment_intent.canceled.json",
	"payment_method.attached":                  "/payment_method.attached.json",
}

// BuildFromFixture creates a new fixture struct for a file
func BuildFromFixture(fs afero.Fs, apiKey, jsonFile string) (*Fixture, error) {
	fixture, err := NewFixture(
		fs,
		apiKey,
		stripe.DefaultAPIBaseURL,
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
