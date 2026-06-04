package tui

import (
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/assert"
)

func TestNewMouseEventFilter_PassesNonMouseEvents(t *testing.T) {
	filter := NewMouseEventFilter()
	msg := tea.WindowSizeMsg{Width: 80, Height: 24}
	assert.Equal(t, msg, filter(nil, msg))
}

func TestNewMouseEventFilter_PassesFirstMouseEvent(t *testing.T) {
	filter := NewMouseEventFilter()
	msg := tea.MouseWheelMsg{Button: tea.MouseWheelDown}
	assert.Equal(t, msg, filter(nil, msg))
}

func TestNewMouseEventFilter_DropsMouseEventWithinThrottle(t *testing.T) {
	filter := NewMouseEventFilter()
	msg := tea.MouseWheelMsg{Button: tea.MouseWheelDown}
	filter(nil, msg)
	assert.Nil(t, filter(nil, msg))
}

func TestNewMouseEventFilter_PassesMouseEventAfterThrottle(t *testing.T) {
	filter := NewMouseEventFilter()
	msg := tea.MouseWheelMsg{Button: tea.MouseWheelDown}
	filter(nil, msg)
	time.Sleep(mouseThrottleDur + 5*time.Millisecond)
	assert.Equal(t, msg, filter(nil, msg))
}
