package tui

import "github.com/rk-senne/moonbase/internal/tools"

// ToolsModel holds state for the Tools view: the curated catalog, the cursor,
// a pending install confirmation (tool ID, empty when none), and the last
// action result to surface to the operator.
type ToolsModel struct {
	Catalog []tools.Tool
	Cursor  int
	Confirm string // ID of the tool awaiting y/n install confirmation
	Result  string // last action message (installed / cancelled / manual)
}

// NewToolsModel builds a ToolsModel from the curated catalog.
func NewToolsModel() ToolsModel {
	return ToolsModel{Catalog: tools.Catalog()}
}

// toolByID returns the catalog tool with the given ID.
func (m ToolsModel) toolByID(id string) (tools.Tool, bool) {
	for _, t := range m.Catalog {
		if t.ID == id {
			return t, true
		}
	}
	return tools.Tool{}, false
}
