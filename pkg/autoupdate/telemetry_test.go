package autoupdate

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
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
		assert.Equal(t, "curl", r.Form.Get("install_method"))
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
