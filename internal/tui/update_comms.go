package tui

import (
	"fmt"
	"os"
	"strings"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"github.com/rk-senne/moonbase/internal/snippets"
)

// handleCommsKeys handles key messages when the view is ViewComms.
func (a App) handleCommsKeys(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	// Context file input mode
	if a.views.CtxFile.Active {
		switch {
		case key.Matches(msg, a.keys.ContextCancel):
			a.views.CtxFile.Active = false
			a.views.CtxFile.Input.Blur()
		case key.Matches(msg, a.keys.ContextConfirm):
			path := a.views.CtxFile.Input.Value()
			if data, err := os.ReadFile(path); err == nil {
				inject := fmt.Sprintf("[attached: %s]\n```\n%s\n```", path, string(data))
				a.views.Comms.Input.SetValue(a.views.Comms.Input.Value() + inject)
			} else {
				a.addIntel("File not found: %s", path)
			}
			a.views.CtxFile.Active = false
			a.views.CtxFile.Input.Reset()
			a.views.CtxFile.Input.Blur()
		default:
			var cmd tea.Cmd
			a.views.CtxFile.Input, cmd = a.views.CtxFile.Input.Update(msg)
			return a, cmd
		}
		return a, nil
	}
	// Snippet picker mode
	if a.views.SnippetPick.Active {
		switch {
		case key.Matches(msg, a.keys.SnippetCancel):
			a.views.SnippetPick.Active = false
		case key.Matches(msg, a.keys.SnippetUp):
			if a.views.SnippetPick.Cursor > 0 {
				a.views.SnippetPick.Cursor--
			}
		case key.Matches(msg, a.keys.SnippetDown):
			if a.views.SnippetPick.Cursor < len(a.views.SnippetPick.List)-1 {
				a.views.SnippetPick.Cursor++
			}
		case key.Matches(msg, a.keys.SnippetConfirm):
			if len(a.views.SnippetPick.List) > 0 {
				a.views.Comms.Input.SetValue(a.views.SnippetPick.List[a.views.SnippetPick.Cursor].Content)
				a.views.SnippetPick.Active = false
			}
		}
		return a, nil
	}
	switch {
	case key.Matches(msg, a.keys.Back):
		a.view = ViewDossier
		a.views.Comms.Input.Blur()
	case key.Matches(msg, a.keys.SendMessage):
		if !a.views.Comms.State.streaming {
			val := a.views.Comms.Input.Value()
			// @agent — switch active agent
			if strings.HasPrefix(val, "@") {
				agentName := strings.TrimPrefix(val, "@")
				a.switchCommsAgent(agentName)
				a.views.Comms.Input.Reset()
				return a, nil
			}
			// >>agent message — relay custom message to agent
			if strings.HasPrefix(val, ">>") {
				parts := strings.SplitN(strings.TrimPrefix(val, ">>"), " ", 2)
				if len(parts) >= 2 {
					a.views.Comms.Input.Reset()
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
				a.views.Comms.Input.Reset()
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
		a.views.CtxFile.Active = true
		a.views.CtxFile.Input.Focus()
		return a, textinput.Blink
	case key.Matches(msg, a.keys.SnippetPicker):
		a.views.SnippetPick.List = snippets.ForAgent(a.views.Comms.State.agent)
		a.views.SnippetPick.Cursor = 0
		a.views.SnippetPick.Active = true
	case key.Matches(msg, a.keys.CommsQuit):
		return a, tea.Quit
	default:
		if !a.views.Comms.State.streaming {
			var cmd tea.Cmd
			a.views.Comms.Input, cmd = a.views.Comms.Input.Update(msg)
			return a, cmd
		}
	}
	return a, nil
}

// handleStreamChunk processes incoming stream tokens for COMMS.
func (a App) handleStreamChunk(msg streamChunkMsg) (tea.Model, tea.Cmd) {
	if a.views.Comms.State == nil {
		return a, nil
	}
	if msg.err != nil {
		a.views.Comms.State.buffer += fmt.Sprintf("\n[ERROR: %s]", msg.err)
		a.views.Comms.State.FinishStream(a.theme.Data)
		return a, nil
	}
	if msg.done {
		a.views.Comms.State.FinishStream(a.theme.Data)
		a.ringBell()
		return a, nil
	}
	a.views.Comms.State.AppendStreamToken(msg.text, a.theme.Data)
	// Continue polling the stream
	return a, pollStream(a.views.Pipeline.StreamCh)
}
