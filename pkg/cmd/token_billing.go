package cmd

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
	"strings"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/stripe/stripe-cli/pkg/ansi"
	"github.com/stripe/stripe-cli/pkg/requests"
	"github.com/stripe/stripe-cli/pkg/stripe"
	"github.com/stripe/stripe-cli/pkg/validators"
)

const tokenBillingOnboardingInitPath = "/v1/token_billing/onboarding/init"

type tokenBillingCmd struct {
	cmd *cobra.Command
}

type tokenBillingInitCmd struct {
	cmd *cobra.Command
	rb  requests.Base

	planName                   string
	models                     []string
	defaultMarkupPercent       string
	priceTrackingPreference    string
	subscriptionFeeAmount      string
	creditGrantPerPeriodAmount string
	interval                   string
}

type tokenBillingOnboardingStatus struct {
	Status                     string            `json:"status"`
	Message                    string            `json:"message"`
	AIGatewayEnabled           bool              `json:"ai_gateway_enabled"`
	TestRequestCompleted       bool              `json:"test_request_completed"`
	PricingConfigured          bool              `json:"pricing_configured"`
	TokenMetersConfigured      bool              `json:"token_meters_configured"`
	ZeroCreditRejectionEnabled bool              `json:"zero_credit_rejection_enabled"`
	WebhookHealthy             bool              `json:"webhook_healthy"`
	InferenceMode              string            `json:"inference_mode"`
	GatewayEndpoint            string            `json:"gateway_endpoint"`
	EnvironmentVariables       map[string]string `json:"environment_variables"`
	Resources                  map[string]string `json:"resources"`
	NextSteps                  []string          `json:"next_steps"`
}

func newTokenBillingCmd() *tokenBillingCmd {
	tbc := &tokenBillingCmd{}
	tbc.cmd = &cobra.Command{
		Use:   "token-billing",
		Short: "Initialize and inspect Token Billing setup",
		Long: `Initialize and inspect Token Billing setup.

Token Billing lets you meter AI gateway usage and connect it to Stripe Billing
resources for testing token-based pricing flows.`,
		Args: validators.NoArgs,
	}

	tbc.cmd.AddCommand(newTokenBillingInitCmd().cmd)
	return tbc
}

func newTokenBillingInitCmd() *tokenBillingInitCmd {
	ic := &tokenBillingInitCmd{
		planName:                   "AI starter",
		models:                     []string{"openai/gpt-4o-mini"},
		defaultMarkupPercent:       "20.0",
		priceTrackingPreference:    "disabled",
		subscriptionFeeAmount:      "0",
		creditGrantPerPeriodAmount: "1000",
		interval:                   "month",
	}

	ic.rb = requests.Base{
		Method:         http.MethodPost,
		Profile:        &Config.Profile,
		SuppressOutput: true,
	}

	ic.cmd = &cobra.Command{
		Use:   "init",
		Short: "Initialize Token Billing resources for the current account",
		Long: `Initialize Token Billing resources for the current account.

This command creates or reuses the Token Billing pricing plan and meter setup
needed for the AI gateway testmode onboarding flow, then prints resource IDs,
the gateway endpoint, suggested environment variables, and next steps.`,
		Example: `  stripe token-billing init

  stripe token-billing init \
    --plan-name "AI starter" \
    --model openai/gpt-4o-mini \
    --default-markup-percent 20.0 \
    --subscription-fee-amount 0 \
    --credit-grant-per-period-amount 1000 \
    --interval month`,
		RunE: ic.runTokenBillingInitCmd,
		Args: validators.NoArgs,
	}

	ic.cmd.Flags().StringVar(&ic.planName, "plan-name", ic.planName, "Name for the Token Billing pricing plan")
	ic.cmd.Flags().StringArrayVar(&ic.models, "model", ic.models, "AI model to configure, formatted as publisher/model. Repeat for multiple models")
	ic.cmd.Flags().StringVar(&ic.defaultMarkupPercent, "default-markup-percent", ic.defaultMarkupPercent, "Default markup percentage for model prices")
	ic.cmd.Flags().StringVar(&ic.priceTrackingPreference, "price-tracking-preference", ic.priceTrackingPreference, "Price tracking preference: disabled, migrate_all, or new_customers_only")
	ic.cmd.Flags().StringVar(&ic.subscriptionFeeAmount, "subscription-fee-amount", ic.subscriptionFeeAmount, "Subscription fee amount in USD minor units")
	ic.cmd.Flags().StringVar(&ic.creditGrantPerPeriodAmount, "credit-grant-per-period-amount", ic.creditGrantPerPeriodAmount, "Credit grant amount per billing period in USD minor units")
	ic.cmd.Flags().StringVar(&ic.interval, "interval", ic.interval, "Billing interval for the pricing plan")
	ic.cmd.Flags().BoolVar(&ic.rb.Livemode, "live", false, "Make a live request (default: test)")
	ic.cmd.Flags().BoolVar(&ic.rb.DryRun, "dry-run", false, "Preview the request without sending it")
	ic.cmd.Flags().BoolVar(&ic.rb.DarkStyle, "dark-style", false, "Use a darker color scheme better suited for lighter command-lines")
	ic.cmd.Flags().StringVar(&ic.rb.APIBaseURL, "api-base", stripe.DefaultAPIBaseURL, "Sets the API base URL")
	ic.cmd.Flags().MarkHidden("api-base") // #nosec G104

	return ic
}

