package backend

import (
	"strings"
	"testing"

	"github.com/rk-senne/moonbase/internal/discovery"
)

// OpenAI.DeployWithUsage is the path the flywheel uses to record token cost, so
// it needs coverage beyond the raw-prompt variant. OPENAI_BASE_URL lets these
// tests run against a stub server without touching the real API.

func TestOpenAI_DeployWithUsage_ReturnsContentAndUsage(t *testing.T) {
	srv := sseChatServer(t, []string{"first", "-second"}, 21, 9)
	t.Setenv("OPENAI_API_KEY", "test-key")
	t.Setenv("OPENAI_BASE_URL", srv.URL)

	out, usage, err := (&OpenAI{}).DeployWithUsage(testAgent(), &discovery.ProjectContext{}, "task")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "first") || !strings.Contains(out, "second") {
		t.Errorf("output = %q, want both streamed chunks", out)
	}
	if usage == nil {
		t.Fatal("expected usage info for cost tracking")
	}
	if usage.PromptTokens != 21 || usage.CompletionTokens != 9 || usage.TotalTokens != 30 {
		t.Errorf("usage = %d/%d/%d, want 21/9/30",
			usage.PromptTokens, usage.CompletionTokens, usage.TotalTokens)
	}
}

func TestOpenAI_DeployWithUsage_RequiresAPIKey(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "")

	if _, _, err := (&OpenAI{}).DeployWithUsage(testAgent(), &discovery.ProjectContext{}, "task"); err == nil ||
		!strings.Contains(err.Error(), "OPENAI_API_KEY") {
		t.Fatalf("expected missing-key error, got %v", err)
	}
}

func TestOpenAI_Deploy_ReturnsContent(t *testing.T) {
	srv := sseChatServer(t, []string{"plain"}, 2, 1)
	t.Setenv("OPENAI_API_KEY", "test-key")
	t.Setenv("OPENAI_BASE_URL", srv.URL)

	out, err := (&OpenAI{}).Deploy(testAgent(), &discovery.ProjectContext{}, "task")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "plain") {
		t.Errorf("output = %q", out)
	}
}

func TestOpenAI_ImplementsUsageInterfaces(t *testing.T) {
	var o any = &OpenAI{}
	if _, ok := o.(UsageReporter); !ok {
		t.Error("OpenAI must implement UsageReporter")
	}
	if _, ok := o.(RawUsageReporter); !ok {
		t.Error("OpenAI must implement RawUsageReporter")
	}
}
