//go:build gen_resources
// +build gen_resources

package main

import (
	"bytes"
	"fmt"
	"go/format"
	"net/http"
	"os"
	"path/filepath"
	"sort"
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
	Name      string // operation name, e.g. "create"
	Path      string
	HTTPVerb  string
	IsPreview bool
	ServerURL string
	VarName   string
	Params    map[string]*resource.ParamSpec
}

// StripeVersionTemplateData stores the stripe version parsed from api spec
type StripeVersionTemplateData struct {
	StripeVersion        string
	StripePreviewVersion string
}

const (
	pathTemplate = "../gen/resources_cmds.go.tpl"
	pathName     = "resources_cmds.go.tpl"

	pathSpecTemplate = "../gen/resource_specs.go.tpl"
	pathSpecName     = "resource_specs.go.tpl"

	// output directory for generated files (relative to pkg/cmd/ working dir)
	outputDir = "resources"

	// stripe version parsing
	stripeVersionTemplatePath = "../gen/stripe_version_header.go.tpl"
	stripeVersionTemplateName = "stripe_version_header.go.tpl"
	stripeVersionPath         = "../requests/stripe_version_header.go"
)

// templateFuncs is the shared function map used across all generator templates.
var templateFuncs = template.FuncMap{
	"ToCamel": strcase.ToCamel,
	"quote":   func(s string) string { return fmt.Sprintf("%q", s) },
	"upper":   strings.ToUpper,
}

var test_helpers_path = "test_helpers"

func main() {
	// This is the script that generates resource commands from the OpenAPI spec files.
	templateData, gaVersion, previewVersion, err := getTemplateData()
	if err != nil {
		panic(err)
	}

	// Generate the stripe version header
	generateStripeVersionHeader(gaVersion, previewVersion)

	// Ensure output directory exists
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		panic(err)
	}

	// Remove stale generated files before writing new ones so that renamed or
	// removed namespaces don't leave orphaned *_gen.go files behind.
	if err := cleanGeneratedFiles(outputDir); err != nil {
		panic(err)
	}

	// Load the wiring template
	tmpl := template.Must(template.
		New(pathName).
		Funcs(templateFuncs).
		ParseFiles(pathTemplate))

	// Execute the wiring template into resources/resources_gen.go
	var result bytes.Buffer
	err = tmpl.Execute(&result, templateData)
	if err != nil {
		panic(err)
	}

	formatted, err := format.Source(result.Bytes())
	if err != nil {
		panic(fmt.Errorf("format error in resources_gen.go: %w\n%s", err, result.String()))
	}

	wiringPath := filepath.Join(outputDir, "resources_gen.go")
	fmt.Printf("writing %s\n", wiringPath)
	if err := os.WriteFile(wiringPath, formatted, 0644); err != nil {
		panic(err)
	}

	// Generate spec files (one per api-namespace × sub-namespace group)
	if err := generateSpecFiles(templateData, outputDir); err != nil {
		panic(err)
	}
}

