package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/rk-senne/moonbase/internal/snippets"
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
	style := a.IntelFlashStyle(moonbaseTheme)
	_ = style // just ensure no panic

	a.intelFlash = 3
	style2 := a.IntelFlashStyle(moonbaseTheme)
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
	cs := newCommsState("numbuh-1", "system", 80, 40)
	cs.AddUserMessage("hello agent", moonbaseTheme)
	if len(cs.conv.Messages) != 1 {
		t.Errorf("expected 1 message, got %d", len(cs.conv.Messages))
	}
}

func TestCommsState_AppendStreamToken(t *testing.T) {
	cs := newCommsState("numbuh-1", "system", 80, 40)
	cs.streaming = true
	cs.AppendStreamToken("hello ", moonbaseTheme)
	cs.AppendStreamToken("world", moonbaseTheme)
	if cs.buffer != "hello world" {
		t.Errorf("expected buffer 'hello world', got '%s'", cs.buffer)
	}
}

func TestCommsState_FinishStream(t *testing.T) {
	cs := newCommsState("numbuh-1", "system", 80, 40)
	cs.streaming = true
	cs.buffer = "response content"
	cs.FinishStream(moonbaseTheme)
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

func Test_newDocsState(t *testing.T) {
	ds := newDocsState(80, 40)
	if ds == nil {
		t.Fatal("expected non-nil DocsState")
	}
	// It may or may not have files depending on CWD
}

// === ProjectsState tests ===

func Test_newProjectsState(t *testing.T) {
	ps := newProjectsState()
	if ps == nil {
		t.Fatal("expected non-nil ProjectsState")
	}
}

// === Comms key handling: snippet picker ===

func TestCommsKeys_SnippetPickerNav(t *testing.T) {
	app := NewApp()
	app.view = ViewComms
	app.boot.Ready = true
	app.views.Comms.State = newCommsState("test", "prompt", 80, 40)
	app.views.Comms.Input.Focus()
	app.views.SnippetPick.Active = true
	app.views.SnippetPick.List = []snippets.Snippet{
		{Name: "s1", Content: "content1"},
		{Name: "s2", Content: "content2"},
	}
	app.views.SnippetPick.Cursor = 0

	// Navigate down
	model, _ := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	result := model.(App)
	if result.views.SnippetPick.Cursor != 1 {
		t.Errorf("expected snippetPick.Cursor=1, got %d", result.views.SnippetPick.Cursor)
	}

	// Navigate up
	model, _ = result.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	result = model.(App)
	if result.views.SnippetPick.Cursor != 0 {
		t.Errorf("expected snippetPick.Cursor=0, got %d", result.views.SnippetPick.Cursor)
	}

	// Select with enter
	model, _ = result.Update(tea.KeyMsg{Type: tea.KeyEnter})
	result = model.(App)
	if result.views.SnippetPick.Active {
		t.Error("expected snippetPick.Active=false after enter")
	}
	if result.views.Comms.Input.Value() != "content1" {
		t.Errorf("expected comms input set to snippet content, got '%s'", result.views.Comms.Input.Value())
	}
}

// === Comms key handling: relay commands ===

func TestCommsKeys_RelayCommand(t *testing.T) {
	app := NewApp()
	app.view = ViewComms
	app.boot.Ready = true
	app.views.Comms.State = newCommsState("test", "prompt", 80, 40)
	app.views.Comms.Input.Focus()
	app.views.Comms.Input.SetValue(">>numbuh-1 hello there")

	model, _ := app.Update(tea.KeyMsg{Type: tea.KeyEnter})
	result := model.(App)
	if result.views.Comms.Input.Value() != "" {
		t.Errorf("expected comms input reset after relay, got '%s'", result.views.Comms.Input.Value())
	}
}

func TestCommsKeys_RelayLastResponse(t *testing.T) {
	app := NewApp()
	app.view = ViewComms
	app.boot.Ready = true
	app.views.Comms.State = newCommsState("test", "prompt", 80, 40)
	app.views.Comms.Input.Focus()
	app.views.Comms.Input.SetValue(">numbuh-2")

	model, _ := app.Update(tea.KeyMsg{Type: tea.KeyEnter})
	result := model.(App)
	if result.views.Comms.Input.Value() != "" {
		t.Errorf("expected comms input reset after relay, got '%s'", result.views.Comms.Input.Value())
	}
}

func TestCommsKeys_CtrlS_SnippetPicker(t *testing.T) {
	app := NewApp()
	app.view = ViewComms
	app.boot.Ready = true
	app.views.Comms.State = newCommsState("test", "prompt", 80, 40)
	app.views.Comms.Input.Focus()

	model, _ := app.Update(tea.KeyMsg{Type: tea.KeyCtrlS})
	result := model.(App)
	if !result.views.SnippetPick.Active {
		t.Error("expected snippetPick.Active=true after ctrl+s")
	}
}

func TestCommsKeys_CtrlF_ContextFile(t *testing.T) {
	app := NewApp()
	app.view = ViewComms
	app.boot.Ready = true
	app.views.Comms.State = newCommsState("test", "prompt", 80, 40)
	app.views.Comms.Input.Focus()

	model, _ := app.Update(tea.KeyMsg{Type: tea.KeyCtrlF})
	result := model.(App)
	if !result.views.CtxFile.Active {
		t.Error("expected ctxFile.Active=true after ctrl+f")
	}
}
