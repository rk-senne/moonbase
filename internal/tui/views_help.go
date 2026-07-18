package tui

import (
	"github.com/charmbracelet/lipgloss"
)

func (a App) renderHelp() string {
	header := a.renderHeader("Operations Manual")

	help := `
  ◆ NAVIGATION                          ◆ MISSIONS
  ─────────────────────                  ──────────────────────
  ↑↓ / jk    Navigate roster            m         New mission
  0-9        Jump to operative           n         Next phase
  enter      Open dossier / deploy       r         Retry phase
  esc        Back / close                s         Skip phase
  /          Search operatives           H         Mission history
  tab        Cycle panel focus
  q          Disconnect

  ◆ VIEWS                               ◆ TOOLS
  ─────────────────────                  ──────────────────────
  p          Project navigator           L         lazygit
  W          Document viewer             B         btop
  C          Open COMMS (chat)           V         nvim
  ?          This manual                 tab       TERMINAL (in main)
  T          Cycle theme

  ◆ COMMS (CHAT)                        ◆ SYSTEM
  ─────────────────────                  ──────────────────────
  enter      Send message                P         Create PR (personal)
  ctrl+f     Attach file                 d         Git diff
  ctrl+s     Snippet picker              g         Git status
  @name      Switch agent                w         Toggle file watcher
  >name      Relay last response to agent
  >>name msg Relay custom message to agent
  esc        Back to dossier

  ◆ THE KND WAY
  ──────────────────────
  Sector V = core pipeline.  Specialists = cross-cutting.
  Council = full lifecycle.  "We fight for kids everywhere."`

	body := lipgloss.NewStyle().Foreground(ColorInfo).Render(help)
	statusBar := a.renderStatusBar("[esc] CLOSE MANUAL")
	return lipgloss.JoinVertical(lipgloss.Left, header, "\n"+body+"\n", statusBar)
}
