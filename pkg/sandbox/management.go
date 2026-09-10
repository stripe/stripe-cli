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

// AccessLevel describes the OAuth account's access in the workspace hierarchy.
type AccessLevel int

const (
	AccessLevelDirect          AccessLevel = 1
	AccessLevelSandboxChildren AccessLevel = 2
	AccessLevelNone            AccessLevel = 3
)

// ManagedSandbox is the normalized representation shared by sandbox list and
// delete. WorkspaceID is retained for later targeting but must not be shown to
// users.
type ManagedSandbox struct {
	WorkspaceID string
	AccountID   string
	Name        string
	AccessLevel AccessLevel
}

// DeleteTarget is a sandbox selected for deletion. Only user-facing account
// information is exported; the workspace and originating live account remain
// private to the management client.
type DeleteTarget struct {
	AccountID string
	Name      string

	workspaceID   string
	liveAccountID string
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
	sandboxes, _, err := c.listAccessible(ctx)
	return sandboxes, err
}

// PrepareDelete finds one sandbox account within the active live account's
// accessible sandbox scope and binds it to that live account for deletion.
func (c *ManagementClient) PrepareDelete(ctx context.Context, accountID string) (*DeleteTarget, error) {
	accountID = strings.TrimSpace(accountID)
	if !validAccountID(accountID) {
		return nil, errorcategory.New(errorcategory.UserInput, "a valid sandbox account id (acct_...) is required; run `stripe sandbox list` to find one")
	}

	sandboxes, creds, err := c.listAccessible(ctx)
	if err != nil {
		return nil, err
	}

	var match *ManagedSandbox
	for i := range sandboxes {
		if sandboxes[i].AccountID != accountID {
			continue
		}
		if match != nil {
			return nil, errorcategory.New(errorcategory.UserInput, "more than one accessible sandbox matches that account; reauthenticate and try again")
		}
		match = &sandboxes[i]
	}
	if match == nil {
		return nil, errorcategory.New(errorcategory.UserInput, "no accessible sandbox matches that account; run `stripe sandbox list` to see available sandboxes")
	}

	return &DeleteTarget{
		AccountID:     match.AccountID,
		Name:          match.Name,
		workspaceID:   match.WorkspaceID,
		liveAccountID: creds.OAKContext,
	}, nil
}

// Delete closes a prepared sandbox after verifying that the active live
// account has not changed since selection.
func (c *ManagementClient) Delete(ctx context.Context, target *DeleteTarget) error {
	if target == nil ||
		!validAccountID(target.AccountID) ||
		strings.TrimSpace(target.Name) == "" ||
		!validTestmodeWorkspaceID(target.workspaceID) ||
		!validAccountID(target.liveAccountID) {
		return errorcategory.New(errorcategory.API, "could not delete sandbox: the prepared target was invalid")
	}

	creds, err := c.resolveDeleteCredentials(target)
	if err != nil {
		return err
	}

	response, err := c.closeSandbox(ctx, target, creds)
	if statusCode, ok := requestStatusCode(err); ok && statusCode == http.StatusUnauthorized && config.OAuthTokenRefresher != nil {
		if refreshErr := config.OAuthTokenRefresher(c.Profile); refreshErr != nil {
			return errorcategory.New(errorcategory.Auth, "could not delete sandbox: OAuth authorization could not be refreshed; run `stripe login` or reauthorize the CLI")
		}
		creds, err = c.resolveDeleteCredentials(target)
		if err != nil {
			return err
		}
		response, err = c.closeSandbox(ctx, target, creds)
	}
	if err != nil {
		return safeDependencyError("could not delete sandbox", err)
	}

	var parsed struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(response, &parsed); err != nil || parsed.ID != target.workspaceID {
		return errorcategory.New(errorcategory.API, "could not delete sandbox: the response was invalid")
	}
	return nil
}

func (c *ManagementClient) listAccessible(ctx context.Context) ([]ManagedSandbox, stripe.Credentials, error) {
	creds, err := c.resolveCredentials()
	if err != nil {
		return nil, stripe.Credentials{}, err
	}

	workspaceContext, err := requests.GetWorkspaceContext(ctx, c.APIBaseURL, c.Profile, creds, true)
	if err != nil {
		return nil, stripe.Credentials{}, safeDependencyError("could not resolve the active live workspace", err)
	}
	if !validLiveWorkspaceID(workspaceContext.WorkspaceID) {
		return nil, stripe.Credentials{}, errorcategory.New(errorcategory.API, "could not resolve the active live workspace: the response was invalid")
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
		return nil, stripe.Credentials{}, safeDependencyError("could not list accessible sandboxes", err)
	}

	var parsed accessibleSandboxesResponse
	if err := json.Unmarshal(response, &parsed); err != nil {
		return nil, stripe.Credentials{}, errorcategory.New(errorcategory.API, "could not list accessible sandboxes: the response was invalid")
	}

	sandboxes, err := normalizeAccessibleSandboxes(parsed)
	if err != nil {
		return nil, stripe.Credentials{}, err
	}
	return sandboxes, creds, nil
}

func (c *ManagementClient) resolveDeleteCredentials(target *DeleteTarget) (stripe.Credentials, error) {
	creds, err := c.resolveCredentials()
	if err != nil {
		return stripe.Credentials{}, err
	}
	if creds.OAKContext != target.liveAccountID {
		return stripe.Credentials{}, errorcategory.New(errorcategory.UserInput, "could not delete sandbox because the active live account changed; run the command again")
	}
	return stripe.NewOAKCredentials(creds.Token, target.AccountID, false), nil
}

func (c *ManagementClient) closeSandbox(ctx context.Context, target *DeleteTarget, creds stripe.Credentials) ([]byte, error) {
	base := &requests.Base{
		Method:         http.MethodPost,
		SuppressOutput: true,
		APIBaseURL:     c.APIBaseURL,
		Livemode:       false,
	}
	path := "/v2/workspaces/undocumented/testmode/" + url.PathEscape(target.workspaceID) + "/close"
	return base.MakeRequest(ctx, creds, path, &requests.RequestParameters{}, nil, true, nil)
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
	Workspaces []accessibleSandbox `json:"workspaces"`
}

type accessibleSandbox struct {
	WorkspaceID string      `json:"id"`
	AccountID   string      `json:"merchant_id"`
	Name        string      `json:"name"`
	AccessLevel AccessLevel `json:"access_level"`
}

func normalizeAccessibleSandboxes(response accessibleSandboxesResponse) ([]ManagedSandbox, error) {
	records := make([]accessibleSandbox, 0, len(response.Workspaces))
	records = append(records, response.Workspaces...)
	for _, organization := range response.Organizations {
		records = append(records, organization.Workspaces...)
	}

	validated := make([]ManagedSandbox, 0, len(records))
	for _, record := range records {
		if !validTestmodeWorkspaceID(record.WorkspaceID) ||
			!validAccountID(record.AccountID) ||
			strings.TrimSpace(record.Name) == "" ||
			record.AccessLevel < AccessLevelDirect ||
			record.AccessLevel > AccessLevelNone {
			return nil, errorcategory.New(errorcategory.API, "could not list accessible sandboxes: the response contained an invalid sandbox")
		}

		validated = append(validated, ManagedSandbox(record))
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
