// Package doctor implements the migration doctor: rule-based detection of
// Stripe API migrations across seven languages, judged against read-only
// account facts. Each finding's verdict names the exact code change to make
// by hand; doctor does not edit code itself.
//
// The CLI surface is two commands — doctor (diagnose) and guide (the agent
// playbook). See README.md for the architecture and reports.go for the JSON
// contract.
package doctor
