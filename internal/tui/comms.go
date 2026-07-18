package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	"github.com/charmbracelet/lipgloss"
	"github.com/f5508037/moonbase/internal/chat"
)

// CommsState holds the COMMS panel state
type CommsState struct {
	conv      *chat.Conversation
	viewport  viewport.Model
	streaming bool
	buffer    string // accumulates streaming tokens for current response
	content   string // full rendered chat content
	agent     string
}

func newCommsState(agent, systemPrompt string, width, height int) *CommsState {
	vpWidth := width - 4
	vpHeight := height - 6
	if vpWidth < 10 {
		vpWidth = 10
	}
	if vpHeight < 4 {
		vpHeight = 4
	}
	vp := viewport.New(vpWidth, vpHeight)
	vp.SetContent("  Awaiting transmission...\n")

	return &CommsState{
		conv:     chat.NewConversation(agent, systemPrompt),
		viewport: vp,
		agent:    agent,
	}
}

func (c *CommsState) AddUserMessage(msg string) {
	c.conv.Add(chat.RoleUser, msg)
	c.rebuildContent()
}

func (c *CommsState) AppendStreamToken(token string) {
	c.buffer += token
	c.rebuildContent()
}

func (c *CommsState) FinishStream() {
	if c.buffer != "" {
		c.conv.Add(chat.RoleAssistant, c.buffer)
		c.buffer = ""
	}
	c.streaming = false
	c.rebuildContent()
	chat.Save(c.conv)
}

func (c *CommsState) rebuildContent() {
	var b strings.Builder
	dimStyle := lipgloss.NewStyle().Foreground(ColorDim)
	userStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#888888"))
	agentStyle := lipgloss.NewStyle().Foreground(ColorActive)
	nameStyle := lipgloss.NewStyle().Foreground(ColorBrand).Bold(true)

	for _, msg := range c.conv.Messages {
		ts := msg.Timestamp.Format("15:04")
		if msg.Role == chat.RoleUser {
			b.WriteString(dimStyle.Render(ts) + " " + userStyle.Render("you") + "\n")
			for _, line := range strings.Split(msg.Content, "\n") {
				b.WriteString("  " + line + "\n")
			}
		} else {
			b.WriteString(dimStyle.Render(ts) + " " + nameStyle.Render(c.agent) + "\n")
			rendered := renderMarkdown(msg.Content, c.viewport.Width-4)
			b.WriteString(rendered)
		}
		b.WriteString("\n")
	}

	// Streaming buffer (in progress)
	if c.buffer != "" {
		b.WriteString(dimStyle.Render("···") + " " + nameStyle.Render(c.agent) + "\n")
		for _, line := range strings.Split(c.buffer, "\n") {
			b.WriteString("  " + agentStyle.Render(line) + "\n")
		}
	}

	c.content = b.String()
	c.viewport.SetContent(c.content)
	c.viewport.GotoBottom()
}

// renderComms renders the COMMS view
func (a App) renderComms() string {
	header := a.renderHeader("Comms › " + a.comms.agent)

	// Chat viewport
	vpStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ColorInfo).
		Width(a.width - 2).
		Height(a.height - 5)

	chatView := vpStyle.Render(a.comms.viewport.View())

	// Input bar
	inputPrefix := lipgloss.NewStyle().Foreground(ColorBrand).Bold(true).Render(" > ")
	var inputBar string

	if a.contextFile {
		inputBar = lipgloss.NewStyle().Foreground(ColorInfo).Render(" 📎 Attach: ") + a.contextInput.View()
	} else if a.snippetPicker {
		inputBar = a.renderSnippetPicker()
	} else if a.comms.streaming {
		typingAnim := lipgloss.NewStyle().Foreground(ColorActive).Render(a.anim.RenderTyping())
		inputBar = inputPrefix + typingAnim + lipgloss.NewStyle().Foreground(ColorDim).Render(" streaming...")
	} else {
		inputBar = inputPrefix + a.commsInput.View()
	}

	hints := "[enter] SEND  [@agent] SWITCH  [>agent] RELAY  [>>agent msg] RELAY+MSG  [esc] BACK"
	statusBar := a.renderStatusBar(hints)

	return lipgloss.JoinVertical(lipgloss.Left, header, chatView, inputBar, statusBar)
}

func (a App) renderSnippetPicker() string {
	labelStyle := lipgloss.NewStyle().Foreground(ColorInfo).Bold(true)
	var b strings.Builder
	b.WriteString(labelStyle.Render(" 📋 SNIPPETS: "))
	if len(a.snippetList) == 0 {
		b.WriteString(lipgloss.NewStyle().Foreground(ColorDim).Render("(none saved — use: moonbase snippet save <name>)"))
	} else {
		for i, s := range a.snippetList {
			if i == a.snippetCursor {
				b.WriteString(lipgloss.NewStyle().Foreground(ColorActive).Bold(true).Render(fmt.Sprintf(" [%s] ", s.Name)))
			} else {
				b.WriteString(lipgloss.NewStyle().Foreground(ColorDim).Render(fmt.Sprintf("  %s  ", s.Name)))
			}
		}
		b.WriteString(lipgloss.NewStyle().Foreground(ColorDim).Render(" ↑↓/enter"))
	}
	return b.String()
}
