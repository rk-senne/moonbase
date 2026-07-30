package tui

import "charm.land/bubbles/v2/textinput"

// SearchModel holds the operative-search/filter state for the dashboard.
// Extracted from App to keep the top-level struct focused on orchestration.
type SearchModel struct {
	Input    textinput.Model
	Active   bool
	Filtered []int
}

// Reset clears the search state, blurring and emptying the input and
// discarding any filtered indices.
func (m *SearchModel) Reset() {
	m.Active = false
	m.Input.Reset()
	m.Input.Blur()
	m.Filtered = nil
}
