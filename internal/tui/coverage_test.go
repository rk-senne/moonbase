package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/f5508037/moonbase/internal/snippets"
)

// === AnimState tests ===

func TestAnimState_Advance(t *testing.T) {
	a := &AnimState{}
	a.TriggerIntelFlash()
	a.TriggerSelectPulse()
	a.TriggerReveal()

	if a.intelFlash != 6 {
		t.Errorf("expected intelFlash=6, got %d", a.intelFlash)
	}

	a.Advance()
	if a.frame != 1 {
		t.Errorf("expected frame=1, got %d", a.frame)
	}
	if a.intelFlash != 5 {
		t.Errorf("expected intelFlash=5 after advance, got %d", a.intelFlash)
	}
	if a.selectPulse != 7 {
		t.Errorf("expected selectPulse=7, got %d", a.selectPulse)
	}
	if !a.revealing {
		t.Error("expected revealing=true")
	}

	// Advance 6 more times to decay intel flash to 0
	for i := 0; i < 6; i++ {
		a.Advance()
	}
	if a.intelFlash != 0 {
		t.Errorf("expected intelFlash=0, got %d", a.intelFlash)
	}
}

func TestAnimState_RenderTyping(t *testing.T) {
	a := &AnimState{frame: 0}
	result := a.RenderTyping()
	if result == "" {
		t.Error("expected non-empty typing indicator")
	}
}

func TestAnimState_IntelFlashStyle(t *testing.T) {
	a := &AnimState{intelFlash: 0}
	style := a.IntelFlashStyle()
	_ = style // just ensure no panic

	a.intelFlash = 3
	style2 := a.IntelFlashStyle()
	_ = style2
}

func TestAnimState_PulseBadge(t *testing.T) {
	a := &AnimState{selectPulse: 0}
	badge := a.PulseBadge()
	if badge != BadgeActive {
		t.Errorf("expected BadgeActive when no pulse, got %s", badge)
	}

	a.selectPulse = 3
	badge2 := a.PulseBadge()
	if badge2 == "" {
		t.Error("expected non-empty badge during pulse")
	}
}

func TestAnimState_TypewriterText(t *testing.T) {
	a := &AnimState{typewriterAt: 0}
	result := a.TypewriterText("Hello World")
	// typewriterAt=0 means not started, but the function returns full[:0]+"█" = "█"
	// Actually: if typewriterAt >= len(full), return full. 0 < 11, so returns full[:0]+"█"
	// Let's test with typewriterAt >= len to get full text
	a.typewriterAt = 100
	result = a.TypewriterText("Hello World")
	if result != "Hello World" {
		t.Errorf("expected full text when typewriterAt>=len, got %s", result)
	}

	a.typewriterAt = 5
	result = a.TypewriterText("Hello World")
	if result != "Hello█" {
		t.Errorf("expected 'Hello█', got %s", result)
	}
}

func TestAnimState_TriggerTypewriter(t *testing.T) {
	a := &AnimState{}
	a.TriggerTypewriter()
	if a.typewriterAt != 1 {
		t.Errorf("expected typewriterAt=1, got %d", a.typewriterAt)
	}
}

// === CommsState tests ===

func TestCommsState_AddUserMessage(t *testing.T) {
	cs := NewCommsState("numbuh-1", "system", 80, 40)
	cs.AddUserMessage("hello agent")
	if len(cs.conv.Messages) != 1 {
		t.Errorf("expected 1 message, got %d", len(cs.conv.Messages))
	}
}

func TestCommsState_AppendStreamToken(t *testing.T) {
	cs := NewCommsState("numbuh-1", "system", 80, 40)
	cs.streaming = true
	cs.AppendStreamToken("hello ")
	cs.AppendStreamToken("world")
	if cs.buffer != "hello world" {
		t.Errorf("expected buffer 'hello world', got '%s'", cs.buffer)
	}
}

