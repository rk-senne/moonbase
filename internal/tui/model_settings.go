package tui

import (
	"runtime"

	"github.com/rk-senne/moonbase/internal/tools"
)

// settingsActionReboot is the fixed first row in the Settings view (cursor 0):
// the "reboot & update moonbase" action.
const settingsActionReboot = 0

// settingsRowKind classifies a navigable Settings row.
type settingsRowKind int

const (
	rowReboot settingsRowKind = iota // the moonbase reboot/update action
	rowInstallAll                    // an "install all" action for one OS section
	rowTool                          // a single dev tool within an OS section
)

// settingsRow is one line in the Settings view. OS is "" for the reboot action
// and "darwin"/"linux" for the two dev-environment sections.
type settingsRow struct {
	Kind settingsRowKind
	OS   string
	Tool tools.Tool
}

// selectable reports whether the row can be focused/actioned on the current
// host. The reboot action is always selectable; a dev-tool or install-all row is
// selectable only in the section matching the running OS — the other OS section
// is shown for reference but grayed out and skipped by navigation.
func (r settingsRow) selectable(currentOS string) bool {
	switch r.Kind {
	case rowReboot:
		return true
	case rowInstallAll, rowTool:
		return r.OS == currentOS
	}
	return false
}

// settingsSectionOrder is the stable render order of the OS sections so the view
// is deterministic regardless of host (the current OS is highlighted, not
// reordered).
var settingsSectionOrder = []string{"darwin", "linux"}

// SettingsModel holds state for the Settings view: the reboot action plus two
// OS dev-environment sections (macOS, Linux). The section matching the running
// OS is active; the other is grayed out.
type SettingsModel struct {
	CurrentOS string
	Rows      []settingsRow
	Cursor    int    // index into Rows; navigation only lands on selectable rows
	Confirm   string // "" | "reboot" | "installall:<os>" | tool ID awaiting y/n
	Result    string // last action message

	// RebootRequested is set once a reinstall succeeds; after the tea program
	// exits, main re-execs RebootBin so the TUI relaunches on the new binary.
	RebootRequested bool
	RebootBin       string
}

// NewSettingsModel builds a SettingsModel for the running OS.
func NewSettingsModel() SettingsModel {
	return newSettingsModelFor(runtime.GOOS)
}

// newSettingsModelFor builds the model for a specific OS (testable).
func newSettingsModelFor(goos string) SettingsModel {
	rows := []settingsRow{{Kind: rowReboot}}
	for _, os := range settingsSectionOrder {
		rows = append(rows, settingsRow{Kind: rowInstallAll, OS: os})
		for _, t := range tools.ToolsForOS(os) {
			rows = append(rows, settingsRow{Kind: rowTool, OS: os, Tool: t})
		}
	}
	return SettingsModel{CurrentOS: goos, Rows: rows, Cursor: settingsActionReboot}
}

// moveCursor moves the cursor by dir (±1) to the next selectable row, staying
// put when there is none in that direction.
func (m *SettingsModel) moveCursor(dir int) {
	i := m.Cursor + dir
	for i >= 0 && i < len(m.Rows) {
		if m.Rows[i].selectable(m.CurrentOS) {
			m.Cursor = i
			return
		}
		i += dir
	}
}

// current returns the row under the cursor.
func (m SettingsModel) current() settingsRow {
	if m.Cursor < 0 || m.Cursor >= len(m.Rows) {
		return settingsRow{}
	}
	return m.Rows[m.Cursor]
}

// toolByID returns the catalog tool with the given ID from any section.
func (m SettingsModel) toolByID(id string) (tools.Tool, bool) {
	for _, r := range m.Rows {
		if r.Kind == rowTool && r.Tool.ID == id {
			return r.Tool, true
		}
	}
	return tools.Tool{}, false
}
