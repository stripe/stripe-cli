// Package signals scans a repo for pack-declared code signals: expected
// webhook events. These are honest substring checks, reported as signals —
// never verdicts. It depends on catalog for the declared-signal shapes and
// on scan only to know which file extensions the AST engine (and so this
// package) can read.
//
// Frontend tokens (legacy client-side signals like Card Element usage),
// package-version-floor checks, and the which-migration triage aren't
// supported pending the removed non-dpm topics returning — see git history.
package signals

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/stripe/stripe-cli/pkg/doctor/catalog"
	"github.com/stripe/stripe-cli/pkg/doctor/scan"
)

// ScanSignals walks the scanned directory for the pack's declared signals.
// These are honest substring checks, reported as signals — never verdicts.
func ScanSignals(root string, sig *catalog.PackSignals) *HandlerSignals {
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
