package tui

import (
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

// DashboardModel owns the roster navigation state: which agent is highlighted
// (Cursor) and which is selected (Selected) for detail views. It is a value type
// with Update method — deliberately NOT a tea.Model implementation.
type DashboardModel struct {
	Cursor   int
	Selected int
}

// NewDashboardModel constructs a DashboardModel with defaults.
func NewDashboardModel() DashboardModel {
	return DashboardModel{
		Cursor:   0,
		Selected: 0,
	}
}

// Update handles roster navigation key messages (Up/Down/JumpToAgent).
// Returns the updated model. Other dashboard keys (Enter, theme, etc.) remain
// in App's handleDashboardKeys because they need access to App-level state.
func (m DashboardModel) Update(msg tea.KeyMsg, ctx AppContext) (DashboardModel, tea.Cmd) {
	switch {
	case key.Matches(msg, ctx.Keys.Up):
		if m.Cursor > 0 {
			m.Cursor--
		}
		m.Selected = m.Cursor
	case key.Matches(msg, ctx.Keys.Down):
		if ctx.Registry != nil && m.Cursor < ctx.Registry.Count()-1 {
			m.Cursor++
		}
		m.Selected = m.Cursor
	}
	return m, nil
}

// MoveUp moves the cursor up if possible and syncs selected.
func (m DashboardModel) MoveUp() DashboardModel {
	if m.Cursor > 0 {
		m.Cursor--
	}
	m.Selected = m.Cursor
	return m
}

// MoveDown moves the cursor down within bounds and syncs selected.
func (m DashboardModel) MoveDown(maxIndex int) DashboardModel {
	if m.Cursor < maxIndex {
		m.Cursor++
	}
	m.Selected = m.Cursor
	return m
}

// JumpTo sets cursor and selected to the given index.
func (m DashboardModel) JumpTo(idx int) DashboardModel {
	m.Cursor = idx
	m.Selected = idx
	return m
}
