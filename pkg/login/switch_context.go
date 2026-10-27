package login

import (
	"context"
	"fmt"
	"strings"
	"unicode/utf8"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/stripe/stripe-cli/pkg/config"
	"github.com/stripe/stripe-cli/pkg/errorcategory"
)

// SwitchResult describes the account and mode a SwitchContext call activated.
type SwitchResult struct {
	Account config.AuthorizedAccount
	// Mode is the API's mode value ("test" or "live"); use DisplayMode for the CLI's
	// user-facing terminology.
	Mode string
}

// DisplayMode maps Mode to the CLI's user-facing terminology, e.g. "test" -> "sandbox".
func (r *SwitchResult) DisplayMode() string {
	return displayMode(r.Mode)
}

// SwitchContext updates the active OAuth context and returns the account and mode that became
// active. If accountID is non-empty it selects that account directly (test mode by default, live
// if livemode is true). Otherwise it shows an interactive list, returning a nil result if the
// user cancels it.
func SwitchContext(ctx context.Context, accessBaseURL, apiBaseURL string, cfg *config.Config, accountID string, livemode bool) (*SwitchResult, error) {
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

	if accountID != "" {
		return switchByID(cfg, accounts, accountID, livemode)
	}

	ac, _ := cfg.Profile.GetActiveContext()
	parentAccountIDs, _ := ResolveParentAccounts(ctx, apiBaseURL, &cfg.Profile, uat, ac, accounts)
	return runSwitchContextTUI(cfg, accounts, parentAccountIDs)
}

func switchByID(cfg *config.Config, accounts []config.AuthorizedAccount, accountID string, livemode bool) (*SwitchResult, error) {
	wantMode := "test"
	if livemode {
		wantMode = "live"
	}
	for _, a := range accounts {
		if a.ID != accountID {
			continue
		}
		for _, m := range a.Modes {
			if m == wantMode {
				if err := applyContext(cfg, a, m); err != nil {
					return nil, err
				}
				return &SwitchResult{Account: a, Mode: m}, nil
			}
		}
		if livemode {
			return nil, errorcategory.Errorf(errorcategory.Auth, "Account %s does not have live mode access", accountID)
		}
		return nil, errorcategory.Errorf(errorcategory.Auth, "Account %s does not have sandbox access; use --live to switch to live mode", accountID)
	}
	return nil, errorcategory.Errorf(errorcategory.Auth, "Account %s not found in your authorized accounts", accountID)
}

func applyContext(cfg *config.Config, account config.AuthorizedAccount, mode string) error {
	livemode := mode == "live"
	if err := cfg.Profile.SaveActiveContext(account.ID, livemode); err != nil {
		return err
	}
	// Preserve the existing UAT: writeProfile removes the keychain entry when UAT is empty.
	uat, _ := cfg.Profile.GetUAT()
	cfg.Profile.UAT = uat
	cfg.Profile.AccountID = account.ID
	cfg.Profile.DisplayName = account.Name
	return cfg.Profile.CreateProfile()
}

// --- TUI ---

// switchNestedIndent marks a row that belongs to another account shown above
// it (a sandbox under its parent, or a legacy account's own test-mode row
// under its live row) — the same tree-connector convention as
// PrintGroupedAuthorizedAccounts, scaled down for the TUI's tighter layout.
const switchNestedIndent = "└─ "

var switchIndentWidth = utf8.RuneCountInString(switchNestedIndent)

type switchRow struct {
	name   string
	id     string
	mode   string
	active bool
	nested bool
}

type switchContextModel struct {
	rows   []switchRow
	cursor int
	nameW  int
	idW    int
	done   bool
	quit   bool
}

// switchRowsForAccount builds the row(s) for a single authorized account,
// matching the grouping rules PrintGroupedAuthorizedAccounts uses: a legacy
// combined account's test-mode row is labeled "Test mode" and nested under
// its own live row.
func switchRowsForAccount(a config.AuthorizedAccount, nested bool, activeID, activeMode string) []switchRow {
	modes := a.Modes
	if len(modes) == 0 {
		modes = []string{"test"}
	}
	isLegacyCombined := accountHasLiveMode(modes)
	modes = orderModesLiveFirst(modes)

	rows := make([]switchRow, 0, len(modes))
	for _, m := range modes {
		name := a.Name
		rowNested := nested
		if m == "test" && isLegacyCombined {
			name = "Test mode"
			rowNested = true
		}
		rows = append(rows, switchRow{
			name:   name,
			id:     a.ID,
			mode:   m,
			active: a.ID == activeID && m == activeMode,
			nested: rowNested,
		})
	}
	return rows
}

// buildSwitchRows builds the interactive selector's rows, grouping sandboxes
// underneath their parent (live) account the same way whoami and login list
// do (see classifyAccountsByParent). A sandbox whose parent can't be
// resolved is shown as its own standalone row rather than being grouped
// under a catch-all heading.
func buildSwitchRows(accounts []config.AuthorizedAccount, parentAccountIDs map[string]string, activeID, activeMode string) (rows []switchRow, nameW, idW int) {
	classified := classifyAccountsByParent(accounts, parentAccountIDs)

	for _, a := range classified.headers {
		rows = append(rows, switchRowsForAccount(a, false, activeID, activeMode)...)
		for _, child := range classified.childrenByParent[a.ID] {
			rows = append(rows, switchRowsForAccount(child, true, activeID, activeMode)...)
		}
	}

	for _, r := range rows {
		if len(r.name) > nameW {
			nameW = len(r.name)
		}
		if len(r.id) > idW {
			idW = len(r.id)
		}
	}
	return rows, nameW, idW
}

