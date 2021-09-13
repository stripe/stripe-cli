//go:build events_list
// +build events_list

package main

import (
	"bytes"
	"fmt"
	"go/format"
	"io/ioutil"
	"text/template"

	"github.com/stripe/stripe-cli/pkg/spec"
)

type TemplateData struct {
	Events []string
}

const (
	pathStripeSpec = "../../api/openapi-spec/spec3.sdk.json"

	pathTemplate = "../gen/events_list.go.tpl"

	pathName = "events_list.go.tpl"

	pathOutput = "../proxy/events_list.go"
)

func main() {
	// generate `events_list.go` from OpenAPI spec file
	// code for this func from gen_resources_cmds.go

	// load spec
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
	err = ioutil.WriteFile(pathOutput, formatted, 0644)
	if err != nil {
		panic(err)
	}

}

func getTemplateData() (*TemplateData, error) {
	data := &TemplateData{
		Events: make([]string, 0),
	}

	// load API spec
	api, err := spec.LoadSpec(pathStripeSpec)
	if err != nil {
		return nil, err
	}

	postRequest := api.Paths["/v1/webhook_endpoints"]["post"]
	requestSchema := postRequest.RequestBody.Content["application/x-www-form-urlencoded"].Schema
	events := requestSchema.Properties["enabled_events"].Items.Enum
	for _, e := range events {
		data.Events = append(data.Events, e.(string))
	}

	return data, nil

}
