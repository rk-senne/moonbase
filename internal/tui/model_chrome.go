package tui

import "time"

// ChromeModel holds header/animation/focus chrome state for rendering.
// Extracted from App to keep the top-level struct focused on orchestration.
type ChromeModel struct {
	Clock     string
	StartTime time.Time
	Focus     FocusPanel
	Blink     bool
	Anim      AnimState
}
