package cmd

import (
	"bytes"
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
	"time"

	"github.com/google/uuid"
	"github.com/logrusorgru/aurora"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/stripe/stripe-cli/pkg/ansi"
	"github.com/stripe/stripe-cli/pkg/config"
	"github.com/stripe/stripe-cli/pkg/login"
	"github.com/stripe/stripe-cli/pkg/open"
	"github.com/stripe/stripe-cli/pkg/requests"
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
	cmd            *cobra.Command
	email          string
	fromGit        bool
	name           string
	nonInteractive bool
	baseURL        string
	dashboardURL   string
}

func newSandboxCmd() *sandboxCmd {
	sc := &sandboxCmd{}
	sc.cmd = &cobra.Command{
		Use:   "sandbox",
		Short: "Manage Stripe sandbox environments",
		Args:  validators.NoArgs,
		Annotations: map[string]string{
			AIAgentHelpAnnotationKey: "  Use `stripe sandbox create --from-git` to provision a sandbox using your git email.\n" +
				"  Use `stripe sandbox create --email you@example.com` to provision with an explicit email.\n" +
				"  If provisioning fails, falls back to browser login (like stripe login).\n" +
				"  If already logged in, opens the sandbox management page.",
		},
	}

	createCmd := newSandboxCreateCmd()
	claimCmd := newSandboxClaimCmd()
	sc.cmd.AddCommand(createCmd.cmd)
	sc.cmd.AddCommand(claimCmd.cmd)

	// `sandbox new` is an experimental POC. It is gated behind an env flag so it
	// is not registered at all (invisible AND unrunnable — `unknown command`)
	// unless explicitly enabled, on top of being Hidden. This keeps it out of
	// users' hands pre-GA; the authoritative access gate remains server-side
	// (the UAT allowlist + hzn_sandbox_create), not the client.
	if sandboxNewEnabled() {
		sc.cmd.AddCommand(newSandboxNewCmd().cmd)
	}
	return sc
}

// sandboxNewEnabled reports whether the experimental `stripe sandbox new` command
// should be registered. Mirrors the STRIPE_CLI_CANARY env-gating pattern.
func sandboxNewEnabled() bool {
	return os.Getenv("STRIPE_CLI_ENABLE_SANDBOX_NEW") == "true"
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
	scc.cmd.Flags().BoolVar(&scc.fromGit, "from-git", false, "Infer email and full name from git config")
	scc.cmd.Flags().StringVar(&scc.name, "full-name", "", "Your full name (optional)")
	scc.cmd.Flags().BoolVar(&scc.nonInteractive, "non-interactive", false, "Print output directly without waiting for input")

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
		fmt.Printf("Your sandbox session has expired.\nRun `stripe login` to continue with a claimed sandbox, or run `stripe sandbox create` again to create a new one.\n")
		return nil

	default:
		// Active claimable sandbox that hasn't expired. Show existing keys
		// and claim URL — one sandbox at a time.
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
		expiresAt := viper.GetString(Config.Profile.GetConfigField(config.SandboxExpiresAtName))
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
		return "", fmt.Errorf("--email and --from-git are mutually exclusive")
	}

	var email string
	switch {
	case scc.fromGit:
		gitEmail := sandbox.GitConfigFunc("user.email")
		if gitEmail == "" {
			return "", fmt.Errorf("--from-git requires git config user.email to be set, but it was not found")
		}
		fmt.Printf("Using email: %s (from git config)\n", gitEmail)
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
		return fmt.Errorf("browser login unavailable in SSH session")
	}

	if scc.nonInteractive {
		return login.InitiateLogin(cmd.Context(), scc.dashboardURL, &Config)
	}
	return login.Login(cmd.Context(), scc.dashboardURL, &Config)
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
		return fmt.Errorf("no secret key in server response")
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
	if viper.GetString(Config.Profile.GetConfigField(config.SandboxClaimURLName)) != "" {
		return true
	}
	return viper.GetString(Config.Profile.GetConfigField(config.SandboxExpiresAtName)) != ""
}

