package cmd

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/mail"
	"net/url"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/logrusorgru/aurora"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/stripe/stripe-cli/pkg/ansi"
	"github.com/stripe/stripe-cli/pkg/config"
	"github.com/stripe/stripe-cli/pkg/errorcategory"
	"github.com/stripe/stripe-cli/pkg/login"
	"github.com/stripe/stripe-cli/pkg/open"
	"github.com/stripe/stripe-cli/pkg/sandbox"
	"github.com/stripe/stripe-cli/pkg/stripe"
	"github.com/stripe/stripe-cli/pkg/validators"
)

const (
	defaultSandboxBaseURL        = "https://ai.stripe.com"
	sandboxAlreadyClaimedMessage = "This sandbox has already been claimed. Run `stripe login` to authenticate with your claimed account."
	sandboxExpiredMessage        = "Your sandbox session has expired.\nRun `stripe login` to continue with a claimed sandbox, or run `stripe sandbox create` again to create a new one."
	// RetrieveClaimableSandboxStatus shipped in the 2026-08-26 snapshot.
	sandboxClaimStatusVersion = "2026-08-26.preview"
)

var openBrowserFunc = open.Browser
var canOpenBrowserFunc = open.CanOpenBrowser

type sandboxCmd struct {
	cmd *cobra.Command
}

type sandboxCreateCmd struct {
	cmd            *cobra.Command
	email          string
	fromGit        bool
	name           string
	nonInteractive bool
	createBlank    bool
	country        string
	baseURL        string
	apiBaseURL     string
	dashboardURL   string
	client         sandboxCreateClient
	reauth         func(context.Context, string, string) error
	isInteractive  func(*cobra.Command) bool
}

func newSandboxCmd() *sandboxCmd {
	sc := &sandboxCmd{}
	sc.cmd = &cobra.Command{
		Use:   "sandbox",
		Short: "Manage Stripe sandbox environments",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				return errorcategory.Errorf(errorcategory.UserInput, "unknown command %q for %q", args[0], cmd.CommandPath())
			}
			return cmd.Help()
		},
		Annotations: map[string]string{
			AIAgentHelpAnnotationKey: "  For new integrations, use separate general sandboxes to isolate settings and test data from live mode. Use the test mode sandbox only for existing integrations or features that require it.\n" +
				"  Use separate sandboxes for local development and continuous integration (CI) as this avoids undesired interaction between your test environments.\n" +
				"  Reuse sandboxes across test runs.\n" +
				"  With an active live OAuth account, use `stripe sandbox create \"My sandbox\"`.\n" +
				"  Use `stripe sandbox create --from-git` to provision a sandbox using your git email.\n" +
				"  Use `stripe sandbox create --email [you@example.com](mailto:you@example.com)` to provision with an explicit email.\n" +
				"  If anonymous provisioning fails, falls back to browser login (like stripe login).",
		},
	}

	createCmd := newSandboxCreateCmd()
	claimCmd := newSandboxClaimCmd()
	sc.cmd.AddCommand(createCmd.cmd)
	sc.cmd.AddCommand(claimCmd.cmd)
	sc.cmd.AddCommand(newSandboxListCmd().cmd)
	sc.cmd.AddCommand(newSandboxDeleteCmd().cmd)
	return sc
}

