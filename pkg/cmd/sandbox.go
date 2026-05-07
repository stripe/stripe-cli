package cmd

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/logrusorgru/aurora"
	"github.com/spf13/cobra"

	"github.com/stripe/stripe-cli/pkg/ansi"
	"github.com/stripe/stripe-cli/pkg/login"
	"github.com/stripe/stripe-cli/pkg/login/keys"
	"github.com/stripe/stripe-cli/pkg/open"
	"github.com/stripe/stripe-cli/pkg/sandbox"
	"github.com/stripe/stripe-cli/pkg/stripe"
	"github.com/stripe/stripe-cli/pkg/validators"
)

// defaultSandboxBaseURL is the production endpoint for sandbox provisioning.
// Confirmed reachable as a public HTTPS service by the DEeP team.
const defaultSandboxBaseURL = "https://ai.stripe.com"

type sandboxCmd struct {
	cmd *cobra.Command
}

type sandboxCreateCmd struct {
	cmd          *cobra.Command
	email        string
	name         string
	baseURL      string
	dashboardURL string
	dashboard    bool
}

func newSandboxCmd() *sandboxCmd {
	sc := &sandboxCmd{}
	sc.cmd = &cobra.Command{
		Use:   "sandbox",
		Short: "Manage Stripe sandbox environments",
		Args:  validators.NoArgs,
		Annotations: map[string]string{
			AIAgentHelpAnnotationKey: "  Use `stripe sandbox create --email auto` to provision a sandbox without authentication.\n" +
				"  If provisioning fails, falls back to browser login (like stripe login).\n" +
				"  Use `stripe sandbox create --dashboard` to skip provisioning and use browser login directly.",
		},
	}

	createCmd := newSandboxCreateCmd()
	sc.cmd.AddCommand(createCmd.cmd)
	return sc
}

func newSandboxCreateCmd() *sandboxCreateCmd {
	scc := &sandboxCreateCmd{}
	scc.cmd = &cobra.Command{
		Use:   "create",
		Short: "Provision a new sandbox environment",
		Long: `Create a new Stripe sandbox with test API keys.

By default, uses a proof-of-work challenge to provision a temporary sandbox
without authentication. If that fails, automatically falls back to browser login.

Use --dashboard to skip provisioning and connect an existing Stripe account
via browser login directly (same flow as stripe login).

Keys are saved to the CLI config so subsequent stripe commands work immediately.`,
		Example: `  stripe sandbox create --email auto
  stripe sandbox create --email you@example.com --name "Jane Smith"
  stripe sandbox create --dashboard`,
		Args: validators.NoArgs,
		Annotations: map[string]string{
			AIAgentHelpAnnotationKey: "  Provisions a sandbox and saves keys to the CLI config (same as stripe login).\n" +
				"  Pass --email auto to resolve your email from git config user.email.\n" +
				"  Falls back to browser login if provisioning fails. Use --dashboard to go directly to browser.",
		},
		RunE: scc.runSandboxCreateCmd,
	}

	scc.cmd.Flags().StringVar(&scc.email, "email", "", "Your email address (required for provisioning). Use \"auto\" to infer from git config user.email")
	scc.cmd.Flags().StringVar(&scc.name, "name", "", "Your full name (optional). Use \"auto\" to infer from git config user.name")
	scc.cmd.Flags().BoolVar(&scc.dashboard, "dashboard", false, "Skip provisioning and connect via browser login")

	scc.cmd.Flags().StringVar(&scc.baseURL, "base-url", defaultSandboxBaseURL, "Sets the sandbox API base URL")
	_ = scc.cmd.Flags().MarkHidden("base-url")

	scc.cmd.Flags().StringVar(&scc.dashboardURL, "dashboard-base", stripe.DefaultDashboardBaseURL, "Sets the dashboard base URL")
	_ = scc.cmd.Flags().MarkHidden("dashboard-base")

	return scc
}

func (scc *sandboxCreateCmd) runSandboxCreateCmd(cmd *cobra.Command, args []string) error {
	color := ansi.Color(cmd.ErrOrStderr())

	if scc.dashboard {
		return scc.runDashboardFlow(cmd, color)
	}

	// PoW flow with fallback to dashboard
	email, err := resolveAutoValue(scc.email, "user.email", "--email")
	if err != nil {
		return err
	}

	var name string
	if scc.name != "" {
		name, err = resolveAutoValue(scc.name, "user.name", "--name")
		if err != nil {
			return err
		}
	}

	result, err := scc.runProvisionFlow(cmd, color, email, name)
	if err != nil {
		fmt.Fprintf(cmd.ErrOrStderr(), "\nProvisioning failed: %v\n", err)
		fmt.Fprintln(cmd.ErrOrStderr(), color.Yellow("Falling back to browser login..."))
		return scc.runDashboardFlow(cmd, color)
	}

	return scc.outputResult(cmd, color, result)
}

