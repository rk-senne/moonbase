package tui

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// AnimState holds all animation frame counters
type AnimState struct {
	frame        int  // global frame counter (increments every tick)
	radarFrame   int  // satellite spinner frame
	intelFlash   int  // frames remaining for intel flash effect
	selectPulse  int  // frames remaining for selection pulse
	revealChars  int  // characters revealed in panel reveal animation
	revealing    bool // whether a reveal is in progress
	typewriterAt int  // position in typewriter text
}

// Animation tick - fires every 150ms for smooth animation
type animTickMsg time.Time

func animTick() tea.Cmd {
	return tea.Every(150*time.Millisecond, func(t time.Time) tea.Msg {
		return animTickMsg(t)
	})
}

// Radar/satellite spinner frames
var radarFrames = []string{"◜", "◝", "◞", "◟"}

// Typing indicator frames for streaming
var typingFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

// RenderRadar returns the current radar animation frame
func (a *AnimState) RenderRadar() string {
	return radarFrames[a.radarFrame%len(radarFrames)]
}

// RenderTyping returns the current typing indicator
func (a *AnimState) RenderTyping() string {
	return typingFrames[a.frame%len(typingFrames)]
}

// IntelFlashStyle returns a highlighted style if flash is active, otherwise normal.
// Accepts a Theme to derive colours from rather than using package globals.
func (a *AnimState) IntelFlashStyle(t Theme) lipgloss.Style {
	if a.intelFlash > 0 {
		return lipgloss.NewStyle().Foreground(t.Brand).Bold(true)
	}
	return lipgloss.NewStyle().Foreground(t.Dim)
}

// PulseBadge returns a pulsing badge character
func (a *AnimState) PulseBadge() string {
	if a.selectPulse > 0 {
		pulseFrames := []string{"◉", "◎", "●", "◎"}
		return pulseFrames[a.frame%len(pulseFrames)]
	}
	return BadgeActive
}

// TypewriterText returns text revealed up to the current position
func (a *AnimState) TypewriterText(full string) string {
	if a.typewriterAt >= len(full) {
		return full
	}
	return full[:a.typewriterAt] + "█"
}

// Advance increments all animation state by one frame
func (a *AnimState) Advance() {
	a.frame++

	// Radar rotates every 3 frames
	if a.frame%3 == 0 {
		a.radarFrame++
	}

	// Decay flash/pulse counters
	if a.intelFlash > 0 {
		a.intelFlash--
	}
	if a.selectPulse > 0 {
		a.selectPulse--
	}

	// Reveal animation
	if a.revealing {
		a.revealChars += 8
	}

	// Typewriter advance
	if a.typewriterAt > 0 {
		a.typewriterAt++
	}
}

// TriggerIntelFlash starts the flash effect
func (a *AnimState) TriggerIntelFlash() {
	a.intelFlash = 6
}

// TriggerSelectPulse starts the selection pulse
func (a *AnimState) TriggerSelectPulse() {
	a.selectPulse = 8
}

// TriggerReveal starts a panel reveal
func (a *AnimState) TriggerReveal() {
	a.revealing = true
	a.revealChars = 0
}

// TriggerTypewriter starts a typewriter effect
func (a *AnimState) TriggerTypewriter() {
	a.typewriterAt = 1
}
