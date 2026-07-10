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
	"text/tabwriter"
	"time"

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

	// `sandbox new` is an experimental POC. It stays Hidden (kept out of
	// help/completion) but is always registered — this ships only to the
	// long-lived sandboxes-cli feature branch, never to master pre-GA, so no
	// client-side env gate is needed. The authoritative access gate remains
	// server-side (the UAT allowlist + hzn_sandbox_create), not the client.
	sc.cmd.AddCommand(newSandboxNewCmd().cmd)
	sc.cmd.AddCommand(newSandboxListCmd().cmd)
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
	copyLiveAccount  bool
	createBlank      bool
	activate         bool
	batch            int
	stripeVersion    string
	apiBase          string
}

type sandboxListCmd struct {
	cmd           *cobra.Command
	stripeAccount string
	stripeVersion string
	apiBase       string
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
	_ = snc.cmd.MarkFlagRequired("name")
	// Mode selectors mirror the dashboard's create-sandbox modal: copy a live
	// account, or create a blank sandbox for a country. Exactly one is required.
	snc.cmd.Flags().BoolVar(&snc.copyLiveAccount, "copy-live-account", false, "Copy your live account into a new sandbox")
	snc.cmd.Flags().BoolVar(&snc.createBlank, "create-blank", false, "Create a fresh blank sandbox (requires --business-location)")
	snc.cmd.Flags().StringVar(&snc.businessLocation, "business-location", "", "Country for a --create-blank sandbox (e.g. US)")
	snc.cmd.Flags().StringVar(&snc.replicaOf, "replica-of", "", "Livemode workspace ID (wksp_...) to copy; defaults to your logged-in account")
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

	if err := snc.validateFlags(); err != nil {
		return err
	}
	replicaOf := strings.TrimSpace(snc.replicaOf)
	businessLocation := strings.TrimSpace(snc.businessLocation)

	baseURL, err := url.Parse(snc.apiBase)
	if err != nil {
		return fmt.Errorf("invalid --api-base %q: %w", snc.apiBase, err)
	}

	// Low-level client with an empty APIKey so it doesn't set a Bearer auth
	// header; the UAT is injected per-request as a STRIPE-V2-SIG token instead.
	client := &stripe.Client{
		BaseURL: baseURL,
		APIKey:  "",
	}

	// authConfigure sets the UAT auth + version headers shared by every call.
	// The context-resolution GETs (user_accessible, playground) are self-scoped
	// by the UAT and take no Stripe-Context; only the create call sets it.
	authConfigure := func(req *http.Request) error {
		req.Header.Set("Authorization", "STRIPE-V2-SIG "+uat)
		req.Header.Set("Stripe-Version", snc.stripeVersion)
		req.Header.Set("Content-Type", stripe.V2ContentType)
		return nil
	}

	// Resolve the live workspace to copy / derive the playground from (both modes
	// need it). Order: --replica-of override, the livemode compartment saved at
	// login, then GET /v2/compartments/user_accessible.
	liveWorkspace := replicaOf
	if liveWorkspace == "" {
		liveWorkspace, err = resolveLiveWorkspace(cmd.Context(), client, authConfigure)
		if err != nil {
			return err
		}
	}

	// Never copy an organization: replica_of / playground resolution both require a
	// workspace (wksp_). Guard every resolution path (login context, user_accessible,
	// --replica-of) at one choke point so an org_ can never slip through.
	if !strings.HasPrefix(liveWorkspace, "wksp_") {
		return fmt.Errorf("resolved live parent %q is not a workspace (wksp_...); sandboxes can only copy an account, not an organization", liveWorkspace)
	}
	// Surface the auto-resolved account so the default is never silent.
	if snc.copyLiveAccount {
		fmt.Fprintf(cmd.ErrOrStderr(), "Copying live workspace %s\n", liveWorkspace)
	}

	// Resolve the internal playground (play_) for the live workspace. Playground
	// ids are not user-facing, so there is no override: the command always derives it.
	stripeContext, err := snc.resolvePlayground(cmd.Context(), client, authConfigure, liveWorkspace)
	if err != nil {
		return err
	}
	if !strings.HasPrefix(stripeContext, "play_") {
		return fmt.Errorf("resolved a non-playground context %q for %s", stripeContext, liveWorkspace)
	}

	// Mirror the dashboard's create-sandbox request body, including the
	// idempotency_token derived from the create inputs (see newIdempotencyToken).
	reqBody := map[string]interface{}{
		"name":              snc.name,
		"activate_sandbox":  snc.activate,
		"idempotency_token": newIdempotencyToken(snc.name, businessLocation, liveWorkspace),
	}
	if snc.createBlank {
		reqBody["business_location"] = businessLocation
	} else {
		reqBody["replica_of"] = liveWorkspace
	}
	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return err
	}

	// The create call additionally scopes to the resolved playground.
	createConfigure := func(req *http.Request) error {
		if cfgErr := authConfigure(req); cfgErr != nil {
			return cfgErr
		}
		req.Header.Set("Stripe-Context", stripeContext)
		return nil
	}

	resp, err := client.PerformRequest(cmd.Context(), http.MethodPost, "/v2/sandboxes", string(bodyBytes), createConfigure)
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

	// Lead with the actionable ids (name, the acct_ to target, the sandbox id), then
	// include the full response for anything else the caller needs.
	var created struct {
		ID          string `json:"id"`
		V1AccountID string `json:"v1_account_id"`
		Name        string `json:"name"`
	}
	_ = json.Unmarshal(respBytes, &created)
	out := cmd.OutOrStdout()
	if created.ID != "" {
		fmt.Fprintf(out, "Created sandbox %q\n", created.Name)
		if created.V1AccountID != "" {
			fmt.Fprintf(out, "  account: %s\n", created.V1AccountID)
		}
		fmt.Fprintf(out, "  sandbox: %s\n", created.ID)
	}
	// Pretty-print the JSON response when possible; otherwise print as-is.
	var pretty bytes.Buffer
	if json.Indent(&pretty, respBytes, "", "  ") == nil {
		fmt.Fprintln(out, pretty.String())
	} else {
		fmt.Fprintln(out, string(respBytes))
	}

	return nil
}

