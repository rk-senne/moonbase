package tui

import "github.com/rk-senne/moonbase/internal/backend"

// BackendModel holds the available AI backends and the active selection.
// Extracted from App to keep the top-level struct focused on orchestration.
type BackendModel struct {
	Available []backend.Backend
	Active    backend.Backend
}
