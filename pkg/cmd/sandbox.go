package cmd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/mail"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/logrusorgru/aurora"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

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

	existingKey, _ := Config.Profile.GetAPIKey(false)

	switch {
	case existingKey == "":
		// No key — proceed to provision a new sandbox below.

	case !isClaimableSandbox():
		// Logged in with a real key (sk_test_, rk_test_ from stripe login).
		// Direct to dashboard — sandbox creation requires an empty profile.
		sandboxURL := scc.dashboardURL + "/sandboxes"
		fmt.Fprintf(cmd.ErrOrStderr(), "You're already authenticated; sandbox management is available in Dashboard.\n\n")
		if canOpenBrowserFunc() {
			fmt.Fprintf(cmd.ErrOrStderr(), "Press Enter to open the browser or visit %s", sandboxURL)
			buf := make([]byte, 1)
			cmd.InOrStdin().Read(buf)
			openBrowserFunc(sandboxURL)
		} else {
			fmt.Fprintf(cmd.ErrOrStderr(), "Visit %s\n", sandboxURL)
		}
		return nil

	case isExpiredSandbox():
		// Claimable sandbox has expired. Clear the stale config so the user
		// can provision a fresh one or login with a claimed account.
		clearExpiredSandboxProfile()
		fmt.Fprintf(cmd.ErrOrStderr(), "Your sandbox session has expired.\nRun `stripe login` to continue with a claimed sandbox, or run `stripe sandbox create` again to create a new one.\n")
		return nil

	default:
		// Active claimable sandbox that hasn't expired. Show existing keys
		// and claim URL — one sandbox at a time.
		pubKey, _ := Config.Profile.GetPublishableKey(false)
		accountID, _ := Config.Profile.GetAccountID()
		fmt.Fprintf(cmd.ErrOrStderr(), "You already have an active sandbox.\n\n")
		fmt.Fprintf(cmd.ErrOrStderr(), "Secret key:      %s\n", existingKey)
		if pubKey != "" {
			fmt.Fprintf(cmd.ErrOrStderr(), "Publishable key: %s\n", pubKey)
		}
		if accountID != "" {
			fmt.Fprintf(cmd.ErrOrStderr(), "Account ID:      %s\n", accountID)
		}
		claimURL := viper.GetString(Config.Profile.GetConfigField("sandbox_claim_url"))
		if claimURL != "" {
			fmt.Fprintf(cmd.ErrOrStderr(), "\nWhen you're ready, claim your sandbox at:\n  %s\n", claimURL)
		}
		expiresAt := viper.GetString(Config.Profile.GetConfigField("sandbox_expires_at"))
		if expiresAt != "" {
			fmt.Fprintf(cmd.ErrOrStderr(), "Expires: %s\n", expiresAt)
		}
		return nil
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
		log.WithFields(log.Fields{"error": err}).Debug("sandbox: provisioning failed, falling back to browser")
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

	var email string
	switch {
	case scc.fromGit:
		gitEmail := sandbox.GitConfigFunc("user.email")
		if gitEmail == "" {
			return "", fmt.Errorf("--from-git requires git config user.email to be set, but it was not found")
		}
		fmt.Fprintf(cmd.ErrOrStderr(), "Using email: %s (from git config)\n", gitEmail)
		email = gitEmail
	case scc.email != "":
		email = scc.email
	default:
		return "", fmt.Errorf("email is required, pass --email your@email.com or use --from-git to infer from git config user.email")
	}

	if _, err := mail.ParseAddress(email); err != nil {
		return "", fmt.Errorf("invalid email %q: %w", email, err)
	}
	return email, nil
}

func (scc *sandboxCreateCmd) runProvisionFlow(cmd *cobra.Command, color aurora.Aurora, email, name string) (*sandbox.ProvisionResponse, error) {
	client := sandbox.NewClient(scc.baseURL)

	challengeResp, err := client.GetChallenge(cmd.Context(), email)
	if err != nil {
		return nil, err
	}

	fmt.Fprint(cmd.ErrOrStderr(), "Setting up your sandbox...")
	solution, err := sandbox.SolveChallenge(cmd.Context(), challengeResp.Algorithm, challengeResp.Challenge, challengeResp.Salt)
	if err != nil {
		fmt.Fprintln(cmd.ErrOrStderr())
		return nil, err
	}
	fmt.Fprintln(cmd.ErrOrStderr(), " done.")

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
	return authenticator.Login(cmd.Context(), links)
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
		ExpiresAt      string `json:"expires_at,omitempty"`
	}{
		SecretKey:      result.GetSecretKey(),
		PublishableKey: result.GetPublishableKey(),
		ClaimURL:       result.GetClaimURL(),
		AccountID:      result.GetAccountID(),
		ExpiresAt:      result.GetExpiresAt(),
	}
	out, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		return err
	}
	fmt.Fprintln(cmd.OutOrStdout(), string(out))

	fmt.Fprintf(cmd.ErrOrStderr(), "\n%s Use the keys above to start building your integration.\n", color.Green("Provisioned!"))
	if result.GetClaimURL() != "" {
		expiresAt := result.GetExpiresAt()
		if expiresAt != "" {
			fmt.Fprintf(cmd.ErrOrStderr(), "\nWhen you're ready, claim your sandbox at:\n  %s\n", result.GetClaimURL())
			fmt.Fprintf(cmd.ErrOrStderr(), "Expires: %s\n", expiresAt)
		} else {
			fmt.Fprintf(cmd.ErrOrStderr(), "\nWhen you're ready, claim your sandbox at:\n  %s\n", result.GetClaimURL())
			fmt.Fprintf(cmd.ErrOrStderr(), "Expires in 7 days.\n")
		}
	}

	return nil
}

