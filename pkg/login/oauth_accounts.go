package login

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"unicode/utf8"

	"github.com/stripe/stripe-cli/pkg/ansi"
	"github.com/stripe/stripe-cli/pkg/config"
	"github.com/stripe/stripe-cli/pkg/errorcategory"
)

type listAccountsResponse struct {
	Accounts []config.AuthorizedAccount `json:"accounts"`
	HasMore  bool                       `json:"has_more"`
}

type accountsRequestError struct {
	statusCode int
	body       string
}

func (e *accountsRequestError) Error() string {
	return fmt.Sprintf("accounts request failed (status %d): %s", e.statusCode, e.body)
}

func IsAuthorizedAccountsUnauthorized(err error) bool {
	var requestErr *accountsRequestError
	return errors.As(err, &requestErr) && requestErr.statusCode == http.StatusUnauthorized
}

// ListAuthorizedAccounts returns the Stripe accounts accessible to accessToken.
func ListAuthorizedAccounts(ctx context.Context, accessBaseURL, accessToken string) ([]config.AuthorizedAccount, error) {
	return fetchAuthorizedAccounts(ctx, accessBaseURL, accessToken)
}

func fetchAuthorizedAccounts(ctx context.Context, accessBaseURL, accessToken string) ([]config.AuthorizedAccount, error) {
	endpoint, err := url.Parse(accessBaseURL + accessAPNPath + "/token/accounts")
	if err != nil {
		return nil, err
	}

	var accounts []config.AuthorizedAccount
	startingAfter := ""
	for {
		query := endpoint.Query()
		query.Set("limit", "100")
		if startingAfter != "" {
			query.Set("starting_after", startingAfter)
		}
		endpoint.RawQuery = query.Encode()

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("Authorization", "Bearer "+accessToken)

		resp, err := accessSrvHTTPClient.Do(req)
		if err != nil {
			return nil, err
		}
		body, readErr := io.ReadAll(resp.Body)
		resp.Body.Close()
		if readErr != nil {
			return nil, readErr
		}

		if resp.StatusCode != http.StatusOK {
			return nil, errorcategory.With(&accountsRequestError{statusCode: resp.StatusCode, body: string(body)}, errorcategory.Auth)
		}

		var result listAccountsResponse
		if err := json.Unmarshal(body, &result); err != nil {
			return nil, fmt.Errorf("failed to parse accounts response: %w", err)
		}
		accounts = append(accounts, result.Accounts...)
		if !result.HasMore {
			return accounts, nil
		}
		if len(result.Accounts) == 0 {
			return nil, errorcategory.Errorf(errorcategory.API, "accounts response indicated more results but returned an empty page")
		}
		startingAfter = result.Accounts[len(result.Accounts)-1].ID
	}
}

// AuthorizedAccountsResult bundles the accounts a user has authorized with which account and
// mode are currently active.
type AuthorizedAccountsResult struct {
	Accounts        []config.AuthorizedAccount
	ActiveAccountID string
	ActiveLivemode  bool
}

// ListAuthorizedAccountsForActiveSession fetches the accounts authorized for cfg's stored OAuth
// session, along with which account and mode are currently active. Returns an error if cfg
// isn't logged in via OAuth.
func ListAuthorizedAccountsForActiveSession(ctx context.Context, accessBaseURL string, cfg *config.Config) (*AuthorizedAccountsResult, error) {
	uat, err := cfg.Profile.GetUAT()
	if err != nil {
		return nil, err
	}
	if !strings.HasPrefix(uat, "oak_") {
		return nil, errorcategory.Errorf(errorcategory.Auth, "not logged in; run 'stripe login' first")
	}

	accounts, err := ListAuthorizedAccounts(ctx, accessBaseURL, uat)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch authorized accounts: %w", err)
	}

	result := &AuthorizedAccountsResult{Accounts: accounts}
	if ac, _ := cfg.Profile.GetActiveContext(); ac != nil {
		result.ActiveAccountID = ac.AccountID
		result.ActiveLivemode = ac.Livemode
	}
	return result, nil
}

