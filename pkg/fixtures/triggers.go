package fixtures

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/spf13/afero"

	"github.com/stripe/stripe-cli/pkg/stripe"
)

//go:embed triggers/*
var triggers embed.FS

// Events is a mapping of pre-built trigger events and the corresponding json file.
// It is built automatically from the embedded triggers/ directory: the event name
// is derived from the filename (e.g. customer.created.json → customer.created).
// Fixture files may declare additional event aliases in _meta.aliases when a single
// fixture logically serves multiple events.
var Events = buildEventsMap()

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
		var data FixtureData
		if json.Unmarshal(b, &data) == nil {
			for _, alias := range data.Meta.Aliases {
				m[alias] = path
			}
		}
	}
	return m
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
	for name := range Events {
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
		if file, ok := Events[event]; ok {
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
	for name, file := range Events {
		reversed[file] = name
	}

	return reversed
}
