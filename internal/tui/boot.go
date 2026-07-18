package tui

import (
	"math/rand"
	"strings"
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

// Core boot sequence (always shown in order)
var bootCore = []string{
	"Establishing uplink to Moonbase...",
	"Authenticating operative credentials...",
	"Loading agent configurations...",
	"Scanning AI backends...",
	"Initializing Sector V...",
	"Deploying specialist division...",
}

// Easter egg messages — one is randomly inserted into boot sequence
var bootEasterEggs = []string{
	"Father detected on network... just kidding. 😏",
	"Numbuh 13 tripped over a cable. Rerouting...",
	"Supreme Leader 362 authorized your access. Welcome aboard.",
	"Numbuh 4 says: \"Just punch the code in already!\"",
	"DCFDTL intrusion attempt blocked. Nice try, Delightfuls.",
	"Hamsters are nominal. Powering treehouse generators.",
	"Soda supply: CRITICAL. Deploying resupply mission.",
	"Toiletnator attempted login. Access denied. Again.",
	"Rainbow Monkeys protocol engaged... wait, wrong channel.",
	"Numbuh 5 says Numbuh 5 thinks the system looks good.",
	"Sector Z signal detected... and lost. As always.",
	"Code module S.P.I.C.E. loaded. Spicy operations ready.",
	"Teen Ninja firewall holding. No adults detected.",
	"Moonbase gravity: stable. Don't look out the window.",
	"KND handbook rule #42: Never trust a grown-up's IDE.",
}

// bootMessages is generated fresh each boot with a random easter egg
var bootMessages []string

// cascadeChars for the matrix-style falling effect
const cascadeChars = "01アイウエオカキクケコサシスセソタチツテトナニヌネノ█▓▒░╗╔═║╚╝"

func init() {
	bootMessages = generateBootSequence()
}

func generateBootSequence() []string {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	msgs := make([]string, len(bootCore))
	copy(msgs, bootCore)

	// Insert a random easter egg at position 3 or 4
	egg := bootEasterEggs[r.Intn(len(bootEasterEggs))]
	pos := 3 + r.Intn(2)
	msgs = append(msgs[:pos], append([]string{egg}, msgs[pos:]...)...)

	// Final message
	msgs = append(msgs, "Systems online. Welcome, operative.")
	return msgs
}

// generateCascade creates a frame of the data cascade animation
func generateCascade(width, height, frame int) string {
	r := rand.New(rand.NewSource(int64(frame * 7919)))
	runes := []rune(cascadeChars)

	var b strings.Builder
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			// Characters fall from top - density decreases with frame progress
			density := 90 - (frame * 8)
			if density < 5 {
				density = 5
			}
			if r.Intn(100) < density {
				ch := runes[r.Intn(len(runes))]
				b.WriteRune(ch)
			} else {
				b.WriteRune(' ')
			}
		}
		if y < height-1 {
			b.WriteRune('\n')
		}
	}
	return b.String()
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
