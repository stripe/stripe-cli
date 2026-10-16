package doctor

import (
	"testing"

	"github.com/stripe/stripe-cli/pkg/doctor/catalog"
	"github.com/stripe/stripe-cli/pkg/doctor/signals"
)

// TestDpmSignals pins dpm's code-signal layer: the fixtures keep their
// original three-events-present behavior.
func TestDpmSignals(t *testing.T) {
	h := signals.ScanSignals("testdata", &catalog.DpmSignals)
	if !h.AllPresent {
		t.Errorf("dpm: expected all three delayed-notification events present: %+v", h.Events)
	}
}
