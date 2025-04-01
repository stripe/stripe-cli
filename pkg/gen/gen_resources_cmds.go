//go:build gen_resources
// +build gen_resources

package main

import (
	"bytes"
	"fmt"
	"go/format"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"text/template"

	"github.com/iancoleman/strcase"

	"github.com/stripe/stripe-cli/pkg/cmd/resource"
	"github.com/stripe/stripe-cli/pkg/gen"
	"github.com/stripe/stripe-cli/pkg/spec"
)

type SpecVersion = string

const (
	V1Spec        SpecVersion = "v1"
	V2Spec        SpecVersion = "v2"
	V2PreviewSpec SpecVersion = "v2-preview"
)

type TemplateData struct {
	Versions map[SpecVersion]*VersionData
}

type VersionData struct {
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
	EnumFlags map[string][]spec.StripeEnumValue
}

// StripeVersionTemplateData stores the stripe version parsed from api spec
type StripeVersionTemplateData struct {
	StripeVersion        string
	StripePreviewVersion string
}

const (
	pathStripeSpec = "../../api/openapi-spec/spec3.sdk.json"

	pathStripeSpecV2 = "../../api/openapi-spec/spec3.v2.sdk.json"

	pathStripeSpecV2Preview = "../../api/openapi-spec/spec3.v2.sdk.preview.json"

	pathTemplate = "../gen/resources_cmds.go.tpl"

	pathName = "resources_cmds.go.tpl"

	pathOutput = "resources_cmds.go"

	// stripe version parsing
	stripeVersionTemplatePath = "../gen/stripe_version_header.go.tpl"
	stripeVersionTemplateName = "stripe_version_header.go.tpl"
	stripeVersionPath         = "../requests/stripe_version_header.go"
)

var test_helpers_path = "test_helpers"

