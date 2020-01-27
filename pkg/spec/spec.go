package spec

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
)

//
// Public values
//

// A set of constants for the different types of possible OpenAPI parameters.
const (
	ParameterPath  = "path"
	ParameterQuery = "query"
)

// A set of constant for the named types available in JSON Schema.
const (
	TypeArray   = "array"
	TypeBoolean = "boolean"
	TypeInteger = "integer"
	TypeNumber  = "number"
	TypeObject  = "object"
	TypeString  = "string"
)

//
// Public types
//

// Components is a struct for the components section of an OpenAPI
// specification.
type Components struct {
	Schemas map[string]*Schema `json:"schemas"`
}

// ExpansionResources is a struct for possible expansions in a resource.
type ExpansionResources struct {
	OneOf []*Schema `json:"oneOf"`
}

// Fixtures is a struct for a set of companion fixtures for an OpenAPI
// specification.
type Fixtures struct {
	Resources map[ResourceID]interface{} `json:"resources"`
}

// HTTPVerb is a type for an HTTP verb like GET, POST, etc.
type HTTPVerb string

// Info is the `info` portion of an OpenAPI specification that contains meta
// information about it.
type Info struct {
	// Version is the Stripe API version represented in the specification. It
	// takes a date-based form like `2019-02-19`.
	Version string `json:"version"`
}

// StripeOperation is a struct for possible operations on a resource.
type StripeOperation struct {
	MethodName string   `json:"method_name"`
	MethodOn   string   `json:"method_on"`
	MethodType string   `json:"method_type"`
	Operation  HTTPVerb `json:"operation"`
	Path       string   `json:"path"`
}

// This is a list of fields that either we handle properly or we're confident
// it's safe to ignore. If a field not in this list appears in the OpenAPI spec,
// then we'll get an error so we remember to update stripe-mock to support it.
var supportedSchemaFields = []string{
	"$ref",
	"additionalProperties",
	"anyOf",
	"description",
	"enum",
	"format",
	"items",
	"maxLength",
	"nullable",
	"pattern",
	"properties",
	"required",
	"title",
	"type",
	"x-expandableFields",
	"x-expansionResources",
	"x-resourceId",
	"x-stripeOperations",

	// This is currently being used to store additional metadata for our SDKs. It's
	// passed through our Spec and should be ignored
	"x-stripeParam",
	"x-stripeResource",

	// This is currently a hint for the server-side so I haven't included it in
	// Schema yet. If we do start validating responses that come out of
	// stripe-mock, we may need to observe this as well.
	"x-stripeBypassValidation",
}

// Schema is a struct representing a JSON schema.
type Schema struct {
	// AdditionalProperties is either a `false` to indicate that no additional
	// properties in the object are allowed (beyond what's in Properties), or a
	// JSON schema that describes the expected format of any additional properties.
	//
	// We currently just read it as an `interface{}` because we're not using it
	// for anything right now.
	AdditionalProperties interface{} `json:"additionalProperties,omitempty"`

	AnyOf      []*Schema          `json:"anyOf,omitempty"`
	Enum       []interface{}      `json:"enum,omitempty"`
	Format     string             `json:"format,omitempty"`
	Items      *Schema            `json:"items,omitempty"`
	MaxLength  int                `json:"maxLength,omitempty"`
	Nullable   bool               `json:"nullable,omitempty"`
	Pattern    string             `json:"pattern,omitempty"`
	Properties map[string]*Schema `json:"properties,omitempty"`
	Required   []string           `json:"required,omitempty"`
	Type       string             `json:"type,omitempty"`

	// Ref is populated if this JSON Schema is actually a JSON reference, and
	// it defines the location of the actual schema definition.
	Ref string `json:"$ref,omitempty"`

	XExpandableFields   *[]string           `json:"x-expandableFields,omitempty"`
	XExpansionResources *ExpansionResources `json:"x-expansionResources,omitempty"`
	XResourceID         string              `json:"x-resourceId,omitempty"`
	XStripeOperations   *[]StripeOperation  `json:"x-stripeOperations,omitempty"`
}

func (s *Schema) String() string {
	js, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		panic(err)
	}

	return string(js)
}

// UnmarshalJSON is a custom JSON unmarshaling implementation for Schema that
// provides better error messages instead of silently ignoring fields.
func (s *Schema) UnmarshalJSON(data []byte) error {
	var rawFields map[string]interface{}

	err := json.Unmarshal(data, &rawFields)
	if err != nil {
		return err
	}

	for _, supportedField := range supportedSchemaFields {
		delete(rawFields, supportedField)
	}

	for unsupportedField := range rawFields {
		return fmt.Errorf(
			"unsupported field in JSON schema: '%s'", unsupportedField)
	}

	// Define a second type that's identical to Schema, but distinct, so that when
	// we call json.Unmarshal it will call the default implementation of
	// unmarshalling a Schema object instead of recursively calling this
	// UnmarshalJSON function again.
	type schemaAlias Schema

	var inner schemaAlias

	err = json.Unmarshal(data, &inner)
	if err != nil {
		return err
	}

	*s = Schema(inner)

	return nil
}

// MediaType is a struct bucketing a request or response by media type in an
// OpenAPI specification.
type MediaType struct {
	Schema *Schema `json:"schema"`
}

// Operation is a struct representing a possible HTTP operation in an OpenAPI
// specification.
type Operation struct {
	Deprecated  *bool                   `json:"deprecated"`
	Description string                  `json:"description"`
	OperationID string                  `json:"operation_id"`
	Parameters  []*Parameter            `json:"parameters"`
	RequestBody *RequestBody            `json:"requestBody"`
	Responses   map[StatusCode]Response `json:"responses"`
}

// Parameter is a struct representing a request parameter to an HTTP operation
// in an OpenAPI specification.
type Parameter struct {
	Description string  `json:"description"`
	In          string  `json:"in"`
	Name        string  `json:"name"`
	Required    bool    `json:"required"`
	Schema      *Schema `json:"schema"`
}

// Path is a type for an HTTP path in an OpenAPI specification.
type Path string

// RequestBody is a struct representing the body of a request in an OpenAPI
// specification.
type RequestBody struct {
	Content  map[string]MediaType `json:"content"`
	Required bool                 `json:"required"`
}

// Response is a struct representing the response of an HTTP operation in an
// OpenAPI specification.
type Response struct {
	Description string               `json:"description"`
	Content     map[string]MediaType `json:"content"`
}

// ResourceID is a type for the ID of a response resource in an OpenAPI
// specification.
type ResourceID string

// Spec is a struct representing an OpenAPI specification.
type Spec struct {
	Components Components                       `json:"components"`
	Info       *Info                            `json:"info"`
	Paths      map[Path]map[HTTPVerb]*Operation `json:"paths"`
}

// StatusCode is a type for the response status code of an HTTP operation in an
// OpenAPI specification.
type StatusCode string

//
// Public functions
//

// LoadSpec loads and returns the OpenAPI spec.
func LoadSpec(specPath string) (*Spec, error) {
	data, err := ioutil.ReadFile(specPath)
	if err != nil {
		return nil, err
	}

	var stripeSpec Spec

	err = json.Unmarshal(data, &stripeSpec)
	if err != nil {
		return nil, fmt.Errorf("error decoding spec: %v", err)
	}

	return &stripeSpec, nil
}