// PrintAuthorizedContexts fetches the authorized accounts for accessToken and
// prints them as a formatted list, grouped by parent account where that can
// be resolved (see PrintAuthorizedAccounts), marking the active context.
func PrintAuthorizedContexts(ctx context.Context, accessBaseURL, apiBaseURL, accessToken string, profile *config.Profile) error {
	accounts, err := ListAuthorizedAccounts(ctx, accessBaseURL, accessToken)
	if err != nil {
		return fmt.Errorf("failed to fetch authorized accounts: %w", err)
	}

	PrintAuthorizedAccounts(ctx, apiBaseURL, profile, accessToken, accounts)
	return nil
}

// PrintAuthorizedContextsList prints already-fetched authorized accounts as a
// formatted list, marking the active context.
func PrintAuthorizedContextsList(profile *config.Profile, accounts []config.AuthorizedAccount) {
	ac, _ := profile.GetActiveContext()
	activeID, activeMode := "", "test"
	if ac != nil {
		activeID = ac.AccountID
		if ac.Livemode {
			activeMode = "live"
		}
	}

	type row struct {
		name   string
		id     string
		mode   string
		active bool
	}
	var rows []row
	nameW, idW := 0, 0
	for _, a := range accounts {
		modes := a.Modes
		if len(modes) == 0 {
			modes = []string{"test"}
		}
		for _, m := range modes {
			rows = append(rows, row{
				name:   a.Name,
				id:     a.ID,
				mode:   displayMode(m),
				active: a.ID == activeID && m == activeMode,
			})
			if len(a.Name) > nameW {
				nameW = len(a.Name)
			}
			if len(a.ID) > idW {
				idW = len(a.ID)
			}
		}
	}

	color := ansi.Color(os.Stdout)
	fmt.Printf("Authorized (%d accounts):\n", len(rows))
	for _, r := range rows {
		if r.active {
			fmt.Printf("  %-*s  %-*s  %-7s  %s\n", nameW, r.name, idW, r.id, r.mode, color.Green("● active"))
		} else {
			fmt.Printf("  %-*s  %-*s  %s\n", nameW, r.name, idW, r.id, r.mode)
		}
	}
}

// displayMode maps the API's "test" mode value to the CLI's "sandbox" terminology.
func displayMode(m string) string {
	if m == "test" {
		return "sandbox"
	}
	return m
}

// Indents used when rendering accountRow. They differ in width, so printRow
// compensates to keep the id and mode columns aligned across rows.
const (
	headerIndent = "  "
	nestedIndent = "    └─ "
)

type accountRow struct {
	name   string
	id     string
	mode   string
	active bool
	// nested marks a row that belongs to another account shown above it
	// (a sandbox under its parent, or a legacy account's own test-mode row
	// under its live row), so it renders with a tree connector.
	nested bool
}

// accountHasLiveMode reports whether modes includes "live".
func accountHasLiveMode(modes []string) bool {
	for _, m := range modes {
		if m == "live" {
			return true
		}
	}
	return false
}

// orderModesLiveFirst returns modes with "live" moved to the front (other
// modes keep their relative order). For a legacy combined account, this
// guarantees the live row renders before its nested test-mode row — the API
// doesn't promise any particular mode ordering.
func orderModesLiveFirst(modes []string) []string {
	ordered := make([]string, 0, len(modes))
	for _, m := range modes {
		if m == "live" {
			ordered = append(ordered, m)
		}
	}
	for _, m := range modes {
		if m != "live" {
			ordered = append(ordered, m)
		}
	}
	return ordered
}

