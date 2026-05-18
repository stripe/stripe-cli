package cmd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/logrusorgru/aurora"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"

	"github.com/stripe/stripe-cli/pkg/ansi"
	"github.com/stripe/stripe-cli/pkg/login"
	"github.com/stripe/stripe-cli/pkg/login/keys"
	"github.com/stripe/stripe-cli/pkg/open"
	"github.com/stripe/stripe-cli/pkg/sandbox"
	"github.com/stripe/stripe-cli/pkg/stripe"
	"github.com/stripe/stripe-cli/pkg/validators"
)

const defaultSandboxBaseURL = "https://ai.stripe.com"

var openBrowserFunc = open.Browser
var canOpenBrowserFunc = open.CanOpenBrowser

type sandboxCmd struct {
	cmd *cobra.Command
}

type sandboxCreateCmd struct {
	cmd          *cobra.Command
	email        string
	fromGit      bool
	name         string
	baseURL      string
	dashboardURL string
}

func newSandboxCmd() *sandboxCmd {
	sc := &sandboxCmd{}
	sc.cmd = &cobra.Command{
		Use:    "sandbox",
		Short:  "Manage Stripe sandbox environments",
		Hidden: true,
		Args:   validators.NoArgs,
		Annotations: map[string]string{
			AIAgentHelpAnnotationKey: "  Use `stripe sandbox create --from-git` to provision a sandbox using your git email.\n" +
				"  Use `stripe sandbox create --email you@example.com` to provision with an explicit email.\n" +
				"  If provisioning fails, falls back to browser login (like stripe login).\n" +
				"  If already logged in, opens the sandbox management page.",
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

If you are already logged in (have a configured API key), opens the
sandbox management page in your browser instead.

Otherwise, uses a proof-of-work challenge to provision a temporary sandbox
without authentication. If that fails, automatically falls back to
browser-based signup/login.

Keys are saved to the current CLI profile so subsequent stripe commands
work immediately.`,
		Example: `stripe sandbox create --email you@example.com
  stripe sandbox create --from-git`,
		Args: validators.NoArgs,
		Annotations: map[string]string{
			AIAgentHelpAnnotationKey: "  Provisions a sandbox and saves keys to the current CLI profile.\n" +
				"  Pass --from-git to resolve your email from git config user.email.\n" +
				"  Pass --email to provide an explicit email address.\n" +
				"  Falls back to browser login on server errors.",
		},
		RunE: scc.runSandboxCreateCmd,
	}

	scc.cmd.Flags().StringVar(&scc.email, "email", "", "Your email address")
	scc.cmd.Flags().BoolVar(&scc.fromGit, "from-git", false, "Infer email from git config user.email")
	scc.cmd.Flags().StringVar(&scc.name, "name", "", "Your full name (optional)")

	scc.cmd.Flags().StringVar(&scc.baseURL, "base-url", defaultSandboxBaseURL, "Sets the sandbox API base URL")
	_ = scc.cmd.Flags().MarkHidden("base-url")

	scc.cmd.Flags().StringVar(&scc.dashboardURL, "dashboard-base", stripe.DefaultDashboardBaseURL, "Sets the dashboard base URL")
	_ = scc.cmd.Flags().MarkHidden("dashboard-base")

	return scc
}

func (scc *sandboxCreateCmd) runSandboxCreateCmd(cmd *cobra.Command, args []string) error {
	color := ansi.Color(cmd.ErrOrStderr())

	// If logged in with a real account key (sk_test_, rk_test_ from stripe login),
	// redirect to dashboard.
	existingKey, _ := Config.Profile.GetAPIKey(false)
	if existingKey != "" && !strings.HasPrefix(existingKey, "rkcs_") {
		sandboxURL := scc.dashboardURL + "/sandboxes"
		fmt.Fprintf(cmd.ErrOrStderr(), "Already logged in.\n\n")
		fmt.Fprintf(cmd.ErrOrStderr(), "Press Enter to open the browser or visit %s", sandboxURL)
		fmt.Scanln()
		if canOpenBrowserFunc() {
			openBrowserFunc(sandboxURL)
		}
		return nil
	}

	// If an existing sandbox key exists, copy the current profile to a named
	// backup (by account ID) before overwriting, so old sandboxes remain
	// accessible via --project-name.
	if existingKey != "" && strings.HasPrefix(existingKey, "rkcs_") {
		existingAccountID, _ := Config.Profile.GetAccountID()
		if existingAccountID != "" {
			Config.CopyProfile(Config.Profile.ProfileName, existingAccountID)
		}
	}

	// Resolve email — --email and --from-git are mutually exclusive.
	email, err := scc.resolveEmail(cmd)
	if err != nil {
		return err
	}

	var name string
	if scc.name != "" {
		name = scc.name
	} else if scc.fromGit {
		name = sandbox.GitConfigFunc("user.name")
	}

	// Primary path: proof-of-work provisioning against developer-ai-srv.
	// This gives the user a temporary sandbox without any browser interaction.
	result, err := scc.runProvisionFlow(cmd, color, email, name)
	if err != nil {
		// Don't fallback if the user canceled (Ctrl+C) or the context expired.
		// Only fallback on server/network errors — the user intentionally interrupted.
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return err
		}

		// Fallback: open browser for signup/login. This uses the existing
		// stripe login infrastructure (POST /stripecli/auth + polling).
		// Any server-side failure (429, 500, network) triggers this path
		// so the user always has a way to get keys.
		fmt.Fprintln(cmd.ErrOrStderr(), color.Yellow("\nOpening browser to set up your account..."))
		return scc.runDashboardFlow(cmd, color, email)
	}

	return scc.outputResult(cmd, color, result)
}

// resolveEmail determines the email from flags. --email and --from-git are
// mutually exclusive; providing both is an error.
func (scc *sandboxCreateCmd) resolveEmail(cmd *cobra.Command) (string, error) {
	if scc.fromGit && scc.email != "" {
		return "", fmt.Errorf("--email and --from-git are mutually exclusive")
	}

	switch {
	case scc.fromGit:
		gitEmail := sandbox.GitConfigFunc("user.email")
		if gitEmail == "" {
			return "", fmt.Errorf("--from-git requires git config user.email to be set, but it was not found")
		}
		fmt.Fprintf(cmd.ErrOrStderr(), "Using email: %s (from git config)\n", gitEmail)
		return gitEmail, nil
	case scc.email != "":
		return scc.email, nil
	default:
		return "", fmt.Errorf("email is required, pass --email your@email.com or use --from-git to infer from git config user.email")
	}
}

func (scc *sandboxCreateCmd) runProvisionFlow(cmd *cobra.Command, color aurora.Aurora, email, name string) (*sandbox.ProvisionResponse, error) {
	client := sandbox.NewClient(scc.baseURL)

	challengeResp, err := client.GetChallenge(cmd.Context(), email)
	if err != nil {
		return nil, err
	}

	solution, err := sandbox.SolveChallenge(cmd.Context(), challengeResp.Algorithm, challengeResp.Challenge, challengeResp.Salt)
	if err != nil {
		return nil, err
	}

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

// runDashboardFlow is the browser-based fallback. It calls
// login.Authenticator.Login() with a modified BrowserURL that points to
// /register with the user's email pre-filled, so new users land on signup
// instead of a login gate. Everything else (spinners, polling, key saving,
// success messages) uses the standard stripe login flow.
func (scc *sandboxCreateCmd) runDashboardFlow(cmd *cobra.Command, color aurora.Aurora, email string) error {
	if isSSHSession() {
		fmt.Fprintln(cmd.ErrOrStderr(), "SSH session detected. Cannot open browser.")
		fmt.Fprintln(cmd.ErrOrStderr(), "Use `stripe login --interactive` or set STRIPE_API_KEY instead.")
		return fmt.Errorf("browser login unavailable in SSH session")
	}

	links, err := login.GetLinks(cmd.Context(), scc.dashboardURL, "stripe-sandbox")
	if err != nil {
		return err
	}

	// Override BrowserURL to point to /register with email pre-filled.
	// After registration, the redirect sends the user to the confirm_auth page.
	confirmPath := "/stripecli/confirm_auth"
	if parsed, err := url.Parse(links.BrowserURL); err == nil {
		confirmPath = parsed.RequestURI()
	}
	params := url.Values{}
	params.Set("redirect", confirmPath)
	if email != "" {
		params.Set("email", email)
	}
	links.BrowserURL = fmt.Sprintf("%s/register?%s", scc.dashboardURL, params.Encode())

	// Use the standard login flow — handles "Press Enter", spinners,
	// polling, key saving, and success messaging.
	configurer := keys.NewRAKConfigurer(&Config, afero.NewOsFs())
	transfer := keys.NewRAKTransfer(configurer)
	authenticator := login.NewAuthenticator(transfer)
	if err := authenticator.Login(cmd.Context(), links); err != nil {
		return err
	}

	// Clear stale sandbox fields — the user now has a real account.
	Config.Profile.DeleteConfigField("sandbox_claim_url")
	Config.Profile.DeleteConfigField("sandbox_expires_at")
	return nil
}

func (scc *sandboxCreateCmd) outputResult(cmd *cobra.Command, color aurora.Aurora, result *sandbox.ProvisionResponse) error {
	if err := saveSandboxToConfig(result); err != nil {
		return err
	}

	output := struct {
		SecretKey      string `json:"secret_key"`
		PublishableKey string `json:"publishable_key"`
		ClaimURL       string `json:"claim_url,omitempty"`
		AccountID      string `json:"account_id,omitempty"`
	}{
		SecretKey:      result.GetSecretKey(),
		PublishableKey: result.PublishableKey,
		ClaimURL:       result.ClaimURL,
		AccountID:      result.MerchantToken,
	}
	out, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		return err
	}
	fmt.Fprintln(cmd.OutOrStdout(), string(out))

	fmt.Fprintf(cmd.ErrOrStderr(), "\n%s Use the keys above to start building your integration.\n", color.Green("Provisioned!"))
	if result.ClaimURL != "" {
		fmt.Fprintf(cmd.ErrOrStderr(), "\nWhen you're ready, claim your sandbox at:\n  %s\n", result.ClaimURL)
		fmt.Fprintf(cmd.ErrOrStderr(), "Expires in 7 days.\n")
	}

	return nil
}

// saveSandboxToConfig writes the provisioned sandbox keys to the current
// profile. Only called when no keys exist (the already-configured check
// ensures we never overwrite).
func saveSandboxToConfig(result *sandbox.ProvisionResponse) error {
	secretKey := result.GetSecretKey()
	if secretKey == "" {
		return fmt.Errorf("no secret key in server response")
	}

	accountID := result.AccountID
	if accountID == "" {
		accountID = result.MerchantToken
	}

	Config.Profile.TestModeAPIKey = secretKey
	Config.Profile.TestModePublishableKey = result.PublishableKey
	if accountID != "" {
		Config.Profile.AccountID = accountID
		Config.Profile.DisplayName = accountID
	}
	if err := Config.Profile.CreateProfile(); err != nil {
		return err
	}

	// Write sandbox-specific metadata (non-fatal if these fail)
	if result.ClaimURL != "" {
		Config.Profile.WriteConfigField("sandbox_claim_url", result.ClaimURL)
	}
	if result.ExpiresAt != "" {
		Config.Profile.WriteConfigField("sandbox_expires_at", result.ExpiresAt)
	}

	return nil
}

func isSSHSession() bool {
	return os.Getenv("SSH_TTY") != "" || os.Getenv("SSH_CONNECTION") != "" || os.Getenv("SSH_CLIENT") != ""
}
