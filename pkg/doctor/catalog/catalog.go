package catalog

// catalog.go embeds a snapshot of the dpm rule in the shape the live
// doctor_rules API (GET /v2/integ_hlth/doctor_rules/dpm) actually serves
// it in — see apirule.go for the parser and judge.FetchDPMRule for the
// live fetch. This embedded copy is the offline/fallback source: used
// directly with --offline, or when the live fetch fails for any reason.

import (
	_ "embed"
	"fmt"
)

//go:embed dpm.json
var dpmCatalogJSON []byte

// DpmRule is parsed from the embedded snapshot at startup, so it's always
// available as a fallback even without network access.
var DpmRule = mustParseAPIRule(dpmCatalogJSON)

func mustParseAPIRule(raw []byte) Rule {
	rule, err := ParseAPIRule(raw)
	if err != nil {
		panic(fmt.Sprintf("doctor: embedded dpm.json: %v", err))
	}
	return rule
}
