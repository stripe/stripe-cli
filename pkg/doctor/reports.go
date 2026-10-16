package doctor

import (
	"github.com/stripe/stripe-cli/pkg/doctor/catalog"
	"github.com/stripe/stripe-cli/pkg/doctor/drill"
	"github.com/stripe/stripe-cli/pkg/doctor/judge"
	"github.com/stripe/stripe-cli/pkg/doctor/signals"
)

// reports.go — the machine contract. Every subcommand produces one of these
// structs; --json marshals it to stdout verbatim (logs go to stderr), and the
// text renderers in cmd.go present the same data to humans. Exit codes:
//
//	0  clean / verified
//	1  findings present / verification failed
//	2  operational error
//
// Agents should treat these schemas plus the exit codes as the interface;
// `stripe doctor guide` documents the step-by-step playbook.

// ScanReport is scan-stage data (kept for internal use and tests).
type ScanReport struct {
	Command  string            `json:"command"`
	Findings []catalog.Finding `json:"findings"`
	Stats    ScanStats         `json:"stats"`
}

type ScanStats struct {
	FilesScanned int `json:"files_scanned"`
	FilesParsed  int `json:"files_parsed"`
	Skipped      int `json:"skipped_by_prefilter"`
}

// DoctorReport is the output of `stripe doctor`: scan + account facts +
// verdicts, optionally with live behavioral checks. Degraded is set (and
// Account empty) when account facts were unavailable — findings then carry
// verdict_class UNKNOWN.
type DoctorReport struct {
	Command   string               `json:"command"`
	Topic     string               `json:"topic"`
	Degraded  string               `json:"degraded,omitempty"`
	Account   judge.AccountSummary `json:"account,omitzero"`
	Findings  []judge.Finding      `json:"findings"`
	Summary   map[string]int       `json:"summary"` // verdict class -> count
	Stats     ScanStats            `json:"stats"`
	LiveDrill *drill.Report        `json:"live_drill,omitempty"`
	// Code signals (substring-level, honest about their strength):
	WebhookHandlers *signals.HandlerSignals `json:"webhook_handlers,omitempty"`
}
