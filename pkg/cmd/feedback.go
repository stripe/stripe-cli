package cmd

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"runtime"
	"strings"

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

var feedbackFeatureAreas = []string{
	"billing",
	"connect",
	"payments",
	"radar",
	"issuing",
	"terminal",
	"webhooks",
	"docs",
	"cli",
	"dashboard",
	"other",
}

var feedbackSentiments = []string{"positive", "negative", "neutral"}

// feedbackResponse is the JSON body returned by the feedback API.
type feedbackResponse struct {
	ID      string `json:"id"`
	Success bool   `json:"success"`
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

func newFeedbackCmd() *feedbackCmd {
	fc := &feedbackCmd{}

	fc.cmd = &cobra.Command{
		Use:   "feedback",
		Args:  validators.NoArgs,
		Short: "Share feedback on any part of Stripe",
		Long: `Share feedback on any part of Stripe (docs, APIs, the Dashboard, the CLI, and more) directly from your terminal.

This is not a support channel and Stripe cannot respond to feedback submitted here.
For help with your account or integration, use https://support.stripe.com`,
		RunE: fc.runFeedbackCmd,
	}

	fc.cmd.Flags().StringVar(&fc.feature, "feature", "", fmt.Sprintf("Product area this feedback is about (%s)", strings.Join(feedbackFeatureAreas, ", ")))
	fc.cmd.Flags().StringVar(&fc.sentiment, "sentiment", "", fmt.Sprintf("Sentiment of the feedback (%s)", strings.Join(feedbackSentiments, ", ")))
	fc.cmd.Flags().StringVar(&fc.message, "message", "", "Your feedback message")
	fc.cmd.Flags().StringVar(&fc.context, "context", "", "What were you trying to do?")
	fc.cmd.Flags().StringVar(&fc.actor, "actor", "", "Who is submitting this feedback: human or agent")
	fc.cmd.Flags().BoolVar(&fc.jsonOutput, "json", false, "Output the result as JSON")

	fc.cmd.Flags().StringVar(&fc.apiBaseURL, "api-base", stripe.DefaultAPIBaseURL, "Sets the API base URL")
	fc.cmd.Flags().MarkHidden("api-base") // #nosec G104

	return fc
}

func (fc *feedbackCmd) runFeedbackCmd(cmd *cobra.Command, args []string) error {
	apiKey, err := Config.Profile.GetAPIKey(false)
	if err != nil {
		return err
	}

	fc.isInteractive = term.IsTerminal(int(os.Stdin.Fd()))

	if fc.isInteractive && fc.sentiment == "" && fc.message == "" {
		if err := fc.promptForFeedback(cmd); err != nil {
			return err
		}
		// A human is physically present to answer the prompts, so we know
		// who is submitting the feedback without having to ask.
		fc.actor = "human"
	} else {
		if err := fc.validateNonInteractive(); err != nil {
			return err
		}
	}

	resp, err := fc.submitFeedback(cmd, apiKey)
	if err != nil {
		return err
	}

	fc.printResult(cmd, resp)

	return nil
}

func (fc *feedbackCmd) validateNonInteractive() error {
	var missing []string
	if fc.sentiment == "" {
		missing = append(missing, "--sentiment")
	}
	if fc.message == "" {
		missing = append(missing, "--message")
	}
	if fc.actor == "" {
		missing = append(missing, "--actor")
	}
	if len(missing) > 0 {
		return fmt.Errorf("%s required when running non-interactively", strings.Join(missing, ", "))
	}

	return nil
}

func (fc *feedbackCmd) promptForFeedback(cmd *cobra.Command) error {
	out := cmd.OutOrStdout()
	reader := bufio.NewReader(cmd.InOrStdin())

	fmt.Fprintln(out, ansi.Color(out).Yellow("This is not a support channel. Stripe cannot respond to feedback submitted here."))
	fmt.Fprintln(out, "For help with your account or integration, use: https://support.stripe.com")
	fmt.Fprintln(out)

	feature, err := promptLine(out, reader, fmt.Sprintf("What is this feedback about? [%s]", strings.Join(feedbackFeatureAreas, ", ")))
	if err != nil {
		return err
	}
	fc.feature = feature

	sentiment, err := promptLine(out, reader, fmt.Sprintf("Sentiment: [%s]", strings.Join(feedbackSentiments, ", ")))
	if err != nil {
		return err
	}
	fc.sentiment = sentiment

	message, err := promptLine(out, reader, "Your feedback:")
	if err != nil {
		return err
	}
	fc.message = message

	context, err := promptLine(out, reader, "What were you trying to do? (optional, press Enter to skip)")
	if err != nil {
		return err
	}
	fc.context = context

	return nil
}

func promptLine(out io.Writer, reader *bufio.Reader, prompt string) (string, error) {
	fmt.Fprintln(out, prompt)
	fmt.Fprint(out, "> ")

	line, err := reader.ReadString('\n')
	if err != nil && err != io.EOF {
		return "", err
	}
	fmt.Fprintln(out)

	return strings.TrimSpace(line), nil
}

func (fc *feedbackCmd) submitFeedback(cmd *cobra.Command, apiKey string) (*feedbackResponse, error) {
	baseURL, err := url.Parse(fc.apiBaseURL)
	if err != nil {
		return nil, err
	}

	client := &stripe.Client{
		BaseURL: baseURL,
		APIKey:  apiKey,
	}

	form := url.Values{}
	form.Set("sentiment", fc.sentiment)
	form.Set("message", fc.message)
	form.Set("channel", "cli")
	form.Set("actor", fc.actor)

	if fc.feature != "" {
		form.Set("feature_area", fc.feature)
	}
	if fc.context != "" {
		form.Set("context", fc.context)
	}

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

func (fc *feedbackCmd) printResult(cmd *cobra.Command, resp *feedbackResponse) {
	if fc.jsonOutput {
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
