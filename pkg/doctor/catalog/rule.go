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

// A Rule is a lint check over request parameters. Action declares what the
// finding means: "remove" rules name a parameter that should be deleted by
// hand, judged by the finding's intent alone (see the judge package).
// "advise" rules detect and explain a rename or value change for the human
// to apply (not currently produced by ParseAPIRule — every topic today is
// "remove"). IntroducedIn is the API version that made the change — packs
// become selectable by the user's version window. There is no field here
// for what a remediation should actually do (e.g. a Dashboard deep link
// to surface, or which param to touch): that's defined directly in the
// RemediationTopic's own Go implementation instead — see
// pkg/doctor/remediation/dpm.
type Rule struct {
	ID           string
	Severity     string
	Action       string // "remove" (judged by finding intent) | "advise" (detect-only)
	IntroducedIn string // API version the change shipped in, "" = not versioned
	// Message and ActionText are the API's own "message"/"action" fields,
	// kept separate (not concatenated) so doctor can render them as two
	// distinct lines up front. ActionText is the API's human-readable
	// remediation instruction (e.g. "Remove payment_method_types and let
	// the Dashboard manage payment methods") — unrelated to Action, above,
	// which is the engine's own remove/advise classification.
	Message    string
	ActionText string
	Docs       string
	Match      []ParamMatch
	// RemediationTopic is the API's "doctorTopic" field — a bare string
	// (e.g. "dpm") naming which pkg/doctor/remediation.Implementation to
	// run against this rule. "" means none is declared. doctor's engine
	// doesn't act on it itself — it just looks this string up in the
	// remediation registry (see pkg/doctor/remediation).
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
