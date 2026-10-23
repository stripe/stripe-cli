package doctor

import (
	"github.com/stripe/stripe-cli/pkg/doctor/catalog"
	"github.com/stripe/stripe-cli/pkg/doctor/judge"
	"github.com/stripe/stripe-cli/pkg/doctor/remediation"
	"github.com/stripe/stripe-cli/pkg/doctor/webhookcheck"
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

// DoctorReport is the output of `stripe doctor`: scan + verdicts,
// optionally with live behavioral checks. Message/Action/Docs are the
// migration rule's own top-level fields, straight from the doctor_rules
// API response — rendered up front, ahead of the per-finding verdicts.
// DoctorTopic is the rule's own "doctorTopic" field — the lookup key into
// pkg/doctor/remediation's registry. RemediationResult is what doctor
// actually computed by running that topic's Implementation, if one is
// registered — see that package for how a topic becomes a Result.
// There's no generic top-level code-signals field: a remediation that
// cares about code signals (e.g. webhook handlers) declares and scans for
// its own — see pkg/doctor/remediation/paymentmethods — and reports it
// via its own Result instead.
type DoctorReport struct {
	Command           string               `json:"command"`
	Topic             string               `json:"topic"`
	Message           string               `json:"message,omitempty"`
	Action            string               `json:"action,omitempty"`
	Docs              string               `json:"docs,omitempty"`
	DoctorTopic       string               `json:"doctorTopic,omitempty"`
	RemediationResult *remediation.Result  `json:"remediation_result,omitempty"`
	Findings          []judge.Finding      `json:"findings"`
	Summary           map[string]int       `json:"summary"` // verdict class -> count
	Stats             ScanStats            `json:"stats"`
	WebhookCheck      *webhookcheck.Report `json:"webhook_check,omitempty"`
}
