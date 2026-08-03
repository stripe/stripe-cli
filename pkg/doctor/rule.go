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
// remediation is: "remove" rules are auto-fixable (span deletion); "advise"
// rules detect and explain but never edit (renames, value changes, and
// anything whose fix needs a primitive the engine doesn't have yet).
// IntroducedIn is the API version that made the change — packs become
// selectable by the user's version window.
type Rule struct {
	ID           string
	Severity     string
	Action       string // "remove" (fixable) | "advise" (detect-only)
	IntroducedIn string // API version the change shipped in, "" = not versioned
	Message      string
	Docs         string
	Match        []ParamMatch
	// Companion, when set, forks `fix` by account API version: at/after the
	// rule's IntroducedIn cutoff the parameter is simply removed (the new
	// behavior is the default there); below it — or when the version is
	// unknowable — the removal becomes a REPLACEMENT that inserts this
	// parameter, because bare removal would silently change behavior.
	Companion *Companion
}

// A Companion describes the parameter `fix` inserts in place of a removed one
// on accounts whose traffic predates the rule's cutoff.
type Companion struct {
	Param     string   // canonical name, e.g. "automatic_payment_methods"
	ForParam  string   // only replacements of THIS matched param (top-level)
	Resources []string // API resources that accept it, e.g. payment_intents
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
