package tui

import (
	"fmt"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/stripe/stripe-cli/internal/markdown"
)

func TestUpdate_QuitStagesMouseReset(t *testing.T) {
	r, err := markdown.NewRenderer()
	require.NoError(t, err)

	long := "# Title\n"
	for i := range 50 {
		long += fmt.Sprintf("Line %d\n", i)
	}

	m := New(
		WithRenderer(r),
		WithPage(Page{Content: []byte(long)}),
	)
	result, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 10})
	model := result.(Model)

	result, cmd := model.Update(tea.KeyPressMsg{Code: 'q', Text: "q"})
	model = result.(Model)

	require.NotNil(t, cmd)
	assert.True(t, model.quitting)
	assert.Equal(t, tea.MouseModeNone, model.View().MouseMode)

	_, quitCmd := model.Update(quitAfterMouseResetMsg{})
	require.NotNil(t, quitCmd)
	assert.IsType(t, tea.QuitMsg{}, quitCmd())
}

func TestUpdate_QuittingIgnoresFurtherMouseInput(t *testing.T) {
	r, err := markdown.NewRenderer()
	require.NoError(t, err)

	long := "# Title\n"
	for i := range 50 {
		long += fmt.Sprintf("Line %d\n", i)
	}

	m := New(
		WithRenderer(r),
		WithPage(Page{Content: []byte(long)}),
	)
	result, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 10})
	model := result.(Model)

	result, _ = model.Update(tea.KeyPressMsg{Code: 'q', Text: "q"})
	model = result.(Model)

	offset := model.viewport.YOffset()
	result, cmd := model.Update(tea.MouseWheelMsg{Button: tea.MouseWheelDown})
	model = result.(Model)

	assert.Nil(t, cmd)
	assert.Equal(t, offset, model.viewport.YOffset())
}

func TestUpdate_QuittingIgnoresFurtherKeyInput(t *testing.T) {
	r, err := markdown.NewRenderer()
	require.NoError(t, err)

	long := "# Title\n"
	for i := range 50 {
		long += fmt.Sprintf("Line %d\n", i)
	}

	m := New(
		WithRenderer(r),
		WithPage(Page{Content: []byte(long)}),
	)
	result, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 10})
	model := result.(Model)

	result, _ = model.Update(tea.KeyPressMsg{Code: 'q', Text: "q"})
	model = result.(Model)

	offset := model.viewport.YOffset()
	result, _ = model.Update(tea.KeyPressMsg{Code: 'j', Text: "j"})
	model = result.(Model)

	assert.Equal(t, offset, model.viewport.YOffset())
}
