package resource

// OperationSpec is the self-describing definition of one API operation.
// Produced by codegen; consumed both by NewOperationCmd and future rich-output features.
type OperationSpec struct {
	Name      string // cobra command name, e.g. "create"
	Path      string // e.g. "/v1/customers"
	Method    string // e.g. "POST"
	IsPreview bool
	ServerURL string // non-empty for operations that use a different server
	Summary   string // one-line description from the spec
	Params    map[string]*ParamSpec
}

// ParamSpec describes a single parameter of an API operation.
type ParamSpec struct {
	Type        string // "string" | "integer" | "boolean" | "number" | "array"
	Description string
	Required    bool
	Format      string // e.g. "date-time", "unix-time"
	Enum        []EnumSpec
}

// EnumSpec describes a single valid value for an enum parameter.
type EnumSpec struct {
	Value       string
	Description string // from x-stripeEnum; empty if only plain enum was available
}
