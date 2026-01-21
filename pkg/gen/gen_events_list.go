//go:build events_list
// +build events_list

package main

import (
	"bytes"
	"fmt"
	"go/format"
	"os"
	"sort"
	"text/template"

	"github.com/stripe/stripe-cli/pkg/gen"
	"github.com/stripe/stripe-cli/pkg/spec"
)

type TemplateData struct {
	Events            []string
	ThinEvents        []string
	PreviewEvents     []string
	PreviewThinEvents []string
}

const (
	pathTemplate = "../gen/events_list.go.tpl"

	pathName = "events_list.go.tpl"

	pathOutput = "../proxy/events_list.go"
)

func main() {
	// generate `events_list.go` from OpenAPI spec file
	// code for this func from gen_resources_cmds.go
	templateData, err := getTemplateDataFromUnifiedSpec()
	if err != nil {
		panic(err)
	}

	// load template
	tmpl := template.Must(template.
		New(pathName).
		ParseFiles(pathTemplate))

	// execute template
	var result bytes.Buffer
	err = tmpl.Execute(&result, templateData)
	if err != nil {
		panic(err)
	}

	// format template output
	formatted, err := format.Source(result.Bytes())
	if err != nil {
		panic(err)
	}

	// write formatted code to disk
	fmt.Printf("writing %s\n", pathOutput)
	err = os.WriteFile(pathOutput, formatted, 0644)
	if err != nil {
		panic(err)
	}

}

// getTemplateData loads event data from separate V1 and V2 spec files and returns
// template data with sorted event lists for code generation.
func getTemplateData() (*TemplateData, error) {
	eventsV1Set, err := getV1Events(gen.PathStripeSpec)
	if err != nil {
		return nil, err
	}
	eventsV2Set, err := getThinEvents(gen.PathStripeSpecV2)
	if err != nil {
		return nil, err
	}
	previewEventsV2Set, err := getThinEvents(gen.PathStripeSpecV2Preview)
	if err != nil {
		return nil, err
	}

	data := &TemplateData{
		Events:            setToSortedSlice(eventsV1Set),
		ThinEvents:        setToSortedSlice(eventsV2Set),
		PreviewThinEvents: setToSortedSlice(previewEventsV2Set),
	}

	return data, nil
}

// getTemplateDataFromUnifiedSpec loads event data from unified spec files (which combine
// V1 and V2 events) and returns template data with sorted event lists for code generation.
// Preview events are filtered to only include events that don't exist in the non-preview specs.
// V1 preview events are excluded for now.
func getTemplateDataFromUnifiedSpec() (*TemplateData, error) {
	eventsV1Set, err := getV1Events(gen.PathUnifiedSpec)
	if err != nil {
		return nil, err
	}

	thinEventsSet, err := getThinEvents(gen.PathUnifiedSpec)
	if err != nil {
		return nil, err
	}
	previewThinEventsSet, err := getThinEvents(gen.PathUnifiedPreviewSpec)
	if err != nil {
		return nil, err
	}

	// Filter preview thin events to only include events not present in the non-preview list
	filteredPreviewThinEventsSet := make(map[string]struct{})
	for e := range previewThinEventsSet {
		if _, exists := thinEventsSet[e]; !exists {
			filteredPreviewThinEventsSet[e] = struct{}{}
		}
	}

	data := &TemplateData{
		Events:            setToSortedSlice(eventsV1Set),
		ThinEvents:        setToSortedSlice(thinEventsSet),
		PreviewEvents:     nil, // V1 preview events excluded for now
		PreviewThinEvents: setToSortedSlice(filteredPreviewThinEventsSet),
	}
	return data, nil
}

// getV1Events extracts V1 (classic) event types from an OpenAPI spec file by reading
// the enabled_events property from the webhook_endpoints endpoint schema.
// Returns a set (map[string]struct{}) of unique event type strings.
func getV1Events(pathSpec string) (map[string]struct{}, error) {
	api, err := spec.LoadSpec(pathSpec)
	if err != nil {
		return nil, err
	}

	postRequest := api.Paths["/v1/webhook_endpoints"]["post"]
	requestSchema := postRequest.RequestBody.Content["application/x-www-form-urlencoded"].Schema
	events := requestSchema.Properties["enabled_events"].Items.Enum
	eventSet := make(map[string]struct{}, len(events))
	for _, e := range events {
		eventSet[e.(string)] = struct{}{}
	}
	return eventSet, nil
}

// getThinEvents extracts V2 thin event types from an OpenAPI spec file by iterating
// through all schemas and finding those marked with x-stripeEvent extension where
// eventKind is "thin".
// Returns a set (map[string]struct{}) of unique event type strings.
func getThinEvents(pathSpec string) (map[string]struct{}, error) {
	api, err := spec.LoadSpec(pathSpec)
	if err != nil {
		return nil, err
	}

	eventSet := make(map[string]struct{})
	// Iterate over every schema
	for _, schema := range api.Components.Schemas {
		// Skip schemas that do not have x-stripeEvent or are not thin events
		if schema.XStripeEvent == nil || schema.XStripeEvent.EventKind == nil || *schema.XStripeEvent.EventKind != "thin" {
			continue
		}

		eventType := schema.XStripeEvent.EventType
		eventSet[eventType] = struct{}{}
	}

	return eventSet, nil
}

// setToSortedSlice converts a set (map[string]struct{}) to a sorted slice of strings.
// Returns a lexicographically sorted []string containing all keys from the input set.
func setToSortedSlice(set map[string]struct{}) []string {
	slice := make([]string, 0, len(set))
	for key := range set {
		slice = append(slice, key)
	}
	sort.Strings(slice)
	return slice
}
