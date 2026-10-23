// Package codesignals scans a repo for caller-declared code signals:
// expected webhook events. These are honest substring checks, reported as
// signals — never verdicts. Callers (today: the paymentmethods
// remediation) declare their own PackSignals rather than doctor's engine
// wiring one in generically — see pkg/doctor/remediation/paymentmethods
// for the dpm example. This package only depends on scan, to know which
// file extensions the AST engine (and so this package) can read.
//
// Frontend tokens (legacy client-side signals like Card Element usage) and
// package-version-floor checks aren't supported yet.
package codesignals

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/stripe/stripe-cli/pkg/doctor/scan"
)

// PackSignals are a caller's declared code signals: which webhook events
// the migrated integration must handle.
type PackSignals struct {
	WebhookEvents []WebhookEvent
}

// WebhookEvent is one event the migrated integration is expected to handle,
// with the caller's explanation of what happens if the handler is missing.
type WebhookEvent struct {
	Event string
	Note  string
}

// ScanSignals walks the scanned directory for sig's declared signals.
// These are honest substring checks, reported as signals — never verdicts.
func ScanSignals(root string, sig *PackSignals) *HandlerSignals {
	if sig == nil {
		return nil
	}
	h := &HandlerSignals{}
	for _, ev := range sig.WebhookEvents {
		h.Events = append(h.Events, EventSignal{Event: ev.Event, Note: ev.Note})
	}

	_ = filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			switch info.Name() {
			case "node_modules", "vendor", ".git", "dist", "build":
				return filepath.SkipDir
			}
			return nil
		}
		if !scan.SupportsExt(strings.ToLower(filepath.Ext(path))) {
			return nil
		}
		src, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		text := string(src)
		for i := range h.Events {
			if strings.Contains(text, h.Events[i].Event) {
				h.Events[i].Files = append(h.Events[i].Files, path)
				h.Events[i].Present = true
			}
		}
		return nil
	})
	h.AllPresent = len(h.Events) > 0
	for _, e := range h.Events {
		if !e.Present {
			h.AllPresent = false
		}
	}
	return h
}