func main() {
	// This is the script that generates the `resources.go` file from the
	// OpenAPI spec files.

	// Load the v1 OpenAPI spec
	v1Spec, err := spec.LoadSpec(pathStripeSpec)
	if err != nil {
		panic(err)
	}

	// Load the v2 OpenAPI spec
	v2Spec, err := spec.LoadSpec(pathStripeSpecV2)
	if err != nil {
		panic(err)
	}

	v2PreviewSpec, err := spec.LoadSpec(pathStripeSpecV2Preview)
	if err != nil {
		panic(err)
	}

	// Generate the stripe version header
	v2Version := v2Spec.Info.Version
	v2PreviewVersion := v2PreviewSpec.Info.Version

	generateStripeVersionHeader(v2Version, v2PreviewVersion)

	// Prepare the template data
	specs := map[SpecVersion]*spec.Spec{
		V1Spec:        v1Spec,
		V2Spec:        v2Spec,
		V2PreviewSpec: v2PreviewSpec,
	}
	templateData, err := getTemplateData(specs)
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

func generateStripeVersionHeader(version string, previewVersion string) {
	// This generates `stripe_version_header.go`
	stripeVersionTemplateData := &StripeVersionTemplateData{
		StripeVersion:        version,
		StripePreviewVersion: previewVersion,
	}

	tmpl := template.Must(template.
		New(stripeVersionTemplateName).
		Funcs(template.FuncMap{
			"ToCamel": strcase.ToCamel,
		}).
		ParseFiles(stripeVersionTemplatePath))

	// Execute the template
	var result bytes.Buffer
	err := tmpl.Execute(&result, stripeVersionTemplateData)
	if err != nil {
		panic(err)
	}

	// Format the output of the template execution
	formatted, err := format.Source(result.Bytes())
	if err != nil {
		panic(err)
	}

	fmt.Printf("writing %s\n", stripeVersionPath)
	err = os.WriteFile(stripeVersionPath, formatted, 0644)
	if err != nil {
		panic(err)
	}
}

func getTemplateData(apiSpecs map[SpecVersion]*spec.Spec) (*TemplateData, error) {
	data := &TemplateData{
		Versions: make(map[SpecVersion]*VersionData),
	}

	for version, apiSpec := range apiSpecs {
		// Iterate over every resource schema
		for name, schema := range apiSpec.Components.Schemas {
			// Skip resources that don't have any operations
			if schema.XStripeOperations == nil {
				continue
			}

			// Skip non-public resources except for preview resources
			if schema.XStripeNotPublic && version != V2PreviewSpec {
				continue
			}

			err := genCmdTemplate(version, name, name, data, apiSpec)
			if err != nil {
				return nil, err
			}

			alias := resource.GetCmdAlias(name)

			if alias != "" {
				// Aliased commands write a second entry into the resource commands, and use post-processing to hide the
				// command from the index (e.g. when running `stripe resources`)
				err := genCmdTemplate(version, name, alias, data, apiSpec)
				if err != nil {
					return nil, err
				}
			}
		}
	}

	return data, nil
}

func genCmdTemplate(specVersion SpecVersion, schemaName string, cmdName string, data *TemplateData, stripeAPI *spec.Spec) error {
	origNsName, origResName := parseSchemaName(cmdName)
	schema := stripeAPI.Components.Schemas[schemaName]

	// Iterate over every operation for the resource
	for _, op := range *schema.XStripeOperations {
		// We're only implementing "service" operations
		if op.MethodOn != "service" {
			continue
		}

		nsName := origNsName
		resName := origResName
		subResName := ""

		if strings.Contains(resName, ".") {
			components := strings.SplitN(resName, ".", 2)
			resName = components[0]
			subResName = components[1]
		} else if strings.Contains(op.Path, test_helpers_path) && test_helpers_path != nsName {
			// create entry in the test_helpers namespace
			if nsName != "" {
				err := addToTemplateData(data, specVersion, test_helpers_path, nsName, resName, stripeAPI, op)
				if err != nil {
					return err
				}
			} else {
				err := addToTemplateData(data, specVersion, test_helpers_path, resName, "", stripeAPI, op)
				if err != nil {
					return err
				}
			}

			// add test_helpers as a sub-resource to the current namespace-resource entry
			subResName = test_helpers_path
		}

		err := addToTemplateData(data, specVersion, nsName, resName, subResName, stripeAPI, op)
		if err != nil {
			return err
		}
	}

	return nil
}

func addToTemplateData(data *TemplateData, specVersion SpecVersion, nsName, resName, subResName string, stripeAPI *spec.Spec, op spec.StripeOperation) error {
	hasSubResources := subResName != ""

	if _, ok := data.Versions[specVersion]; !ok {
		data.Versions[specVersion] = &VersionData{
			Namespaces: make(map[string]*NamespaceData),
		}
	}

	if _, ok := data.Versions[specVersion].Namespaces[nsName]; !ok {
		data.Versions[specVersion].Namespaces[nsName] = &NamespaceData{
			Resources: make(map[string]*ResourceData),
		}
	}

	// If we haven't seen the resource before, initialize it
	resCmdName := resource.GetResourceCmdName(resName)
	if _, ok := data.Versions[specVersion].Namespaces[nsName].Resources[resCmdName]; !ok {
		data.Versions[specVersion].Namespaces[nsName].Resources[resCmdName] = &ResourceData{

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
		if _, ok := data.Versions[specVersion].Namespaces[nsName].Resources[resCmdName].SubResources[subResCmdName]; !ok {
			data.Versions[specVersion].Namespaces[nsName].Resources[resCmdName].SubResources[subResCmdName] = &ResourceData{
				Operations: make(map[string]*OperationData),
			}
		}
		_, operationExists = data.Versions[specVersion].Namespaces[nsName].Resources[resCmdName].SubResources[subResCmdName].Operations[op.MethodName]
	} else {
		_, operationExists = data.Versions[specVersion].Namespaces[nsName].Resources[resCmdName].Operations[op.MethodName]
	}

	// If we haven't seen the operation before, initialize it
	if !operationExists {
		httpString := string(op.Operation)
		specOp := stripeAPI.Paths[spec.Path(op.Path)][spec.HTTPVerb(httpString)]
		// Skip deprecated methods
		if specOp.Deprecated != nil && *specOp.Deprecated == true {
			return nil
		}

		properties, enums := getMethodProperties(specVersion, specOp, op)

		if hasSubResources {
			data.Versions[specVersion].Namespaces[nsName].Resources[resCmdName].SubResources[subResCmdName].Operations[op.MethodName] = &OperationData{
				Path:      op.Path,
				HTTPVerb:  httpString,
				PropFlags: properties,
				EnumFlags: enums,
			}
		} else {
			data.Versions[specVersion].Namespaces[nsName].Resources[resCmdName].Operations[op.MethodName] = &OperationData{
				Path:      op.Path,
				HTTPVerb:  httpString,
				PropFlags: properties,
				EnumFlags: enums,
			}
		}
	}

	return nil
}

func getMethodProperties(specVersion SpecVersion, specOp *spec.Operation, op spec.StripeOperation) (map[string]string, map[string][]spec.StripeEnumValue) {
	httpString := string(op.Operation)
	properties := make(map[string]string)
	enumValues := make(map[string][]spec.StripeEnumValue)

	if strings.ToUpper(httpString) == http.MethodPost {
		if specOp.RequestBody == nil {
			return properties, enumValues
		}

		mediaType := getMediaType(specVersion)
		requestContent := specOp.RequestBody.Content

		if media, ok := requestContent[mediaType]; ok {
			for propName, schema := range media.Schema.Properties {
				// If property is metadata or expand, skip it
				if propName == "metadata" || propName == "expand" {
					continue
				}

				if schema.XStripeNotPublic {
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

					// Save enum values if they exist
					if len(schema.XStripeEnum) > 0 {
						enumValues[propName] = schema.XStripeEnum
					}
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

	return properties, enumValues
}

func getMediaType(specVersion SpecVersion) string {
	mediaType := "application/x-www-form-urlencoded"
	if specVersion == V2Spec || specVersion == V2PreviewSpec {
		mediaType = "application/json"
	}

	return mediaType
}

func parseSchemaName(name string) (string, string) {
	if strings.HasPrefix(name, "v2.") {
		name = strings.TrimPrefix(name, "v2.")
	}

	if strings.Contains(name, ".") {
		components := strings.SplitN(name, ".", 2)
		return components[0], components[1]
	}
	return "", name
}
