package cmd

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"time"

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
		Example: `  stripe sandbox create --from-git
  stripe sandbox create --email you@example.com`,
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

	// If already logged in, just open the sandboxes page
	existingKey, _ := Config.Profile.GetAPIKey(false)
	if existingKey != "" {
		sandboxURL := scc.dashboardURL + "/test/sandboxes"
		fmt.Fprintf(cmd.ErrOrStderr(), "Already logged in. Opening sandbox management page...\n")
		fmt.Fprintf(cmd.ErrOrStderr(), "  %s\n", sandboxURL)
		if canOpenBrowserFunc() {
			openBrowserFunc(sandboxURL)
		}
		return nil
	}

	// Resolve email
	var email string
	switch {
	case scc.fromGit:
		gitEmail := sandbox.GitConfigFunc("user.email")
		if gitEmail == "" {
			return fmt.Errorf("--from-git requires git config user.email to be set, but it was not found")
		}
		email = gitEmail
		fmt.Fprintf(cmd.ErrOrStderr(), "Using email: %s (from git config)\n", email)
	case scc.email != "":
		email = scc.email
	default:
		return fmt.Errorf("email is required, pass --email your@email.com or use --from-git to infer from git config user.email")
	}

	var name string
	if scc.name != "" {
		name = scc.name
	}

	result, err := scc.runProvisionFlow(cmd, color, email, name)
	if err != nil {
		fmt.Fprintf(cmd.ErrOrStderr(), "\nProvisioning failed: %v\n", err)
		fmt.Fprintln(cmd.ErrOrStderr(), color.Yellow("Falling back to browser login..."))
		return scc.runDashboardFlow(cmd, color, email)
	}

	return scc.outputResult(cmd, color, result)
}

func (scc *sandboxCreateCmd) runProvisionFlow(cmd *cobra.Command, color aurora.Aurora, email, name string) (*sandbox.ProvisionResponse, error) {
	client := sandbox.NewClient(scc.baseURL)

	challengeResp, err := client.GetChallenge(cmd.Context(), email)
	if err != nil {
		return nil, err
	}

	fmt.Fprintln(cmd.ErrOrStderr(), color.Yellow("Solving proof-of-work..."))
	start := time.Now()

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

	// Build a signup URL with email pre-filled and redirect to CLI auth confirmation
	confirmPath := "/stripecli/confirm_auth"
	if parsed, err := url.Parse(links.BrowserURL); err == nil {
		confirmPath = parsed.RequestURI()
	}
	params := url.Values{}
	params.Set("redirect", confirmPath)
	if email != "" {
		params.Set("email", email)
	}
	signupURL := fmt.Sprintf("%s/register?%s", scc.dashboardURL, params.Encode())

	fmt.Fprintf(cmd.ErrOrStderr(), "\nOpening browser to create or log in to your Stripe account...\n")
	fmt.Fprintf(cmd.ErrOrStderr(), "  1. Sign up or log in at the page that opens\n")
	fmt.Fprintf(cmd.ErrOrStderr(), "  2. Confirm the pairing code: %s\n", color.Bold(links.VerificationCode))
	fmt.Fprintf(cmd.ErrOrStderr(), "  3. Return here — your keys will appear automatically\n\n")

	if canOpenBrowserFunc() {
		if err := openBrowserFunc(signupURL); err != nil {
			fmt.Fprintf(cmd.ErrOrStderr(), "Could not open browser: %v\n", err)
		}
	}
	fmt.Fprintf(cmd.ErrOrStderr(), "If the browser doesn't open, visit:\n  %s\n\n", signupURL)

	fmt.Fprintln(cmd.ErrOrStderr(), color.Yellow("Waiting for confirmation..."))

	response, _, err := keys.PollForKey(cmd.Context(), links.PollURL, 2*time.Second, 20*60/2)
	if err != nil {
		return err
	}

	// Save to current profile (same as stripe login)
	configurer := keys.NewRAKConfigurer(&Config, afero.NewOsFs())
	if err := configurer.SaveLoginDetails(response); err != nil {
		fmt.Fprintf(cmd.ErrOrStderr(), "Warning: could not save to config: %v\n", err)
	}

	displayName := response.AccountDisplayName
	if displayName == "" {
		displayName = response.AccountID
	}
	fmt.Fprintf(cmd.ErrOrStderr(), "\n%s\n", color.Green(fmt.Sprintf("Connected to %q!", displayName)))
	fmt.Fprintf(cmd.ErrOrStderr(), "Your test API key is now configured. Run `stripe` commands to get started.\n")

	return nil
}

func (scc *sandboxCreateCmd) outputResult(cmd *cobra.Command, color aurora.Aurora, result *sandbox.ProvisionResponse) error {
	if err := saveSandboxToConfig(result); err != nil {
		return err
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
	expiry := result.ExpiresAt
	if expiry == "" {
		expiry = time.Now().AddDate(0, 0, 7).Format("2006-01-02")
	}
	fmt.Fprintf(cmd.ErrOrStderr(), "%s This sandbox expires on %s. Claim it before then to keep it.\n", color.Yellow("⚠"), expiry)

	return nil
}

func saveSandboxToConfig(result *sandbox.ProvisionResponse) error {
	secretKey := result.GetSecretKey()
	if secretKey == "" {
		return fmt.Errorf("no secret key in server response")
	}

	// Save to current profile (overwrite test mode keys)
	Config.Profile.TestModeAPIKey = secretKey
	Config.Profile.TestModePublishableKey = result.PublishableKey
	if result.AccountID != "" {
		Config.Profile.AccountID = result.AccountID
		Config.Profile.DisplayName = result.AccountID
	}
	if err := Config.Profile.CreateProfile(); err != nil {
		return err
	}

	// Write additional sandbox metadata (non-fatal if these fail)
	if result.ClaimURL != "" {
		Config.Profile.WriteConfigField("sandbox_claim_url", result.ClaimURL)
	}
	sandboxExpiry := result.ExpiresAt
	if sandboxExpiry == "" {
		sandboxExpiry = time.Now().AddDate(0, 0, 7).Format("2006-01-02")
	}
	Config.Profile.WriteConfigField("sandbox_expires_at", sandboxExpiry)

	return nil
}

func isSSHSession() bool {
	return os.Getenv("SSH_TTY") != "" || os.Getenv("SSH_CONNECTION") != "" || os.Getenv("SSH_CLIENT") != ""
}
