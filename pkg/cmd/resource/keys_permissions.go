package resource

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/spf13/cobra"

	"github.com/stripe/stripe-cli/pkg/cmdutil"
	"github.com/stripe/stripe-cli/pkg/config"
	"github.com/stripe/stripe-cli/pkg/stripe"
	"github.com/stripe/stripe-cli/pkg/validators"
)

type keysPermissionsResponse struct {
	Permissions         []string             `json:"permissions"`
	UnresolvedEndpoints []unresolvedEndpoint `json:"unresolved_endpoints"`
}

type unresolvedEndpoint struct {
	Endpoint string `json:"endpoint"`
	Reason   string `json:"reason"`
}

// KeysPermissionsCmd resolves the RAK permissions required for a set of API endpoints.
type KeysPermissionsCmd struct {
	cfg        *config.Config
	cmd        *cobra.Command
	apiBaseURL string
}

// NewKeysPermissionsCmd returns a new keys permissions command.
func NewKeysPermissionsCmd(parentCmd *cobra.Command, cfg *config.Config) {
	kpc := &KeysPermissionsCmd{cfg: cfg}

	kpc.cmd = &cobra.Command{
		Use:   "permissions [endpoints...]",
		Short: "Resolve required restricted key permissions for API endpoints",
		Long:  `Given one or more API endpoints (e.g. "GET /v1/customers"), resolves the minimal set of restricted key permissions required.`,
		Example: `  stripe keys permissions "GET /v1/customers" "GET /v1/balance"
    stripe keys permissions "GET /v1/charges/:id"`,
		Args:   cobra.MinimumNArgs(1),
		RunE:   kpc.runKeysPermissionsCmd,
		Hidden: true,
	}

	kpc.cmd.Flags().StringVar(&kpc.apiBaseURL, "api-base", stripe.DefaultAPIBaseURL, "Sets the API base URL")
	kpc.cmd.Flags().MarkHidden("api-base") // #nosec G104

	parentCmd.AddCommand(kpc.cmd)
}

func (kpc *KeysPermissionsCmd) runKeysPermissionsCmd(cmd *cobra.Command, args []string) error {
	apiKey, err := kpc.cfg.Profile.GetAPIKey(false)
	if err != nil {
		return err
	}

	err = validators.APIKeyNotRestricted(apiKey)
	if err != nil {
		return err
	}

	baseURL, _ := url.Parse(kpc.apiBaseURL)
	client := &stripe.Client{
		BaseURL: baseURL,
		APIKey:  apiKey,
	}

	// Build form-encoded body
	form := url.Values{}
	for i, endpoint := range args {
		form.Add(fmt.Sprintf("endpoints[%d]", i), endpoint)
	}

	resp, err := client.PerformRequest(
		cmd.Context(),
		http.MethodPost,
		"/v1/stripecli/key_permissions",
		form.Encode(),
		nil,
	)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}

	var result keysPermissionsResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	// Print unresolved endpoints as warnings
	for _, ue := range result.UnresolvedEndpoints {
		fmt.Fprintf(cmd.ErrOrStderr(), "Warning: %s — %s\n", ue.Endpoint, ue.Reason)
	}

	if len(result.Permissions) == 0 {
		fmt.Println("No permissions resolved.")
		return nil
	}

	// Print permissions
	fmt.Println("Required permissions:")
	for _, p := range result.Permissions {
		fmt.Printf("  %s\n", p)
	}

	return nil
}

// AddKeysSubCmds adds custom subcommands to the `keys` command.
func AddKeysSubCmds(rootCmd *cobra.Command, cfg *config.Config) {
	// The `keys` resource command may not exist (it's auto-generated from the OpenAPI spec).
	// If it doesn't exist, create a top-level `keys` command to host our subcommand.
	keysCmd, ok := cmdutil.FindSubCmd(rootCmd, "keys")
	if !ok {
		keysCmd = &cobra.Command{
			Use:   "keys",
			Short: "Manage API keys",
		}
		rootCmd.AddCommand(keysCmd)
	}

	NewKeysPermissionsCmd(keysCmd, cfg)
}
