package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"runtime"
	"strings"
	"unicode"

	"github.com/manifoldco/promptui"
	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/stripe/stripe-cli/pkg/ansi"
	"github.com/stripe/stripe-cli/pkg/requests"
	"github.com/stripe/stripe-cli/pkg/stripe"
	"github.com/stripe/stripe-cli/pkg/useragent"
	"github.com/stripe/stripe-cli/pkg/validators"
	"github.com/stripe/stripe-cli/pkg/version"
)

const feedbackEndpoint = "/v1/_unstable/feedback"

// feedbackMinLen and feedbackMaxLen bound the length of the free-text
// message and context fields, after sanitization.
const (
	feedbackMinLen = 10
	feedbackMaxLen = 2000
)

var feedbackFeatureAreas = []string{
	"payments",
	"connect",
	"subscriptions",
	"payouts",
	"reporting",
	"invoicing",
	"checkout",
	"tax",
	"radar",
	"issuing",
	"terminal",
	"treasury",
	"capital",
	"docs",
	"webhooks",
	"cli",
	"dashboard",
	"crypto",
	"stablecoins",
	"managed-payments",
	"identity",
	"other",
}

var feedbackSentiments = []string{"positive", "negative", "neutral"}

// feedbackResponse is the JSON body returned by the feedback API.
type feedbackResponse struct {
	ID      string `json:"id"`
	Success bool   `json:"success"`
}

func sanitizeText(s string) string {
	trimmed := strings.TrimSpace(s)

	return strings.Map(func(character rune) rune {
		if character == '\n' || unicode.IsPrint(character) {
			return character
		}
		return -1
	}, trimmed)
}

// validateFeedbackText enforces feedbackMinLen/feedbackMaxLen on a required
// free-text field after sanitizing it. field names the field in the error
// message (eg. "feedback must be at least 10 characters"). It's shared by
// validateFeedbackMessage/validateFeedbackContext below, which are in turn
// used both for non-interactive flag validation and as promptui.Validate
// callbacks in promptForFeedback.
func validateFeedbackText(field, s string) error {
	if err := validators.Length(feedbackMinLen, feedbackMaxLen)(sanitizeText(s)); err != nil {
		return fmt.Errorf("%s %w", field, err)
	}

	return nil
}

func validateFeedbackMessage(s string) error {
	return validateFeedbackText("feedback", s)
}

func validateFeedbackContext(s string) error {
	return validateFeedbackText("context", s)
}

type feedbackCmd struct {
	cmd *cobra.Command

	feature    string
	sentiment  string
	message    string
	context    string
	actor      string
	jsonOutput bool
	apiBaseURL string

	// isInteractive tracks whether we detected a TTY and went through the
	// prompt flow instead of relying on flags.
	isInteractive bool
}

// missingRequiredFields returns the flag names of any required fields that
// are still empty. It's used both to decide whether to fall back to
// interactive prompts and to validate flags supplied non-interactively.
func (feedback *feedbackCmd) missingRequiredFields() []string {
	requiredFields := []struct{ flag, value string }{
		{"--sentiment", feedback.sentiment},
		{"--message", feedback.message},
		{"--context", feedback.context},
		{"--actor", feedback.actor},
	}

	var missing []string
	for _, field := range requiredFields {
		if field.value == "" {
			missing = append(missing, field.flag)
		}
	}

	return missing
}

func newFeedbackCmd() *feedbackCmd {
	feedback := &feedbackCmd{}

	feedback.cmd = &cobra.Command{
		Use:   "feedback",
		Args:  validators.NoArgs,
		Short: "Share feedback on any part of Stripe",
		Long: `Share feedback on any part of Stripe (docs, APIs, the Dashboard, the CLI, and more) directly from your terminal.

This is not a support channel and Stripe cannot respond to feedback submitted here.
For help with your account or integration, use https://support.stripe.com`,
		RunE: feedback.runFeedbackCmd,
	}

	feedback.cmd.Flags().StringVar(&feedback.feature, "feature", "", fmt.Sprintf("Product area this feedback is about (%s)", strings.Join(feedbackFeatureAreas, ", ")))
	feedback.cmd.Flags().StringVar(&feedback.sentiment, "sentiment", "", fmt.Sprintf("Sentiment of the feedback (%s)", strings.Join(feedbackSentiments, ", ")))
	feedback.cmd.Flags().StringVar(&feedback.message, "message", "", "Your feedback message")
	feedback.cmd.Flags().StringVar(&feedback.context, "context", "", "What were you trying to do?")
	feedback.cmd.Flags().StringVar(&feedback.actor, "actor", "", "Who is submitting this feedback: human or agent")
	feedback.cmd.Flags().BoolVar(&feedback.jsonOutput, "json", false, "Output the result as JSON")

	feedback.cmd.Flags().StringVar(&feedback.apiBaseURL, "api-base", stripe.DefaultAPIBaseURL, "Sets the API base URL")
	feedback.cmd.Flags().MarkHidden("api-base") // #nosec G104

	return feedback
}

