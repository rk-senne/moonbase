package tui

import "charm.land/bubbles/v2/textinput"

// ContextFileModel holds the file-attachment input state for COMMS.
// Extracted from App to keep the top-level struct focused on orchestration.
type ContextFileModel struct {
	Active bool
	Input  textinput.Model
}
