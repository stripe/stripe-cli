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
	"github.com/stripe/stripe-cli/pkg/gen"
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

	pathTemplate = "../gen/resources_cmds.go.tpl"

	pathName = "resources_cmds.go.tpl"

	pathOutput = "resources_cmds.go"
)

var test_helpers_path = "test_helpers"

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

		origNsName, origResName := parseSchemaName(name)

		// Iterate over every operation for the resource
		for _, op := range *schema.XStripeOperations {
			// We're only implementing "service" operations
			if op.MethodOn != "service" {
				continue
			}

			nsName := origNsName
			resName := origResName
			subResName := ""

			if strings.Contains(op.Path, test_helpers_path) && test_helpers_path != nsName {
				// create entry in the test_helpers namespace
				if nsName != "" {
					data, err = addToTemplateData(data, test_helpers_path, nsName, resName, stripeAPI, op)
				} else {
					data, err = addToTemplateData(data, test_helpers_path, resName, "", stripeAPI, op)
				}

				// add test_helpers as a sub-resource to the current namespace-resource entry
				subResName = test_helpers_path
			}

			data, err = addToTemplateData(data, nsName, resName, subResName, stripeAPI, op)
		}
	}

	return data, nil
}

func addToTemplateData(data *TemplateData, nsName, resName, subResName string, stripeAPI *spec.Spec, op spec.StripeOperation) (*TemplateData, error) {
	hasSubResources := subResName != ""

	if _, ok := data.Namespaces[nsName]; !ok {
		data.Namespaces[nsName] = &NamespaceData{
			Resources: make(map[string]*ResourceData),
		}
	}

	// If we haven't seen the resource before, initialize it
	resCmdName := resource.GetResourceCmdName(resName)
	if _, ok := data.Namespaces[nsName].Resources[resCmdName]; !ok {
		data.Namespaces[nsName].Resources[resCmdName] = &ResourceData{
			Operations:   make(map[string]*OperationData),
			SubResources: make(map[string]*ResourceData),
		}
	}

	// check if operations already exists
	operationExists := true
	subResCmdName := ""

	if hasSubResources {
		// If we haven't seen the sub-resource before, initialize it
		subResCmdName = resource.GetResourceCmdName(subResName)
		if _, ok := data.Namespaces[nsName].Resources[resCmdName].SubResources[subResCmdName]; !ok {
			data.Namespaces[nsName].Resources[resCmdName].SubResources[subResCmdName] = &ResourceData{
				Operations: make(map[string]*OperationData),
			}
		}
		_, operationExists = data.Namespaces[nsName].Resources[resCmdName].SubResources[subResCmdName].Operations[op.MethodName]
	} else {
		_, operationExists = data.Namespaces[nsName].Resources[resCmdName].Operations[op.MethodName]
	}

	// If we haven't seen the operation before, initialize it
	if !operationExists {
		httpString := string(op.Operation)
		properties := make(map[string]string)

		specOp := stripeAPI.Paths[spec.Path(op.Path)][spec.HTTPVerb(httpString)]

		// Skip deprecated methods
		if specOp.Deprecated != nil && *specOp.Deprecated == true {
			return data, nil
		}

		if strings.ToUpper(httpString) == http.MethodPost {
			requestContent := specOp.RequestBody.Content

			if media, ok := requestContent["application/x-www-form-urlencoded"]; ok {
				for propName, schema := range media.Schema.Properties {
					// If property is metadata or expand, skip it
					if propName == "metadata" || propName == "expand" {
						continue
					}

					if schema.Type == "object" {
						denormalizedProps := gen.DenormalizeObject(propName, schema)
						for prop, propType := range denormalizedProps {
							properties[prop] = propType
						}

					} else {
						scalarType := gen.GetType(schema)

						if scalarType == nil {
							continue
						}

						properties[propName] = *scalarType
					}
				}
			}
		} else {
			for _, param := range specOp.Parameters {
				// Only create flags for query string parameters
				if param.In != "query" {
					continue
				}

				// Skip metadata and expand params
				if param.Name == "metadata" || param.Name == "expand" {
					continue
				}

				schema := param.Schema
				scalarType := gen.GetType(schema)

				if scalarType == nil {
					continue
				}

				properties[param.Name] = *scalarType
			}
		}

		if hasSubResources {
			data.Namespaces[nsName].Resources[resCmdName].SubResources[subResCmdName].Operations[op.MethodName] = &OperationData{
				Path:      op.Path,
				HTTPVerb:  httpString,
				PropFlags: properties,
			}
		} else {
			data.Namespaces[nsName].Resources[resCmdName].Operations[op.MethodName] = &OperationData{
				Path:      op.Path,
				HTTPVerb:  httpString,
				PropFlags: properties,
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
