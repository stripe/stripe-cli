package tui

import (
	"sync"
	"time"

	tea "charm.land/bubbletea/v2"
)

const mouseThrottleDur = 15 * time.Millisecond

// NewMouseEventFilter rate-limits mouse events before they reach the model.
// This reduces the amount of in-flight wheel input during shutdown.
func NewMouseEventFilter() func(tea.Model, tea.Msg) tea.Msg {
	var (
		mu   sync.Mutex
		last time.Time
	)

	return func(_ tea.Model, msg tea.Msg) tea.Msg {
		if _, ok := msg.(tea.MouseMsg); !ok {
			return msg
		}

		mu.Lock()
		defer mu.Unlock()

		now := time.Now()
		if now.Sub(last) < mouseThrottleDur {
			return nil
		}
		last = now
		return msg
	}
}
