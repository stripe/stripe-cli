package cmd

import (
	"bufio"
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/stripe/stripe-cli/pkg/config"
	"github.com/stripe/stripe-cli/pkg/requests"
)

// setupFeedbackTestConfig configures the package-level Config for a test and
// returns a cleanup function to restore the previous state.
func setupFeedbackTestConfig(t *testing.T, apiKey string) func() {
	t.Helper()

	origAPIKey := Config.Profile.APIKey
	origDeviceName := Config.Profile.DeviceName

	viper.Reset()
	Config.Profile = config.Profile{
		ProfileName: "default",
		APIKey:      apiKey,
		DeviceName:  "test-device",
	}

	return func() {
		Config.Profile.APIKey = origAPIKey
		Config.Profile.DeviceName = origDeviceName
		viper.Reset()
	}
}

func newTestFeedbackCmd(serverURL string) *feedbackCmd {
	fc := newFeedbackCmd()
	fc.apiBaseURL = serverURL
	return fc
}

func TestFeedbackNonInteractiveRequiresFlags(t *testing.T) {
	cleanup := setupFeedbackTestConfig(t, "sk_test_123456789012")
	defer cleanup()

	fc := newTestFeedbackCmd("http://example.invalid")

	// No sentiment, message, or actor supplied, and stdin is not a TTY in
	// tests, so this should be treated as non-interactive and fail fast.
	buf := new(bytes.Buffer)
	fc.cmd.SetIn(strings.NewReader(""))
	fc.cmd.SetOut(buf)

	err := fc.runFeedbackCmd(fc.cmd, []string{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "--sentiment")
	assert.Contains(t, err.Error(), "--message")
	assert.Contains(t, err.Error(), "--actor")
}

func TestFeedbackRequiresAuth(t *testing.T) {
	cleanup := setupFeedbackTestConfig(t, "")
	defer cleanup()

	fc := newTestFeedbackCmd("http://example.invalid")
	fc.sentiment = "positive"
	fc.message = "this is a test feedback message"
	fc.actor = "human"

	err := fc.runFeedbackCmd(fc.cmd, []string{})
	require.Error(t, err)
}

func TestFeedbackSubmitSendsExpectedRequest(t *testing.T) {
	var receivedForm url.Values
	var receivedVersionHeader string
	var receivedPath string

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedPath = r.URL.Path
		receivedVersionHeader = r.Header.Get("Stripe-Version")

		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		receivedForm, err = url.ParseQuery(string(body))
		require.NoError(t, err)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"id":"pfbk_test123","success":true}`))
	}))
	defer ts.Close()

	cleanup := setupFeedbackTestConfig(t, "sk_test_123456789012")
	defer cleanup()

	fc := newTestFeedbackCmd(ts.URL)
	fc.sentiment = "positive"
	fc.message = "this is a test feedback message"
	fc.actor = "human"
	fc.feature = "cli"
	fc.context = "testing the feedback command"

	buf := new(bytes.Buffer)
	fc.cmd.SetOut(buf)

	err := fc.runFeedbackCmd(fc.cmd, []string{})
	require.NoError(t, err)

	assert.Equal(t, "/v1/_unstable/feedback", receivedPath)
	assert.Equal(t, requests.StripePreviewVersionHeaderValue, receivedVersionHeader)
	assert.Equal(t, "positive", receivedForm.Get("sentiment"))
	assert.Equal(t, "this is a test feedback message", receivedForm.Get("message"))
	assert.Equal(t, "cli", receivedForm.Get("channel"))
	assert.Equal(t, "human", receivedForm.Get("actor"))
	assert.Equal(t, "cli", receivedForm.Get("feature_area"))
	assert.Equal(t, "testing the feedback command", receivedForm.Get("context"))
	assert.Equal(t, "test-device", receivedForm.Get("device_name"))
	assert.NotEmpty(t, receivedForm.Get("os"))

	assert.Contains(t, buf.String(), "pfbk_test123")
}

func TestFeedbackSubmitJSONOutput(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"id":"pfbk_test456","success":true}`))
	}))
	defer ts.Close()

	cleanup := setupFeedbackTestConfig(t, "sk_test_123456789012")
	defer cleanup()

	fc := newTestFeedbackCmd(ts.URL)
	fc.sentiment = "neutral"
	fc.message = "testing json output format here"
	fc.actor = "human"
	fc.jsonOutput = true

	buf := new(bytes.Buffer)
	fc.cmd.SetOut(buf)

	err := fc.runFeedbackCmd(fc.cmd, []string{})
	require.NoError(t, err)

	var result feedbackResponse
	require.NoError(t, json.Unmarshal(bytes.TrimSpace(buf.Bytes()), &result))
	assert.True(t, result.Success)
	assert.Equal(t, "pfbk_test456", result.ID)
}

func TestFeedbackActorAgentAccepted(t *testing.T) {
	var receivedForm url.Values

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		receivedForm, err = url.ParseQuery(string(body))
		require.NoError(t, err)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"id":"pfbk_test789","success":true}`))
	}))
	defer ts.Close()

	cleanup := setupFeedbackTestConfig(t, "sk_test_123456789012")
	defer cleanup()

	fc := newTestFeedbackCmd(ts.URL)
	fc.sentiment = "positive"
	fc.message = "agent submitted feedback test message"
	fc.actor = "agent"

	err := fc.runFeedbackCmd(fc.cmd, []string{})
	require.NoError(t, err)
	assert.Equal(t, "agent", receivedForm.Get("actor"))
}

func TestFeedbackServerError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":"invalid sentiment"}`))
	}))
	defer ts.Close()

	cleanup := setupFeedbackTestConfig(t, "sk_test_123456789012")
	defer cleanup()

	fc := newTestFeedbackCmd(ts.URL)
	fc.sentiment = "positive"
	fc.message = "this is a test feedback message"
	fc.actor = "human"

	err := fc.runFeedbackCmd(fc.cmd, []string{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "400")
}

func TestPromptLineTrimsInput(t *testing.T) {
	buf := new(bytes.Buffer)
	reader := bufio.NewReader(strings.NewReader("  hello world  \n"))

	line, err := promptLine(buf, reader, "prompt")
	require.NoError(t, err)
	assert.Equal(t, "hello world", line)
}
