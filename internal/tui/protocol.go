package tui

import (
	"strings"

	"charm.land/lipgloss/v2"
)

func (a App) renderProtocol() string {
	header := a.renderHeader("Protocol")

	brand := lipgloss.NewStyle().Foreground(a.theme.Data.Brand).Bold(true)
	info := lipgloss.NewStyle().Foreground(a.theme.Data.Info)
	dim := lipgloss.NewStyle().Foreground(a.theme.Data.Dim)

	var b strings.Builder

	b.WriteString(brand.Render("  ╔══════════════════════════════════════════════════════════════╗") + "\n")
	b.WriteString(brand.Render("  ║          K.N.D. MOONBASE — OPERATIONS PROTOCOL              ║") + "\n")
	b.WriteString(brand.Render("  ╚══════════════════════════════════════════════════════════════╝") + "\n\n")

	// MODES
	b.WriteString(info.Render("  ◆ MODES") + "\n")
	b.WriteString(dim.Render("  ───────────────────────────────────────────────") + "\n")
	b.WriteString("  KND (File Browser)    Default. Browse files, preview, edit.\n")
	b.WriteString("  Terminal              ` toggle. Run shell commands inline.\n")
	b.WriteString("  COMMS                 C key. Chat with AI agents, streaming.\n")
	b.WriteString("  Pipeline              m key. Multi-phase mission execution.\n")
	b.WriteString("  Dossier               enter. View operative details + deploy.\n\n")

	// GLOBAL KEYS
	b.WriteString(info.Render("  ◆ GLOBAL KEYS") + "\n")
	b.WriteString(dim.Render("  ───────────────────────────────────────────────") + "\n")
	b.WriteString("  ?          Help overlay          F1    This protocol\n")
	b.WriteString("  q/ctrl+c   Quit                  esc   Back/close\n")
	b.WriteString("  tab        Cycle panel focus     T     Cycle theme\n")
	b.WriteString("  /          Search operatives     0-9   Jump to agent\n\n")

	// KND FILE BROWSER
	b.WriteString(info.Render("  ◆ KND FILE BROWSER") + "\n")
	b.WriteString(dim.Render("  ───────────────────────────────────────────────") + "\n")
	b.WriteString("  j/k ↑↓     Navigate files        enter  Open directory\n")
	b.WriteString("  backspace  Parent directory       e      Edit in $EDITOR\n")
	b.WriteString("  `          Switch to terminal     .      Refresh\n\n")

	// TERMINAL
	b.WriteString(info.Render("  ◆ TERMINAL") + "\n")
	b.WriteString(dim.Render("  ───────────────────────────────────────────────") + "\n")
	b.WriteString("  enter      Execute command        cd     Change directory\n")
	b.WriteString("  clear      Clear output           `      Switch to KND\n")
	b.WriteString("  esc        Exit terminal mode\n\n")

	// COMMS
	b.WriteString(info.Render("  ◆ COMMS (AI CHAT)") + "\n")
	b.WriteString(dim.Render("  ───────────────────────────────────────────────") + "\n")
	b.WriteString("  enter      Send message           ctrl+f  Attach file\n")
	b.WriteString("  ctrl+s     Snippet picker         @name   Switch agent\n")
	b.WriteString("  >name      Relay last response    >>name  Relay with message\n")
	b.WriteString("  esc        Back to dossier\n\n")

	// VIEWS
	b.WriteString(info.Render("  ◆ VIEWS") + "\n")
	b.WriteString(dim.Render("  ───────────────────────────────────────────────") + "\n")
	b.WriteString("  p          Project navigator      W      Document viewer\n")
	b.WriteString("  H          Mission history        C      Open COMMS\n")
	b.WriteString("  m          New mission            enter  Dossier\n\n")

	// TOOLS
	b.WriteString(info.Render("  ◆ TOOLS & SYSTEM") + "\n")
	b.WriteString(dim.Render("  ───────────────────────────────────────────────") + "\n")
	b.WriteString("  L          lazygit                B      btop\n")
	b.WriteString("  V          nvim                   d      git diff\n")
	b.WriteString("  M          tmux (moonbase)        F      fish shell\n")
	b.WriteString("  g          git status             w      Toggle watcher\n")
	b.WriteString("  P          Create PR (personal)   \n\n")

	// CLI
	b.WriteString(info.Render("  ◆ CLI COMMANDS") + "\n")
	b.WriteString(dim.Render("  ───────────────────────────────────────────────") + "\n")
	b.WriteString("  moonbase              Launch TUI\n")
	b.WriteString("  moonbase list         Operative roster\n")
	b.WriteString("  moonbase deploy <n>   Deploy agent by numbuh\n")
	b.WriteString("  moonbase export <id>  Export mission report\n")
	b.WriteString("  moonbase snippet      Manage snippets\n")
	b.WriteString("  moonbase install      Symlink agents to ~/.kiro/agents\n")
	b.WriteString("  echo 'task' | moonbase   Pipe mode\n\n")

	// THEMES
	b.WriteString(info.Render("  ◆ THEMES (T to cycle)") + "\n")
	b.WriteString(dim.Render("  ───────────────────────────────────────────────") + "\n")
	b.WriteString("  moonbase    Cyan/green/gold (default)\n")
	b.WriteString("  treehouse   Forest green/brown\n")
	b.WriteString("  classified  Red/crimson\n")
	b.WriteString("  nerv        Orange/purple/magenta\n")

	body := a.theme.Styles.Panel.Width(a.width - 2).Render(b.String())
	statusBar := a.renderContextualStatusBar()
	return lipgloss.JoinVertical(lipgloss.Left, header, body, statusBar)
}
