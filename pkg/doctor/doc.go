// Package doctor implements the migration doctor: rule-based detection of
// Stripe API migrations, judged against the migration rule. Each
// finding's verdict names the exact code change to make by hand; doctor
// does not edit code itself. See README.md for the architecture.
package doctor
