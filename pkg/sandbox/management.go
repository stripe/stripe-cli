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
	"unicode/utf16"

	"github.com/google/uuid"

	"github.com/stripe/stripe-cli/pkg/config"
	"github.com/stripe/stripe-cli/pkg/errorcategory"
	"github.com/stripe/stripe-cli/pkg/requests"
	"github.com/stripe/stripe-cli/pkg/stripe"
)

const (
	accessibleSandboxesPath = "/v2/compartments/user_accessible_sandboxes"
	createSandboxPath       = "/v2/sandboxes"
	workspaceIDPrefix       = "wksp_"
	testmodeWorkspacePrefix = "wksp_test_"
	accountIDPrefix         = "acct_"
	playgroundIDPrefix      = "play_"
	maxSandboxNameLength    = 100
	maxSandboxesCreatedCode = "max_sandboxes_created"
	maxSandboxesCreatedCopy = "Could not create sandbox: your account has reached the limit for sandboxes. Delete a sandbox to create a new one."
)

// CreateOptions describes one authenticated sandbox creation request.
type CreateOptions struct {
	Name    string
	Blank   bool
	Country string
}

// CreatedSandbox is the public identifier returned after creation.
type CreatedSandbox struct {
	AccountID string
}

// DeletedSandbox is the public result returned after deletion is confirmed.
type DeletedSandbox struct {
	AccountID string
	Name      string
}

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
	WorkspaceID      string
	AccountID        string
	Name             string
	AccessLevel      SandboxAccessLevel
	IsLegacyTestmode bool
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