func (ic *tokenBillingInitCmd) runTokenBillingInitCmd(cmd *cobra.Command, args []string) error {
	if err := stripe.ValidateAPIBaseURL(ic.rb.APIBaseURL); err != nil {
		return err
	}

	if ic.shouldPromptForInit(cmd) {
		if err := ic.promptForInitConfig(cmd); err != nil {
			return err
		}
	}

	if len(ic.models) == 0 {
		return fmt.Errorf("at least one --model is required")
	}

	apiKey, err := ic.rb.Profile.GetAPIKey(ic.rb.Livemode)
	if err != nil {
		return err
	}

	body := ic.buildRequestBody()
	if ic.rb.DryRun {
		output, err := ic.rb.BuildDryRunOutput(apiKey, ic.rb.APIBaseURL, tokenBillingOnboardingInitPath, &requests.RequestParameters{}, body)
		if err != nil {
			return err
		}
		b, err := json.MarshalIndent(output, "", "  ")
		if err != nil {
			return err
		}
		fmt.Fprintln(cmd.OutOrStdout(), string(b))
		return nil
	}

	spinner := ansi.StartNewSpinner("Initializing Token Billing resources...", cmd.OutOrStdout())
	resp, err := ic.rb.MakeRequest(cmd.Context(), apiKey, tokenBillingOnboardingInitPath, &requests.RequestParameters{}, body, true, nil)
	if err != nil {
		ansi.StopSpinner(spinner, "Token Billing initialization failed", cmd.OutOrStdout())
		return err
	}
	ansi.StopSpinner(spinner, "Token Billing resources ready", cmd.OutOrStdout())

	var status tokenBillingOnboardingStatus
	if err := json.Unmarshal(resp, &status); err != nil {
		return fmt.Errorf("failed to parse Token Billing onboarding response: %w", err)
	}

	ic.printStatus(cmd, status)
	return nil
}

func (ic *tokenBillingInitCmd) shouldPromptForInit(cmd *cobra.Command) bool {
	if !cmdInputIsTerminal(cmd) {
		return false
	}

	for _, flagName := range []string{
		"plan-name",
		"model",
		"default-markup-percent",
		"price-tracking-preference",
		"subscription-fee-amount",
		"credit-grant-per-period-amount",
		"interval",
	} {
		flag := cmd.Flags().Lookup(flagName)
		if flag != nil && flag.Changed {
			return false
		}
	}

	return true
}

