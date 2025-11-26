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

type ApiNamespace = string

const (
	V1Namespace        ApiNamespace = "v1"
	V1PreviewNamespace ApiNamespace = "v1-preview"
	V2Namespace        ApiNamespace = "v2"
	V2PreviewNamespace ApiNamespace = "v2-preview"
)

type TemplateData struct {
	ApiNamespaces map[ApiNamespace]*ApiNamespaceData
}

type ApiNamespaceData struct {
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
	pathTemplate = "../gen/resources_cmds.go.tpl"
	pathName     = "resources_cmds.go.tpl"
	pathOutput   = "resources_cmds.go"

	// stripe version parsing
	stripeVersionTemplatePath = "../gen/stripe_version_header.go.tpl"
	stripeVersionTemplateName = "stripe_version_header.go.tpl"
	stripeVersionPath         = "../requests/stripe_version_header.go"
)

var test_helpers_path = "test_helpers"

func main() {
	// This is the script that generates the `resources.go` file from the
	// OpenAPI spec files.
	//
	// There are two loading strategies:
	// 1. Legacy: Load separate V1, V2, and V2 Preview spec files
	// 2. Unified: Load unified spec files that contain both V1 and V2 together

	templateData, gaVersion, previewVersion, err := getTemplateDataFromUnifiedSpec()
	if err != nil {
		panic(err)
	}

	// Generate the stripe version header
	generateStripeVersionHeader(gaVersion, previewVersion)

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

// getTemplateData loads resource data from separate v1, v2, and v2 preview spec files.
// This is the legacy approach that requires three separate spec files.
// Returns template data along with GA and preview version strings for the stripe version header.
func getTemplateData() (*TemplateData, string, string, error) {
	// Load the v1 OpenAPI spec
	v1Spec, err := spec.LoadSpec(gen.PathStripeSpec)
	if err != nil {
		return nil, "", "", err
	}

	// Load the v2 OpenAPI spec
	v2Spec, err := spec.LoadSpec(gen.PathStripeSpecV2)
	if err != nil {
		return nil, "", "", err
	}

	v2PreviewSpec, err := spec.LoadSpec(gen.PathStripeSpecV2Preview)
	if err != nil {
		return nil, "", "", err
	}

	data := &TemplateData{
		ApiNamespaces: make(map[ApiNamespace]*ApiNamespaceData),
	}

	// Map API namespaces to their respective specs
	apiSpecs := map[ApiNamespace]*spec.Spec{
		V1Namespace:        v1Spec,
		V2Namespace:        v2Spec,
		V2PreviewNamespace: v2PreviewSpec,
	}

	// Process each spec
	for apiNamespace, apiSpec := range apiSpecs {
		// Iterate over every resource schema
		for name, schema := range apiSpec.Components.Schemas {
			// Skip resources that don't have any operations
			if schema.XStripeOperations == nil {
				continue
			}

			// Skip non-public resources except for preview resources
			if schema.XStripeNotPublic && apiNamespace != V2PreviewNamespace {
				continue
			}

			err := genCmdTemplate(apiNamespace, name, name, data, apiSpec)
			if err != nil {
				return nil, "", "", err
			}

			alias := resource.GetCmdAlias(name)
			if alias != "" {
				// Aliased commands write a second entry into the resource commands, and use post-processing to hide the
				// command from the index (e.g. when running `stripe resources`)
				err := genCmdTemplate(apiNamespace, name, alias, data, apiSpec)
				if err != nil {
					return nil, "", "", err
				}
			}
		}
	}

	return data, v2Spec.Info.Version, v2PreviewSpec.Info.Version, nil
}

// getTemplateDataFromUnifiedSpec loads resource data from unified spec files that contain
// both v1 and v2 APIs together. This approach:
//   - Uses path-based detection to determine v1 vs v2 (see getApiNamespaceFromOperations)
//   - Processes GA spec for v1/v2 commands, preview spec for v1-preview/v2-preview commands
//
// Returns template data along with GA and preview version strings for the stripe version header.
func getTemplateDataFromUnifiedSpec() (*TemplateData, string, string, error) {
	// Load both unified specs
	gaSpec, err := spec.LoadSpec(gen.PathUnifiedSpec)
	if err != nil {
		return nil, "", "", err
	}

	previewSpec, err := spec.LoadSpec(gen.PathUnifiedPreviewSpec)
	if err != nil {
		return nil, "", "", err
	}

	data := &TemplateData{
		ApiNamespaces: make(map[ApiNamespace]*ApiNamespaceData),
	}

	// Process GA spec - resources are split into V1 and V2 based on operation paths
	for name, schema := range gaSpec.Components.Schemas {
		if schema.XStripeOperations == nil {
			continue
		}
		if schema.XStripeNotPublic {
			continue
		}

		// Determine API namespace from operation paths (v1 vs v2)
		apiNamespace := getApiNamespaceFromOperations(schema.XStripeOperations, false)

		err := genCmdTemplate(apiNamespace, name, name, data, gaSpec)
		if err != nil {
			return nil, "", "", err
		}

		alias := resource.GetCmdAlias(name)
		if alias != "" {
			err := genCmdTemplate(apiNamespace, name, alias, data, gaSpec)
			if err != nil {
				return nil, "", "", err
			}
		}
	}

	// Process Preview spec - only v2-preview resources (exclude v1-preview for now)
	for name, schema := range previewSpec.Components.Schemas {
		if schema.XStripeOperations == nil {
			continue
		}

		// Determine API namespace from operation paths (v1Preview vs v2Preview)
		apiNamespace := getApiNamespaceFromOperations(schema.XStripeOperations, true)

		// Skip v1-preview resources for now
		if apiNamespace == V1PreviewNamespace {
			continue
		}

		err := genCmdTemplate(apiNamespace, name, name, data, previewSpec)
		if err != nil {
			return nil, "", "", err
		}

		alias := resource.GetCmdAlias(name)
		if alias != "" {
			err := genCmdTemplate(apiNamespace, name, alias, data, previewSpec)
			if err != nil {
				return nil, "", "", err
			}
		}
	}

	return data, gaSpec.Info.Version, previewSpec.Info.Version, nil
}

// getApiNamespaceFromOperations determines the ApiNamespace by examining operation paths.
//
// The operation path is the authoritative source for determining API namespace, and guides
// the path to the CLI command.
//
// Parameters:
//   - ops: the list of operations from a schema's x-stripeOperations (must be non-nil and non-empty)
//   - isPreview: whether this schema comes from the preview spec (determines Preview suffix)
//
// Returns V1Namespace/V2Namespace for GA, or V1PreviewNamespace/V2PreviewNamespace for preview.
func getApiNamespaceFromOperations(ops *[]spec.StripeOperation, isPreview bool) ApiNamespace {
	if ops == nil || len(*ops) == 0 {
		panic("getApiNamespaceFromOperations called with nil or empty operations slice")
	}

	for _, op := range *ops {
		if strings.HasPrefix(op.Path, "/v2/") {
			if isPreview {
				return V2PreviewNamespace
			}
			return V2Namespace
		}
	}
	// Default to v1 if no /v2/ paths found (all paths are /v1/*)
	if isPreview {
		return V1PreviewNamespace
	}
	return V1Namespace
}

func genCmdTemplate(apiNamespace ApiNamespace, schemaName string, cmdName string, data *TemplateData, stripeAPI *spec.Spec) error {
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
				err := addToTemplateData(data, apiNamespace, test_helpers_path, nsName, resName, stripeAPI, op)
				if err != nil {
					return err
				}
			} else {
				err := addToTemplateData(data, apiNamespace, test_helpers_path, resName, "", stripeAPI, op)
				if err != nil {
					return err
				}
			}

			// add test_helpers as a sub-resource to the current namespace-resource entry
			subResName = test_helpers_path
		}

		err := addToTemplateData(data, apiNamespace, nsName, resName, subResName, stripeAPI, op)
		if err != nil {
			return err
		}
	}

	return nil
}

