package tui

import (
	"github.com/rk-senne/moonbase/internal/mux"
	"github.com/rk-senne/moonbase/internal/pipeline"
)

// specialistPaneCommand builds the shell command run in a split pane for one
// triggered specialist. The KND operative is deployed via its native Kiro agent
// (loaded from .kiro/agents) with the mission task as the opening prompt, so the
// operator can watch and drive it live. Values are passed to mux.SplitRun as a
// single command string; mux hands them to the multiplexer as discrete argv
// entries (no shell-injection surface).
func specialistPaneCommand(agentName, task string) string {
	cmd := "kiro-cli chat --agent " + agentName
	if task != "" {
		cmd += " " + shellQuote(task)
	}
	return cmd
}

// shellQuote single-quotes a value for safe inclusion in the pane command string
// (the pane runs it through a shell). Embedded single quotes are escaped.
func shellQuote(s string) string {
	out := make([]byte, 0, len(s)+2)
	out = append(out, '\'')
	for i := 0; i < len(s); i++ {
		if s[i] == '\'' {
			out = append(out, '\'', '\\', '\'', '\'') // ' -> '\''
			continue
		}
		out = append(out, s[i])
	}
	out = append(out, '\'')
	return string(out)
}

// usePanesForFanOut reports whether triggered specialists should be launched into
// split panes instead of run headless: the operator opted in (config) AND a
// multiplexer session is actually available to split.
func usePanesForFanOut(optIn bool, m mux.Mux) bool {
	return optIn && m.Available() && m.InSession()
}

// specialistPaneCommands builds the ordered pane commands for triggered phases.
func specialistPaneCommands(triggered []pipeline.Phase, task string) []string {
	cmds := make([]string, 0, len(triggered))
	for _, p := range triggered {
		cmds = append(cmds, specialistPaneCommand(p.AgentName, task))
	}
	return cmds
}