func newSandboxCreateCmd() *sandboxCreateCmd {
	scc := &sandboxCreateCmd{
		reauth:        login.ReauthImmediately,
		isInteractive: sandboxCommandIsInteractive,
	}
	scc.cmd = &cobra.Command{
		Use:   "create [name]",
		Short: "Provision a new sandbox environment",
		Long: `Create a new Stripe sandbox.

With an active live OAuth account, provide a name to create a sandbox under
that account. By default, settings and data are copied from the live account;
pass --create-blank and --country to create a blank sandbox instead.

Without OAuth, use --email or --from-git to provision a temporary claimable
sandbox with test API keys. If that fails, the command falls back to
browser-based signup or login.

For a claimable sandbox, keys are saved to the current CLI profile so
subsequent stripe commands work immediately.`,
		Example: `stripe sandbox create "My sandbox"
  stripe sandbox create "My blank sandbox" --create-blank --country US
  stripe sandbox create --email you@example.com
  stripe sandbox create --from-git`,
		Args: validators.MaximumNArgs(1),
		Annotations: map[string]string{
			AIAgentHelpAnnotationKey: "  With an active live OAuth account, pass a name to create a managed sandbox.\n" +
				"  Without OAuth, provisions a claimable sandbox and saves keys to the current CLI profile.\n" +
				"  Pass --from-git to resolve your email from git config user.email.\n" +
				"  Pass --email to provide an explicit email address.\n" +
				"  Falls back to browser login on server errors.",
		},
		RunE: scc.runSandboxCreateCmd,
	}

	scc.cmd.Flags().StringVar(&scc.email, "email", "", "Your email address")
	scc.cmd.Flags().BoolVar(&scc.fromGit, "from-git", false, "Infer email and full name from git config")
	scc.cmd.Flags().StringVar(&scc.name, "full-name", "", "Your full name (optional)")
	scc.cmd.Flags().BoolVar(&scc.nonInteractive, "non-interactive", false, "Print output directly without waiting for input")
	scc.cmd.Flags().BoolVar(&scc.createBlank, "create-blank", false, "Create a blank sandbox instead of copying the active live account")
	scc.cmd.Flags().StringVar(&scc.country, "country", "", "Two-letter country code for a blank sandbox")

	scc.cmd.Flags().StringVar(&scc.baseURL, "base-url", defaultSandboxBaseURL, "Sets the sandbox API base URL")
	_ = scc.cmd.Flags().MarkHidden("base-url")

	scc.cmd.Flags().StringVar(&scc.apiBaseURL, "api-base", stripe.DefaultAPIBaseURL, "Sets the API base URL")
	_ = scc.cmd.Flags().MarkHidden("api-base")

	scc.cmd.Flags().StringVar(&scc.dashboardURL, "dashboard-base", stripe.DefaultDashboardBaseURL, "Sets the dashboard base URL")
	_ = scc.cmd.Flags().MarkHidden("dashboard-base")

	return scc
}

func (scc *sandboxCreateCmd) runSandboxCreateCmd(cmd *cobra.Command, args []string) error {
	if err := login.ValidateAccessBaseURL(rootAccessBaseURL); err != nil {
		return err
	}

	if !Config.Profile.HasOverrideAPIKey() {
		uat, err := Config.Profile.GetUAT()
		if err != nil {
			return err
		}
		if strings.HasPrefix(uat, "oak_") {
			return scc.runAuthenticatedSandboxCreateCmd(cmd, args)
		}
	}

	if len(args) > 0 {
		return errorcategory.New(errorcategory.UserInput, "sandbox name is only valid with an active live OAuth account; run `stripe login` first")
	}
	for _, flagName := range []string{"create-blank", "country"} {
		if cmd.Flags().Changed(flagName) {
			return errorcategory.Errorf(errorcategory.UserInput, "--%s is only valid with an active live OAuth account; run `stripe login` first", flagName)
		}
	}

	return scc.runAnonymousSandboxCreateCmd(cmd)
}

