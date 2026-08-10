package login

import (
	"context"
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/stripe/stripe-cli/pkg/config"
)

// SwitchContext updates the active OAuth context. If accountID is non-empty
// it selects that account directly (test mode by default, live if livemode is
// true). Otherwise it shows an interactive list.
func SwitchContext(ctx context.Context, accessBaseURL string, cfg *config.Config, accountID string, livemode bool) error {
	uat, err := cfg.Profile.GetUAT()
	if err != nil {
		return err
	}
	if !strings.HasPrefix(uat, "oak_") {
		return fmt.Errorf("not logged in with OAuth; run 'stripe login' first")
	}

	accounts, err := ListAuthorizedAccounts(ctx, accessBaseURL, uat)
	if err != nil {
		return fmt.Errorf("failed to fetch authorized accounts: %w", err)
	}

	if accountID != "" {
		return switchByID(cfg, accounts, accountID, livemode)
	}
	return runSwitchContextTUI(cfg, accounts)
}

func switchByID(cfg *config.Config, accounts []config.AuthorizedAccount, accountID string, livemode bool) error {
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
				return applyContext(cfg, a, m)
			}
		}
		if livemode {
			return fmt.Errorf("account %s is not authorized for livemode", accountID)
		}
		return fmt.Errorf("account %s is not authorized for test mode; use --live to switch to livemode", accountID)
	}
	return fmt.Errorf("account %s not found in your authorized accounts", accountID)
}

func applyContext(cfg *config.Config, account config.AuthorizedAccount, mode string) error {
	livemode := mode == "live"
	if err := config.SaveActiveContext(account.ID, livemode); err != nil {
		return err
	}
	// Preserve the existing UAT: writeProfile removes the keychain entry when UAT is empty.
	uat, _ := cfg.Profile.GetUAT()
	cfg.Profile.UAT = uat
	cfg.Profile.AccountID = account.ID
	cfg.Profile.DisplayName = account.Name
	if err := cfg.Profile.CreateProfile(); err != nil {
		return err
	}
	fmt.Printf("Active context: %s · %s (%s)\n", account.Name, mode, account.ID)
	return nil
}

// --- TUI ---

type switchRow struct {
	isSeparator    bool
	isSectionLabel bool
	label          string
	name           string
	id             string
	mode           string
	active         bool
}

func (r switchRow) selectable() bool {
	return !r.isSeparator && !r.isSectionLabel
}

type switchContextModel struct {
	rows   []switchRow
	cursor int
	nameW  int
	idW    int
	done   bool
	quit   bool
}

func buildSwitchRows(accounts []config.AuthorizedAccount, activeID, activeMode string) (rows []switchRow, nameW, idW int) {
	var liveRows, sandboxRows []switchRow
	for _, a := range accounts {
		for _, m := range a.Modes {
			row := switchRow{
				name:   a.Name,
				id:     a.ID,
				mode:   m,
				active: a.ID == activeID && m == activeMode,
			}
			if len(a.Name) > nameW {
				nameW = len(a.Name)
			}
			if len(a.ID) > idW {
				idW = len(a.ID)
			}
			if m == "live" {
				liveRows = append(liveRows, row)
			} else {
				sandboxRows = append(sandboxRows, row)
			}
		}
	}

	rows = append(rows, liveRows...)
	if len(sandboxRows) > 0 {
		if len(rows) > 0 {
			rows = append(rows, switchRow{isSeparator: true})
		}
		rows = append(rows, switchRow{isSectionLabel: true, label: "Sandboxes"})
		rows = append(rows, sandboxRows...)
	}
	return rows, nameW, idW
}

func newSwitchContextModel(accounts []config.AuthorizedAccount) switchContextModel {
	activeID, activeMode := "", "test"
	if ac, _ := config.GetActiveContext(); ac != nil {
		activeID = ac.AccountID
		if ac.Livemode {
			activeMode = "live"
		}
	}

	rows, nameW, idW := buildSwitchRows(accounts, activeID, activeMode)

	m := switchContextModel{rows: rows, nameW: nameW, idW: idW, cursor: -1}
	for i, r := range rows {
		if !r.selectable() {
			continue
		}
		if m.cursor == -1 {
			m.cursor = i
		}
		if r.active {
			m.cursor = i
			break
		}
	}
	if m.cursor == -1 {
		m.cursor = 0
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
		for i := m.cursor - 1; i >= 0; i-- {
			if m.rows[i].selectable() {
				m.cursor = i
				break
			}
		}
	case key.Code == tea.KeyDown || key.Code == 'j':
		for i := m.cursor + 1; i < len(m.rows); i++ {
			if m.rows[i].selectable() {
				m.cursor = i
				break
			}
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
	switchSectionStyle = lipgloss.NewStyle().Faint(true)
	switchCursorStyle  = lipgloss.NewStyle().Bold(true)
	switchActiveStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("2"))
)

func (m switchContextModel) View() tea.View {
	var sb strings.Builder

	// pointer(2) + name(nameW) + sep(2) + id(idW) + sep(2) = nameW+idW+6 before mode
	// Header "Account" starts at col 2 (after pointer space), "Mode" at nameW+idW+6
	header := fmt.Sprintf("  %-*sMode", m.nameW+m.idW+4, "Account")
	sb.WriteString(switchHeaderStyle.Render(header))
	sb.WriteByte('\n')

	divLen := 2 + m.nameW + 2 + m.idW + 2 + 7 + 2 + 8 // ends after "● active"
	sb.WriteString(switchDividerStyle.Render(strings.Repeat("─", divLen)))
	sb.WriteByte('\n')

	for i, r := range m.rows {
		switch {
		case r.isSeparator:
			sb.WriteString(switchDividerStyle.Render("──"))
			sb.WriteByte('\n')
		case r.isSectionLabel:
			sb.WriteString(switchSectionStyle.Render(r.label))
			sb.WriteByte('\n')
		default:
			pointer := "  "
			if i == m.cursor {
				pointer = "▶ "
			}
			activeMarker := ""
			if r.active {
				activeMarker = "  " + switchActiveStyle.Render("● active")
			}
			line := fmt.Sprintf("%s%-*s  %-*s  %-7s%s", pointer, m.nameW, r.name, m.idW, r.id, r.mode, activeMarker)
			line = strings.TrimRight(line, " ")
			if i == m.cursor {
				line = switchCursorStyle.Render(line)
			}
			sb.WriteString(line)
			sb.WriteByte('\n')
		}
	}

	sb.WriteString("\n↑↓ to navigate · enter to select · esc to cancel")
	return tea.NewView(sb.String())
}

func runSwitchContextTUI(cfg *config.Config, accounts []config.AuthorizedAccount) error {
	if len(accounts) == 0 {
		return fmt.Errorf("no authorized accounts found")
	}
	m := newSwitchContextModel(accounts)

	final, err := tea.NewProgram(m).Run()
	if err != nil {
		return fmt.Errorf("context selector: %w", err)
	}
	result, ok := final.(switchContextModel)
	if !ok || result.quit || !result.done {
		return nil
	}
	if result.cursor < 0 || result.cursor >= len(result.rows) {
		return nil
	}
	sel := result.rows[result.cursor]
	for _, a := range accounts {
		if a.ID == sel.id {
			return applyContext(cfg, a, sel.mode)
		}
	}
	return fmt.Errorf("selected account not found")
}