// validateFlags checks --batch and the mutually-exclusive mode selectors
// (--copy-live-account / --create-blank) and their inputs, mirroring the
// dashboard's create-sandbox modal. Extracted from runSandboxNewCmd to keep its
// cyclomatic complexity manageable.
func (snc *sandboxNewCmd) validateFlags() error {
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

	// Mode selection mirrors the dashboard's create-sandbox modal: copy a live
	// account, or create a blank sandbox for a country. Exactly one is required.
	replicaOf := strings.TrimSpace(snc.replicaOf)
	businessLocation := strings.TrimSpace(snc.businessLocation)
	switch {
	case snc.copyLiveAccount && snc.createBlank:
		return fmt.Errorf("--copy-live-account and --create-blank are mutually exclusive")
	case !snc.copyLiveAccount && !snc.createBlank:
		return fmt.Errorf("pass one of --copy-live-account (copy your live account) or --create-blank (a fresh sandbox)")
	case snc.createBlank && businessLocation == "":
		return fmt.Errorf("--create-blank requires --business-location (e.g. US)")
	case snc.createBlank && replicaOf != "":
		return fmt.Errorf("--replica-of is only valid with --copy-live-account")
	case snc.copyLiveAccount && businessLocation != "":
		return fmt.Errorf("--business-location is only valid with --create-blank")
	case replicaOf != "" && !strings.HasPrefix(replicaOf, "wksp_"):
		return fmt.Errorf("--replica-of must be a livemode workspace id (wksp_...), got %q", replicaOf)
	}
	return nil
}

// accessibleWorkspace is the subset of a user_accessible / user_accessible_sandboxes
// entry the sandbox resolvers need. The same shape appears as a standalone workspace,
// nested under organizations[].workspaces, and as a sandbox entry.
type accessibleWorkspace struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	MerchantID string `json:"merchant_id"`
	ReplicaOf  string `json:"replica_of"`
	Livemode   string `json:"livemode"`
}

// fetchAccessibleWorkspaces returns the caller's accessible live workspaces (standalone
// plus org-nested) from GET /v2/compartments/user_accessible.
func fetchAccessibleWorkspaces(ctx context.Context, client *stripe.Client, configure func(*http.Request) error) ([]accessibleWorkspace, error) {
	resp, err := client.PerformRequest(ctx, http.MethodGet, "/v2/compartments/user_accessible", "", configure)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("could not list your accounts: %s\n%s", resp.Status, string(respBytes))
	}
	var parsed struct {
		StandaloneWorkspaces []accessibleWorkspace `json:"standalone_workspaces"`
		Organizations        []struct {
			Workspaces []accessibleWorkspace `json:"workspaces"`
		} `json:"organizations"`
	}
	if err := json.Unmarshal(respBytes, &parsed); err != nil {
		return nil, fmt.Errorf("could not parse accounts response: %w", err)
	}
	out := append([]accessibleWorkspace{}, parsed.StandaloneWorkspaces...)
	for _, org := range parsed.Organizations {
		out = append(out, org.Workspaces...)
	}
	return out, nil
}