func generateStripeVersionHeader(version string, previewVersion string) {
	stripeVersionTemplateData := &StripeVersionTemplateData{
		StripeVersion:        version,
		StripePreviewVersion: previewVersion,
	}

	tmpl := template.Must(template.
		New(stripeVersionTemplateName).
		Funcs(templateFuncs).
		ParseFiles(stripeVersionTemplatePath))

	var result bytes.Buffer
	err := tmpl.Execute(&result, stripeVersionTemplateData)
	if err != nil {
		panic(err)
	}

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

// getTemplateData loads resource data from unified spec files.
func getTemplateData() (*TemplateData, string, string, error) {
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

	// Process GA spec
	for name, schema := range gaSpec.Components.Schemas {
		if schema.XStripeOperations == nil {
			continue
		}
		if schema.XStripeNotPublic {
			continue
		}

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

	// Process Preview spec - only v2-preview resources
	for name, schema := range previewSpec.Components.Schemas {
		if schema.XStripeOperations == nil {
			continue
		}

		apiNamespace := getApiNamespaceFromOperations(schema.XStripeOperations, true)

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
	if isPreview {
		return V1PreviewNamespace
	}
	return V1Namespace
}

func genCmdTemplate(apiNamespace ApiNamespace, schemaName string, cmdName string, data *TemplateData, stripeAPI *spec.Spec) error {
	origNsName, origResName := parseSchemaName(cmdName)
	schema := stripeAPI.Components.Schemas[schemaName]

	for _, op := range *schema.XStripeOperations {
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

	resCmdName := resource.GetResourceCmdName(resName)
	if _, ok := data.ApiNamespaces[apiNamespace].Namespaces[nsName].Resources[resCmdName]; !ok {
		data.ApiNamespaces[apiNamespace].Namespaces[nsName].Resources[resCmdName] = &ResourceData{
			Operations:   make(map[string]*OperationData),
			SubResources: make(map[string]*ResourceData),
		}
	}

	operationExists := true
	subResCmdName := ""

	if hasSubResources {
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

	if !operationExists {
		httpString := string(op.Operation)
		specOp := stripeAPI.Paths[spec.Path(op.Path)][spec.HTTPVerb(httpString)]
		if specOp.Deprecated != nil && *specOp.Deprecated == true {
			return nil
		}

		isPreview := apiNamespace == V1PreviewNamespace || apiNamespace == V2PreviewNamespace
		params := getOperationParams(apiNamespace, specOp, op)

		serverURL := ""
		if len(specOp.Servers) > 0 {
			serverURL = specOp.Servers[0].URL
		}

		varName := computeVarName(apiNamespace, nsName, resCmdName, subResCmdName, op.MethodName)

		opData := &OperationData{
			Name:      op.MethodName,
			Path:      op.Path,
			HTTPVerb:  httpString,
			IsPreview: isPreview,
			ServerURL: serverURL,
			VarName:   varName,
			Params:    params,
		}

		if hasSubResources {
			data.ApiNamespaces[apiNamespace].Namespaces[nsName].Resources[resCmdName].SubResources[subResCmdName].Operations[op.MethodName] = opData
		} else {
			data.ApiNamespaces[apiNamespace].Namespaces[nsName].Resources[resCmdName].Operations[op.MethodName] = opData
		}
	}

	return nil
}

// getOperationParams extracts parameter metadata from a spec operation, returning a map of
// parameter name → ParamSpec. Parameter names use dot notation for nested fields.
func getOperationParams(apiNamespace ApiNamespace, specOp *spec.Operation, op spec.StripeOperation) map[string]*resource.ParamSpec {
	params := make(map[string]*resource.ParamSpec)
	httpString := string(op.Operation)

	if strings.ToUpper(httpString) == http.MethodPost {
		if specOp.RequestBody == nil {
			return params
		}

		mediaType := getMediaType(apiNamespace)
		requestContent := specOp.RequestBody.Content

		if media, ok := requestContent[mediaType]; ok {
			requiredSet := make(map[string]bool)
			for _, req := range media.Schema.Required {
				requiredSet[req] = true
			}

			for propName, schema := range media.Schema.Properties {
				if propName == "metadata" || propName == "expand" {
					continue
				}
				if schema.XStripeNotPublic {
					continue
				}

				if schema.Type == "object" {
					addDenormalizedParams(params, propName, schema)
				} else {
					scalarType := gen.GetType(schema)
					if scalarType == nil {
						continue
					}

					ps := &resource.ParamSpec{
						Type:     *scalarType,
						Required: requiredSet[propName],
						Format:   schema.Format,
						Enum:     mergeEnumValues(schema),
					}
					params[propName] = ps
				}
			}
		}
	} else {
		for _, param := range specOp.Parameters {
			if param.In != "query" {
				continue
			}
			if param.Name == "metadata" || param.Name == "expand" {
				continue
			}

			schema := param.Schema
			scalarType := gen.GetType(schema)
			if scalarType == nil {
				continue
			}

			ps := &resource.ParamSpec{
				Type:     *scalarType,
				Required: param.Required,
				Format:   schema.Format,
				Enum:     mergeEnumValues(schema),
			}
			params[param.Name] = ps
		}
	}

	return params
}

// addDenormalizedParams recursively flattens an object schema into dot-notation params.
func addDenormalizedParams(params map[string]*resource.ParamSpec, prefix string, schema *spec.Schema) {
	for propName, propSchema := range schema.Properties {
		key := prefix + "." + propName

		if propSchema.Type == "object" {
			addDenormalizedParams(params, key, propSchema)
		} else {
			scalarType := gen.GetType(propSchema)
			if scalarType == nil {
				continue
			}

			isRequired := containsStr(schema.Required, propName)
			ps := &resource.ParamSpec{
				Type:     *scalarType,
				Required: isRequired,
				Format:   propSchema.Format,
				Enum:     mergeEnumValues(propSchema),
			}
			params[key] = ps
		}
	}
}

// mergeEnumValues builds the EnumSpec slice from a schema's enum/x-stripeEnum fields.
// enum provides the complete set of values; x-stripeEnum provides descriptions for a subset.
func mergeEnumValues(schema *spec.Schema) []resource.EnumSpec {
	if len(schema.Enum) == 0 && len(schema.XStripeEnum) == 0 {
		return nil
	}

	if len(schema.Enum) > 0 {
		var enums []resource.EnumSpec
		for _, rawVal := range schema.Enum {
			val, ok := rawVal.(string)
			if !ok || val == "" {
				continue
			}
			enums = append(enums, resource.EnumSpec{
				Value: val,
			})
		}
		return enums
	}

	// Fall back to x-stripeEnum only
	var enums []resource.EnumSpec
	for _, ev := range schema.XStripeEnum {
		enums = append(enums, resource.EnumSpec{
			Value: ev.Value,
		})
	}
	return enums
}

func containsStr(slice []string, s string) bool {
	for _, v := range slice {
		if v == s {
			return true
		}
	}
	return false
}

// computeVarName returns the exported Go variable name for an OperationSpec.
// Format: {ApiNsPrefix}{NsCamel}{ResCamel}{SubResCamel}{OpCamel}
// e.g. V1BillingMetersCreate, V1CustomersCreate, V1ChargesTestHelpersCapture
func computeVarName(apiNamespace ApiNamespace, nsName, resCmdName, subResCmdName, opName string) string {
	prefix := apiNsPrefix(apiNamespace)
	nsCase := strcase.ToCamel(nsName)
	resCase := strcase.ToCamel(resCmdName)
	subResCase := ""
	if subResCmdName != "" {
		subResCase = strcase.ToCamel(subResCmdName)
	}
	opCase := strcase.ToCamel(opName)
	return prefix + nsCase + resCase + subResCase + opCase
}

func apiNsPrefix(apiNamespace ApiNamespace) string {
	switch apiNamespace {
	case V1Namespace:
		return "V1"
	case V1PreviewNamespace:
		return "V1Preview"
	case V2Namespace:
		return "V2"
	case V2PreviewNamespace:
		return "V2Preview"
	default:
		return "V" + strcase.ToCamel(strings.ReplaceAll(apiNamespace, "-", " "))
	}
}

type varEntry struct {
	VarName string
	Op      *OperationData
}

// cleanGeneratedFiles removes all *_gen.go files from dir so stale outputs
// from previous generator runs (e.g. after a namespace rename or removal) are
// not left behind.
func cleanGeneratedFiles(dir string) error {
	matches, err := filepath.Glob(filepath.Join(dir, "*_gen.go"))
	if err != nil {
		return err
	}
	for _, f := range matches {
		if err := os.Remove(f); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("removing stale generated file %s: %w", f, err)
		}
	}
	return nil
}

// generateSpecFiles writes a single specs_gen.go file containing all operation
// spec variables, sorted by VarName for stable output.
func generateSpecFiles(data *TemplateData, outputDir string) error {
	tmpl := template.Must(template.
		New(pathSpecName).
		Funcs(templateFuncs).
		ParseFiles(pathSpecTemplate))

	var entries []varEntry
	for _, nsData := range data.ApiNamespaces {
		for _, nsDataInner := range nsData.Namespaces {
			for _, resData := range nsDataInner.Resources {
				for _, opData := range resData.Operations {
					entries = append(entries, varEntry{VarName: opData.VarName, Op: opData})
				}
				for _, subResData := range resData.SubResources {
					for _, opData := range subResData.Operations {
						entries = append(entries, varEntry{VarName: opData.VarName, Op: opData})
					}
				}
			}
		}
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].VarName < entries[j].VarName
	})

	var result bytes.Buffer
	if err := tmpl.Execute(&result, entries); err != nil {
		return fmt.Errorf("spec file specs_gen.go: %w", err)
	}

	formatted, err := format.Source(result.Bytes())
	if err != nil {
		return fmt.Errorf("format error in specs_gen.go: %w\n%s", err, result.String())
	}

	filePath := filepath.Join(outputDir, "specs_gen.go")
	fmt.Printf("writing %s\n", filePath)
	return os.WriteFile(filePath, formatted, 0644)
}

// getMediaType returns the content type for request bodies based on API namespace.
func getMediaType(apiNamespace ApiNamespace) string {
	if apiNamespace == V2Namespace || apiNamespace == V2PreviewNamespace {
		return "application/json"
	}
	return "application/x-www-form-urlencoded"
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
