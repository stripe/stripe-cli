package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/stripe/stripe-cli/pkg/agentsetup"
	"github.com/stripe/stripe-cli/pkg/agentskills"
)

const (
	skillsScopeLocal  = "local"
	skillsScopeGlobal = "global"
)

type rowKind int

const (
	rowAgent rowKind = iota
	rowSkills
)

type selectRow struct {
	kind     rowKind
	status   agentsetup.Status // set when kind == rowAgent
	label    string
	detail   string
	selected bool
	disabled bool   // shown but not selectable (e.g. unsupported version)
	hint     string // explanation shown when disabled
}

type selectModel struct {
	rows   []selectRow
	cursor int
	done   bool
	quit   bool
}

// Selection is the outcome of the agent-setup checklist.
type Selection struct {
	Agents        []agentsetup.Status
	InstallSkills bool
}

func newSelectModel(statuses []agentsetup.Status, skills skillsScopes) selectModel {
	rows := make([]selectRow, 0, len(statuses)+1)
	for _, s := range statuses {
		row := selectRow{
			kind:   rowAgent,
			status: s,
			label:  s.DisplayName,
		}
		switch {
		case s.Plugin.Installed:
			row.disabled = true
			row.hint = "plugin already installed"
		case s.Error != "" && s.Status != agentsetup.StatusError:
			row.disabled = true
			row.hint = s.Error
		default:
			row.detail = "plugin not installed"
		}
		row.selected = !row.disabled
		rows = append(rows, row)
	}

	skillsLabel, skillsDetail, skillsSelected := skillsRowPresentation(statuses, skills)
	rows = append(rows, selectRow{
		kind:     rowSkills,
		label:    skillsLabel,
		detail:   skillsDetail,
		selected: skillsSelected,
	})

	return selectModel{rows: rows}
}

func skillsRowPresentation(statuses []agentsetup.Status, skills skillsScopes) (label, detail string, selected bool) {
	label = "Install Stripe skills"

	hasOutOfDate := skillsScopeNeedsUpdate(skills.Local) || skillsScopeNeedsUpdate(skills.Global)
	if hasOutOfDate {
		return label, "Detected outdated Stripe skills", true
	}

	if skills.Local.Status == agentskills.StatusCurrent && skills.Global.Status == agentskills.StatusCurrent {
		return "Stripe skills", "up to date", false
	}

	return label, "", len(statuses) == 0
}

func (m selectModel) Init() tea.Cmd { return nil }

func (m selectModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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
	case key.Code == tea.KeySpace:
		if len(m.rows) > 0 && !m.rows[m.cursor].disabled {
			m.rows[m.cursor].selected = !m.rows[m.cursor].selected
		}
	case key.Code == 'a':
		m.toggleAll()
	case key.Code == tea.KeyEnter:
		m.done = true
		return m, tea.Quit
	case key.Code == 'q' || key.Code == tea.KeyEscape || (key.Code == 'c' && key.Mod == tea.ModCtrl):
		m.quit = true
		return m, tea.Quit
	}

	return m, nil
}

// toggleAll selects every non-disabled row when any is unselected, otherwise
// deselects all non-disabled rows.
func (m *selectModel) toggleAll() {
	anyUnselected := false
	for _, r := range m.rows {
		if !r.selected && !r.disabled {
			anyUnselected = true
			break
		}
	}
	for i := range m.rows {
		if !m.rows[i].disabled {
			m.rows[i].selected = anyUnselected
		}
	}
}

func (m selectModel) selection() Selection {
	sel := Selection{}
	for _, r := range m.rows {
		if !r.selected {
			continue
		}
		switch r.kind {
		case rowAgent:
			sel.Agents = append(sel.Agents, r.status)
		case rowSkills:
			sel.InstallSkills = true
		}
	}
	return sel
}

var (
	checkedStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("2"))
	uncheckedStyle = lipgloss.NewStyle().Faint(true)
	cursorRowStyle = lipgloss.NewStyle().Bold(true)
	dividerStyle   = lipgloss.NewStyle().Faint(true)
	containerStyle = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Padding(1, 2)
)

func (m selectModel) View() tea.View {
	body := "Select agents to install the Stripe plugin:\n\n"

	skillsDividerShown := false
	for i, r := range m.rows {
		if r.kind == rowSkills && !skillsDividerShown {
			body += "\n" + dividerStyle.Render("──────────────────── OR ────────────────────") + "\n"
			skillsDividerShown = true
		}

		pointer := "  "
		if i == m.cursor {
			pointer = "▶ "
		}

		if r.disabled {
			line := fmt.Sprintf("%s[-] %-22s %s", pointer, r.label, r.hint)
			body += line + "\n"
			continue
		}

		box := uncheckedStyle.Render("·")
		if r.selected {
			box = checkedStyle.Render("✔")
		}

		line := fmt.Sprintf("%s[%s] %-22s %s", pointer, box, r.label, r.detail)
		if i == m.cursor {
			line = cursorRowStyle.Render(line)
		}
		body += line + "\n"
	}

	body += "\nspace toggle · a select all · enter confirm · q quit"

	return tea.NewView(containerStyle.Render(body))
}