// rowsForAccount builds the row(s) for a single authorized account. A legacy
// combined account shares one account ID across both modes (no distinct
// sandbox account); its test-mode row is labeled "Test mode" and marked
// nested, since it belongs to the live row above it rather than standing on
// its own.
func rowsForAccount(a config.AuthorizedAccount, activeID, activeMode string) []accountRow {
	modes := a.Modes
	if len(modes) == 0 {
		modes = []string{"test"}
	}
	isLegacyCombined := accountHasLiveMode(modes)
	modes = orderModesLiveFirst(modes)

	rows := make([]accountRow, 0, len(modes))
	for _, m := range modes {
		name := a.Name
		nested := false
		if m == "test" && isLegacyCombined {
			name = "Test mode"
			nested = true
		}
		rows = append(rows, accountRow{
			name:   name,
			id:     a.ID,
			mode:   displayMode(m),
			active: a.ID == activeID && m == activeMode,
			nested: nested,
		})
	}
	return rows
}

// classifiedAccounts partitions accounts by parent: header accounts (in
// original order) and each header's nested child accounts (also in original
// order, keyed by the header's ID). Shared by every presentation of the
// authorized-accounts list (whoami, login, switch) so they all group the
// same way.
type classifiedAccounts struct {
	headers          []config.AuthorizedAccount
	childrenByParent map[string][]config.AuthorizedAccount
}

// classifyAccountsByParent sorts accounts into grouping headers (accounts
// with a live mode of their own, ones referenced as a parent, or standalone
// sandboxes with no resolvable parent) and the accounts that nest under each,
// using parentAccountIDs to resolve sandboxes to their parent. A sandbox with
// no known parent is shown as its own standalone account rather than being
// grouped under a catch-all heading.
func classifyAccountsByParent(accounts []config.AuthorizedAccount, parentAccountIDs map[string]string) classifiedAccounts {
	byID := make(map[string]config.AuthorizedAccount, len(accounts))
	for _, a := range accounts {
		byID[a.ID] = a
	}

	// An account is a grouping header if it has a live mode of its own, or if
	// some sandbox resolved it as its parent.
	isHeader := make(map[string]bool, len(accounts))
	for _, a := range accounts {
		if accountHasLiveMode(a.Modes) {
			isHeader[a.ID] = true
		}
	}
	for _, parentID := range parentAccountIDs {
		if _, ok := byID[parentID]; ok {
			isHeader[parentID] = true
		}
	}

	result := classifiedAccounts{childrenByParent: make(map[string][]config.AuthorizedAccount)}
	for _, a := range accounts {
		if isHeader[a.ID] {
			result.headers = append(result.headers, a)
			continue
		}

		parentID, hasParent := parentAccountIDs[a.ID]
		if hasParent && isHeader[parentID] {
			result.childrenByParent[parentID] = append(result.childrenByParent[parentID], a)
			continue
		}

		result.headers = append(result.headers, a)
	}

	return result
}

// groupedAccountRows holds the result of grouping accounts by parent, ready
// to render. headerRows and childRows are both keyed by the parent/header
// account's ID.
type groupedAccountRows struct {
	headerRows map[string][]accountRow
	childRows  map[string][]accountRow
	totalRows  int
}

// groupAccountsByParent builds the printable rows for each account, grouped
// by parent per classifyAccountsByParent.
func groupAccountsByParent(accounts []config.AuthorizedAccount, parentAccountIDs map[string]string, activeID, activeMode string) groupedAccountRows {
	classified := classifyAccountsByParent(accounts, parentAccountIDs)

	result := groupedAccountRows{
		headerRows: make(map[string][]accountRow, len(classified.headers)),
		childRows:  make(map[string][]accountRow, len(classified.childrenByParent)),
	}

	for _, a := range classified.headers {
		rows := rowsForAccount(a, activeID, activeMode)
		result.headerRows[a.ID] = rows
		result.totalRows += len(rows)
	}
	for parentID, children := range classified.childrenByParent {
		for _, a := range children {
			rows := rowsForAccount(a, activeID, activeMode)
			for i := range rows {
				rows[i].nested = true
			}
			result.childRows[parentID] = append(result.childRows[parentID], rows...)
			result.totalRows += len(rows)
		}
	}

	return result
}