func addToTemplateData(data *TemplateData, apiNamespace ApiNamespace, nsName, resName, subResName string, stripeAPI *spec.Spec, op spec.StripeOperation) error {
	hasSubResources := subResName != ""

	if _, ok := data.ApiNamespaces[apiNamespace]; !ok {
		data.ApiNamespaces[apiNamespace] = &ApiNamespaceData{
			Namespaces: make(map[string]*NamespaceData),
		}
	}

	if _, ok := data.ApiNamespaces[apiNamespace].Namespaces[nsName]; !ok {
		data.ApiNamespaces[apiNamespace].Namespaces[nsName] = &NamespaceData{
			Resources: make(map[string]*ResourceData),
		}
	}

	// If we haven't seen the resource before, initialize it
	resCmdName := resource.GetResourceCmdName(resName)
	if _, ok := data.ApiNamespaces[apiNamespace].Namespaces[nsName].Resources[resCmdName]; !ok {
		data.ApiNamespaces[apiNamespace].Namespaces[nsName].Resources[resCmdName] = &ResourceData{

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
		if _, ok := data.ApiNamespaces[apiNamespace].Namespaces[nsName].Resources[resCmdName].SubResources[subResCmdName]; !ok {
			data.ApiNamespaces[apiNamespace].Namespaces[nsName].Resources[resCmdName].SubResources[subResCmdName] = &ResourceData{
				Operations: make(map[string]*OperationData),
			}
		}
		_, operationExists = data.ApiNamespaces[apiNamespace].Namespaces[nsName].Resources[resCmdName].SubResources[subResCmdName].Operations[op.MethodName]
	} else {
		_, operationExists = data.ApiNamespaces[apiNamespace].Namespaces[nsName].Resources[resCmdName].Operations[op.MethodName]
	}

	// If we haven't seen the operation before, initialize it
	if !operationExists {
		httpString := string(op.Operation)
		specOp := stripeAPI.Paths[spec.Path(op.Path)][spec.HTTPVerb(httpString)]
		// Skip deprecated methods
		if specOp.Deprecated != nil && *specOp.Deprecated == true {
			return nil
		}

		properties, enums := getMethodProperties(apiNamespace, specOp, op)

		if hasSubResources {
			data.ApiNamespaces[apiNamespace].Namespaces[nsName].Resources[resCmdName].SubResources[subResCmdName].Operations[op.MethodName] = &OperationData{
				Path:      op.Path,
				HTTPVerb:  httpString,
				PropFlags: properties,
				EnumFlags: enums,
			}
		} else {
			data.ApiNamespaces[apiNamespace].Namespaces[nsName].Resources[resCmdName].Operations[op.MethodName] = &OperationData{
				Path:      op.Path,
				HTTPVerb:  httpString,
				PropFlags: properties,
				EnumFlags: enums,
			}
		}
	}

	return nil
}

func getMethodProperties(apiNamespace ApiNamespace, specOp *spec.Operation, op spec.StripeOperation) (map[string]string, map[string][]spec.StripeEnumValue) {
	httpString := string(op.Operation)
	properties := make(map[string]string)
	enumValues := make(map[string][]spec.StripeEnumValue)

	if strings.ToUpper(httpString) == http.MethodPost {
		if specOp.RequestBody == nil {
			return properties, enumValues
		}

		mediaType := getMediaType(apiNamespace)
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

// getMediaType returns the content type for request bodies based on API namespace.
// v1 APIs use form-urlencoded, v2 APIs use JSON.
func getMediaType(apiNamespace ApiNamespace) string {
	mediaType := "application/x-www-form-urlencoded"
	if apiNamespace == V2Namespace || apiNamespace == V2PreviewNamespace {
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