// Create creates a sandbox beneath the active live OAuth account.
func (c *ManagementClient) Create(ctx context.Context, options CreateOptions) (CreatedSandbox, error) {
	name := strings.TrimSpace(options.Name)
	if name == "" {
		return CreatedSandbox{}, errorcategory.New(errorcategory.UserInput, "sandbox name cannot be blank")
	}
	// Dashboard uses JavaScript String.length for this limit, so count UTF-16
	// code units instead of Go bytes to keep the two creation surfaces aligned.
	if len(utf16.Encode([]rune(name))) > maxSandboxNameLength {
		return CreatedSandbox{}, errorcategory.New(errorcategory.UserInput, "sandbox name must be 100 characters or fewer")
	}
	if normalizedName := strings.ToLower(name); normalizedName == "test mode" || normalizedName == "testmode" {
		return CreatedSandbox{}, errorcategory.New(errorcategory.UserInput, `sandbox name cannot be "Test mode"; choose a different name`)
	}
	if options.Blank {
		if !validCountryCode(options.Country) {
			return CreatedSandbox{}, errorcategory.New(errorcategory.UserInput, "blank sandbox creation requires a two-letter country code")
		}
	} else if options.Country != "" {
		return CreatedSandbox{}, errorcategory.New(errorcategory.UserInput, "country is only valid for blank sandbox creation")
	}

	creds, err := c.resolveCredentials()
	if err != nil {
		return CreatedSandbox{}, err
	}

	playgroundContext, err := requests.GetPlaygroundContext(ctx, c.APIBaseURL, c.Profile, creds, true)
	if err != nil {
		return CreatedSandbox{}, safeDependencyError("could not resolve the active playground", err)
	}
	if !validPlaygroundID(playgroundContext.PlaygroundID) {
		return CreatedSandbox{}, errorcategory.New(errorcategory.API, "could not resolve the active playground: the response was invalid")
	}

	body := map[string]interface{}{
		"name":             name,
		"activate_sandbox": !options.Blank,
	}
	if options.Blank {
		body["business_location"] = options.Country
	} else {
		creds, err = c.resolveCredentials()
		if err != nil {
			return CreatedSandbox{}, err
		}
		workspaceContext, workspaceErr := requests.GetWorkspaceContext(ctx, c.APIBaseURL, c.Profile, creds, true)
		if workspaceErr != nil {
			return CreatedSandbox{}, safeDependencyError("could not resolve the active live workspace", workspaceErr)
		}
		if !validLiveWorkspaceID(workspaceContext.WorkspaceID) {
			return CreatedSandbox{}, errorcategory.New(errorcategory.API, "could not resolve the active live workspace: the response was invalid")
		}
		body["replica_of"] = workspaceContext.WorkspaceID
	}

	creds, err = c.resolveCredentials()
	if err != nil {
		return CreatedSandbox{}, err
	}
	params := &requests.RequestParameters{}
	params.SetIdempotency(uuid.NewString())
	base := &requests.Base{
		Profile:        c.Profile,
		Method:         http.MethodPost,
		SuppressOutput: true,
		APIBaseURL:     c.APIBaseURL,
		Livemode:       true,
	}
	response, err := base.MakeRequest(
		ctx,
		creds,
		createSandboxPath,
		params,
		body,
		true,
		func(request *http.Request) error {
			request.Header.Del("Stripe-Account")
			request.Header.Set("Stripe-Context", playgroundContext.PlaygroundID)
			return nil
		},
	)
	if err != nil {
		return CreatedSandbox{}, safeCreateError(err)
	}

	var parsed struct {
		AccountID string `json:"v1_account_id"`
	}
	if err := json.Unmarshal(response, &parsed); err != nil || !validAccountID(parsed.AccountID) {
		return CreatedSandbox{}, errorcategory.New(errorcategory.API, "sandbox creation could not be confirmed; check Dashboard before retrying")
	}

	return CreatedSandbox{AccountID: parsed.AccountID}, nil
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
			"include_legacy_testmode":               true,
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

// Delete closes a sandbox beneath the active live OAuth account.
func (c *ManagementClient) Delete(ctx context.Context, accountID string) (DeletedSandbox, error) {
	accountID = strings.TrimSpace(accountID)
	if !validAccountID(accountID) {
		return DeletedSandbox{}, errorcategory.New(errorcategory.UserInput, "sandbox account must be an account id (acct_...)")
	}

	sandboxes, err := c.ListAccessible(ctx)
	if err != nil {
		return DeletedSandbox{}, err
	}

	matches := make([]ManagedSandbox, 0, 1)
	for _, managedSandbox := range sandboxes {
		if managedSandbox.AccountID == accountID {
			matches = append(matches, managedSandbox)
		}
	}
	if len(matches) == 0 {
		return DeletedSandbox{}, errorcategory.New(errorcategory.API, "no sandbox found under the active live account; run `stripe sandbox list` to see available sandboxes")
	}
	if len(matches) > 1 {
		return DeletedSandbox{}, errorcategory.New(errorcategory.API, "sandbox deletion could not identify a unique target; run `stripe sandbox list` before retrying")
	}

	managedSandbox := matches[0]
	if managedSandbox.IsLegacyTestmode {
		return DeletedSandbox{}, errorcategory.New(errorcategory.UserInput, "test mode cannot be deleted")
	}

	creds, err := c.resolveCredentials()
	if err != nil {
		return DeletedSandbox{}, err
	}

	path := "/v2/workspaces/undocumented/testmode/" + url.PathEscape(managedSandbox.WorkspaceID) + "/close"
	base := &requests.Base{
		Profile:        c.Profile,
		Method:         http.MethodPost,
		SuppressOutput: true,
		APIBaseURL:     c.APIBaseURL,
		Livemode:       true,
	}
	response, err := base.MakeRequest(
		ctx,
		creds,
		path,
		&requests.RequestParameters{},
		map[string]interface{}{},
		true,
		func(request *http.Request) error {
			// Livemode: true keeps credential resolution and refresh anchored to the
			// active live account, while the close itself targets the sandbox directly
			// in test mode. This callback is reapplied after a reactive refresh.
			request.Header.Del("Stripe-Account")
			request.Header.Set("Stripe-Context", managedSandbox.AccountID)
			request.Header.Set("Stripe-Livemode", "false")
			return nil
		},
	)
	if err != nil {
		return DeletedSandbox{}, safeDeleteError(err)
	}

	var parsed struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(response, &parsed); err != nil || parsed.ID != managedSandbox.WorkspaceID {
		return DeletedSandbox{}, errorcategory.New(errorcategory.API, "sandbox deletion could not be confirmed; run `stripe sandbox list` before retrying")
	}

	return DeletedSandbox{AccountID: managedSandbox.AccountID, Name: managedSandbox.Name}, nil
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
	IsLegacyTestmode  bool               `json:"is_legacy_testmode_compartment"`
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
			WorkspaceID:      record.sandbox.WorkspaceID,
			AccountID:        record.sandbox.AccountID,
			Name:             record.sandbox.Name,
			AccessLevel:      sandboxAccessLevelFromLabels(record.accessLabels),
			IsLegacyTestmode: record.sandbox.IsLegacyTestmode,
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

func validPlaygroundID(id string) bool {
	return len(id) > len(playgroundIDPrefix) && strings.HasPrefix(id, playgroundIDPrefix)
}

func validCountryCode(country string) bool {
	return len(country) == 2 &&
		country[0] >= 'A' && country[0] <= 'Z' &&
		country[1] >= 'A' && country[1] <= 'Z'
}

func safeCreateError(err error) error {
	if requestErr, ok := requestError(err); ok {
		if requestErr.ErrorCode == maxSandboxesCreatedCode {
			return errorcategory.New(errorcategory.API, maxSandboxesCreatedCopy)
		}
		return safeDependencyError("could not create sandbox", err)
	}

	category := errorcategory.API
	var urlErr *url.Error
	var netErr net.Error
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) ||
		errors.As(err, &urlErr) || errors.As(err, &netErr) {
		category = errorcategory.Network
	}
	return errorcategory.New(category, "sandbox creation could not be confirmed; check Dashboard before retrying")
}

func safeDeleteError(err error) error {
	if _, ok := requestStatusCode(err); ok {
		return safeDependencyError("could not delete sandbox", err)
	}

	category := errorcategory.API
	var urlErr *url.Error
	var netErr net.Error
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) ||
		errors.As(err, &urlErr) || errors.As(err, &netErr) {
		category = errorcategory.Network
	}

	// A close request can reach the API before a transport or timeout failure. Ask
	// the user to list before retrying so an unknown outcome is not repeated blindly.
	return errorcategory.New(category, "sandbox deletion outcome is unknown; run `stripe sandbox list` before retrying")
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
	requestErr, ok := requestError(err)
	if !ok {
		return 0, false
	}
	return requestErr.StatusCode, true
}

func requestError(err error) (requests.RequestError, bool) {
	var value requests.RequestError
	if errors.As(err, &value) {
		return value, true
	}
	var pointer *requests.RequestError
	if errors.As(err, &pointer) && pointer != nil {
		return *pointer, true
	}
	return requests.RequestError{}, false
}
