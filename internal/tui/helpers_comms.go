package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/f5508037/moonbase/internal/agents"
	"github.com/f5508037/moonbase/internal/chat"
)

func (a *App) openComms() {
	agent := a.registry.Get(a.selected)

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
		a.comms = &CommsState{
			conv:     conv,
			viewport: viewport.New(vpWidth, vpHeight),
			agent:    agent.Name,
		}
		a.comms.rebuildContent()
	} else {
		a.comms = newCommsState(agent.Name, agent.Prompt, a.width, a.height)
	}
	a.view = ViewComms
	a.commsInput.Focus()
	a.commsInput.Width = a.width - 8
	if a.commsInput.Width < 10 {
		a.commsInput.Width = 10
	}
}

func (a *App) sendCommsMessage() tea.Cmd {
	msg := a.commsInput.Value()
	if msg == "" {
		return nil
	}
	a.commsInput.Reset()
	a.comms.AddUserMessage(msg)
	a.comms.streaming = true

	// Start streaming and store channel for continued polling
	a.streamCh = chat.Stream(a.comms.conv)
	return pollStream(a.streamCh)
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
			a.comms.agent = agent.Name
			a.comms.conv.System = agent.Prompt
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

	fromAgent := a.comms.agent
	if msg == "" {
		for i := len(a.comms.conv.Messages) - 1; i >= 0; i-- {
			if a.comms.conv.Messages[i].Role == chat.RoleAssistant {
				msg = a.comms.conv.Messages[i].Content
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
	a.comms = &CommsState{
		conv:     conv,
		viewport: viewport.New(a.width-6, a.height-6),
		agent:    target.Name,
	}
	a.comms.rebuildContent()

	a.comms.AddUserMessage(relayMsg)
	a.comms.streaming = true
	a.addIntel("Relayed to %s from %s", target.Name, fromAgent)

	a.streamCh = chat.Stream(a.comms.conv)
	return pollStream(a.streamCh)
}
