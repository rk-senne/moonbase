package tui

import "github.com/rk-senne/moonbase/internal/tools"

// settingsActionReboot is the fixed first row in the Settings view (cursor 0):
// the "reboot & update moonbase" action. Catalog entries follow at cursor 1..N.
const settingsActionReboot = 0

// SettingsModel holds state for the Settings view: the reboot action plus the
// broader dev-environment catalog (Homebrew + runtimes + terminal tools).
type SettingsModel struct {
	Catalog []tools.Tool
	Cursor  int    // 0 = reboot action; 1..len(Catalog) index Catalog[cursor-1]
	Confirm string // "" | "reboot" | tool ID awaiting y/n
	Result  string // last action message

	// RebootRequested is set once a reinstall succeeds; after the tea program
	// exits, main re-execs RebootBin so the TUI relaunches on the new binary.
	RebootRequested bool
	RebootBin       string
}

// NewSettingsModel builds a SettingsModel from the dev catalog.
func NewSettingsModel() SettingsModel {
	return SettingsModel{Catalog: tools.DevCatalog()}
}

// rowCount is the number of navigable rows (reboot action + catalog).
func (m SettingsModel) rowCount() int { return len(m.Catalog) + 1 }

// selectedTool returns the catalog tool under the cursor, or ok=false when the
// cursor is on the reboot action.
func (m SettingsModel) selectedTool() (tools.Tool, bool) {
	if m.Cursor <= settingsActionReboot || m.Cursor-1 >= len(m.Catalog) {
		return tools.Tool{}, false
	}
	return m.Catalog[m.Cursor-1], true
}

// toolByID returns the catalog tool with the given ID.
func (m SettingsModel) toolByID(id string) (tools.Tool, bool) {
	for _, t := range m.Catalog {
		if t.ID == id {
			return t, true
		}
	}
	return tools.Tool{}, false
}