// RunSelectionTUI shows the checklist and returns the user's selection. It
// returns nil (not an error) when the user quits without confirming.
func RunSelectionTUI(statuses []agentsetup.Status, skills skillsScopes) (*Selection, error) {
	m := newSelectModel(statuses, skills)
	final, err := tea.NewProgram(m).Run()
	if err != nil {
		return nil, fmt.Errorf("agent selection: %w", err)
	}

	result, ok := final.(selectModel)
	if !ok || result.quit || !result.done {
		return nil, nil
	}
	sel := result.selection()
	return &sel, nil
}

// scopeModel is a small radio prompt for choosing where to install skills.
type scopeModel struct {
	options []string
	labels  []string
	title   string
	cursor  int
	done    bool
	quit    bool
}

func newScopeModel(skills skillsScopes) scopeModel {
	localPath := ".agents/skills"
	if cwd, err := os.Getwd(); err == nil {
		localPath = filepath.Join(cwd, ".agents", "skills")
	}
	globalPath := ".agents/skills"
	if home, err := os.UserHomeDir(); err == nil {
		globalPath = filepath.Join(home, ".agents", "skills")
	}

	title := "Install Stripe skills where?"
	if skillsScopeNeedsUpdate(skills.Local) || skillsScopeNeedsUpdate(skills.Global) {
		title = "Update Stripe skills where?"
	}

	cursor := 0
	if preferred := preferredSkillsScope(skills); preferred == skillsScopeGlobal {
		cursor = 1
	}

	return scopeModel{
		options: []string{skillsScopeLocal, skillsScopeGlobal},
		labels: []string{
			scopeOptionLabel("This project", localPath, skills.Local),
			scopeOptionLabel("Global", globalPath, skills.Global),
		},
		title:  title,
		cursor: cursor,
	}
}

func scopeOptionLabel(prefix, path string, status agentskills.DirStatus) string {
	if hint := scopeStatusHint(status); hint != "" {
		return fmt.Sprintf("%-14s %s  %s", prefix, path, hint)
	}
	return fmt.Sprintf("%-14s %s", prefix, path)
}

func scopeStatusHint(status agentskills.DirStatus) string {
	switch status.Status {
	case agentskills.StatusCurrent:
		return "(up to date)"
	case agentskills.StatusOutOfDate:
		return "(out of date)"
	case agentskills.StatusPartial:
		return "(partially installed)"
	case agentskills.StatusNotInstalled:
		return "(not installed)"
	default:
		return ""
	}
}

func preferredSkillsScope(skills skillsScopes) string {
	if skillsScopeNeedsUpdate(skills.Local) {
		return skillsScopeLocal
	}
	if skillsScopeNeedsUpdate(skills.Global) {
		return skillsScopeGlobal
	}
	if skillsScopeNeedsInstall(skills.Local) {
		return skillsScopeLocal
	}
	if skillsScopeNeedsInstall(skills.Global) {
		return skillsScopeGlobal
	}
	return skillsScopeLocal
}

func (m scopeModel) Init() tea.Cmd { return nil }

func (m scopeModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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
		if m.cursor < len(m.options)-1 {
			m.cursor++
		}
	case key.Code == tea.KeyEnter:
		m.done = true
		return m, tea.Quit
	case key.Code == 'q' || key.Code == tea.KeyEscape || (key.Code == 'c' && key.Mod == tea.ModCtrl):
		m.quit = true
		return m, tea.Quit
	}
	return m, nil
}

func (m scopeModel) View() tea.View {
	body := m.title + "\n\n"
	for i := range m.options {
		marker := "( )"
		if i == m.cursor {
			marker = "(•)"
		}
		line := fmt.Sprintf("  %s %s", marker, m.labels[i])
		if i == m.cursor {
			line = cursorRowStyle.Render(line)
		}
		body += line + "\n"
	}
	body += "\n↑/↓ move · enter confirm · q cancel"
	return tea.NewView(containerStyle.Render(body))
}

// RunSkillsScopeTUI prompts for the skills install scope, returning "local" or
// "global". ok is false when the user cancels.
func RunSkillsScopeTUI(skills skillsScopes) (scope string, ok bool, err error) {
	final, runErr := tea.NewProgram(newScopeModel(skills)).Run()
	if runErr != nil {
		return "", false, fmt.Errorf("skills scope selection: %w", runErr)
	}
	result, isModel := final.(scopeModel)
	if !isModel || result.quit || !result.done {
		return "", false, nil
	}
	return result.options[result.cursor], true, nil
}
