// Package catalog is pure data: migration rules (what to find and how to
// judge it) plus the packs that register them under a topic name. The
// scanning engine and the judgment engine both consume this package but
// never the other way around.
package catalog

// ParamMatch is one (parameter, operations) pair. Operations are hand-copied
// from api/openapi-spec/spec3.cli.json and verified against it; a follow-up
// wires this into go generate alongside resources_cmds.go.
type ParamMatch struct {
	Param      string   // canonical snake_case, dotted for nested params
	Operations []string // "POST /v1/payment_intents"
}

// A Rule is a lint check over request parameters. "remove" rules name a
// parameter to delete by hand, judged by finding intent; "advise" rules
// detect a value change to apply (not currently produced by
// ParseAPIRule). What a remediation actually does lives in that topic's
// own Go code, not here.
type Rule struct {
	ID           string
	Severity     string
	Action       string // "remove" | "advise"
	IntroducedIn string // API version the change shipped in, "" = not versioned
	Message      string // API's own "message" field
	ActionText   string // API's own "action" field
	Docs         string
	Match        []ParamMatch
	// RemediationTopic names which pkg/doctor/remediation.Implementation
	// to run — currently just the rule's own topic (the API has no
	// separate field for this), or "" for none.
	RemediationTopic string
}

type Finding struct {
	File     string `json:"file"`
	Line     int    `json:"line"`
	Col      int    `json:"col"`
	RuleID   string `json:"rule"`
	Severity string `json:"severity"`
	Param    string `json:"param"`
	Anchor   string `json:"anchor"` // the matched param-bag text, trimmed
	Via      string `json:"via"`    // how it resolved: direct, var:<name>, type:<name>
	Value    string `json:"value"`  // the parameter's value expression, for intent ranking
	Message  string `json:"message"`
	Docs     string `json:"docs"`
}