// saveSandboxToConfig backs up the current profile and writes the new
// sandbox keys. Uses the same CopyProfile pattern as SaveLoginDetails:
// the current profile is copied to a backup named by its DisplayName
// (the account ID), then the active profile is overwritten. Repeated
// runs with the same sandbox produce the same backup name (no churn).
// A new sandbox overwrites the backup with the new account ID.
func saveSandboxToConfig(result *sandbox.ProvisionResponse) error {
	secretKey := result.GetSecretKey()
	if secretKey == "" {
		return fmt.Errorf("no secret key in server response")
	}

	accountID := result.GetAccountID()

	// Back up current profile before overwriting. Uses DisplayName as the
	// backup profile name — same behavior as SaveLoginDetails (line 38 of
	// configurer.go). Idempotent: same display name = same backup target.
	Config.CopyProfile(Config.Profile.ProfileName, Config.Profile.GetDisplayName())

	Config.Profile.TestModeAPIKey = secretKey
	Config.Profile.TestModePublishableKey = result.GetPublishableKey()
	if accountID != "" {
		Config.Profile.AccountID = accountID
		Config.Profile.DisplayName = accountID
	}
	if err := Config.Profile.CreateProfile(); err != nil {
		return err
	}

	// Write sandbox-specific metadata (non-fatal if these fail)
	if result.GetClaimURL() != "" {
		Config.Profile.WriteConfigField("sandbox_claim_url", result.GetClaimURL())
	}
	if expiresAt := result.GetExpiresAt(); expiresAt != "" {
		Config.Profile.WriteConfigField("sandbox_expires_at", expiresAt)
	}

	return nil
}

// isClaimableSandbox returns true if the current profile looks like a
// CLI-created claimable sandbox (not a real account from stripe login).
// Key prefix is authoritative — if the key is sk_test_ or rk_test_,
// it's a real account regardless of leftover sandbox metadata.
func isClaimableSandbox() bool {
	key, _ := Config.Profile.GetAPIKey(false)
	if key != "" && !strings.HasPrefix(key, "rkcs_") {
		return false
	}
	if strings.HasPrefix(key, "rkcs_") {
		return true
	}
	// No key — check metadata for partially-cleared sandbox state
	if viper.GetString(Config.Profile.GetConfigField("sandbox_claim_url")) != "" {
		return true
	}
	return viper.GetString(Config.Profile.GetConfigField("sandbox_expires_at")) != ""
}

// isExpiredSandbox returns true if the sandbox_expires_at date has passed.
func isExpiredSandbox() bool {
	expiresAt := viper.GetString(Config.Profile.GetConfigField("sandbox_expires_at"))
	if expiresAt == "" {
		return false
	}
	// Try date-only first, then RFC3339 (older sandboxes may have full timestamps)
	expiry, err := time.Parse("2006-01-02", expiresAt)
	if err != nil {
		expiry, err = time.Parse(time.RFC3339, expiresAt)
		if err != nil {
			return false
		}
	}
	return time.Now().UTC().After(expiry)
}

// clearExpiredSandboxProfile removes sandbox-specific fields from the current
// profile without affecting other profiles. Narrowly scoped — only clears
// fields that sandbox create wrote.
func clearExpiredSandboxProfile() {
	Config.Profile.DeleteConfigField("test_mode_api_key")
	Config.Profile.DeleteConfigField("test_mode_pub_key")
	Config.Profile.DeleteConfigField("sandbox_claim_url")
	Config.Profile.DeleteConfigField("sandbox_expires_at")
	Config.Profile.DeleteConfigField("account_id")
	Config.Profile.DeleteConfigField("display_name")
}

func isSSHSession() bool {
	return os.Getenv("SSH_TTY") != "" || os.Getenv("SSH_CONNECTION") != "" || os.Getenv("SSH_CLIENT") != ""
}
