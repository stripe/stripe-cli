package dpm

import (
	"testing"

	"github.com/stripe/stripe-cli/pkg/doctor/codesignals"
)

// TestWebhookSignals pins this kind's own code-signal layer: the fixtures
// keep their original three-events-present behavior.
func TestWebhookSignals(t *testing.T) {
	h := codesignals.ScanSignals("testdata", webhookSignals)
	if !h.AllPresent {
		t.Errorf("expected all three delayed-notification events present: %+v", h.Events)
	}
}
