package tui

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/rk-senne/moonbase/internal/snippets"
)

// handleCommsKeys handles key messages when the view is ViewComms.
func (a App) handleCommsKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Context file input mode
	if a.ctxFile.Active {
		switch {
		case key.Matches(msg, a.keys.ContextCancel):
			a.ctxFile.Active = false
			a.ctxFile.Input.Blur()
		case key.Matches(msg, a.keys.ContextConfirm):
			path := a.ctxFile.Input.Value()
			if data, err := os.ReadFile(path); err == nil {
				inject := fmt.Sprintf("[attached: %s]\n```\n%s\n```", path, string(data))
				a.comms.Input.SetValue(a.comms.Input.Value() + inject)
			} else {
				a.addIntel("File not found: %s", path)
			}
			a.ctxFile.Active = false
			a.ctxFile.Input.Reset()
			a.ctxFile.Input.Blur()
		default:
			var cmd tea.Cmd
			a.ctxFile.Input, cmd = a.ctxFile.Input.Update(msg)
			return a, cmd
		}
		return a, nil
	}
	// Snippet picker mode
	if a.snippetPick.Active {
		switch {
		case key.Matches(msg, a.keys.SnippetCancel):
			a.snippetPick.Active = false
		case key.Matches(msg, a.keys.SnippetUp):
			if a.snippetPick.Cursor > 0 {
				a.snippetPick.Cursor--
			}
		case key.Matches(msg, a.keys.SnippetDown):
			if a.snippetPick.Cursor < len(a.snippetPick.List)-1 {
				a.snippetPick.Cursor++
			}
		case key.Matches(msg, a.keys.SnippetConfirm):
			if len(a.snippetPick.List) > 0 {
				a.comms.Input.SetValue(a.snippetPick.List[a.snippetPick.Cursor].Content)
				a.snippetPick.Active = false
			}
		}
		return a, nil
	}
	switch {
	case key.Matches(msg, a.keys.Back):
		a.view = ViewDossier
		a.comms.Input.Blur()
	case key.Matches(msg, a.keys.SendMessage):
		if !a.comms.State.streaming {
			val := a.comms.Input.Value()
			// @agent — switch active agent
			if strings.HasPrefix(val, "@") {
				agentName := strings.TrimPrefix(val, "@")
				a.switchCommsAgent(agentName)
				a.comms.Input.Reset()
				return a, nil
			}
			// >>agent message — relay custom message to agent
			if strings.HasPrefix(val, ">>") {
				parts := strings.SplitN(strings.TrimPrefix(val, ">>"), " ", 2)
				if len(parts) >= 2 {
					a.comms.Input.Reset()
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
				a.comms.Input.Reset()
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
		a.ctxFile.Active = true
		a.ctxFile.Input.Focus()
		return a, textinput.Blink
	case key.Matches(msg, a.keys.SnippetPicker):
		a.snippetPick.List = snippets.ForAgent(a.comms.State.agent)
		a.snippetPick.Cursor = 0
		a.snippetPick.Active = true
	case key.Matches(msg, a.keys.CommsQuit):
		return a, tea.Quit
	default:
		if !a.comms.State.streaming {
			var cmd tea.Cmd
			a.comms.Input, cmd = a.comms.Input.Update(msg)
			return a, cmd
		}
	}
	return a, nil
}

// handleStreamChunk processes incoming stream tokens for COMMS.
func (a App) handleStreamChunk(msg streamChunkMsg) (tea.Model, tea.Cmd) {
	if a.comms.State == nil {
		return a, nil
	}
	if msg.err != nil {
		a.comms.State.buffer += fmt.Sprintf("\n[ERROR: %s]", msg.err)
		a.comms.State.FinishStream(a.themeData)
		return a, nil
	}
	if msg.done {
		a.comms.State.FinishStream(a.themeData)
		a.ringBell()
		return a, nil
	}
	a.comms.State.AppendStreamToken(msg.text, a.themeData)
	// Continue polling the stream
	return a, pollStream(a.pipeline.StreamCh)
}