// PrintGroupedAuthorizedAccounts prints already-fetched authorized accounts as
// a formatted list, grouping sandboxes underneath their parent (live)
// account. parentAccountIDs maps a sandbox's account ID to its parent account
// ID, as resolved via the Stripe CLI context status API (see
// requests.GetContextStatuses). A sandbox missing from the map, or whose
// resolved parent isn't itself an authorized account, is shown as its own
// standalone account instead of being grouped under a catch-all heading.
func PrintGroupedAuthorizedAccounts(profile *config.Profile, accounts []config.AuthorizedAccount, parentAccountIDs map[string]string) {
	ac, _ := profile.GetActiveContext()
	activeID, activeMode := "", "test"
	if ac != nil {
		activeID = ac.AccountID
		if ac.Livemode {
			activeMode = "live"
		}
	}

	grouped := groupAccountsByParent(accounts, parentAccountIDs, activeID, activeMode)

	nameW, idW := 0, 0
	measure := func(rows []accountRow) {
		for _, r := range rows {
			if len(r.name) > nameW {
				nameW = len(r.name)
			}
			if len(r.id) > idW {
				idW = len(r.id)
			}
		}
	}
	for _, rows := range grouped.headerRows {
		measure(rows)
	}
	for _, rows := range grouped.childRows {
		measure(rows)
	}

	// Indents vary in width (a plain "  " header indent vs. the wider
	// "    └─ " tree connector), but the id and mode columns should still
	// line up regardless of indent. Widen the name field to make up the
	// difference, so indent width + padded name width is constant.
	maxIndentWidth := 0
	for _, indent := range []string{headerIndent, nestedIndent} {
		if w := utf8.RuneCountInString(indent); w > maxIndentWidth {
			maxIndentWidth = w
		}
	}

	color := ansi.Color(os.Stdout)
	printRow := func(indent string, r accountRow) {
		namePad := nameW + (maxIndentWidth - utf8.RuneCountInString(indent))
		if r.active {
			fmt.Printf("%s%-*s  %-*s  %-7s  %s\n", indent, namePad, r.name, idW, r.id, r.mode, color.Green("● active"))
		} else {
			fmt.Printf("%s%-*s  %-*s  %s\n", indent, namePad, r.name, idW, r.id, r.mode)
		}
	}

	fmt.Printf("Authorized (%d accounts):\n", grouped.totalRows)
	for _, a := range accounts {
		rows, ok := grouped.headerRows[a.ID]
		if !ok {
			continue
		}
		for _, r := range rows {
			indent := headerIndent
			if r.nested {
				indent = nestedIndent
			}
			printRow(indent, r)
		}
		for _, r := range grouped.childRows[a.ID] {
			printRow(nestedIndent, r)
		}
	}
}

// pickActiveContext selects the active account and livemode from the list of
// authorized accounts. It prefers the first test-mode account; if none is
// found it falls back to the first account's first mode.
func pickActiveContext(accounts []config.AuthorizedAccount) (accountID string, livemode bool) {
	for _, a := range accounts {
		for _, m := range a.Modes {
			if m == "test" {
				return a.ID, false
			}
		}
	}
	if len(accounts) > 0 {
		a := accounts[0]
		if len(a.Modes) > 0 {
			return a.ID, a.Modes[0] == "live"
		}
		return a.ID, false
	}
	return "", false
}

// populateProfileFromAccounts saves the active OAuth context and updates the
// profile with the active account's display info.
func populateProfileFromAccounts(cfg *config.Config, accounts []config.AuthorizedAccount, activeID string, activeLivemode bool) error {
	if len(accounts) == 0 {
		return errorcategory.Errorf(errorcategory.Auth, "no authorized accounts returned")
	}

	if err := cfg.Profile.SaveActiveContext(activeID, activeLivemode); err != nil {
		return err
	}

	cfg.Profile.AccountID = activeID
	for _, a := range accounts {
		if a.ID == activeID {
			cfg.Profile.DisplayName = a.Name
			break
		}
	}
	return cfg.Profile.CreateProfile()
}