// resolveLiveWorkspace determines the livemode workspace/org compartment to use
// as the sandbox's live parent. It first reads the livemode compartment saved at
// login (no network), then falls back to GET /v2/compartments/user_accessible.
// It returns an error if zero or more than one livemode workspace is found so the
// caller can disambiguate with --replica-of.
func resolveLiveWorkspace(ctx context.Context, client *stripe.Client, configure func(*http.Request) error) (string, error) {
	// 1. The livemode compartment pinned at login (what the user was scoped to).
	if ui, uiErr := Config.Profile.GetUserInfo(); uiErr == nil && ui != nil {
		for _, c := range ui.Compartments {
			if c.Livemode && strings.HasPrefix(strings.TrimSpace(c.CompartmentID), "wksp_") {
				return c.CompartmentID, nil
			}
		}
	}

	// 2. Ask the server which accounts this credential can access.
	resp, err := client.PerformRequest(ctx, http.MethodGet, "/v2/compartments/user_accessible", "", configure)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("could not list your accounts: %s\n%s", resp.Status, string(respBytes))
	}

	// standalone_workspaces are already filtered to livemode roots by the backend.
	var parsed struct {
		StandaloneWorkspaces []struct {
			ID string `json:"id"`
		} `json:"standalone_workspaces"`
	}
	if err := json.Unmarshal(respBytes, &parsed); err != nil {
		return "", fmt.Errorf("could not parse accounts response: %w", err)
	}

	var workspaces []string
	for _, w := range parsed.StandaloneWorkspaces {
		if strings.HasPrefix(w.ID, "wksp_") {
			workspaces = append(workspaces, w.ID)
		}
	}
	switch len(workspaces) {
	case 0:
		return "", fmt.Errorf("no livemode workspace found for your account; run `stripe login`, or pass --replica-of with a wksp_ id")
	case 1:
		return workspaces[0], nil
	default:
		return "", fmt.Errorf("you have multiple livemode workspaces; pass --replica-of wksp_... to choose one")
	}
}

// resolvePlayground resolves the internal playground compartment (play_) for a
// livemode workspace/org via GET /v2/compartments/playground/:id.
func (snc *sandboxNewCmd) resolvePlayground(ctx context.Context, client *stripe.Client, configure func(*http.Request) error, compartmentID string) (string, error) {
	if compartmentID == "" {
		return "", fmt.Errorf("could not determine a live workspace to resolve the playground from; pass --replica-of with a wksp_ id")
	}
	resp, err := client.PerformRequest(ctx, http.MethodGet, "/v2/compartments/playground/"+url.PathEscape(compartmentID), "", configure)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("could not resolve the playground for %s: %s\n%s", compartmentID, resp.Status, string(respBytes))
	}
	var parsed struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(respBytes, &parsed); err != nil {
		return "", fmt.Errorf("could not parse playground response: %w", err)
	}
	if parsed.ID == "" {
		return "", fmt.Errorf("no playground found for %s; run 'stripe login' again or pass a different --replica-of wksp_ id", compartmentID)
	}
	return parsed.ID, nil
}

// newIdempotencyToken mirrors the dashboard's create-sandbox flow
// (CreateSandboxFlow.tsx): the token is derived from the create inputs (name,
// business_location country, and the replica_of parent workspace) plus a
// wall-clock second. An accidental double-submit of the same command within the
// same second collapses to one create; distinct invocations create distinct
// sandboxes. Sent as the body idempotency_token field, matching the dashboard's
// v2CreateSandbox call (which does not use the Idempotency-Key header).
func newIdempotencyToken(name, country, parent string) string {
	return fmt.Sprintf("%s-%s-%s-%d", name, country, parent, time.Now().Unix())
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

	slc.cmd.Flags().StringVar(&slc.stripeAccount, "stripe-account", "", "Live account (acct_...) whose sandboxes to list; defaults to your logged-in account")
	slc.cmd.Flags().StringVar(&slc.stripeVersion, "stripe-version", requests.StripeVersionHeaderValue, "Sets the Stripe-Version header")
	_ = slc.cmd.Flags().MarkHidden("stripe-version")

	slc.cmd.Flags().StringVar(&slc.apiBase, "api-base", stripe.DefaultAPIBaseURL, "Sets the Stripe API base URL")
	_ = slc.cmd.Flags().MarkHidden("api-base")

	return slc
}

