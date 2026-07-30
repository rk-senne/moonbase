package tui

import "charm.land/bubbles/v2/textinput"

// MissionModel holds the mission-briefing input and recent mission history
// for the dashboard's right panel and the mission view. Extracted from App
// to keep the top-level struct focused on orchestration.
type MissionModel struct {
	Input   textinput.Model
	History []MissionEntry
}