func (scc *sandboxCreateCmd) runProvisionFlow(cmd *cobra.Command, color aurora.Aurora, email, name string) (*sandbox.ProvisionResponse, error) {
	client := sandbox.NewClient(scc.baseURL)

	fmt.Fprintln(cmd.ErrOrStderr(), color.Yellow("Solving proof-of-work..."))
	start := time.Now()

	challengeResp, err := client.GetChallenge(cmd.Context(), email)
	if err != nil {
		return nil, err
	}

	solution, err := sandbox.SolveChallenge(cmd.Context(), challengeResp.Algorithm, challengeResp.Challenge, challengeResp.Salt)
	if err != nil {
		return nil, err
	}

	elapsed := time.Since(start)
	fmt.Fprintf(cmd.ErrOrStderr(), "Solved in %s\n", elapsed.Round(time.Millisecond))

	provisionReq := sandbox.ProvisionRequest{
		Algorithm: challengeResp.Algorithm,
		Challenge: challengeResp.Challenge,
		Salt:      challengeResp.Salt,
		Signature: challengeResp.Signature,
		Number:    solution,
		Email:     email,
		Name:      name,
	}

	return client.Provision(cmd.Context(), provisionReq)
}

func (scc *sandboxCreateCmd) runDashboardFlow(cmd *cobra.Command, color aurora.Aurora) error {
	links, err := login.GetLinks(cmd.Context(), scc.dashboardURL, "stripe-sandbox")
	if err != nil {
		return err
	}

	fmt.Fprintf(cmd.ErrOrStderr(), "Your pairing code is: %s\n", links.VerificationCode)
	fmt.Fprintf(cmd.ErrOrStderr(), "\nOpening browser to connect your Stripe account...\n")
	fmt.Fprintf(cmd.ErrOrStderr(), "If the browser doesn't open, visit:\n  %s\n\n", links.BrowserURL)

	if open.CanOpenBrowser() {
		_ = open.Browser(links.BrowserURL)
	}

	fmt.Fprintln(cmd.ErrOrStderr(), color.Yellow("Waiting for confirmation..."))

	response, _, err := keys.PollForKey(cmd.Context(), links.PollURL, 0, 0)
	if err != nil {
		return err
	}

	result := &sandbox.ProvisionResponse{
		SecretKey:      response.TestModeAPIKey,
		PublishableKey: response.TestModePublishableKey,
		AccountID:      response.AccountID,
	}

	return scc.outputResult(cmd, color, result)
}

func (scc *sandboxCreateCmd) outputResult(cmd *cobra.Command, color aurora.Aurora, result *sandbox.ProvisionResponse) error {
	if err := saveSandboxToConfig(result); err != nil {
		fmt.Fprintf(cmd.ErrOrStderr(), "Warning: could not save to config: %v\n", err)
	}

	out, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return err
	}
	fmt.Fprintln(cmd.OutOrStdout(), string(out))

	fmt.Fprintf(cmd.ErrOrStderr(), "\n%s\n", color.Green("Provisioned!"))
	if result.ClaimURL != "" {
		fmt.Fprintf(cmd.ErrOrStderr(), "Claim your sandbox: %s\n", result.ClaimURL)
	}
	if result.ExpiresAt != "" {
		fmt.Fprintf(cmd.ErrOrStderr(), "Expires: %s\n", result.ExpiresAt)
	}

	return nil
}

func saveSandboxToConfig(result *sandbox.ProvisionResponse) error {
	Config.Profile.TestModeAPIKey = result.SecretKey
	Config.Profile.TestModePublishableKey = result.PublishableKey
	if result.AccountID != "" {
		Config.Profile.AccountID = result.AccountID
		Config.Profile.DisplayName = result.AccountID
	}
	if err := Config.Profile.CreateProfile(); err != nil {
		return err
	}
	if result.ClaimURL != "" {
		if err := Config.Profile.WriteConfigField("sandbox_claim_url", result.ClaimURL); err != nil {
			return err
		}
	}
	return nil
}

func resolveAutoValue(value, gitKey, flagName string) (string, error) {
	if value == "" {
		return "", fmt.Errorf("%s is required", flagName)
	}

	if value == "auto" {
		resolved := sandbox.GitConfigFunc(gitKey)
		if resolved == "" {
			return "", fmt.Errorf("%s auto requires git config %s to be set, but it was not found", flagName, gitKey)
		}
		return resolved, nil
	}

	return value, nil
}
