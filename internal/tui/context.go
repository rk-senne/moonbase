package tui

import (
	"github.com/f5508037/moonbase/internal/agents"
	"github.com/f5508037/moonbase/internal/backend"
	"github.com/f5508037/moonbase/internal/discovery"
)

// AppContext is a read-only struct passed to sub-model Update/View methods so they
// depend only on what they use (Interface Segregation), not the whole App.
// Sub-models MUST NOT mutate anything reachable through AppContext.
//
// If AppContext ever needs to grow past ~8 fields, that is a signal to split it
// by consumer (ISP) rather than widen it.
type AppContext struct {
	Registry   *agents.Registry
	Backend    backend.Backend
	ProjectCtx *discovery.ProjectContext
	Styles     Styles
	Keys       KeyMap
	Width      int
	Height     int
}

// appContext builds an AppContext from the current App state.
func (a App) appContext() AppContext {
	return AppContext{
		Registry:   a.registry,
		Backend:    a.activeBackend,
		ProjectCtx: a.projectCtx,
		Styles:     a.styles,
		Keys:       a.keys,
		Width:      a.width,
		Height:     a.height,
	}
}