func (slc *sandboxListCmd) runSandboxListCmd(cmd *cobra.Command, args []string) error {
	if config.KeyRing == nil {
		return fmt.Errorf("credential store unavailable; run `stripe login` first")
	}
	uatBytes, err := config.KeyRing.Get(config.UATKeychainItemKey)
	if err != nil || len(uatBytes) == 0 {
		return fmt.Errorf("no user access token found; run `stripe login` first")
	}
	uat := strings.TrimSpace(string(uatBytes))

	stripeAccount := strings.TrimSpace(slc.stripeAccount)
	if stripeAccount != "" {
		if strings.HasPrefix(stripeAccount, "org_") {
			return fmt.Errorf("--stripe-account must be an account (acct_...), not an organization (org_...)")
		}
		if !strings.HasPrefix(stripeAccount, "acct_") {
			return fmt.Errorf("--stripe-account must be an account id (acct_...), got %q", stripeAccount)
		}
	}

	baseURL, err := url.Parse(slc.apiBase)
	if err != nil {
		return fmt.Errorf("invalid --api-base %q: %w", slc.apiBase, err)
	}

	client := &stripe.Client{
		BaseURL: baseURL,
		APIKey:  "",
	}

	authConfigure := func(req *http.Request) error {
		req.Header.Set("Authorization", "STRIPE-V2-SIG "+uat)
		req.Header.Set("Stripe-Version", slc.stripeVersion)
		req.Header.Set("Content-Type", stripe.V2ContentType)
		return nil
	}

	// Fetch accessible workspaces once and build maps for translation
	accessible, err := fetchAccessibleWorkspaces(cmd.Context(), client, authConfigure)
	if err != nil {
		return err
	}
	acctByWksp := make(map[string]string, len(accessible))
	nameByWksp := make(map[string]string, len(accessible))
	for _, w := range accessible {
		if w.ID != "" {
			acctByWksp[w.ID] = w.MerchantID
			nameByWksp[w.ID] = w.Name
		}
	}

	// Resolve the live parent (workspace id, acct_, and name)
	var liveWorkspace, liveAccount, liveName string
	if stripeAccount != "" {
		for _, w := range accessible {
			if w.MerchantID == stripeAccount && strings.HasPrefix(w.ID, "wksp_") {
				liveWorkspace, liveName = w.ID, w.Name
				break
			}
		}
		if liveWorkspace == "" {
			return fmt.Errorf("no accessible live account matches %s; check the id or run `stripe login` again", stripeAccount)
		}
		liveAccount = stripeAccount
	} else {
		liveWorkspace, err = resolveLiveWorkspace(cmd.Context(), client, authConfigure)
		if err != nil {
			return err
		}
		liveAccount = acctByWksp[liveWorkspace]
		liveName = nameByWksp[liveWorkspace]
	}
	if !strings.HasPrefix(liveWorkspace, "wksp_") {
		return fmt.Errorf("resolved live parent %q is not a workspace (wksp_...)", liveWorkspace)
	}

	// Transparency line (stderr): prefer name, then acct_, never show wksp_ unless nothing else
	switch {
	case liveName != "":
		fmt.Fprintf(cmd.ErrOrStderr(), "Listing sandboxes under live account %q (%s)\n", liveName, liveAccount)
	case liveAccount != "":
		fmt.Fprintf(cmd.ErrOrStderr(), "Listing sandboxes under live account %s\n", liveAccount)
	default:
		fmt.Fprintf(cmd.ErrOrStderr(), "Listing sandboxes under live workspace %s\n", liveWorkspace)
	}

	query := "live_compartment_parent_id=" + url.QueryEscape(liveWorkspace)
	resp, err := client.PerformRequest(cmd.Context(), http.MethodGet, "/v2/compartments/user_accessible_sandboxes", query, authConfigure)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("list sandboxes failed: %s\n%s", resp.Status, string(respBytes))
	}

	var parsed struct {
		Workspaces    []accessibleWorkspace `json:"workspaces"`
		Organizations []struct {
			Workspaces []accessibleWorkspace `json:"workspaces"`
		} `json:"organizations"`
	}
	if err := json.Unmarshal(respBytes, &parsed); err != nil {
		return fmt.Errorf("could not parse sandboxes response: %w", err)
	}

	var sandboxes []accessibleWorkspace
	sandboxes = append(sandboxes, parsed.Workspaces...)
	for _, org := range parsed.Organizations {
		sandboxes = append(sandboxes, org.Workspaces...)
	}

	if len(sandboxes) == 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "No sandboxes found.")
		return nil
	}

	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ACCOUNT\tNAME\tREPLICA OF")
	for _, s := range sandboxes {
		replicaAcct := ""
		if s.ReplicaOf != "" {
			replicaAcct = acctByWksp[s.ReplicaOf] // parent's acct_; empty if not found — never show wksp_
		}
		fmt.Fprintf(w, "%s\t%s\t%s\n", s.MerchantID, s.Name, replicaAcct)
	}
	w.Flush()

	return nil
}

func isSSHSession() bool {
	return os.Getenv("SSH_TTY") != "" || os.Getenv("SSH_CONNECTION") != "" || os.Getenv("SSH_CLIENT") != ""
}
