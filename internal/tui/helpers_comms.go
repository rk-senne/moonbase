package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/rk-senne/moonbase/internal/agents"
	"github.com/rk-senne/moonbase/internal/chat"
)

func (a *App) openComms() {
	agent := a.registry.Get(a.views.Dashboard.Selected)

	// Guard against zero/negative dimensions
	vpWidth := a.width - 6
	vpHeight := a.height - 6
	if vpWidth < 10 {
		vpWidth = 10
	}
	if vpHeight < 4 {
		vpHeight = 4
	}

	// Load existing chat history or start fresh
	conv := chat.Load(agent.Name, agent.Prompt)
	if conv != nil {
		a.views.Comms.State = &CommsState{
			conv:     conv,
			viewport: viewport.New(vpWidth, vpHeight),
			agent:    agent.Name,
		}
		a.views.Comms.State.rebuildContent(a.theme.Data)
	} else {
		a.views.Comms.State = newCommsState(agent.Name, agent.Prompt, a.width, a.height)
	}
	a.view = ViewComms
	a.views.Comms.Input.Focus()
	a.views.Comms.Input.Width = a.width - 8
	if a.views.Comms.Input.Width < 10 {
		a.views.Comms.Input.Width = 10
	}
}

func (a *App) sendCommsMessage() tea.Cmd {
	msg := a.views.Comms.Input.Value()
	if msg == "" {
		return nil
	}
	a.views.Comms.Input.Reset()
	a.views.Comms.State.AddUserMessage(msg, a.theme.Data)
	a.views.Comms.State.streaming = true

	// Start streaming and store channel for continued polling
	a.views.Pipeline.StreamCh = chat.Stream(a.views.Comms.State.conv)
	return pollStream(a.views.Pipeline.StreamCh)
}

func pollStream(ch <-chan chat.StreamChunk) tea.Cmd {
	return func() tea.Msg {
		chunk, ok := <-ch
		if !ok {
			return streamChunkMsg{done: true}
		}
		return streamChunkMsg{text: chunk.Text, done: chunk.Done, err: chunk.Err}
	}
}

// --- Multi-Agent COMMS ---

func (a *App) switchCommsAgent(name string) {
	for _, agent := range a.registry.All() {
		if strings.Contains(strings.ToLower(agent.Name), strings.ToLower(name)) {
			a.views.Comms.State.agent = agent.Name
			a.views.Comms.State.conv.System = agent.Prompt
			a.addIntel("COMMS switched to: %s", agent.Name)
			return
		}
	}
	a.addIntel("Agent not found: %s", name)
}

// relayToAgent sends a message to another agent's conversation and streams the response.
func (a *App) relayToAgent(targetName, msg string) tea.Cmd {
	var target *agents.Agent
	for _, agent := range a.registry.All() {
		if strings.Contains(strings.ToLower(agent.Name), strings.ToLower(targetName)) {
			ag := agent
			target = &ag
			break
		}
	}
	if target == nil {
		a.addIntel("Relay failed — agent not found: %s", targetName)
		return nil
	}

	fromAgent := a.views.Comms.State.agent
	if msg == "" {
		for i := len(a.views.Comms.State.conv.Messages) - 1; i >= 0; i-- {
			if a.views.Comms.State.conv.Messages[i].Role == chat.RoleAssistant {
				msg = a.views.Comms.State.conv.Messages[i].Content
				break
			}
		}
		if msg == "" {
			a.addIntel("Relay failed — no response to relay")
			return nil
		}
	}

	relayMsg := fmt.Sprintf("[Relayed from %s]\n%s", fromAgent, msg)

	conv := chat.Load(target.Name, target.Prompt)
	if conv == nil {
		conv = chat.NewConversation(target.Name, target.Prompt)
	}
	a.views.Comms.State = &CommsState{
		conv:     conv,
		viewport: viewport.New(a.width-6, a.height-6),
		agent:    target.Name,
	}
	a.views.Comms.State.rebuildContent(a.theme.Data)

	a.views.Comms.State.AddUserMessage(relayMsg, a.theme.Data)
	a.views.Comms.State.streaming = true
	a.addIntel("Relayed to %s from %s", target.Name, fromAgent)

	a.views.Pipeline.StreamCh = chat.Stream(a.views.Comms.State.conv)
	return pollStream(a.views.Pipeline.StreamCh)
}
