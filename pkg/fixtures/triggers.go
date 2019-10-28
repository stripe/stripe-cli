package fixtures

import (
	"fmt"
	"sort"

	"github.com/spf13/afero"

	"github.com/stripe/stripe-cli/pkg/stripe"
)

// Events is a mapping of pre-built trigger events and the corresponding json file
var Events = map[string]string{
	"charge.captured":               "triggers/charge.captured.json",
	"charge.dispute.created":        "triggers/charge.disputed.created.json",
	"charge.failed":                 "triggers/charge.failed.json",
	"charge.refunded":               "triggers/charge.refunded.json",
	"charge.succeeded":              "triggers/charge.succeeded.json",
	"checkout.session.completed":    "triggers/checkout.session.completed.json",
	"customer.created":              "triggers/customer.created.json",
	"customer.deleted":              "triggers/customer.deleted.json",
	"customer.updated":              "triggers/customer.updated.json",
	"customer.source.created":       "triggers/customer.source.created.json",
	"customer.source.updated":       "triggers/customer.source.updated.json",
	"customer.subscription.deleted": "triggers/customer.subscription.deleted.json",
	"customer.subscription.updated": "triggers/customer.subscription.updated.json",
	"invoice.created":               "triggers/invoice.created.json",
	"invoice.finalized":             "triggers/invoice.finalized.json",
	"invoice.payment_failed":        "triggers/invoice.payment_failed.json",
	"invoice.payment_succeeded":     "triggers/invoice.payment_succeeded.json",
	"invoice.updated":               "triggers/invoice.updated.json",
	"payment_intent.created":        "triggers/payment_intent.created.json",
	"payment_intent.payment_failed": "triggers/payment_intent.payment_failed.json",
	"payment_intent.succeeded":      "triggers/payment_intent.succeeded.json",
	"payment_intent.canceled":       "triggers/payment_intent.canceled.json",
	"payment_method.attached":       "triggers/payment_method.attached.json",
}

// BuildFromFixture creates a new fixture struct for a file
func BuildFromFixture(fs afero.Fs, apiKey, jsonFile string) *Fixture {
	fixture, _ := NewFixture(
		fs,
		apiKey,
		stripe.DefaultAPIBaseURL,
		jsonFile,
	)

	return fixture
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

// SupportedEvents returns a map of trigger names to the built fixture for execution
func SupportedEvents(fs afero.Fs, apiKey string) map[string]*Fixture {
	events := make(map[string]*Fixture)
	for event, file := range Events {
		events[event] = BuildFromFixture(fs, apiKey, file)
	}

	return events
}

func reverseMap() map[string]string {
	reversed := make(map[string]string)
	for name, file := range Events {
		reversed[file] = name
	}

	return reversed
}
