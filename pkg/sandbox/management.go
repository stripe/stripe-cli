package sandbox

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"net/url"
	"sort"
	"strings"

	"github.com/stripe/stripe-cli/pkg/config"
	"github.com/stripe/stripe-cli/pkg/errorcategory"
	"github.com/stripe/stripe-cli/pkg/requests"
	"github.com/stripe/stripe-cli/pkg/stripe"
)

const (
	accessibleSandboxesPath = "/v2/compartments/user_accessible_sandboxes"
	workspaceIDPrefix       = "wksp_"
	testmodeWorkspacePrefix = "wksp_test_"
	accountIDPrefix         = "acct_"
)

// SandboxAccessLevel describes the sandbox's default team access setting.
type SandboxAccessLevel int

const (
	SandboxAccessLevelPrivate SandboxAccessLevel = iota + 1
	SandboxAccessLevelGlobal
	SandboxAccessLevelDeveloper
)

// ManagedSandbox is the normalized representation shared by sandbox list and
// delete. WorkspaceID is retained for later targeting but must not be shown to
// users.
type ManagedSandbox struct {
	WorkspaceID string
	AccountID   string
	Name        string
	AccessLevel SandboxAccessLevel
}

// ManagementClient discovers sandboxes authorized by the active live OAuth
// account.
type ManagementClient struct {
	APIBaseURL string
	Profile    *config.Profile
}

// NewManagementClient creates a sandbox management client using the supplied
// Stripe API base URL and CLI profile.
func NewManagementClient(apiBaseURL string, profile *config.Profile) *ManagementClient {
	return &ManagementClient{APIBaseURL: apiBaseURL, Profile: profile}
}

// ListAccessible resolves the active live OAuth workspace and returns the
// sandboxes accessible beneath it.
func (c *ManagementClient) ListAccessible(ctx context.Context) ([]ManagedSandbox, error) {
	creds, err := c.resolveCredentials()
	if err != nil {
		return nil, err
	}

	workspaceContext, err := requests.GetWorkspaceContext(ctx, c.APIBaseURL, c.Profile, creds, true)
	if err != nil {
		return nil, safeDependencyError("could not resolve the active live workspace", err)
	}
	if !validLiveWorkspaceID(workspaceContext.WorkspaceID) {
		return nil, errorcategory.New(errorcategory.API, "could not resolve the active live workspace: the response was invalid")
	}

	base := &requests.Base{
		Profile:        c.Profile,
		Method:         http.MethodGet,
		SuppressOutput: true,
		APIBaseURL:     c.APIBaseURL,
		Livemode:       true,
	}
	response, err := base.MakeRequest(
		ctx,
		creds,
		accessibleSandboxesPath,
		&requests.RequestParameters{},
		map[string]interface{}{
			"live_compartment_parent_id":            workspaceContext.WorkspaceID,
			"recursively_resolve":                   false,
			"include_legacy_testmode":               false,
			"check_user_sandbox_management_actions": false,
			"include_is_dashboard_accessible":       false,
		},
		true,
		nil,
	)
	if err != nil {
		return nil, safeDependencyError("could not list accessible sandboxes", err)
	}

	var parsed accessibleSandboxesResponse
	if err := json.Unmarshal(response, &parsed); err != nil {
		return nil, errorcategory.New(errorcategory.API, "could not list accessible sandboxes: the response was invalid")
	}

	return normalizeAccessibleSandboxes(parsed)
}

func (c *ManagementClient) resolveCredentials() (stripe.Credentials, error) {
	if c == nil || c.Profile == nil {
		return stripe.Credentials{}, errorcategory.New(errorcategory.Auth, "sandbox management requires an active live OAuth account; run `stripe login` first")
	}

	creds, err := c.Profile.ResolveCredentials(true)
	if err != nil {
		var mismatch *config.ActiveContextLivemodeMismatchError
		if errors.As(err, &mismatch) {
			return stripe.Credentials{}, errorcategory.UserInputErrorf("%s", mismatch.Error())
		}
		return stripe.Credentials{}, errorcategory.New(errorcategory.Auth, "sandbox management requires an active live OAuth account; run `stripe login` first")
	}

	if !strings.HasPrefix(creds.Token, "oak_") ||
		creds.OAKLivemode == nil ||
		!*creds.OAKLivemode ||
		!validAccountID(creds.OAKContext) {
		return stripe.Credentials{}, errorcategory.New(errorcategory.Auth, "sandbox management requires an active live OAuth account; run `stripe login` and select a live account")
	}

	return creds, nil
}

type accessibleSandboxesResponse struct {
	Workspaces    []accessibleSandbox   `json:"workspaces"`
	Organizations []sandboxOrganization `json:"organizations"`
}

type sandboxOrganization struct {
	Workspaces        []accessibleSandbox `json:"workspaces"`
	CompartmentLabels []compartmentLabel  `json:"compartment_labels"`
}

type accessibleSandbox struct {
	WorkspaceID       string             `json:"id"`
	AccountID         string             `json:"merchant_id"`
	Name              string             `json:"name"`
	CompartmentLabels []compartmentLabel `json:"compartment_labels"`
}

type compartmentLabel struct {
	UsageType string `json:"usage_type"`
}

const (
	sandboxAccessLabelPrivate   = "sandbox_access_level_private"
	sandboxAccessLabelGlobal    = "sandbox_access_level_global"
	sandboxAccessLabelDeveloper = "sandbox_access_level_developer"
)

