package tui

import (
	"time"

	"github.com/rk-senne/moonbase/internal/platform"
	"github.com/rk-senne/moonbase/internal/watcher"
)

// InfraModel holds infrastructure/platform state — file watcher, platform
// context, and tool availability cache. Extracted from App to keep the
// top-level struct focused on orchestration.
type InfraModel struct {
	Watcher       *watcher.Watcher
	Ctx           platform.Context
	ToolCache     map[string]bool
	ToolCacheTime time.Time
}