func (scc *sandboxCreateCmd) runAnonymousSandboxCreateCmd(cmd *cobra.Command) error {
	// Reject an invalid profile name before provisioning, so we never create a
	// sandbox whose keys cannot be saved.
	if err := Config.Profile.ValidateProfileNameForWrite(); err != nil {
		return err
	}

	color := ansi.Color(cmd.ErrOrStderr())

	existingKey, _ := Config.Profile.GetAPIKey(false)

	switch {
	case existingKey == "":
		// No key — proceed to provision a new sandbox below.

	case !isClaimableSandbox():
		// Logged in with a real key (sk_test_, rk_test_ from stripe login).
		// Direct to dashboard — sandbox creation requires an empty profile.
		sandboxURL := scc.dashboardURL + "/sandboxes"
		fmt.Printf("You're already authenticated; sandbox management is available in Dashboard.\n\n")
		switch {
		case scc.nonInteractive:
			fmt.Printf("%s\n", sandboxURL)
		case canOpenBrowserFunc():
			fmt.Printf("Press Enter to open the browser or visit %s", sandboxURL)
			buf := make([]byte, 1)
			os.Stdin.Read(buf)
			openBrowserFunc(sandboxURL)
		default:
			fmt.Printf("Visit %s\n", sandboxURL)
		}
		return nil

	case isExpiredSandbox():
		// Claimable sandbox has expired. Clear the stale config so the user
		// can provision a fresh one or login with a claimed account.
		clearExpiredSandboxProfile()
		fmt.Printf("%s\n", sandboxExpiredMessage)
		return nil

	default:
		// Active claimable sandbox that hasn't expired. Show existing keys
		// and, if still unclaimed, the claim guidance — one sandbox at a time.
		pubKey, _ := Config.Profile.GetPublishableKey(false)
		accountID, _ := Config.Profile.GetAccountID()
		fmt.Printf("You already have an active sandbox.\n\n")
		fmt.Printf("Secret key:      %s\n", existingKey)
		if pubKey != "" {
			fmt.Printf("Publishable key: %s\n", pubKey)
		}
		if accountID != "" {
			fmt.Printf("Account ID:      %s\n", accountID)
		}
		if sandboxClaimed(cmd.Context(), scc.apiBaseURL) {
			fmt.Printf("\n%s\n", sandboxAlreadyClaimedMessage)
			return nil
		}

		expiresAt := Config.Profile.ReadProfileString(config.SandboxExpiresAtName)
		if expiresAt != "" {
			fmt.Printf("\nThis sandbox expires %s (in 7 days). Claim it before then by running `stripe sandbox claim`.\n", expiresAt)
		} else {
			fmt.Printf("\nRun `stripe sandbox claim` when you're ready to claim your sandbox.\n")
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

	// Primary path: proof-of-work provisioning against ai.stripe.com.
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
		fmt.Println(color.Yellow("\nCould not provision a sandbox automatically. Opening browser to create your Stripe sandbox account instead..."))
		return scc.runDashboardFlow(cmd, color, email)
	}

	return scc.outputResult(cmd, color, result)
}

// resolveEmail determines the email from flags. --email and --from-git are
// mutually exclusive; providing both is an error.
func (scc *sandboxCreateCmd) resolveEmail(cmd *cobra.Command) (string, error) {
	if scc.fromGit && scc.email != "" {
		return "", errorcategory.Errorf(errorcategory.UserInput, "--email and --from-git are mutually exclusive")
	}

	var email string
	switch {
	case scc.fromGit:
		gitEmail := sandbox.GitConfigFunc("user.email")
		if gitEmail == "" {
			return "", errorcategory.Errorf(errorcategory.UserInput, "--from-git requires git config user.email to be set, but it was not found")
		}
		fmt.Printf("Using email: %s (from git config)\n", gitEmail)
		email = gitEmail
	case scc.email != "":
		email = scc.email
	default:
		return "", errorcategory.Errorf(errorcategory.UserInput, "email is required; provide it with --email or use --from-git to infer from git config user.email")
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

	fmt.Print("Setting up your sandbox...")
	solution, err := sandbox.SolveChallenge(cmd.Context(), challengeResp.Algorithm, challengeResp.Challenge, challengeResp.Salt)
	if err != nil {
		fmt.Println()
		return nil, err
	}
	fmt.Println(" done.")

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

// runDashboardFlow is the browser-based fallback. Uses the standard
// stripe login flow directly. Future enhancement: pass email to
// login.Login() to pre-fill and open /register instead of /confirm_auth.
func (scc *sandboxCreateCmd) runDashboardFlow(cmd *cobra.Command, color aurora.Aurora, email string) error {
	if isSSHSession() && !scc.nonInteractive {
		fmt.Println("SSH session detected. Cannot open browser.")
		fmt.Println("Use `stripe login --interactive` or set STRIPE_API_KEY instead.")
		return errorcategory.Errorf(errorcategory.UserInput, "browser login unavailable in SSH session")
	}

	if scc.nonInteractive {
		return login.InitiateLogin(cmd.Context(), scc.dashboardURL, rootAccessBaseURL, &Config)
	}
	return login.Login(cmd.Context(), scc.dashboardURL, rootAccessBaseURL, &Config)
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
	fmt.Println(string(out))

	fmt.Printf("\nUse the keys above to start building your integration.\n")
	if result.GetExpiresAt() != "" {
		fmt.Printf("\nThis sandbox expires %s (in 7 days). Claim it before then by using the above claim_url or running `stripe sandbox claim`.\n", result.GetExpiresAt())
	} else {
		fmt.Printf("\nClaim your sandbox by using the above claim_url or running `stripe sandbox claim`.\n")
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
		return errorcategory.Errorf(errorcategory.API, "no secret key in server response")
	}

	accountID := result.GetAccountID()

	// Back up current profile before overwriting. Uses DisplayName as the
	// backup profile name — same behavior as SaveLoginDetails (line 38 of
	// configurer.go). Idempotent: same display name = same backup target.
	Config.CopyProfile(Config.Profile.ProfileName, Config.Profile.GetDisplayName())

	Config.Profile.TestModeAPIKey = secretKey
	Config.Profile.TestModePublishableKey = result.GetPublishableKey()
	Config.Profile.SandboxClaimURL = result.GetClaimURL()
	Config.Profile.SandboxExpiresAt = result.GetExpiresAt()
	if accountID != "" {
		Config.Profile.AccountID = accountID
		Config.Profile.DisplayName = accountID
	}
	if err := Config.Profile.CreateProfile(); err != nil {
		return err
	}

	// Also write sandbox fields via WriteConfigField to update the global
	// viper instance (CreateProfile writes to a local viper copy).
	if result.GetClaimURL() != "" {
		Config.Profile.WriteConfigField(config.SandboxClaimURLName, result.GetClaimURL())
	}
	if result.GetExpiresAt() != "" {
		Config.Profile.WriteConfigField(config.SandboxExpiresAtName, result.GetExpiresAt())
	}
	Config.Profile.WriteConfigField(config.TestModeAPIKeyName, secretKey)
	if pubKey := result.GetPublishableKey(); pubKey != "" {
		Config.Profile.WriteConfigField(config.TestModePubKeyName, pubKey)
	}
	if accountID != "" {
		Config.Profile.WriteConfigField(config.AccountIDName, accountID)
		Config.Profile.WriteConfigField(config.DisplayNameName, accountID)
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
	if Config.Profile.ReadProfileString(config.SandboxClaimURLName) != "" {
		return true
	}
	return Config.Profile.ReadProfileString(config.SandboxExpiresAtName) != ""
}

func sandboxTestModeAPIKey() string {
	return Config.Profile.ReadProfileString(config.TestModeAPIKeyName)
}

// sandboxClaimed reports whether the profile's claimable sandbox has already been claimed. API failures return false.
func sandboxClaimed(ctx context.Context, apiBaseURL string) bool {
	apiKey := sandboxTestModeAPIKey()
	if apiKey == "" {
		return false
	}
	claimed, err := fetchSandboxClaimStatus(ctx, apiBaseURL, apiKey)
	if err != nil {
		log.WithFields(log.Fields{"error": err}).Debug("sandbox: claim status check failed, falling back to local claim behavior")
		return false
	}
	return claimed
}

// isExpiredSandbox returns true if the sandbox_expires_at date has passed.
func isExpiredSandbox() bool {
	expiresAt := Config.Profile.ReadProfileString(config.SandboxExpiresAtName)
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
	Config.Profile.DeleteConfigField(config.TestModeAPIKeyName)
	Config.Profile.DeleteConfigField(config.TestModePubKeyName)
	Config.Profile.DeleteConfigField(config.SandboxClaimURLName)
	Config.Profile.DeleteConfigField(config.SandboxExpiresAtName)
	Config.Profile.DeleteConfigField(config.AccountIDName)
	Config.Profile.DeleteConfigField(config.DisplayNameName)
}

type sandboxClaimCmd struct {
	cmd            *cobra.Command
	nonInteractive bool
	apiBaseURL     string
}

type sandboxCreateClient interface {
	Create(context.Context, sandbox.CreateOptions) (sandbox.CreatedSandbox, error)
}

type sandboxListClient interface {
	ListAccessible(context.Context) ([]sandbox.ManagedSandbox, error)
}

type sandboxDeleteClient interface {
	Delete(context.Context, string) (sandbox.DeletedSandbox, error)
}

type sandboxListCmd struct {
	cmd     *cobra.Command
	apiBase string
	client  sandboxListClient
}

type sandboxDeleteCmd struct {
	cmd     *cobra.Command
	confirm bool
	apiBase string
	client  sandboxDeleteClient
}

func newSandboxClaimCmd() *sandboxClaimCmd {
	scc := &sandboxClaimCmd{}
	scc.cmd = &cobra.Command{
		Use:   "claim",
		Short: "Claim your sandbox in the browser",
		Long:  "Opens the claim URL for your active sandbox. After claiming, run `stripe login` to get permanent keys.",
		Args:  validators.NoArgs,
		RunE:  scc.runSandboxClaimCmd,
	}
	scc.cmd.Flags().BoolVar(&scc.nonInteractive, "non-interactive", false, "Print output directly without waiting for input")
	scc.cmd.Flags().StringVar(&scc.apiBaseURL, "api-base", stripe.DefaultAPIBaseURL, "Sets the API base URL")
	_ = scc.cmd.Flags().MarkHidden("api-base")
	return scc
}

func (scc *sandboxClaimCmd) runSandboxClaimCmd(cmd *cobra.Command, args []string) error {
	claimURL := Config.Profile.ReadProfileString(config.SandboxClaimURLName)
	if claimURL == "" {
		fmt.Printf("No active sandbox. Run `stripe sandbox create` to get started.\n")
		return nil
	}

	if isExpiredSandbox() {
		clearExpiredSandboxProfile()
		fmt.Printf("%s\n", sandboxExpiredMessage)
		return nil
	}

	if sandboxClaimed(cmd.Context(), scc.apiBaseURL) {
		fmt.Printf("%s\n", sandboxAlreadyClaimedMessage)
		return nil
	}

	accountID, _ := Config.Profile.GetAccountID()

	if accountID != "" {
		fmt.Printf("Claim your sandbox (%s) by visiting the claim link below.\n", accountID)
	} else {
		fmt.Printf("Claim your sandbox by visiting the claim link below.\n")
	}
	fmt.Println()

	switch {
	case scc.nonInteractive:
		fmt.Printf("%s\n", claimURL)
	case canOpenBrowserFunc():
		fmt.Printf("Press Enter to open the browser or visit %s", claimURL)
		buf := make([]byte, 1)
		os.Stdin.Read(buf)
		openBrowserFunc(claimURL)
	default:
		fmt.Printf("Visit %s\n", claimURL)
	}
	return nil
}

func (scc *sandboxCreateCmd) runAuthenticatedSandboxCreateCmd(cmd *cobra.Command, args []string) error {
	for _, flagName := range []string{"email", "from-git", "full-name", "non-interactive", "base-url", "dashboard-base"} {
		if cmd.Flags().Changed(flagName) {
			return errorcategory.Errorf(errorcategory.UserInput, "--%s is only valid for anonymous sandbox provisioning", flagName)
		}
	}
	if len(args) == 0 {
		return errorcategory.New(errorcategory.UserInput, "sandbox name is required; for example: `stripe sandbox create \"My sandbox\"`")
	}

	name := strings.TrimSpace(args[0])
	if name == "" {
		return errorcategory.New(errorcategory.UserInput, "sandbox name cannot be blank")
	}

	country := strings.ToUpper(strings.TrimSpace(scc.country))
	switch {
	case scc.createBlank && !validSandboxCountryCode(country):
		return errorcategory.New(errorcategory.UserInput, "--create-blank requires --country with a two-letter country code")
	case !scc.createBlank && country != "":
		return errorcategory.New(errorcategory.UserInput, "--country is only valid with --create-blank")
	}

	client := scc.client
	if client == nil {
		client = sandbox.NewManagementClient(scc.apiBaseURL, Config.GetProfile())
	}
	created, err := client.Create(cmd.Context(), sandbox.CreateOptions{
		Name:    name,
		Blank:   scc.createBlank,
		Country: country,
	})
	if err != nil {
		return err
	}

	out := cmd.OutOrStdout()
	fmt.Fprintf(out, "Created sandbox %q\n\n", name)
	fmt.Fprintf(out, "Account ID: %s\n", created.AccountID)
	fmt.Fprintln(out, "\nNext step: Run `stripe login` to access this sandbox with the CLI.")
	scc.authorizeCreatedSandbox(cmd)
	return nil
}

func (scc *sandboxCreateCmd) authorizeCreatedSandbox(cmd *cobra.Command) {
	isInteractive := scc.isInteractive
	if isInteractive == nil {
		isInteractive = sandboxCommandIsInteractive
	}
	if !isInteractive(cmd) {
		return
	}

	fmt.Fprint(cmd.OutOrStdout(), "Authorize this sandbox with the CLI now? [y/N]: ")
	input, err := bufio.NewReader(cmd.InOrStdin()).ReadString('\n')
	if err != nil {
		if errors.Is(err, io.EOF) {
			return
		}
		scc.warnAuthorizationFailure(cmd, err)
		return
	}
	input = strings.ToLower(strings.TrimSpace(input))
	if input != "y" && input != "yes" {
		return
	}

	uat, err := Config.Profile.GetUAT()
	if err != nil {
		scc.warnAuthorizationFailure(cmd, err)
		return
	}
	uat, err = config.RefreshUATIfNeeded(&Config.Profile, uat)
	if err != nil {
		scc.warnAuthorizationFailure(cmd, err)
		return
	}
	if !strings.HasPrefix(uat, "oak_") {
		scc.warnAuthorizationFailure(cmd, errorcategory.New(errorcategory.Auth, "no valid OAuth session is available"))
		return
	}

	reauth := scc.reauth
	if reauth == nil {
		reauth = login.ReauthImmediately
	}
	if err := reauth(cmd.Context(), rootAccessBaseURL, uat); err != nil {
		scc.warnAuthorizationFailure(cmd, err)
	}
}

// Follow-up authorization can fail because the keyring is unavailable, the
// OAuth session needs recovery, or the browser/polling flow is interrupted.
// Creation has already succeeded, so keep the warning actionable and return
// success to avoid prompting the caller to create a duplicate sandbox.
func (scc *sandboxCreateCmd) warnAuthorizationFailure(cmd *cobra.Command, err error) {
	fmt.Fprintf(cmd.ErrOrStderr(), "Warning: sandbox creation succeeded, but CLI authorization was not completed: %s\n", err)
	fmt.Fprintln(cmd.ErrOrStderr(), "Run `stripe login` to authorize the existing sandbox.")
}

func validSandboxCountryCode(country string) bool {
	return len(country) == 2 &&
		country[0] >= 'A' && country[0] <= 'Z' &&
		country[1] >= 'A' && country[1] <= 'Z'
}

func newSandboxListCmd() *sandboxListCmd {
	slc := &sandboxListCmd{}
	slc.cmd = &cobra.Command{
		Use:    "list",
		Short:  "List the sandboxes under a live account",
		Args:   validators.NoArgs,
		RunE:   slc.runSandboxListCmd,
		Hidden: true,
	}

	slc.cmd.Flags().StringVar(&slc.apiBase, "api-base", stripe.DefaultAPIBaseURL, "Sets the Stripe API base URL")
	_ = slc.cmd.Flags().MarkHidden("api-base")

	return slc
}

func (slc *sandboxListCmd) runSandboxListCmd(cmd *cobra.Command, args []string) error {
	client := slc.client
	if client == nil {
		client = sandbox.NewManagementClient(slc.apiBase, Config.GetProfile())
	}

	sandboxes, err := client.ListAccessible(cmd.Context())
	if err != nil {
		return err
	}

	accessLevels := make([]string, len(sandboxes))
	for i, managedSandbox := range sandboxes {
		var ok bool
		accessLevels[i], ok = sandboxAccessLevelLabel(managedSandbox.AccessLevel)
		if !ok {
			return errorcategory.New(errorcategory.API, "could not render sandbox list: the response contained an invalid access level")
		}
	}

	if len(sandboxes) == 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "No sandboxes found.")
		return nil
	}

	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "NAME\tACCOUNT\tACCESS")
	for i, managedSandbox := range sandboxes {
		fmt.Fprintf(w, "%s\t%s\t%s\n", managedSandbox.Name, managedSandbox.AccountID, accessLevels[i])
	}
	return w.Flush()
}

func sandboxAccessLevelLabel(accessLevel sandbox.SandboxAccessLevel) (string, bool) {
	switch accessLevel {
	case sandbox.SandboxAccessLevelPrivate:
		return "Private", true
	case sandbox.SandboxAccessLevelGlobal:
		return "All team members", true
	case sandbox.SandboxAccessLevelDeveloper:
		return "Developer", true
	default:
		return "", false
	}
}

func newSandboxDeleteCmd() *sandboxDeleteCmd {
	sdc := &sandboxDeleteCmd{}
	sdc.cmd = &cobra.Command{
		Use:   "delete <account_id>",
		Short: "Delete a sandbox by its account ID",
		Long: `Delete a sandbox created for the logged-in account.

Pass the sandbox account ID (acct_...) shown by ` + "`stripe sandbox list`" + `. This
closes the sandbox's testmode workspace, mirroring the dashboard's delete action;
it never touches your live account.`,
		Example: `stripe sandbox delete acct_123
  stripe sandbox delete acct_123 --confirm`,
		Args:   validators.ExactArgs(1),
		RunE:   sdc.runSandboxDeleteCmd,
		Hidden: true,
	}

	sdc.cmd.Flags().BoolVarP(&sdc.confirm, "confirm", "c", false, "Skip the confirmation prompt")

	sdc.cmd.Flags().StringVar(&sdc.apiBase, "api-base", stripe.DefaultAPIBaseURL, "Sets the Stripe API base URL")
	_ = sdc.cmd.Flags().MarkHidden("api-base")

	return sdc
}

func (sdc *sandboxDeleteCmd) runSandboxDeleteCmd(cmd *cobra.Command, args []string) error {
	stripeAccount := strings.TrimSpace(args[0])
	switch {
	case stripeAccount == "":
		return errorcategory.Errorf(errorcategory.UserInput, "account ID is required (the acct_ shown by `stripe sandbox list`)")
	case strings.HasPrefix(stripeAccount, "org_"):
		return errorcategory.Errorf(errorcategory.UserInput, "account ID must be an account (acct_...), not an organization (org_...)")
	case !strings.HasPrefix(stripeAccount, "acct_") || len(stripeAccount) == len("acct_"):
		return errorcategory.Errorf(errorcategory.UserInput, "account ID must start with acct_")
	}

	confirmed, err := sdc.confirmDelete(cmd, stripeAccount)
	if err != nil {
		return err
	}
	if !confirmed {
		fmt.Fprintln(cmd.OutOrStdout(), "Aborted. No changes were made.")
		return nil
	}

	client := sdc.client
	if client == nil {
		client = sandbox.NewManagementClient(sdc.apiBase, Config.GetProfile())
	}
	deleted, err := client.Delete(cmd.Context(), stripeAccount)
	if err != nil {
		return err
	}

	out := cmd.OutOrStdout()
	if deleted.Name != "" {
		fmt.Fprintf(out, "Deleted sandbox %q (%s)\n", deleted.Name, deleted.AccountID)
	} else {
		fmt.Fprintf(out, "Deleted sandbox %s\n", deleted.AccountID)
	}
	return nil
}

func (sdc *sandboxDeleteCmd) confirmDelete(cmd *cobra.Command, accountID string) (bool, error) {
	if sdc.confirm {
		return true, nil
	}

	if !sandboxCommandIsInteractive(cmd) {
		return false, errorcategory.Errorf(errorcategory.UserInput, "refusing to delete sandbox %s without confirmation; re-run with --confirm", accountID)
	}

	out := cmd.OutOrStdout()
	fmt.Fprintf(out, "Delete sandbox %s?\n", accountID)
	fmt.Fprintln(out, "This action cannot be undone.")
	fmt.Fprint(out, "Continue? [y/N]: ")

	input, err := bufio.NewReader(cmd.InOrStdin()).ReadString('\n')
	if err != nil && err != io.EOF {
		return false, err
	}
	input = strings.ToLower(strings.TrimSpace(input))
	return input == "y" || input == "yes", nil
}

func sandboxCommandIsInteractive(cmd *cobra.Command) bool {
	if cmd.InOrStdin() != os.Stdin {
		return true
	}
	return interactiveHuman(os.Getenv, term.IsTerminal(int(os.Stdin.Fd())))
}

type sandboxClaimStatusResponse struct {
	IsClaimed *bool `json:"is_claimed"`
}

func fetchSandboxClaimStatus(ctx context.Context, apiBaseURL, apiKey string) (bool, error) {
	if err := stripe.ValidateAPIBaseURL(apiBaseURL); err != nil {
		return false, err
	}

	baseURL, err := url.Parse(apiBaseURL)
	if err != nil {
		return false, err
	}

	client := &stripe.Client{
		BaseURL:     baseURL,
		Credentials: stripe.NewAPIKeyCredentials(apiKey),
	}

	resp, err := client.PerformRequest(ctx, http.MethodGet, "/v2/core/claimable_sandboxes/status", "", func(req *http.Request) error {
		req.Header.Set("Stripe-Version", sandboxClaimStatusVersion)
		return nil
	})
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return false, err
	}

	if resp.StatusCode != http.StatusOK {
		return false, errorcategory.Errorf(errorcategory.API, "failed to retrieve sandbox claim status (status %d)", resp.StatusCode)
	}

	var parsed sandboxClaimStatusResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return false, errorcategory.Errorf(errorcategory.API, "failed to parse response: %w", err)
	}
	if parsed.IsClaimed == nil {
		return false, errorcategory.Errorf(errorcategory.API, "invalid sandbox claim status response: missing is_claimed")
	}

	return *parsed.IsClaimed, nil
}

func isSSHSession() bool {
	return os.Getenv("SSH_TTY") != "" || os.Getenv("SSH_CONNECTION") != "" || os.Getenv("SSH_CLIENT") != ""
}