func TestCommsState_FinishStream(t *testing.T) {
	cs := NewCommsState("numbuh-1", "system", 80, 40)
	cs.streaming = true
	cs.buffer = "response content"
	cs.FinishStream()
	if cs.streaming {
		t.Error("expected streaming=false after finish")
	}
	if cs.buffer != "" {
		t.Error("expected buffer cleared after finish")
	}
	if len(cs.conv.Messages) != 1 {
		t.Errorf("expected 1 message recorded, got %d", len(cs.conv.Messages))
	}
}

// === DocsState tests ===

func TestNewDocsState(t *testing.T) {
	ds := NewDocsState(80, 40)
	if ds == nil {
		t.Fatal("expected non-nil DocsState")
	}
	// It may or may not have files depending on CWD
}

// === ProjectsState tests ===

func TestNewProjectsState(t *testing.T) {
	ps := NewProjectsState()
	if ps == nil {
		t.Fatal("expected non-nil ProjectsState")
	}
}

// === Comms key handling: snippet picker ===

func TestCommsKeys_SnippetPickerNav(t *testing.T) {
	app := NewApp()
	app.view = ViewComms
	app.ready = true
	app.comms = NewCommsState("test", "prompt", 80, 40)
	app.commsInput.Focus()
	app.snippetPicker = true
	app.snippetList = []snippets.Snippet{
		{Name: "s1", Content: "content1"},
		{Name: "s2", Content: "content2"},
	}
	app.snippetCursor = 0

	// Navigate down
	model, _ := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	result := model.(App)
	if result.snippetCursor != 1 {
		t.Errorf("expected snippetCursor=1, got %d", result.snippetCursor)
	}

	// Navigate up
	model, _ = result.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	result = model.(App)
	if result.snippetCursor != 0 {
		t.Errorf("expected snippetCursor=0, got %d", result.snippetCursor)
	}

	// Select with enter
	model, _ = result.Update(tea.KeyMsg{Type: tea.KeyEnter})
	result = model.(App)
	if result.snippetPicker {
		t.Error("expected snippetPicker=false after enter")
	}
	if result.commsInput.Value() != "content1" {
		t.Errorf("expected comms input set to snippet content, got '%s'", result.commsInput.Value())
	}
}

// === Comms key handling: relay commands ===

func TestCommsKeys_RelayCommand(t *testing.T) {
	app := NewApp()
	app.view = ViewComms
	app.ready = true
	app.comms = NewCommsState("test", "prompt", 80, 40)
	app.commsInput.Focus()
	app.commsInput.SetValue(">>numbuh-1 hello there")

	model, _ := app.Update(tea.KeyMsg{Type: tea.KeyEnter})
	result := model.(App)
	if result.commsInput.Value() != "" {
		t.Errorf("expected comms input reset after relay, got '%s'", result.commsInput.Value())
	}
}

func TestCommsKeys_RelayLastResponse(t *testing.T) {
	app := NewApp()
	app.view = ViewComms
	app.ready = true
	app.comms = NewCommsState("test", "prompt", 80, 40)
	app.commsInput.Focus()
	app.commsInput.SetValue(">numbuh-2")

	model, _ := app.Update(tea.KeyMsg{Type: tea.KeyEnter})
	result := model.(App)
	if result.commsInput.Value() != "" {
		t.Errorf("expected comms input reset after relay, got '%s'", result.commsInput.Value())
	}
}

func TestCommsKeys_CtrlS_SnippetPicker(t *testing.T) {
	app := NewApp()
	app.view = ViewComms
	app.ready = true
	app.comms = NewCommsState("test", "prompt", 80, 40)
	app.commsInput.Focus()

	model, _ := app.Update(tea.KeyMsg{Type: tea.KeyCtrlS})
	result := model.(App)
	if !result.snippetPicker {
		t.Error("expected snippetPicker=true after ctrl+s")
	}
}

func TestCommsKeys_CtrlF_ContextFile(t *testing.T) {
	app := NewApp()
	app.view = ViewComms
	app.ready = true
	app.comms = NewCommsState("test", "prompt", 80, 40)
	app.commsInput.Focus()

	model, _ := app.Update(tea.KeyMsg{Type: tea.KeyCtrlF})
	result := model.(App)
	if !result.contextFile {
		t.Error("expected contextFile=true after ctrl+f")
	}
}
