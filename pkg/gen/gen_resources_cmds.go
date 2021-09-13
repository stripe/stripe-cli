//go:build gen_resources
// +build gen_resources

package main

import (
	"bytes"
	"fmt"
	"go/format"
	"io/ioutil"
	"net/http"
	"strings"
	"text/template"

	"github.com/iancoleman/strcase"

	"github.com/stripe/stripe-cli/pkg/cmd/resource"
	"github.com/stripe/stripe-cli/pkg/spec"
)

type TemplateData struct {
	Namespaces map[string]*NamespaceData
}

type NamespaceData struct {
	Resources map[string]*ResourceData
}

type ResourceData struct {
	Operations map[string]*OperationData
}

type OperationData struct {
	Path      string
	HTTPVerb  string
	PropFlags map[string]string
}

const (
	pathStripeSpec = "../../api/openapi-spec/spec3.sdk.json"

	pathTemplate = "../gen/resources_cmds.go.tpl"

	pathName = "resources_cmds.go.tpl"

	pathOutput = "resources_cmds.go"
)

var scalarTypes = map[string]bool{
	"boolean": true,
	"integer": true,
	"number":  true,
	"string":  true,
}

func main() {
	// This is the script that generates the `resources.go` file from the
	// OpenAPI spec file.

	// Load the spec and prepare the template data
	templateData, err := getTemplateData()
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
	err = ioutil.WriteFile(pathOutput, formatted, 0644)
	if err != nil {
		panic(err)
	}
}

func getTemplateData() (*TemplateData, error) {
	data := &TemplateData{
		Namespaces: make(map[string]*NamespaceData),
	}

	// Load the JSON OpenAPI spec
	stripeAPI, err := spec.LoadSpec(pathStripeSpec)
	if err != nil {
		return nil, err
	}

	// Iterate over every resource schema
	for name, schema := range stripeAPI.Components.Schemas {
		// Skip resources that don't have any operations
		if schema.XStripeOperations == nil {
			continue
		}

		nsName, resName := parseSchemaName(name)

		// Iterate over every operation for the resource
		for _, op := range *schema.XStripeOperations {
			// We're only implementing "service" operations
			if op.MethodOn != "service" {
				continue
			}

			// If we haven't seen the namespace before, initialize it
			if _, ok := data.Namespaces[nsName]; !ok {
				data.Namespaces[nsName] = &NamespaceData{
					Resources: make(map[string]*ResourceData),
				}
			}

			// If we haven't seen the resource before, initialize it
			resCmdName := resource.GetResourceCmdName(resName)
			if _, ok := data.Namespaces[nsName].Resources[resCmdName]; !ok {
				data.Namespaces[nsName].Resources[resCmdName] = &ResourceData{
					Operations: make(map[string]*OperationData),
				}
			}

			// If we haven't seen the operation before, initialize it
			if _, ok := data.Namespaces[nsName].Resources[resCmdName].Operations[op.MethodName]; !ok {
				httpString := string(op.Operation)
				properties := make(map[string]string)

				specOp := stripeAPI.Paths[spec.Path(op.Path)][spec.HTTPVerb(httpString)]

				// Skip deprecated methods
				if specOp.Deprecated != nil && *specOp.Deprecated == true {
					continue
				}

				if strings.ToUpper(httpString) == http.MethodPost {
					requestContent := specOp.RequestBody.Content

					if media, ok := requestContent["application/x-www-form-urlencoded"]; ok {
						for propName, schema := range media.Schema.Properties {
							scalarType := getScalarType(schema)

							if scalarType == nil {
								continue
							}

							properties[propName] = *scalarType
						}
					}
				} else {
					for _, param := range specOp.Parameters {
						// Only create flags for query string parameters
						if param.In != "query" {
							continue
						}

						schema := param.Schema
						scalarType := getScalarType(schema)

						if scalarType == nil {
							continue
						}

						properties[param.Name] = *scalarType
					}
				}

				data.Namespaces[nsName].Resources[resCmdName].Operations[op.MethodName] = &OperationData{
					Path:      op.Path,
					HTTPVerb:  httpString,
					PropFlags: properties,
				}
			}
		}
	}

	return data, nil
}

func parseSchemaName(name string) (string, string) {
	if strings.Contains(name, ".") {
		components := strings.SplitN(name, ".", 2)
		return components[0], components[1]
	}
	return "", name
}

// getScalarType accepts a schema and returns its scalar type, if it has one.
//
// If the schema is monomorphic, it returns its type if it's scalar.
//
// If the schema is polymorphic, it returns the first scalar type for the
// schema, if there is any.
func getScalarType(schema *spec.Schema) *string {
	if len(schema.AnyOf) > 0 {
		for _, subSchema := range schema.AnyOf {
			scalarType := getScalarType(subSchema)
			if scalarType != nil {
				return scalarType
			}
		}
	} else if scalarTypes[schema.Type] {
		// Special case for string types that only support the "" (empty
		// string) value: we consider these to be non-scalar so we don't
		// generate a flag for those.
		if schema.Type == "string" {
			if len(schema.Enum) == 1 && schema.Enum[0] == "" {
				return nil
			}
		}
		return &schema.Type
	}

	return nil
}
