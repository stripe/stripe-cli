// Package doctor produces read-only, scoped remediation instructions from mocked predicates.
package doctor

import (
	"encoding/json"
	"time"
)

type Citation struct {
	File   string `json:"file"`
	Symbol string `json:"symbol"`
	Start  int    `json:"start"`
	End    int    `json:"end"`
	SHA256 string `json:"sha256"`
}

type Methods struct {
	Status       string     `json:"status"` // known, conditional, absent, unknown
	Values       []string   `json:"values,omitempty"`
	Alternatives [][]string `json:"alternatives,omitempty"`
}

type FlowContext struct {
	Account            string `json:"account"` // unknown, profile, connect
	Mode               string `json:"mode"`    // unknown, test, live
	Configuration      string `json:"configuration,omitempty"`
	DefaultEstablished bool   `json:"defaultEstablished"`
	APIVersion         string `json:"apiVersion,omitempty"`
}

type SourceFlow struct {
	ID         string      `json:"id"`
	Boundary   Citation    `json:"boundary"`
	Method     string      `json:"method"`
	Path       string      `json:"path"`
	Callers    []Citation  `json:"callers"`
	Builders   []Citation  `json:"builders"`
	Pinned     Methods     `json:"pinnedMethods"`
	Automatic  string      `json:"automaticPaymentMethods"` // enabled, disabled, absent, unknown
	Context    FlowContext `json:"context"`
	Unresolved []string    `json:"unresolved"`
	Snapshot   string      `json:"snapshot"`
}

type Dependency struct {
	Manifest   string `json:"manifest"`
	Name       string `json:"name"`
	Version    string `json:"version"`
	Resolution string `json:"resolution"`
}

type Coverage struct {
	Root         string       `json:"root"`
	Files        int          `json:"files"`
	Candidates   int          `json:"candidates"`
	Manifests    []string     `json:"manifests"`
	Dependencies []Dependency `json:"dependencies"`
	Languages    []string     `json:"languages"`
	Excluded     []string     `json:"excluded"`
	Limitations  []string     `json:"limitations"`
	Truncated    bool         `json:"truncated"`
}

type Investigation struct {
	FlowID    string   `json:"flowId"`
	Citation  Citation `json:"citation"`
	Questions []string `json:"questions"`
}

type Discovery struct {
	Snapshot string          `json:"snapshot"`
	Coverage Coverage        `json:"coverage"`
	Flows    []SourceFlow    `json:"flows"`
	Queue    []Investigation `json:"investigationQueue"`
}

// Handoff is evidence supplied by a user-authorized coding agent, not executable policy.
// Citation validation establishes provenance only; supplied claims never authorize edits.
type Handoff struct {
	Snapshot string       `json:"snapshot"`
	Flows    []SourceFlow `json:"flows"`
}

type Diagnostic struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type Task struct {
	Instruction  string   `json:"instruction"`
	Constraints  []string `json:"constraints"`
	Acceptance   []string `json:"acceptance"`
	Verification []string `json:"verification"`
}

type Provenance struct {
	Finding         string          `json:"finding"`
	Declaration     string          `json:"declaration"`
	Kind            string          `json:"kind"`
	Version         int             `json:"version"`
	CLIRevision     string          `json:"cliRevision"`
	HandlerRevision string          `json:"handlerRevision"`
	Options         json.RawMessage `json:"options,omitempty"`
}

type RemediationResult struct {
	Provenance     Provenance     `json:"provenance"`
	Status         string         `json:"status"`
	Flow           SourceFlow     `json:"flow"`
	Coverage       Coverage       `json:"coverage"`
	Configuration  *Configuration `json:"configuration,omitempty"`
	Legacy         *LegacyFix     `json:"legacyFix,omitempty"`
	Lost           []string       `json:"lost"`
	Added          []string       `json:"added"`
	Diagnostics    []Diagnostic   `json:"diagnostics"`
	Tasks          []Task         `json:"tasks"`
	PendingActions []string       `json:"pendingMerchantActions"`
}

type Report struct {
	Input             string              `json:"input"`
	PredicateRevision string              `json:"predicateRevision"`
	GeneratedAt       time.Time           `json:"generatedAt"`
	Status            string              `json:"status"`
	Discovery         Discovery           `json:"discovery"`
	Results           []RemediationResult `json:"results"`
	Diagnostics       []Diagnostic        `json:"diagnostics"`
}
