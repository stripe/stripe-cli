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

	"github.com/stripe/stripe-cli/pkg/spec"
)

type TemplateData struct {
	Events        []string
	ThinEvents    []string
	PreviewEvents []string
}

const (
	pathStripeSpec = "../../api/openapi-spec/spec3.sdk.json"

	pathStripeSpecV2 = "../../api/openapi-spec/spec3.v2.sdk.json"

	pathStripeSpecV2Preview = "../../api/openapi-spec/spec3.v2.sdk.preview.json"

	pathTemplate = "../gen/events_list.go.tpl"

	pathName = "events_list.go.tpl"

	pathOutput = "../proxy/events_list.go"
)

func main() {
	// generate `events_list.go` from OpenAPI spec file
	// code for this func from gen_resources_cmds.go
	templateData, err := getTemplateData()
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

func getTemplateData() (*TemplateData, error) {
	eventsV1, err := getV1EventList()
	if err != nil {
		return nil, err
	}
	eventsV2, err := getV2EventList()
	if err != nil {
		return nil, err
	}
	previewEvents, err := getPreviewEventList()
	if err != nil {
		return nil, err
	}

	data := &TemplateData{
		Events:        eventsV1,
		ThinEvents:    eventsV2,
		PreviewEvents: previewEvents,
	}

	return data, nil
}

func getV1EventList() ([]string, error) {
	api, err := spec.LoadSpec(pathStripeSpec)
	if err != nil {
		return nil, err
	}

	postRequest := api.Paths["/v1/webhook_endpoints"]["post"]
	requestSchema := postRequest.RequestBody.Content["application/x-www-form-urlencoded"].Schema
	events := requestSchema.Properties["enabled_events"].Items.Enum
	eventList := make([]string, 0)
	for _, e := range events {
		eventList = append(eventList, e.(string))
	}
	return eventList, nil
}

func getV2EventList() ([]string, error) {
	api, err := spec.LoadSpec(pathStripeSpecV2)
	if err != nil {
		return nil, err
	}

	eventList := make([]string, 0)
	// Iterate over every resource schema
	for _, schema := range api.Components.Schemas {
		// Skip resources that don't have any operations
		if schema.XStripeEvent == nil {
			continue
		}

		eventType := schema.XStripeEvent.EventType
		eventList = append(eventList, eventType)
	}

	// Sort the eventList so that we have consistent
	// ordering when testing in CI
	sort.Strings(eventList)

	return eventList, nil
}

func getPreviewEventList() ([]string, error) {
	api, err := spec.LoadSpec(pathStripeSpecV2Preview)
	if err != nil {
		return nil, err
	}

	eventList := make([]string, 0)
	// Iterate over every resource schema
	for _, schema := range api.Components.Schemas {
		// Skip resources that don't have any events
		if schema.XStripeEvent == nil {
			continue
		}

		eventType := schema.XStripeEvent.EventType
		eventList = append(eventList, eventType)
	}

	// Sort the eventList so that we have consistent
	// ordering when testing in CI
	sort.Strings(eventList)

	return eventList, nil
}
