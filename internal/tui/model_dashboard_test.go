package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestDashboardModel_Update_Up(t *testing.T) {
	reg := newTestRegistry()

	ctx := AppContext{
		Keys:     DefaultKeyMap(),
		Registry: reg,
	}

	m := DashboardModel{Cursor: 2, Selected: 2}
	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")}, ctx)

	if result.Cursor != 1 {
		t.Errorf("expected Cursor=1, got %d", result.Cursor)
	}
	if result.Selected != 1 {
		t.Errorf("expected Selected=1, got %d", result.Selected)
	}
}

func TestDashboardModel_Update_UpAtZero(t *testing.T) {
	ctx := AppContext{
		Keys:     DefaultKeyMap(),
		Registry: nil,
	}

	m := DashboardModel{Cursor: 0, Selected: 0}
	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")}, ctx)

	if result.Cursor != 0 {
		t.Errorf("expected Cursor=0 (no underflow), got %d", result.Cursor)
	}
}

func TestDashboardModel_Update_Down(t *testing.T) {
	reg := newTestRegistry()

	ctx := AppContext{
		Keys:     DefaultKeyMap(),
		Registry: reg,
	}

	m := DashboardModel{Cursor: 0, Selected: 0}
	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")}, ctx)

	if result.Cursor != 1 {
		t.Errorf("expected Cursor=1, got %d", result.Cursor)
	}
	if result.Selected != 1 {
		t.Errorf("expected Selected=1, got %d", result.Selected)
	}
}

func TestDashboardModel_Update_DownAtMax(t *testing.T) {
	reg := newTestRegistry()
	max := reg.Count() - 1

	ctx := AppContext{
		Keys:     DefaultKeyMap(),
		Registry: reg,
	}

	m := DashboardModel{Cursor: max, Selected: max}
	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")}, ctx)

	if result.Cursor != max {
		t.Errorf("expected Cursor=%d (no overflow), got %d", max, result.Cursor)
	}
}

func TestDashboardModel_Update_UnhandledKey(t *testing.T) {
	ctx := AppContext{
		Keys:     DefaultKeyMap(),
		Registry: nil,
	}

	m := DashboardModel{Cursor: 3, Selected: 3}
	result, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("x")}, ctx)

	if result.Cursor != 3 {
		t.Errorf("expected Cursor=3 (unchanged), got %d", result.Cursor)
	}
	if cmd != nil {
		t.Error("expected nil cmd for unhandled key")
	}
}

func TestDashboardModel_MoveUp(t *testing.T) {
	tests := []struct {
		name       string
		initial    int
		wantCursor int
	}{
		{"from 3 to 2", 3, 2},
		{"from 1 to 0", 1, 0},
		{"from 0 stays 0", 0, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := DashboardModel{Cursor: tt.initial, Selected: tt.initial}
			result := m.MoveUp()
			if result.Cursor != tt.wantCursor {
				t.Errorf("Cursor = %d, want %d", result.Cursor, tt.wantCursor)
			}
			if result.Selected != result.Cursor {
				t.Errorf("Selected = %d, want %d (same as Cursor)", result.Selected, result.Cursor)
			}
		})
	}
}

func TestDashboardModel_MoveDown(t *testing.T) {
	tests := []struct {
		name       string
		initial    int
		maxIndex   int
		wantCursor int
	}{
		{"from 0 to 1 (max 5)", 0, 5, 1},
		{"from 4 to 5 (max 5)", 4, 5, 5},
		{"at max stays", 5, 5, 5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := DashboardModel{Cursor: tt.initial, Selected: tt.initial}
			result := m.MoveDown(tt.maxIndex)
			if result.Cursor != tt.wantCursor {
				t.Errorf("Cursor = %d, want %d", result.Cursor, tt.wantCursor)
			}
			if result.Selected != result.Cursor {
				t.Errorf("Selected = %d, want %d (same as Cursor)", result.Selected, result.Cursor)
			}
		})
	}
}

func TestDashboardModel_JumpTo(t *testing.T) {
	m := DashboardModel{Cursor: 0, Selected: 0}
	result := m.JumpTo(7)

	if result.Cursor != 7 {
		t.Errorf("Cursor = %d, want 7", result.Cursor)
	}
	if result.Selected != 7 {
		t.Errorf("Selected = %d, want 7", result.Selected)
	}
}

func TestNewDashboardModel(t *testing.T) {
	m := NewDashboardModel()
	if m.Cursor != 0 {
		t.Errorf("expected Cursor=0, got %d", m.Cursor)
	}
	if m.Selected != 0 {
		t.Errorf("expected Selected=0, got %d", m.Selected)
	}
}

func TestDashboardModel_Update_DownNilRegistry(t *testing.T) {
	ctx := AppContext{
		Keys:     DefaultKeyMap(),
		Registry: nil,
	}

	m := DashboardModel{Cursor: 0, Selected: 0}
	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")}, ctx)

	// With nil registry, should not advance (Count-1 would panic)
	if result.Cursor != 0 {
		t.Errorf("expected Cursor=0 with nil registry, got %d", result.Cursor)
	}
}
