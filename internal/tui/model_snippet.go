package tui

import "github.com/rk-senne/moonbase/internal/snippets"

// SnippetPickerModel holds the snippet-picker overlay state for COMMS.
// Extracted from App to keep the top-level struct focused on orchestration.
type SnippetPickerModel struct {
	Active bool
	List   []snippets.Snippet
	Cursor int
}
