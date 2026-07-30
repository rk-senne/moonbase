package tui

import "charm.land/bubbles/v2/textinput"

// CommsModel holds the COMMS (AI chat) view state and its input. State is lazily
// created when the view is first opened; Input is always valid.
type CommsModel struct {
	State *CommsState
	Input textinput.Model
}
