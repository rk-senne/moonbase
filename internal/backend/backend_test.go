package backend

import (
	"testing"

	"github.com/f5508037/moonbase/internal/agents"
	clip "github.com/f5508037/moonbase/internal/clipboard"
	"github.com/f5508037/moonbase/internal/discovery"
)

func TestDetectAll_Returns6Backends(t *testing.T) {
	all := DetectAll()
	if len(all) != 6 {
		t.Errorf("expected 6 backends, got %d", len(all))
	}
}

func TestDetectAll_AllHaveNames(t *testing.T) {
	all := DetectAll()
	for _, b := range all {
		if b.Name() == "" {
			t.Error("backend has empty name")
		}
	}
}

func TestDetectAll_ExpectedNames(t *testing.T) {
	all := DetectAll()
	names := make(map[string]bool)
	for _, b := range all {
		names[b.Name()] = true
	}

	expected := []string{"kiro-cli", "codex", "openai", "anthropic", "ollama", "clipboard"}
	for _, name := range expected {
		if !names[name] {
			t.Errorf("expected backend %s not found", name)
		}
	}
}

func TestClipboard_AlwaysAvailable(t *testing.T) {
	c := &Clipboard{}
	if !c.Available() {
		t.Error("clipboard should always be available")
	}
}

func TestPreferred_NeverNil(t *testing.T) {
	b := Preferred()
	if b == nil {
		t.Fatal("Preferred() should never return nil")
	}
	if b.Name() == "" {
		t.Error("Preferred backend should have a name")
	}
}

func TestDetectAvailable_AtLeastClipboard(t *testing.T) {
	available := DetectAvailable()
	if len(available) == 0 {
		t.Fatal("expected at least clipboard to be available")
	}

	hasClipboard := false
	for _, b := range available {
		if b.Name() == "clipboard" {
			hasClipboard = true
		}
	}
	if !hasClipboard {
		t.Error("clipboard should always be in available list")
	}
}

func TestClipboard_Deploy(t *testing.T) {
	c := &Clipboard{}
	agent := agents.Agent{
		Name: "test-agent",
		Role: "Test",
		Prompt: "You are a test agent.",
	}
	ctx := &discovery.ProjectContext{}

	output, err := c.Deploy(agent, ctx, "do the thing")
	// pbcopy may or may not be available in CI, so we accept either success or a known error
	if err != nil {
		if !clip.Available() {
			t.Skip("no clipboard command available")
		}
		t.Fatalf("clipboard deploy failed: %v", err)
	}
	if output == "" {
		t.Error("expected non-empty output from clipboard deploy")
	}
}

func TestKiro_Name(t *testing.T) {
	k := &Kiro{}
	if k.Name() != "kiro-cli" {
		t.Errorf("expected 'kiro-cli', got: %s", k.Name())
	}
}

func TestOllama_Name(t *testing.T) {
	o := &Ollama{}
	if o.Name() != "ollama" {
		t.Errorf("expected 'ollama', got: %s", o.Name())
	}
}
