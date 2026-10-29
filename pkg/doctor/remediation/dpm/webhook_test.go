package dpm

import (
	"testing"

	"github.com/stripe/stripe-cli/pkg/doctor/internal/codesignals"
)

func TestWebhookSignals(t *testing.T) {
	h := codesignals.ScanSignals("testdata", webhookSignals)
	if !h.AllPresent {
		t.Errorf("expected all three delayed-notification events present: %+v", h.Events)
	}
}