func (feedback *feedbackCmd) runFeedbackCmd(cmd *cobra.Command, args []string) error {
	apiKey, err := Config.Profile.GetAPIKey(false)
	if err != nil {
		return err
	}

	// --json signals a scripted invocation, so it always skips the prompt
	// flow, even when attached to a TTY.
	feedback.isInteractive = !feedback.jsonOutput && term.IsTerminal(int(os.Stdin.Fd()))

	// Whatever was already supplied via flags is validated up front,
	// regardless of interactivity, so an invalid --feature/--sentiment/
	// --message/--context fails fast instead of silently falling through
	// to a prompt that would ask the user for it again.
	if err := feedback.validateSuppliedFields(); err != nil {
		return err
	}

	// Fall back to interactive prompts whenever any required field is
	// still missing, not just when every field is — a user who supplied
	// some flags from a terminal should be prompted only for the rest,
	// rather than hit a hard validation error.
	if feedback.isInteractive && len(feedback.missingRequiredFields()) > 0 {
		if err := feedback.promptForFeedback(cmd); err != nil {
			return err
		}
	} else if missing := feedback.missingRequiredFields(); len(missing) > 0 {
		return fmt.Errorf("%s required when running non-interactively", strings.Join(missing, ", "))
	}

	resp, err := feedback.submitFeedback(cmd, apiKey)
	if err != nil {
		return err
	}

	feedback.printResult(cmd, resp)

	return nil
}

// validateSuppliedFields validates the content of any fields already
// supplied via flags (message/context length, sentiment/feature
// membership), independent of whether all required fields are present —
// that's checked separately by missingRequiredFields.
func (feedback *feedbackCmd) validateSuppliedFields() error {
	if err := validators.CallNonEmpty(validateFeedbackMessage, feedback.message); err != nil {
		return err
	}
	if err := validators.CallNonEmpty(validateFeedbackContext, feedback.context); err != nil {
		return err
	}
	if err := validators.CallNonEmpty(validators.OneOf(feedbackSentiments...), feedback.sentiment); err != nil {
		return fmt.Errorf("--sentiment %w", err)
	}
	if err := validators.CallNonEmpty(validators.OneOf(feedbackFeatureAreas...), feedback.feature); err != nil {
		return fmt.Errorf("--feature %w", err)
	}

	return nil
}

// promptForFeedback interactively prompts for any required field that
// wasn't already supplied (and validated) via flags, leaving
// already-supplied fields untouched.
func (feedback *feedbackCmd) promptForFeedback(cmd *cobra.Command) error {
	out := cmd.OutOrStdout()

	fmt.Fprintln(out, ansi.Color(out).Yellow("This is not a support channel. Stripe cannot respond to feedback submitted here."))
	fmt.Fprintln(out, "For help with your account or integration, use: https://support.stripe.com")
	fmt.Fprintln(out)

	if feedback.feature == "" {
		feature, err := selectFeedbackEnum("Which area is this about? (type to filter)", feedbackFeatureAreas)
		if err != nil {
			return err
		}
		feedback.feature = feature
	}

	if feedback.sentiment == "" {
		sentiment, err := selectFeedbackEnum("How are you feeling about it?", feedbackSentiments)
		if err != nil {
			return err
		}
		feedback.sentiment = sentiment
	}

	if feedback.message == "" {
		message, err := textPrompt("What would you like to share?", validateFeedbackMessage)
		if err != nil {
			return err
		}
		feedback.message = message
	}

	if feedback.context == "" {
		context, err := textPrompt("What were you trying to do?", validateFeedbackContext)
		if err != nil {
			return err
		}
		feedback.context = context
	}

	if feedback.actor == "" {
		// A human is physically present to answer the prompts, so we know
		// who is submitting the feedback without having to ask, unless
		// --actor was already supplied.
		feedback.actor = "human"
	}

	// message/context are sanitized once, in submitFeedback, since that's
	// the single point both this interactive path and the non-interactive
	// (flag-driven) path converge on.
	return nil
}

