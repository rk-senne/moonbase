package tui

import (
	"os/exec"

	tea "charm.land/bubbletea/v2"
)

// toolCacheMsg carries a freshly computed tool-availability map back to the
// Update loop after being built off the main goroutine.
type toolCacheMsg struct{ cache map[string]bool }

// isToolAvailable checks tool availability from cache (refreshes every 30s).
func (a App) isToolAvailable(tool string) bool {
	if a.env.Infra.ToolCache == nil {
		return false
	}
	return a.env.Infra.ToolCache[tool]
}

// refreshToolCacheCmd computes tool availability in a goroutine (7+ exec.LookPath
// calls) and returns the result as a toolCacheMsg, keeping the Update loop
// responsive instead of blocking on PATH lookups every 30 seconds.
func refreshToolCacheCmd() tea.Cmd {
	return func() tea.Msg {
		return toolCacheMsg{cache: refreshToolCache()}
	}
}

// refreshToolCache updates the cached tool availability map.
func refreshToolCache() map[string]bool {
	tools := []string{"lazygit", "docker", "btop", "nvim", "cmux", "tmux", "fish"}
	cache := make(map[string]bool, len(tools))
	for _, tool := range tools {
		_, err := exec.LookPath(tool)
		cache[tool] = err == nil
	}
	return cache
}
