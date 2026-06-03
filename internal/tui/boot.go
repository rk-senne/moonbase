package tui

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

const kndLogo = `
    ██╗  ██╗ ███╗   ██╗ ██████╗ 
    ██║ ██╔╝ ████╗  ██║ ██╔══██╗
    █████╔╝  ██╔██╗ ██║ ██║  ██║
    ██╔═██╗  ██║╚██╗██║ ██║  ██║
    ██║  ██╗ ██║ ╚████║ ██████╔╝
    ╚═╝  ╚═╝ ╚═╝  ╚═══╝ ╚═════╝ 
`

const moonbaseLogo = `
  ╔════════════════════════════════════════════════╗
  ║   K I D S   N E X T   D O O R               ║
  ║   TACTICAL OPERATIONS NETWORK                ║
  ╠════════════════════════════════════════════════╣
  ║            🌙 M O O N B A S E 🌙             ║
  ╚════════════════════════════════════════════════╝
`

var bootMessages = []string{
	"Establishing uplink to Moonbase...",
	"Authenticating operative credentials...",
	"Loading agent configurations...",
	"Scanning AI backends...",
	"Initializing Sector V...",
	"Deploying specialist division...",
	"Systems online. Welcome, operative.",
}

type bootTickMsg struct{}
type bootDoneMsg struct{}

func bootTick() tea.Cmd {
	return tea.Tick(300*time.Millisecond, func(t time.Time) tea.Msg {
		return bootTickMsg{}
	})
}

func bootDone() tea.Cmd {
	return tea.Tick(800*time.Millisecond, func(t time.Time) tea.Msg {
		return bootDoneMsg{}
	})
}
