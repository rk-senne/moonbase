package tui

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/f5508037/moonbase/internal/snippets"
)

// handleCommsKeys handles key messages when the view is ViewComms.
func (a App) handleCommsKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Context file input mode
	if a.contextFile {
		switch msg.String() {
		case "esc":
			a.contextFile = false
			a.contextInput.Blur()
		case "enter":
			path := a.contextInput.Value()
			if data, err := os.ReadFile(path); err == nil {
				inject := fmt.Sprintf("[attached: %s]\n```\n%s\n```", path, string(data))
				a.commsInput.SetValue(a.commsInput.Value() + inject)
			} else {
				a.addIntel("File not found: %s", path)
			}
			a.contextFile = false
			a.contextInput.Reset()
			a.contextInput.Blur()
		default:
			var cmd tea.Cmd
			a.contextInput, cmd = a.contextInput.Update(msg)
			return a, cmd
		}
		return a, nil
	}
	// Snippet picker mode
	if a.snippetPicker {
		switch msg.String() {
		case "esc":
			a.snippetPicker = false
		case "up", "k":
			if a.snippetCursor > 0 {
				a.snippetCursor--
			}
		case "down", "j":
			if a.snippetCursor < len(a.snippetList)-1 {
				a.snippetCursor++
			}
		case "enter":
			if len(a.snippetList) > 0 {
				a.commsInput.SetValue(a.snippetList[a.snippetCursor].Content)
				a.snippetPicker = false
			}
		}
		return a, nil
	}
	switch msg.String() {
	case "esc":
		a.view = ViewDossier
		a.commsInput.Blur()
	case "enter":
		if !a.comms.streaming {
			val := a.commsInput.Value()
			// @agent — switch active agent
			if strings.HasPrefix(val, "@") {
				agentName := strings.TrimPrefix(val, "@")
				a.switchCommsAgent(agentName)
				a.commsInput.Reset()
				return a, nil
			}
			// >>agent message — relay custom message to agent
			if strings.HasPrefix(val, ">>") {
				parts := strings.SplitN(strings.TrimPrefix(val, ">>"), " ", 2)
				if len(parts) >= 2 {
					a.commsInput.Reset()
					cmd := a.relayToAgent(parts[0], parts[1])
					if cmd != nil {
						return a, cmd
					}
				}
				return a, nil
			}
			// >agent — relay last assistant response to agent
			if strings.HasPrefix(val, ">") {
				target := strings.TrimPrefix(val, ">")
				a.commsInput.Reset()
				cmd := a.relayToAgent(target, "")
				if cmd != nil {
					return a, cmd
				}
				return a, nil
			}
			cmd := a.sendCommsMessage()
			if cmd != nil {
				return a, cmd
			}
		}
	case "ctrl+f":
		a.contextFile = true
		a.contextInput.Focus()
		return a, textinput.Blink
	case "ctrl+s":
		a.snippetList = snippets.ForAgent(a.comms.agent)
		a.snippetCursor = 0
		a.snippetPicker = true
	case "ctrl+c":
		return a, tea.Quit
	default:
		if !a.comms.streaming {
			var cmd tea.Cmd
			a.commsInput, cmd = a.commsInput.Update(msg)
			return a, cmd
		}
	}
	return a, nil
}

// handleStreamChunk processes incoming stream tokens for COMMS.
func (a App) handleStreamChunk(msg streamChunkMsg) (tea.Model, tea.Cmd) {
	if a.comms == nil {
		return a, nil
	}
	if msg.err != nil {
		a.comms.buffer += fmt.Sprintf("\n[ERROR: %s]", msg.err)
		a.comms.FinishStream()
		return a, nil
	}
	if msg.done {
		a.comms.FinishStream()
		a.ringBell()
		return a, nil
	}
	a.comms.AppendStreamToken(msg.text)
	// Continue polling the stream
	return a, pollStream(a.streamCh)
}