func normalizeAccessibleSandboxes(response accessibleSandboxesResponse) ([]ManagedSandbox, error) {
	type sandboxWithAccessLabels struct {
		sandbox      accessibleSandbox
		accessLabels []compartmentLabel
	}

	records := make([]sandboxWithAccessLabels, 0, len(response.Workspaces))
	for _, workspace := range response.Workspaces {
		records = append(records, sandboxWithAccessLabels{
			sandbox:      workspace,
			accessLabels: workspace.CompartmentLabels,
		})
	}
	for _, organization := range response.Organizations {
		// Account sandboxes in a sandbox organization inherit the organization's
		// default access setting, which is also how Dashboard renders them.
		for _, workspace := range organization.Workspaces {
			records = append(records, sandboxWithAccessLabels{
				sandbox:      workspace,
				accessLabels: organization.CompartmentLabels,
			})
		}
	}

	validated := make([]ManagedSandbox, 0, len(records))
	for _, record := range records {
		if !validTestmodeWorkspaceID(record.sandbox.WorkspaceID) ||
			!validAccountID(record.sandbox.AccountID) ||
			strings.TrimSpace(record.sandbox.Name) == "" {
			return nil, errorcategory.New(errorcategory.API, "could not list accessible sandboxes: the response contained an invalid sandbox")
		}

		validated = append(validated, ManagedSandbox{
			WorkspaceID: record.sandbox.WorkspaceID,
			AccountID:   record.sandbox.AccountID,
			Name:        record.sandbox.Name,
			AccessLevel: sandboxAccessLevelFromLabels(record.accessLabels),
		})
	}

	byWorkspaceID := make(map[string]ManagedSandbox, len(validated))
	for _, sandbox := range validated {
		if existing, ok := byWorkspaceID[sandbox.WorkspaceID]; ok {
			if existing != sandbox {
				return nil, errorcategory.New(errorcategory.API, "could not list accessible sandboxes: the response contained conflicting sandbox records")
			}
			continue
		}
		byWorkspaceID[sandbox.WorkspaceID] = sandbox
	}

	result := make([]ManagedSandbox, 0, len(byWorkspaceID))
	for _, sandbox := range byWorkspaceID {
		result = append(result, sandbox)
	}
	sort.Slice(result, func(i, j int) bool {
		leftName := strings.ToLower(result[i].Name)
		rightName := strings.ToLower(result[j].Name)
		if leftName != rightName {
			return leftName < rightName
		}
		if result[i].AccountID != result[j].AccountID {
			return result[i].AccountID < result[j].AccountID
		}
		return result[i].WorkspaceID < result[j].WorkspaceID
	})

	return result, nil
}

func sandboxAccessLevelFromLabels(labels []compartmentLabel) SandboxAccessLevel {
	for _, label := range labels {
		if label.UsageType == sandboxAccessLabelGlobal {
			return SandboxAccessLevelGlobal
		}
	}
	for _, label := range labels {
		if label.UsageType == sandboxAccessLabelDeveloper {
			return SandboxAccessLevelDeveloper
		}
	}
	return SandboxAccessLevelPrivate
}

func validLiveWorkspaceID(id string) bool {
	return len(id) > len(workspaceIDPrefix) &&
		strings.HasPrefix(id, workspaceIDPrefix) &&
		!strings.HasPrefix(id, testmodeWorkspacePrefix)
}

func validTestmodeWorkspaceID(id string) bool {
	return len(id) > len(testmodeWorkspacePrefix) && strings.HasPrefix(id, testmodeWorkspacePrefix)
}

func validAccountID(id string) bool {
	return len(id) > len(accountIDPrefix) && strings.HasPrefix(id, accountIDPrefix)
}

func safeDependencyError(operation string, err error) error {
	if errors.Is(err, context.Canceled) {
		return errorcategory.New(errorcategory.Network, operation+": request canceled")
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return errorcategory.New(errorcategory.Network, operation+": request timed out")
	}

	if statusCode, ok := requestStatusCode(err); ok {
		switch statusCode {
		case http.StatusUnauthorized:
			return errorcategory.Errorf(errorcategory.Auth, "%s: OAuth authorization is no longer valid; run `stripe login` or reauthorize the CLI", operation)
		case http.StatusForbidden:
			return errorcategory.Errorf(errorcategory.Auth, "%s: the active OAuth account is not authorized; switch context or reauthorize the CLI", operation)
		case http.StatusTooManyRequests:
			return errorcategory.Errorf(errorcategory.RateLimit, "%s: too many requests; try again later", operation)
		default:
			return errorcategory.Errorf(errorcategory.API, "%s: the Stripe API returned an unavailable response", operation)
		}
	}

	var urlErr *url.Error
	if errors.As(err, &urlErr) {
		return errorcategory.Errorf(errorcategory.Network, "%s: network request failed", operation)
	}
	var netErr net.Error
	if errors.As(err, &netErr) {
		return errorcategory.Errorf(errorcategory.Network, "%s: network request failed", operation)
	}

	return errorcategory.Errorf(errorcategory.API, "%s: the response was invalid", operation)
}

func requestStatusCode(err error) (int, bool) {
	var value requests.RequestError
	if errors.As(err, &value) {
		return value.StatusCode, true
	}
	var pointer *requests.RequestError
	if errors.As(err, &pointer) && pointer != nil {
		return pointer.StatusCode, true
	}
	return 0, false
}
