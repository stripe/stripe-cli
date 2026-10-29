package doctor

import (
	"github.com/stripe/stripe-cli/pkg/doctor/internal/catalog"
	"github.com/stripe/stripe-cli/pkg/doctor/internal/judge"
	"github.com/stripe/stripe-cli/pkg/doctor/internal/webhookcheck"
	"github.com/stripe/stripe-cli/pkg/doctor/remediation"
)

// --json marshals one of these structs to stdout verbatim. Exit code 0
// means a report was produced (check its contents for what it found);
// non-zero means doctor failed before producing one.

// ScanReport is scan-stage data, kept for internal use and tests.
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
// migration rule's own top-level fields. DoctorTopic mirrors
// Rule.RemediationTopic, the lookup key into pkg/doctor/remediation's
// registry; RemediationResult is what running that topic's
// Implementation produced, if one is registered.
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