func cmdInputIsTerminal(cmd *cobra.Command) bool {
	f, ok := cmd.InOrStdin().(*os.File)
	return ok && term.IsTerminal(int(f.Fd()))
}

func (ic *tokenBillingInitCmd) promptForInitConfig(cmd *cobra.Command) error {
	out := cmd.OutOrStdout()
	reader := bufio.NewReader(cmd.InOrStdin())
	color := ansi.Color(out)

	fmt.Fprintln(out, color.Bold("Token Billing setup"))
	fmt.Fprintln(out, faint(out, "Create pricing, meter, and AI Gateway testmode resources for this account."))
	fmt.Fprintln(out, faint(out, "Press Enter to accept the suggested default for each prompt."))
	fmt.Fprintln(out)

	var err error
	ic.planName, err = promptString(out, reader, "Plan name", ic.planName)
	if err != nil {
		return err
	}

	models, err := promptString(out, reader, "Models (comma-separated publisher/model values)", strings.Join(ic.models, ", "))
	if err != nil {
		return err
	}
	ic.models = splitCommaSeparatedValues(models)

	ic.defaultMarkupPercent, err = promptString(out, reader, "Default markup percent", ic.defaultMarkupPercent)
	if err != nil {
		return err
	}

	ic.priceTrackingPreference, err = promptString(out, reader, "Price tracking preference", ic.priceTrackingPreference)
	if err != nil {
		return err
	}

	ic.subscriptionFeeAmount, err = promptString(out, reader, "Subscription fee amount in USD minor units", ic.subscriptionFeeAmount)
	if err != nil {
		return err
	}

	ic.creditGrantPerPeriodAmount, err = promptString(out, reader, "Credit grant amount per billing period in USD minor units", ic.creditGrantPerPeriodAmount)
	if err != nil {
		return err
	}

	ic.interval, err = promptString(out, reader, "Billing interval", ic.interval)
	if err != nil {
		return err
	}

	fmt.Fprintln(out)
	fmt.Fprintln(out, color.Bold("Review configuration"))
	printSummaryLine(out, "Plan name", ic.planName)
	printSummaryLine(out, "Models", strings.Join(ic.models, ", "))
	printSummaryLine(out, "Default markup percent", ic.defaultMarkupPercent)
	printSummaryLine(out, "Price tracking preference", ic.priceTrackingPreference)
	printSummaryLine(out, "Subscription fee amount", ic.subscriptionFeeAmount)
	printSummaryLine(out, "Credit grant amount per period", ic.creditGrantPerPeriodAmount)
	printSummaryLine(out, "Billing interval", ic.interval)
	fmt.Fprintln(out)

	confirmed, err := promptConfirm(out, reader, "Proceed with creating Token Billing resources?", true)
	if err != nil {
		return err
	}
	if !confirmed {
		return fmt.Errorf("Token Billing initialization canceled")
	}

	fmt.Fprintln(out)
	return nil
}

func promptString(out io.Writer, reader *bufio.Reader, label string, defaultValue string) (string, error) {
	color := ansi.Color(out)
	fmt.Fprintf(out, "%s %s: ", color.Bold(label), faint(out, "["+defaultValue+"]"))
	input, err := reader.ReadString('\n')
	if err != nil && err != io.EOF {
		return "", err
	}

	value := strings.TrimSpace(input)
	if value == "" {
		return defaultValue, nil
	}
	return value, nil
}

func promptConfirm(out io.Writer, reader *bufio.Reader, label string, defaultYes bool) (bool, error) {
	suffix := "[y/N]"
	if defaultYes {
		suffix = "[Y/n]"
	}

	color := ansi.Color(out)
	fmt.Fprintf(out, "%s %s ", color.Bold(label), faint(out, suffix))
	input, err := reader.ReadString('\n')
	if err != nil && err != io.EOF {
		return false, err
	}

	value := strings.ToLower(strings.TrimSpace(input))
	if value == "" {
		return defaultYes, nil
	}

	return value == "y" || value == "yes", nil
}

