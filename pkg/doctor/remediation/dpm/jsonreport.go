package dpm

// jsonreport.go builds this topic's own --json schema, set as
// remediation.Result.Report.

import (
	"github.com/stripe/stripe-cli/pkg/doctor/remediation"
)

const schemaVersion = "1.0"
const checkID = "dynamic_payment_methods"

// Status strings for this schema, distinct from remediation.Status.
const (
	jsonReviewRequired = "review_required"
	jsonBlocked        = "blocked"
	jsonReady          = "ready"
	jsonUnknown        = "unknown"
)

type jsonReport struct {
	SchemaVersion string                    `json:"schema_version"`
	Check         string                    `json:"check"`
	Status        string                    `json:"status"`
	Summary       string                    `json:"summary"`
	Findings      []remediation.FindingJSON `json:"findings"`
	Dashboard     jsonDashboard             `json:"dashboard"`
	Documentation []jsonDoc                 `json:"documentation"`
}

type jsonDashboard struct {
	EnabledPaymentMethods any    `json:"enabled_payment_methods"`
	RecommendedAction     string `json:"recommended_action"`
	URL                   string `json:"url"`
}

type jsonDoc struct {
	Title string `json:"title"`
	URL   string `json:"url"`
}

func buildJSONReport(ctx remediation.Context, pmf pmConfigFacts, result remediation.Result) *jsonReport {
	var findings []remediation.FindingJSON
	for _, f := range result.Findings {
		if f.ID == "" {
			continue
		}
		findings = append(findings, remediation.FindingsJSON([]remediation.Finding{f}, result.Status)[0])
	}

	status := jsonReady
	if !pmf.ConfiguredOK {
		status = jsonBlocked
	} else if len(findings) > 0 {
		status = jsonReviewRequired
	}

	var enabledMethods any
	if pmf.ConfiguredOK {
		enabledMethods = pmf.EnabledMethods
	}

	return &jsonReport{
		SchemaVersion: schemaVersion,
		Check:         checkID,
		Status:        status,
		Summary:       jsonSummary(ctx, status),
		Findings:      findings,
		Dashboard: jsonDashboard{
			EnabledPaymentMethods: enabledMethods,
			RecommendedAction:     "Review which payment methods to enable in the Dashboard.",
			URL:                   dashboardConfigURL,
		},
		Documentation: []jsonDoc{{Title: title, URL: ctx.Rule.Docs}},
	}
}

func jsonSummary(ctx remediation.Context, status string) string {
	if len(ctx.Findings) == 0 {
		return "No explicit payment method lists were found in the scanned code."
	}
	if status == jsonBlocked {
		return "Explicit payment method lists were found, and no payment methods are currently enabled in the Dashboard."
	}
	return "Explicit payment method lists prevent Dashboard-enabled methods from appearing automatically in these checkout flows."
}
