// Package doctor implements the migration doctor: rule-based detection and
// remediation of Stripe API migrations across seven languages, judged
// against read-only account facts.
//
// The CLI surface is three commands — doctor (diagnose), fix (remediate),
// and guide (the agent playbook). See README.md for the architecture and
// reports.go for the JSON contract.
package doctor