func splitCommaSeparatedValues(value string) []string {
	parts := strings.Split(value, ",")
	values := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			values = append(values, trimmed)
		}
	}
	return values
}

func printSummaryLine(out io.Writer, label string, value string) {
	fmt.Fprintf(out, "  %s %s\n", faint(out, label+":"), value)
}

func faint(out io.Writer, text string) string {
	return ansi.Color(out).Faint(text).String()
}

func (ic *tokenBillingInitCmd) buildRequestBody() map[string]interface{} {
	models := make([]interface{}, 0, len(ic.models))
	for _, model := range ic.models {
		models = append(models, model)
	}

	return map[string]interface{}{
		"plan_name":                      ic.planName,
		"models":                         models,
		"default_markup_percent":         ic.defaultMarkupPercent,
		"price_tracking_preference":      ic.priceTrackingPreference,
		"subscription_fee_amount":        ic.subscriptionFeeAmount,
		"credit_grant_per_period_amount": ic.creditGrantPerPeriodAmount,
		"interval":                       ic.interval,
	}
}

func (ic *tokenBillingInitCmd) printStatus(cmd *cobra.Command, status tokenBillingOnboardingStatus) {
	out := cmd.OutOrStdout()
	color := ansi.Color(out)
	fmt.Fprintf(out, "%s %s\n", color.Green("✔"), color.Bold("Token Billing initialized"))
	if status.Status != "" {
		printSummaryLine(out, "Status", status.Status)
	}
	if status.Message != "" {
		fmt.Fprintf(out, "%s\n", faint(out, status.Message))
	}
	if status.GatewayEndpoint != "" {
		fmt.Fprintf(out, "\n%s\n  %s\n", color.Bold("Gateway endpoint"), status.GatewayEndpoint)
	}

	printStringMap(out, "Resources", status.Resources)
	printStringMap(out, "Suggested environment variables", status.EnvironmentVariables)

	fmt.Fprintf(out, "\n%s\n", color.Bold("Checklist"))
	printChecklistItem(out, "AI Gateway enabled", status.AIGatewayEnabled)
	printChecklistItem(out, "Pricing configured", status.PricingConfigured)
	printChecklistItem(out, "Token meters configured", status.TokenMetersConfigured)
	printChecklistItem(out, "Test request completed", status.TestRequestCompleted)
	printChecklistItem(out, "Zero-credit rejection enabled", status.ZeroCreditRejectionEnabled)
	printChecklistItem(out, "Webhook healthy", status.WebhookHealthy)
	if status.InferenceMode != "" {
		printSummaryLine(out, "Inference mode", status.InferenceMode)
	}

	if len(status.NextSteps) > 0 {
		fmt.Fprintf(out, "\n%s\n", color.Bold("Next steps"))
		for _, step := range status.NextSteps {
			fmt.Fprintf(out, "  %s %s\n", color.Cyan("→"), step)
		}
	}
}

func printStringMap(out interface{ Write([]byte) (int, error) }, title string, values map[string]string) {
	if len(values) == 0 {
		return
	}

	color := ansi.Color(out)
	fmt.Fprintf(out, "\n%s\n", color.Bold(title))
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		value := strings.TrimSpace(values[key])
		if value == "" {
			continue
		}
		fmt.Fprintf(out, "  %s%s\n", faint(out, key+"="), value)
	}
}

func printChecklistItem(out interface{ Write([]byte) (int, error) }, label string, done bool) {
	color := ansi.Color(out)
	marker := color.Yellow("○")
	if done {
		marker = color.Green("✔")
	}
	fmt.Fprintf(out, "  %s %s\n", marker, label)
}
