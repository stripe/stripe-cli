package cmd

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/stripe/stripe-cli/pkg/config"
	"github.com/stripe/stripe-cli/pkg/requests"
)

func setupFeedbackTestConfig(t *testing.T, apiKey string) func() {
	t.Helper()

	originalAPIKey := Config.Profile.APIKey
	originalDeviceName := Config.Profile.DeviceName

	viper.Reset()
	Config.Profile = config.Profile{
		ProfileName: "default",
		APIKey:      apiKey,
		DeviceName:  "test-device",
	}

	return func() {
		Config.Profile.APIKey = originalAPIKey
		Config.Profile.DeviceName = originalDeviceName
		viper.Reset()
	}
}

func newTestFeedbackCmd(serverURL string) *feedbackCmd {
	feedbackCommand := newFeedbackCmd()
	feedbackCommand.apiBaseURL = serverURL
	return feedbackCommand
}

func runFeedback(feedbackCommand *feedbackCmd) error {
	// Downstream code expects a non-nil context.
	feedbackCommand.cmd.SetContext(context.Background())
	return feedbackCommand.runFeedbackCmd(feedbackCommand.cmd, []string{})
}

type feedbackCase struct {
	apiKey     string
	sentiment  string
	message    string
	context    string
	feature    string
	actor      string
	jsonOutput bool
}

func validFeedbackCase() feedbackCase {
	return feedbackCase{
		apiKey:    "sk_test_123456789012",
		sentiment: "positive",
		message:   "this is a test feedback message",
		context:   "this is a test context message",
		actor:     "human",
	}
}

func (feedbackInput feedbackCase) applyTo(feedbackCommand *feedbackCmd) {
	feedbackCommand.sentiment = feedbackInput.sentiment
	feedbackCommand.message = feedbackInput.message
	feedbackCommand.context = feedbackInput.context
	feedbackCommand.feature = feedbackInput.feature
	feedbackCommand.actor = feedbackInput.actor
	feedbackCommand.jsonOutput = feedbackInput.jsonOutput
}

func TestFeedbackNonInteractiveValidation(t *testing.T) {
	tests := []struct {
		name              string
		setup             func() feedbackCase
		wantErrSubstrings []string
	}{
		{
			name: "missing required flags",
			setup: func() feedbackCase {
				feedbackInput := validFeedbackCase()
				feedbackInput.sentiment, feedbackInput.message, feedbackInput.context, feedbackInput.actor = "", "", "", ""
				return feedbackInput
			},
			wantErrSubstrings: []string{"--sentiment", "--message", "--context", "--actor"},
		},
		{
			name: "missing api key",
			setup: func() feedbackCase {
				feedbackInput := validFeedbackCase()
				feedbackInput.apiKey = ""
				return feedbackInput
			},
		},
		{
			name: "message too short",
			setup: func() feedbackCase {
				feedbackInput := validFeedbackCase()
				feedbackInput.message = "too short"
				return feedbackInput
			},
			wantErrSubstrings: []string{"at least 10 characters"},
		},
		{
			name: "message too long",
			setup: func() feedbackCase {
				feedbackInput := validFeedbackCase()
				feedbackInput.message = strings.Repeat("a", feedbackMaxLen+1)
				return feedbackInput
			},
			wantErrSubstrings: []string{"at most 2000 characters"},
		},
		{
			name: "context too short",
			setup: func() feedbackCase {
				feedbackInput := validFeedbackCase()
				feedbackInput.context = "short"
				return feedbackInput
			},
			wantErrSubstrings: []string{"context must be at least 10 characters"},
		},
		{
			name: "invalid sentiment",
			setup: func() feedbackCase {
				feedbackInput := validFeedbackCase()
				feedbackInput.sentiment = "ecstatic"
				return feedbackInput
			},
			wantErrSubstrings: []string{"--sentiment", "not one of the allowed values"},
		},
		{
			name: "invalid feature",
			setup: func() feedbackCase {
				feedbackInput := validFeedbackCase()
				feedbackInput.feature = "not-a-real-area"
				return feedbackInput
			},
			wantErrSubstrings: []string{"--feature", "not one of the allowed values"},
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			feedbackInput := testCase.setup()

			cleanup := setupFeedbackTestConfig(t, feedbackInput.apiKey)
			defer cleanup()

			// ".invalid" never resolves (RFC 2606), so a bug that skipped
			// validation and actually dialed out would fail on a DNS error
			// here rather than silently passing.
			feedbackCommand := newTestFeedbackCmd("http://example.invalid")
			feedbackInput.applyTo(feedbackCommand)

			err := runFeedback(feedbackCommand)
			require.Error(t, err)
			for _, wantSubstring := range testCase.wantErrSubstrings {
				assert.Contains(t, err.Error(), wantSubstring)
			}
		})
	}
}