// isExpiredSandbox returns true if the sandbox_expires_at date has passed.
func isExpiredSandbox() bool {
	expiresAt := viper.GetString(Config.Profile.GetConfigField(config.SandboxExpiresAtName))
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

type sandboxClaimCmd struct {
	cmd            *cobra.Command
	nonInteractive bool
}

type sandboxNewCmd struct {
	cmd              *cobra.Command
	name             string
	replicaOf        string
	businessLocation string
	stripeContext    string
	activate         bool
	batch            int
	stripeVersion    string
	apiBase          string
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
	return scc
}

func (scc *sandboxClaimCmd) runSandboxClaimCmd(cmd *cobra.Command, args []string) error {
	claimURL := viper.GetString(Config.Profile.GetConfigField(config.SandboxClaimURLName))
	if claimURL == "" {
		fmt.Printf("No active sandbox. Run `stripe sandbox create` to get started.\n")
		return nil
	}

	if isExpiredSandbox() {
		clearExpiredSandboxProfile()
		fmt.Printf("Your sandbox session has expired.\nRun `stripe login` to continue with a claimed sandbox, or run `stripe sandbox create` again to create a new one.\n")
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

func newSandboxNewCmd() *sandboxNewCmd {
	snc := &sandboxNewCmd{}
	snc.cmd = &cobra.Command{
		Use:   "new",
		Short: "Create a sandbox for the logged-in account",
		Long: `Create a new sandbox via the authenticated Stripe API using your logged-in session (UAT).

This command creates a sandbox using the authenticated Stripe API. It requires
that you have previously logged in with your Stripe account credentials.`,
		Args: validators.NoArgs,
		RunE: snc.runSandboxNewCmd,
		// Hidden while this is an experimental POC: keep it out of help/completion.
		// The real access gate is the backend (the UAT flag + hzn_sandbox_create).
		// Remove this when the command is ready to GA.
		Hidden: true,
	}

	snc.cmd.Flags().StringVar(&snc.name, "name", "", "Name for the new sandbox")
	snc.cmd.Flags().StringVar(&snc.replicaOf, "replica-of", "", "Livemode workspace ID to replicate (wksp_...); mutually exclusive with --business-location")
	snc.cmd.Flags().StringVar(&snc.businessLocation, "business-location", "", "Country for a fresh blank sandbox (e.g. US); mutually exclusive with --replica-of")
	snc.cmd.Flags().StringVar(&snc.stripeContext, "stripe-context", "", "Playground compartment (play_...) to create the sandbox under")
	snc.cmd.Flags().BoolVar(&snc.activate, "activate", true, "Request capabilities and activate the sandbox after creation")
	snc.cmd.Flags().IntVar(&snc.batch, "batch", 1, "Number of sandboxes to create (currently only 1 is supported)")

	snc.cmd.Flags().StringVar(&snc.stripeVersion, "stripe-version", requests.StripeVersionHeaderValue, "Sets the Stripe-Version header")
	_ = snc.cmd.Flags().MarkHidden("stripe-version")

	snc.cmd.Flags().StringVar(&snc.apiBase, "api-base", stripe.DefaultAPIBaseURL, "Sets the Stripe API base URL")
	_ = snc.cmd.Flags().MarkHidden("api-base")

	return snc
}

func (snc *sandboxNewCmd) runSandboxNewCmd(cmd *cobra.Command, args []string) error {
	// The UAT is stored under the bare keyring key (config.UATKeychainItemKey),
	// not the per-profile field, so read it directly from the keyring rather
	// than going through the profile helpers (which read live/test API keys).
	if config.KeyRing == nil {
		return fmt.Errorf("credential store unavailable; run `stripe login` first")
	}
	uatBytes, err := config.KeyRing.Get(config.UATKeychainItemKey)
	if err != nil || len(uatBytes) == 0 {
		return fmt.Errorf("no user access token found; run `stripe login` first")
	}
	uat := strings.TrimSpace(string(uatBytes))

	if strings.TrimSpace(snc.name) == "" {
		return fmt.Errorf("--name is required")
	}

	// --batch is scaffolded for a future bulk-create shape and currently only
	// supports 1. Future shape: create N sandboxes under the same playground in a
	// single invocation via a bounded fan-out (or a batched backend request) that
	// shares an idempotency prefix, returns the list of created sandboxes, and
	// reports per-item partial failures instead of aborting the whole batch.
	// Until that lands, reject >1 explicitly rather than silently creating one.
	if snc.batch < 1 {
		return fmt.Errorf("--batch must be >= 1")
	}
	if snc.batch > 1 {
		return fmt.Errorf("--batch > 1 is not yet implemented; only --batch 1 is supported today")
	}

	// Stripe-Context must be a playground (play_...) compartment. The backend
	// (CreateSandboxOp) rejects any non-playground context, so validate it here
	// to give a clear error rather than a server-side rejection.
	stripeContext := strings.TrimSpace(snc.stripeContext)
	if stripeContext == "" {
		return fmt.Errorf("--stripe-context is required; pass the playground id (play_...) to create the sandbox under")
	}
	if !strings.HasPrefix(stripeContext, "play_") {
		return fmt.Errorf("--stripe-context must be a playground id (play_...), got %q", stripeContext)
	}

	// The backend requires exactly one of replica_of / business_location (a oneof):
	// replica_of (a livemode wksp_...) clones a live workspace; business_location
	// (a country) creates a fresh blank sandbox. Enforce the oneof up front.
	replicaOf := strings.TrimSpace(snc.replicaOf)
	businessLocation := strings.TrimSpace(snc.businessLocation)
	switch {
	case replicaOf != "" && businessLocation != "":
		return fmt.Errorf("--replica-of and --business-location are mutually exclusive")
	case replicaOf == "" && businessLocation == "":
		return fmt.Errorf("pass one of --replica-of (wksp_...) to clone a live workspace, or --business-location (e.g. US) for a blank sandbox")
	case replicaOf != "" && !strings.HasPrefix(replicaOf, "wksp_"):
		return fmt.Errorf("--replica-of must be a livemode workspace id (wksp_...), got %q", replicaOf)
	}

	baseURL, err := url.Parse(snc.apiBase)
	if err != nil {
		return fmt.Errorf("invalid --api-base %q: %w", snc.apiBase, err)
	}

	// Mirror the dashboard's create-sandbox request body. The idempotency token
	// guards against duplicate creates if the request is retried.
	reqBody := map[string]interface{}{
		"name":              snc.name,
		"activate_sandbox":  snc.activate,
		"idempotency_token": newIdempotencyToken(snc.name),
	}
	if replicaOf != "" {
		reqBody["replica_of"] = replicaOf
	} else {
		reqBody["business_location"] = businessLocation
	}
	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return err
	}

	// Use the low-level client with an empty APIKey so it doesn't set a Bearer
	// Authorization header; the UAT is injected via the configure hook below as
	// a STRIPE-V2-SIG token instead.
	client := &stripe.Client{
		BaseURL: baseURL,
		APIKey:  "",
	}

	configure := func(req *http.Request) error {
		req.Header.Set("Authorization", "STRIPE-V2-SIG "+uat)
		req.Header.Set("Stripe-Version", snc.stripeVersion)
		req.Header.Set("Stripe-Context", stripeContext)
		// Content-Type is set to application/json automatically by the client
		// for /v2/ paths, but set it explicitly to be safe.
		req.Header.Set("Content-Type", stripe.V2ContentType)
		return nil
	}

	resp, err := client.PerformRequest(cmd.Context(), http.MethodPost, "/v2/sandboxes", string(bodyBytes), configure)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("create sandbox failed: %s\n%s", resp.Status, string(respBytes))
	}

	// Pretty-print the JSON response when possible; otherwise print as-is.
	var pretty bytes.Buffer
	if json.Indent(&pretty, respBytes, "", "  ") == nil {
		fmt.Fprintln(cmd.OutOrStdout(), pretty.String())
	} else {
		fmt.Fprintln(cmd.OutOrStdout(), string(respBytes))
	}

	return nil
}

// newIdempotencyToken builds a stable-per-invocation idempotency token from the
// sandbox name and a random UUID so retried requests within a run don't create
// duplicate sandboxes.
func newIdempotencyToken(name string) string {
	suffix := uuid.NewString()
	if name == "" {
		return suffix
	}
	return name + "-" + suffix
}

func isSSHSession() bool {
	return os.Getenv("SSH_TTY") != "" || os.Getenv("SSH_CONNECTION") != "" || os.Getenv("SSH_CLIENT") != ""
}
