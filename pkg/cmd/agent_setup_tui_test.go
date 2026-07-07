package cmd

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/require"

	"github.com/stripe/stripe-cli/pkg/agentsetup"
)

func testStatuses() []agentsetup.Status {
	return []agentsetup.Status{
		{Client: "claude-code", DisplayName: "Claude Code", ExecutablePath: "/usr/local/bin/claude"},
		{Client: "cursor", DisplayName: "Cursor", ExecutablePath: "/usr/local/bin/cursor"},
		{Client: "codex", DisplayName: "Codex CLI", ExecutablePath: "/usr/local/bin/codex"},
	}
}

func pressSelect(m selectModel, code rune) selectModel {
	next, _ := m.Update(tea.KeyPressMsg{Code: code})
	return next.(selectModel)
}

func TestSelectModel_AgentsSelectedSkillsRowNot(t *testing.T) {
	m := newSelectModel(testStatuses())

	// 3 agent rows + 1 skills row.
	require.Len(t, m.rows, 4)
	for i := 0; i < 3; i++ {
		require.Equal(t, rowAgent, m.rows[i].kind)
		require.True(t, m.rows[i].selected)
	}
	require.Equal(t, rowSkills, m.rows[3].kind)
	require.False(t, m.rows[3].selected)
}

func TestSelectModel_NoAgentsPreselectsSkills(t *testing.T) {
	m := newSelectModel(nil)

	require.Len(t, m.rows, 1)
	require.Equal(t, rowSkills, m.rows[0].kind)
	require.True(t, m.rows[0].selected)
}

func TestSelectModel_SelectionSeparatesAgentsAndSkills(t *testing.T) {
	m := newSelectModel(testStatuses())

	// Toggle the skills row (index 3) on.
	m.cursor = 3
	m = pressSelect(m, tea.KeySpace)
	// Toggle Cursor (index 1) off.
	m.cursor = 1
	m = pressSelect(m, tea.KeySpace)

	sel := m.selection()
	require.True(t, sel.InstallSkills)
	require.Len(t, sel.Agents, 2)
	require.Equal(t, "claude-code", sel.Agents[0].Client)
	require.Equal(t, "codex", sel.Agents[1].Client)
}

func TestSelectModel_SelectAllTogglesEveryRow(t *testing.T) {
	m := newSelectModel(testStatuses())

	// Skills row starts unselected, so 'a' selects all.
	m = pressSelect(m, 'a')
	for _, r := range m.rows {
		require.True(t, r.selected)
	}
	// All selected -> 'a' deselects all.
	m = pressSelect(m, 'a')
	for _, r := range m.rows {
		require.False(t, r.selected)
	}
}

func TestSelectModel_CursorClampedAcrossAllRows(t *testing.T) {
	m := newSelectModel(testStatuses())

	m = pressSelect(m, tea.KeyUp)
	require.Equal(t, 0, m.cursor)

	for i := 0; i < 10; i++ {
		m = pressSelect(m, tea.KeyDown)
	}
	require.Equal(t, len(m.rows)-1, m.cursor) // clamped at the skills row
}

func TestSelectModel_EnterConfirmsQuitCancels(t *testing.T) {
	m := newSelectModel(testStatuses())
	next, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	m = next.(selectModel)
	require.True(t, m.done)
	require.False(t, m.quit)
	require.NotNil(t, cmd)

	m2 := newSelectModel(testStatuses())
	next2, cmd2 := m2.Update(tea.KeyPressMsg{Code: 'q'})
	m2 = next2.(selectModel)
	require.True(t, m2.quit)
	require.False(t, m2.done)
	require.NotNil(t, cmd2)
}

func TestScopeModel_DefaultsToLocalAndConfirms(t *testing.T) {
	m := newScopeModel()
	require.Equal(t, skillsScopeLocal, m.options[m.cursor])

	next, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	m = next.(scopeModel)
	require.Equal(t, skillsScopeGlobal, m.options[m.cursor])

	next, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	m = next.(scopeModel)
	require.True(t, m.done)
	require.NotNil(t, cmd)
}

func TestScopeModel_QuitCancels(t *testing.T) {
	m := newScopeModel()
	next, _ := m.Update(tea.KeyPressMsg{Code: 'q'})
	m = next.(scopeModel)
	require.True(t, m.quit)
	require.False(t, m.done)
}