func TestFeedbackSubmit(t *testing.T) {
	tests := []struct {
		name              string
		setup             func() feedbackCase
		handler           http.HandlerFunc
		wantErrSubstrings []string
		wantCheck         func(t *testing.T, output string)
	}{
		{
			name: "sends expected request",
			setup: func() feedbackCase {
				feedbackInput := validFeedbackCase()
				feedbackInput.feature = "cli"
				feedbackInput.context = "testing the feedback command"
				return feedbackInput
			},
			handler: func(responseWriter http.ResponseWriter, request *http.Request) {
				assert.NoError(t, request.ParseForm())
				assert.Equal(t, "positive", request.FormValue("sentiment"))
				assert.Equal(t, "this is a test feedback message", request.FormValue("message"))
				assert.Equal(t, "cli", request.FormValue("channel"))
				assert.Equal(t, "human", request.FormValue("actor"))
				assert.Equal(t, "cli", request.FormValue("feature_area"))
				assert.Equal(t, "testing the feedback command", request.FormValue("context"))
				assert.Equal(t, "test-device", request.FormValue("device_name"))
				assert.NotEmpty(t, request.FormValue("os"))

				responseWriter.Header().Set("Content-Type", "application/json")
				responseWriter.WriteHeader(http.StatusOK)
				fmt.Fprint(responseWriter, `{"id":"pfbk_test123","success":true}`)
			},
			wantCheck: func(t *testing.T, output string) {
				assert.Contains(t, output, "pfbk_test123")
			},
		},
		{
			name: "json output",
			setup: func() feedbackCase {
				feedbackInput := validFeedbackCase()
				feedbackInput.sentiment = "neutral"
				feedbackInput.message = "testing json output format here"
				feedbackInput.jsonOutput = true
				return feedbackInput
			},
			handler: func(responseWriter http.ResponseWriter, request *http.Request) {
				responseWriter.Header().Set("Content-Type", "application/json")
				responseWriter.WriteHeader(http.StatusOK)
				fmt.Fprint(responseWriter, `{"id":"pfbk_test456","success":true}`)
			},
			wantCheck: func(t *testing.T, output string) {
				assert.Contains(t, output, `"id":"pfbk_test456"`)
				assert.Contains(t, output, `"success":true`)
			},
		},
		{
			name: "agent actor accepted",
			setup: func() feedbackCase {
				feedbackInput := validFeedbackCase()
				feedbackInput.message = "agent submitted feedback test message"
				feedbackInput.actor = "agent"
				return feedbackInput
			},
			handler: func(responseWriter http.ResponseWriter, request *http.Request) {
				assert.NoError(t, request.ParseForm())
				assert.Equal(t, "agent", request.FormValue("actor"))

				responseWriter.Header().Set("Content-Type", "application/json")
				responseWriter.WriteHeader(http.StatusOK)
				fmt.Fprint(responseWriter, `{"id":"pfbk_test789","success":true}`)
			},
		},
		{
			name: "server error surfaces status and body",
			setup: func() feedbackCase {
				return validFeedbackCase()
			},
			handler: func(responseWriter http.ResponseWriter, request *http.Request) {
				responseWriter.WriteHeader(http.StatusBadRequest)
				fmt.Fprint(responseWriter, `{"error":"invalid sentiment"}`)
			},
			wantErrSubstrings: []string{"400"},
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			// A real localhost listener, not a mock swapped in for
			// stripe.Client, so this catches request/response bugs a mock
			// would miss. Handlers use assert, not require: they run on
			// httptest's own goroutine, where require's FailNow is unsafe.
			server := httptest.NewServer(http.HandlerFunc(func(responseWriter http.ResponseWriter, request *http.Request) {
				assert.Equal(t, "/v1/_unstable/feedback", request.URL.Path)
				assert.Equal(t, requests.StripePreviewVersionHeaderValue, request.Header.Get("Stripe-Version"))
				testCase.handler(responseWriter, request)
			}))
			defer server.Close()

			cleanup := setupFeedbackTestConfig(t, "sk_test_123456789012")
			defer cleanup()

			feedbackCommand := newTestFeedbackCmd(server.URL)
			testCase.setup().applyTo(feedbackCommand)

			outputBuffer := new(bytes.Buffer)
			feedbackCommand.cmd.SetOut(outputBuffer)

			err := runFeedback(feedbackCommand)

			if len(testCase.wantErrSubstrings) > 0 {
				require.Error(t, err)
				for _, wantSubstring := range testCase.wantErrSubstrings {
					assert.Contains(t, err.Error(), wantSubstring)
				}
				return
			}

			require.NoError(t, err)
			if testCase.wantCheck != nil {
				testCase.wantCheck(t, outputBuffer.String())
			}
		})
	}
}

func TestSanitizeText(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"trims surrounding whitespace", "  hello world  ", "hello world"},
		{"preserves internal newlines", "line one\nline two", "line one\nline two"},
		{"strips control characters", "hello\x00\x07world", "helloworld"},
		{"trims trailing newline after trimspace", "hello\n\n", "hello"},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			assert.Equal(t, testCase.want, sanitizeText(testCase.input))
		})
	}
}

func TestValidateFeedbackMessage(t *testing.T) {
	require.Error(t, validateFeedbackMessage(""))
	require.Error(t, validateFeedbackMessage("too short"))
	require.Error(t, validateFeedbackMessage(strings.Repeat("a", feedbackMaxLen+1)))
	require.NoError(t, validateFeedbackMessage("this message is long enough"))
}

func TestValidateFeedbackContext(t *testing.T) {
	require.Error(t, validateFeedbackContext(""))
	require.Error(t, validateFeedbackContext("   "))
	require.Error(t, validateFeedbackContext("short"))
	require.Error(t, validateFeedbackContext(strings.Repeat("a", feedbackMaxLen+1)))
	require.NoError(t, validateFeedbackContext("this context is long enough"))
}

func TestFeedbackMissingRequiredFields(t *testing.T) {
	feedbackCommand := &feedbackCmd{}
	validFeedbackCase().applyTo(feedbackCommand)
	assert.Empty(t, feedbackCommand.missingRequiredFields())

	feedbackCommand.context = ""
	assert.Equal(t, []string{"--context"}, feedbackCommand.missingRequiredFields())

	feedbackCommand.sentiment = ""
	assert.Equal(t, []string{"--sentiment", "--context"}, feedbackCommand.missingRequiredFields())
}
