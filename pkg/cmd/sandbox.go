package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/stripe/stripe-cli/pkg/ansi"
	"github.com/stripe/stripe-cli/pkg/sandbox"
	"github.com/stripe/stripe-cli/pkg/validators"
)

const defaultSandboxBaseURL = "https://ai.stripe.com"

type sandboxCmd struct {
	cmd *cobra.Command
}

type sandboxCreateCmd struct {
	cmd     *cobra.Command
	email   string
	name    string
	baseURL string
}

func newSandboxCmd() *sandboxCmd {
	sc := &sandboxCmd{}
	sc.cmd = &cobra.Command{
		Use:   "sandbox",
		Short: "Manage Stripe sandbox environments",
		Args:  validators.NoArgs,
		Annotations: map[string]string{
			AIAgentHelpAnnotationKey: "  Use `stripe sandbox create --email auto` to provision a sandbox without authentication.\n" +
				"  Returns test API keys you can use immediately. Claim the sandbox later to keep it.",
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

This command does not require authentication. It uses a proof-of-work challenge
to provision a temporary sandbox environment with working test keys.

Use --email auto to infer your email from git config.`,
		Example: `  stripe sandbox create --email auto
  stripe sandbox create --email you@example.com --name "My Sandbox"`,
		Args: validators.NoArgs,
		RunE: scc.runSandboxCreateCmd,
	}

	scc.cmd.Flags().StringVar(&scc.email, "email", "", "Your email address (required). Use \"auto\" to infer from git config user.email")
	scc.cmd.Flags().StringVar(&scc.name, "name", "", "Display name for the sandbox (optional). Use \"auto\" to infer from git config user.name")

	defaultBase := resolveSandboxBaseURL()
	scc.cmd.Flags().StringVar(&scc.baseURL, "base-url", defaultBase, "Sandbox API base URL")
	scc.cmd.Flags().MarkHidden("base-url") // nolint:errcheck

	return scc
}

func (scc *sandboxCreateCmd) runSandboxCreateCmd(cmd *cobra.Command, args []string) error {
	color := ansi.Color(cmd.ErrOrStderr())

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

	client := sandbox.NewClient(scc.baseURL)

	fmt.Fprintln(cmd.ErrOrStderr(), color.Yellow("Getting challenge..."))

	challengeResp, err := client.GetChallenge(cmd.Context(), email)
	if err != nil {
		return err
	}

	fmt.Fprintln(cmd.ErrOrStderr(), color.Yellow("Solving challenge..."))
	start := time.Now()

	solution, err := sandbox.SolveChallenge(cmd.Context(), challengeResp.Algorithm, challengeResp.Challenge, challengeResp.Salt)
	if err != nil {
		return err
	}

	elapsed := time.Since(start)
	fmt.Fprintf(cmd.ErrOrStderr(), "Solved in %s\n", elapsed.Round(time.Millisecond))

	fmt.Fprintln(cmd.ErrOrStderr(), color.Yellow("Provisioning sandbox..."))

	provisionReq := sandbox.ProvisionRequest{
		Algorithm: challengeResp.Algorithm,
		Challenge: challengeResp.Challenge,
		Salt:      challengeResp.Salt,
		Signature: challengeResp.Signature,
		Number:    solution,
		Email:     email,
		Name:      name,
	}

	result, err := client.Provision(cmd.Context(), provisionReq)
	if err != nil {
		return err
	}

	if err := saveSandboxToConfig(result); err != nil {
		fmt.Fprintf(cmd.ErrOrStderr(), "Warning: could not save to config: %v\n", err)
	}

	out, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return err
	}
	fmt.Fprintln(cmd.OutOrStdout(), string(out))

	fmt.Fprintf(cmd.ErrOrStderr(), "\n%s\n", color.Green("Sandbox provisioned successfully!"))
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

func resolveSandboxBaseURL() string {
	if v := os.Getenv("STRIPE_SANDBOX_BASE_URL"); v != "" {
		return v
	}
	if v := os.Getenv("DEVAI_BASE_URL"); v != "" {
		return v
	}
	return defaultSandboxBaseURL
}
