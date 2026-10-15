package doctor

// Rules are pure data: a rule declares what to find and how to remediate;
// the engine stays language- and rule-agnostic.

// ParamMatch is one (parameter, operations) pair. Operations are hand-copied
// from api/openapi-spec/spec3.cli.json and verified against it; a follow-up
// wires this into go generate alongside resources_cmds.go.
type ParamMatch struct {
	Param      string   // canonical snake_case, dotted for nested params
	Operations []string // "POST /v1/payment_intents"
}

// A Rule is a lint check over request parameters. Action declares what the
// finding means: "remove" rules name a parameter that should be deleted by
// hand (judged against account facts — see verdict.go); "advise" rules
// detect and explain a rename or value change for the human to apply.
// IntroducedIn is the API version that made the change — packs become
// selectable by the user's version window.
type Rule struct {
	ID           string
	Severity     string
	Action       string // "remove" (judged against account facts) | "advise" (detect-only)
	IntroducedIn string // API version the change shipped in, "" = not versioned
	Message      string
	Docs         string
	Match        []ParamMatch
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

// ---------- Engine ----------
