package resource

// OperationSpec is the self-describing definition of one API operation.
// Produced by codegen; consumed both by NewOperationCmd and future rich-output features.
type OperationSpec struct {
	Name      string // cobra command name, e.g. "create"
	Path      string // e.g. "/v1/customers"
	Method    string // e.g. "POST"
	IsPreview bool
	ServerURL string // non-empty for operations that use a different server
	Params    map[string]*ParamSpec
}

// ParamSpec describes a single parameter of an API operation.
//
// Required and MostCommon together drive the --help examples produced by buildExamples.
// The goal is two concise, runnable invocations: a minimal call using only what the API
// requires, and a richer call that also shows the most commonly useful optional parameters.
type ParamSpec struct {
	Type string // "string" | "integer" | "boolean" | "number" | "array"

	// Required is true when the parameter is unconditionally required by the API —
	// the parameter itself and every ancestor object in its dot-path must all be required.
	// A field that is required within an optional parent is not Required here, because the
	// parent need not be provided. This matches what goes in the minimal example line.
	Required bool

	// MostCommon is true when the parameter is worth surfacing in the richer example line,
	// even if not strictly required. The generator sets this via two heuristics:
	//   - Depth-0 params (no dots) explicitly listed in x-stripeMostCommon on the request body.
	//   - Depth-1 params (one dot) that are locally required within a depth-0 MostCommon
	//     object — i.e., if you are going to provide the parent, these fields are needed.
	// Depth 2+ params are never marked; examples stay concise.
	MostCommon bool

	Format string // e.g. "date-time", "unix-time"
	Enum   []EnumSpec
}

// EnumSpec describes a single valid value for an enum parameter.
type EnumSpec struct {
	Value string
}
