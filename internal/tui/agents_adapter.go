package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/rk-senne/moonbase/internal/agents"
)

// AgentsLoadedMsg is the Bubbletea message sent when agents finish loading from disk.
// This lives in the TUI package to keep the agents domain layer free of TUI dependencies.
type AgentsLoadedMsg struct {
	Agents []agents.Agent
	Err    error
}

// LoadAgentsCmd returns a tea.Cmd that loads agents from the registry's directory.
func LoadAgentsCmd(reg *agents.Registry) tea.Cmd {
	return func() tea.Msg {
		err := reg.LoadSync()
		if err != nil {
			return AgentsLoadedMsg{Err: err}
		}
		return AgentsLoadedMsg{Agents: reg.All()}
	}
}

// LoadAgentsMultiCmd returns a tea.Cmd that loads agents from multiple directories.
func LoadAgentsMultiCmd(reg *agents.Registry, dirs ...string) tea.Cmd {
	return func() tea.Msg {
		err := reg.LoadMultipleDirsSync(dirs...)
		if err != nil {
			return AgentsLoadedMsg{Err: err}
		}
		return AgentsLoadedMsg{Agents: reg.All()}
	}
}
