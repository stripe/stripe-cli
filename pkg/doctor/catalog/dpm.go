package catalog

// dpm.go embeds a local snapshot of the dpm migration rule, in the exact
// shape ParseAPIRule parses from the live doctor_rules API's "doctor_rule"
// object. doctor currently always uses this embedded copy rather than
// judge.FetchRule's live fetch, so edits to dpm.json can be tested
// immediately without a round trip to the (not yet public) API. Swap
// cmd.go back to the live fetch when that's no longer needed.

import _ "embed"

//go:embed dpm.json
var dpmRuleJSON []byte

// DpmRule is the dpm migration rule, parsed from the embedded dpm.json.
var DpmRule = mustParseRule(dpmRuleJSON)

func mustParseRule(raw []byte) Rule {
	rule, err := ParseAPIRule(raw)
	if err != nil {
		panic(err)
	}
	return rule
}
