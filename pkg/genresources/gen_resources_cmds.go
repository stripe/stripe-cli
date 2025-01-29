package main

import (
	"bytes"
	"fmt"
	"go/format"
	"os"
	"text/template"

	"github.com/iancoleman/strcase"
	"github.com/stripe/stripe-cli/pkg/genresources/templatedata"
	"github.com/stripe/stripe-cli/pkg/spec"
)

type TemplateData struct {
	Namespaces map[string]*NamespaceData
}

type NamespaceData struct {
	Resources map[string]*ResourceData
}

type ResourceData struct {
	Operations   map[string]*OperationData
	SubResources map[string]*ResourceData
}

type OperationData struct {
	Path      string
	HTTPVerb  string
	PropFlags map[string]string
}

const (
	pathStripeSpec = "../../api/openapi-spec/spec3.sdk.json"

	pathTemplate = "../genresources/resources_cmds.go.tpl"

	pathName = "resources_cmds.go.tpl"

	pathOutput = "resources_cmds.go"
)

var test_helpers_path = "test_helpers"

func main() {
	// This is the script that generates the `resources.go` file from the
	// OpenAPI spec file.

	// Load the JSON OpenAPI spec
	stripeAPI, err := spec.LoadSpec(pathStripeSpec)
	if err != nil {
		panic(err)
	}

	// Prepare the template data
	templateData, err := templatedata.GetTemplateData(stripeAPI)
	if err != nil {
		panic(err)
	}

	// Load the template with a custom function map
	tmpl := template.Must(template.
		// Note that the template name MUST match the file name
		New(pathName).
		Funcs(template.FuncMap{
			// The `ToCamel` function is used to turn snake_case strings to
			// CamelCase strings. The template uses this to form Go variable
			// names.
			"ToCamel": strcase.ToCamel,
		}).
		ParseFiles(pathTemplate))

	// Execute the template
	var result bytes.Buffer
	err = tmpl.Execute(&result, templateData)
	if err != nil {
		panic(err)
	}

	// Format the output of the template execution
	formatted, err := format.Source(result.Bytes())
	if err != nil {
		panic(err)
	}

	// Write the formatted source code to disk
	fmt.Printf("writing %s\n", pathOutput)
	err = os.WriteFile(pathOutput, formatted, 0644)
	if err != nil {
		panic(err)
	}
}
