package tui

import "os/exec"

// isToolAvailable checks tool availability from cache (refreshes every 30s).
func (a App) isToolAvailable(tool string) bool {
	if a.toolCache == nil {
		return false
	}
	return a.toolCache[tool]
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