// selectFeedbackEnum prompts the user to choose one of options, supporting
// arrow-key navigation and typeahead filtering (case-insensitive substring
// match on the option text).
func selectFeedbackEnum(label string, options []string) (string, error) {
	prompt := promptui.Select{
		Label: label,
		Items: options,
		Templates: &promptui.SelectTemplates{
			// label is substituted via Sprintf, so it must not itself
			// contain '%' (the promptui template syntax below, "{{ }}",
			// is untouched by Sprintf and safe).
			Selected: ansi.Faint(fmt.Sprintf("✔ %s: {{ . | bold }} ", label)),
		},
		Searcher: func(input string, index int) bool {
			return strings.Contains(strings.ToLower(options[index]), strings.ToLower(input))
		},
		StartInSearchMode: len(options) > 5,
		Size:              7,
	}

	_, result, err := prompt.Run()
	if err != nil {
		return "", err
	}

	return result, nil
}

// textPrompt prompts the user for a single line of free text, re-prompting
// until validator (which must be non-nil) passes.
func textPrompt(label string, validator promptui.ValidateFunc) (string, error) {
	prompt := promptui.Prompt{
		Label:    label,
		Validate: validator,
	}

	result, err := prompt.Run()
	if err != nil {
		return "", err
	}

	return result, nil
}

func (feedback *feedbackCmd) submitFeedback(cmd *cobra.Command, apiKey string) (*feedbackResponse, error) {
	baseURL, err := url.Parse(feedback.apiBaseURL)
	if err != nil {
		return nil, err
	}

	client := &stripe.Client{
		BaseURL: baseURL,
		APIKey:  apiKey,
	}

	message := sanitizeText(feedback.message)
	context := sanitizeText(feedback.context)

	form := url.Values{}
	form.Set("sentiment", feedback.sentiment)
	form.Set("message", message)
	form.Set("channel", "cli")
	form.Set("actor", feedback.actor)

	if feedback.feature != "" {
		form.Set("feature_area", feedback.feature)
	}
	form.Set("context", context)

	// Explicitly supply metadata as form params; the server does not infer
	// this information from headers.
	if deviceName, err := Config.Profile.GetDeviceName(); err == nil && deviceName != "" {
		form.Set("device_name", deviceName)
	}
	form.Set("cli_version", version.Version)
	form.Set("os", runtime.GOOS)

	if terminalProgram := useragent.DetectTerminalProgram(os.Getenv); terminalProgram != "" {
		form.Set("terminal_program", terminalProgram)
	}

	if aiAgent := useragent.DetectAIAgent(os.Getenv); aiAgent != "" {
		form.Set("ai_agent", aiAgent)
	}

	if os.Getenv("CI") != "" {
		form.Set("is_ci", "true")
	}

	resp, err := client.PerformRequest(cmd.Context(), http.MethodPost, feedbackEndpoint, form.Encode(), func(req *http.Request) error {
		req.Header.Set("Stripe-Version", requests.StripePreviewVersionHeaderValue)
		return nil
	})
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to submit feedback (status %d): %s", resp.StatusCode, string(body))
	}

	var result feedbackResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &result, nil
}

func (feedback *feedbackCmd) printResult(cmd *cobra.Command, resp *feedbackResponse) {
	if feedback.jsonOutput {
		out, err := json.Marshal(resp)
		if err != nil {
			fmt.Fprintln(cmd.ErrOrStderr(), err)
			return
		}
		fmt.Fprintln(cmd.OutOrStdout(), string(out))
		return
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Thanks — feedback recorded (id: %s)\n", resp.ID)
}
