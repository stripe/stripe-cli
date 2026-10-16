package catalog

// Pack-declared code signals: expected webhook events. These are pure data,
// authored alongside a Rule in packs.go or catalog.go; the code that scans
// a repo for them lives in the signals package.
//
// Frontend tokens (legacy client-side signals like Card Element usage),
// package-version-floor checks, and the which-migration triage aren't
// supported pending the removed non-dpm topics returning — see git history.

// PackSignals are a pack's declared code signals: which webhook events the
// migrated integration must handle.
type PackSignals struct {
	WebhookEvents []WebhookEvent
}

// WebhookEvent is one event the migrated integration is expected to handle,
// with the pack's explanation of what happens if the handler is missing.
type WebhookEvent struct {
	Event string
	Note  string
}