func newSwitchContextModel(accounts []config.AuthorizedAccount, profile *config.Profile, parentAccountIDs map[string]string) switchContextModel {
	activeID, activeMode := "", "test"
	if ac, _ := profile.GetActiveContext(); ac != nil {
		activeID = ac.AccountID
		if ac.Livemode {
			activeMode = "live"
		}
	}

	rows, nameW, idW := buildSwitchRows(accounts, parentAccountIDs, activeID, activeMode)

	m := switchContextModel{rows: rows, nameW: nameW, idW: idW}
	for i, r := range rows {
		if r.active {
			m.cursor = i
			break
		}
	}
	return m
}

func (m switchContextModel) Init() tea.Cmd { return nil }

func (m switchContextModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	key, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return m, nil
	}
	switch {
	case key.Code == tea.KeyUp || key.Code == 'k':
		if m.cursor > 0 {
			m.cursor--
		}
	case key.Code == tea.KeyDown || key.Code == 'j':
		if m.cursor < len(m.rows)-1 {
			m.cursor++
		}
	case key.Code == tea.KeyEnter:
		m.done = true
		return m, tea.Quit
	case key.Code == tea.KeyEscape || key.Code == 'q' || (key.Code == 'c' && key.Mod == tea.ModCtrl):
		m.quit = true
		return m, tea.Quit
	}
	return m, nil
}

var (
	switchHeaderStyle  = lipgloss.NewStyle().Faint(true)
	switchDividerStyle = lipgloss.NewStyle().Faint(true)
	switchCursorStyle  = lipgloss.NewStyle().Bold(true)
	switchActiveStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("2"))
)

func (m switchContextModel) View() tea.View {
	var sb strings.Builder

	// The name column is widened by switchIndentWidth so a nested row's
	// "└─ " connector plus name lines up in the same space a top-level row's
	// bare name would occupy — otherwise the id/mode columns would drift
	// out of alignment between nested and top-level rows.
	nameRegion := m.nameW + switchIndentWidth

	// pointer(2) + nameRegion + sep(2) + id(idW) + sep(2) = nameRegion+idW+6 before mode
	// Header "Account" starts at col 2 (after pointer space), "Mode" at nameRegion+idW+6
	header := fmt.Sprintf("  %-*sMode", nameRegion+m.idW+4, "Account")
	sb.WriteString(switchHeaderStyle.Render(header))
	sb.WriteByte('\n')

	divLen := 2 + nameRegion + 2 + m.idW + 2 + 7 + 2 + 8 // ends after "● active"
	sb.WriteString(switchDividerStyle.Render(strings.Repeat("─", divLen)))
	sb.WriteByte('\n')

	for i, r := range m.rows {
		pointer := "  "
		if i == m.cursor {
			pointer = "▶ "
		}
		indent := ""
		if r.nested {
			indent = switchNestedIndent
		}
		namePad := nameRegion - utf8.RuneCountInString(indent)
		activeMarker := ""
		if r.active {
			activeMarker = "  " + switchActiveStyle.Render("● active")
		}
		line := fmt.Sprintf("%s%s%-*s  %-*s  %-7s%s", pointer, indent, namePad, r.name, m.idW, r.id, displayMode(r.mode), activeMarker)
		line = strings.TrimRight(line, " ")
		if i == m.cursor {
			line = switchCursorStyle.Render(line)
		}
		sb.WriteString(line)
		sb.WriteByte('\n')
	}

	sb.WriteString("\nRun 'stripe login' to authorize additional accounts or sandboxes.\n\n")
	sb.WriteString("↑↓ to navigate · enter to select · esc to cancel")
	return tea.NewView(sb.String())
}

func runSwitchContextTUI(cfg *config.Config, accounts []config.AuthorizedAccount, parentAccountIDs map[string]string) (*SwitchResult, error) {
	if len(accounts) == 0 {
		return nil, errorcategory.Errorf(errorcategory.Auth, "no authorized accounts found")
	}
	m := newSwitchContextModel(accounts, &cfg.Profile, parentAccountIDs)

	final, err := tea.NewProgram(m).Run()
	if err != nil {
		return nil, fmt.Errorf("context selector: %w", err)
	}
	result, ok := final.(switchContextModel)
	if !ok || result.quit || !result.done {
		return nil, nil
	}
	if result.cursor < 0 || result.cursor >= len(result.rows) {
		return nil, nil
	}
	sel := result.rows[result.cursor]
	for _, a := range accounts {
		if a.ID == sel.id {
			if err := applyContext(cfg, a, sel.mode); err != nil {
				return nil, err
			}
			return &SwitchResult{Account: a, Mode: sel.mode}, nil
		}
	}
	return nil, errorcategory.Errorf(errorcategory.Auth, "selected account not found")
}
