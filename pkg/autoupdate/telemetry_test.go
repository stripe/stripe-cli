package autoupdate

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/stripe/stripe-cli/pkg/installmethod"
)

func TestSendTelemetryEvent(t *testing.T) {
	var received bool
	var gotBody string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		received = true
		assert.Equal(t, "stripe-cli", r.Header.Get("origin"))
		assert.Equal(t, "application/x-www-form-urlencoded", r.Header.Get("Content-Type"))
		assert.Equal(t, http.MethodPost, r.Method)

		err := r.ParseForm()
		assert.NoError(t, err)
		gotBody = r.Form.Get("event_name")

		assert.Equal(t, "stripe-cli", r.Form.Get("client_id"))
		// Whatever pkg/installmethod reports for this host, not a hardcoded
		// name: the point of detecting it is that these events group with the
		// ones the rest of the CLI sends.
		assert.Equal(t, installmethod.Detect(installmethod.OSEnv()), r.Form.Get("install_method"))
		assert.NotEmpty(t, r.Form.Get("event_id"))
		assert.NotEmpty(t, r.Form.Get("created"))

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	original := telemetryEndpoint
	telemetryEndpoint = server.URL
	defer func() { telemetryEndpoint = original }()

	sendTelemetryEvent("Auto-Update Succeeded", "from=1.0.0 to=1.1.0")

	assert.True(t, received)
	assert.Equal(t, "Auto-Update Succeeded", gotBody)
}

func TestSendTelemetryEvent_OptedOut(t *testing.T) {
	var received bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		received = true
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	original := telemetryEndpoint
	telemetryEndpoint = server.URL
	defer func() { telemetryEndpoint = original }()

	t.Setenv("STRIPE_CLI_TELEMETRY_OPTOUT", "1")

	sendTelemetryEvent("Auto-Update Succeeded", "from=1.0.0 to=1.1.0")

	assert.False(t, received)
}

func TestSendTelemetryEvent_DoNotTrack(t *testing.T) {
	var received bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		received = true
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	original := telemetryEndpoint
	telemetryEndpoint = server.URL
	defer func() { telemetryEndpoint = original }()

	t.Setenv("DO_NOT_TRACK", "true")

	sendTelemetryEvent("Auto-Update Succeeded", "from=1.0.0 to=1.1.0")

	assert.False(t, received)
}

func TestEventValueIsDeterministic(t *testing.T) {
	fields := map[string]string{"to": "1.51.1", "from": "1.51.0", "reason": reasonChecksum}

	// Same fields must render identically every time: map iteration order is
	// randomized, and an event_value whose field order varies cannot be grouped.
	first := eventValue(fields)
	for i := 0; i < 50; i++ {
		assert.Equal(t, first, eventValue(fields))
	}

	assert.Equal(t, "from=1.51.0 reason=checksum to=1.51.1", first)
	assert.Empty(t, eventValue(nil))
}

func TestRecoverAndReportSwallowsPanic(t *testing.T) {
	original := telemetryEndpoint
	telemetryEndpoint = "http://127.0.0.1:1" // nothing listening; the send fails and is ignored
	defer func() { telemetryEndpoint = original }()

	// The panic must not escape: auto-update is not work the user asked for, so
	// it may not take down the command they did ask for.
	didPanic := func() (panicked bool) {
		defer func() {
			if r := recover(); r != nil {
				panicked = true
			}
		}()
		func() {
			defer recoverAndReport("test")
			panic("boom")
		}()
		return false
	}()

	assert.False(t, didPanic, "recoverAndReport must swallow the panic")
}
