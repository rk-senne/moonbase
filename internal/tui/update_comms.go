package tui

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/f5508037/moonbase/internal/snippets"
)

// handleCommsKeys handles key messages when the view is ViewComms.
func (a App) handleCommsKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Context file input mode
	if a.contextFile {
		switch {
		case key.Matches(msg, a.keys.ContextCancel):
			a.contextFile = false
			a.contextInput.Blur()
		case key.Matches(msg, a.keys.ContextConfirm):
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
		switch {
		case key.Matches(msg, a.keys.SnippetCancel):
			a.snippetPicker = false
		case key.Matches(msg, a.keys.SnippetUp):
			if a.snippetCursor > 0 {
				a.snippetCursor--
			}
		case key.Matches(msg, a.keys.SnippetDown):
			if a.snippetCursor < len(a.snippetList)-1 {
				a.snippetCursor++
			}
		case key.Matches(msg, a.keys.SnippetConfirm):
			if len(a.snippetList) > 0 {
				a.commsInput.SetValue(a.snippetList[a.snippetCursor].Content)
				a.snippetPicker = false
			}
		}
		return a, nil
	}
	switch {
	case key.Matches(msg, a.keys.Back):
		a.view = ViewDossier
		a.commsInput.Blur()
	case key.Matches(msg, a.keys.SendMessage):
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
	case key.Matches(msg, a.keys.AttachFile):
		a.contextFile = true
		a.contextInput.Focus()
		return a, textinput.Blink
	case key.Matches(msg, a.keys.SnippetPicker):
		a.snippetList = snippets.ForAgent(a.comms.agent)
		a.snippetCursor = 0
		a.snippetPicker = true
	case key.Matches(msg, a.keys.CommsQuit):
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
		a.comms.FinishStream(a.themeData)
		return a, nil
	}
	if msg.done {
		a.comms.FinishStream(a.themeData)
		a.ringBell()
		return a, nil
	}
	a.comms.AppendStreamToken(msg.text, a.themeData)
	// Continue polling the stream
	return a, pollStream(a.pipeline.StreamCh)
}
