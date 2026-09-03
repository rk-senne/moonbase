package backend

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/rk-senne/moonbase/internal/agents"
	"github.com/rk-senne/moonbase/internal/discovery"
)

// sseChatServer returns a server that speaks the OpenAI-compatible SSE dialect,
// emitting the given content chunks then a usage frame and [DONE].
func sseChatServer(t *testing.T, chunks []string, promptTokens, completionTokens int) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		for _, c := range chunks {
			fmt.Fprintf(w, "data: {\"choices\":[{\"delta\":{\"content\":%q}}]}\n\n", c)
		}
		fmt.Fprintf(w,
			"data: {\"choices\":[{\"delta\":{},\"finish_reason\":\"stop\"}],\"usage\":{\"prompt_tokens\":%d,\"completion_tokens\":%d,\"total_tokens\":%d}}\n\n",
			promptTokens, completionTokens, promptTokens+completionTokens)
		fmt.Fprint(w, "data: [DONE]\n\n")
	}))
	t.Cleanup(srv.Close)
	return srv
}

func testAgent() agents.Agent {
	return agents.Agent{Name: "numbuh-1", Prompt: "You are Numbuh 1."}
}

// === kimiConfig ===

func TestKimiConfig_RequiresAPIKey(t *testing.T) {
	t.Setenv("MOONSHOT_API_KEY", "")

	if _, _, _, err := kimiConfig(); err == nil ||
		!strings.Contains(err.Error(), "MOONSHOT_API_KEY") {
		t.Fatalf("expected missing-key error, got %v", err)
	}
}

func TestKimiConfig_AppliesDefaults(t *testing.T) {
	t.Setenv("MOONSHOT_API_KEY", "k")
	t.Setenv("KIMI_MODEL", "")
	t.Setenv("KIMI_BASE_URL", "")

	key, model, baseURL, err := kimiConfig()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if key != "k" {
		t.Errorf("apiKey = %q", key)
	}
	if model != defaultKimiModel {
		t.Errorf("model = %q, want default %q", model, defaultKimiModel)
	}
	if baseURL != defaultKimiBaseURL {
		t.Errorf("baseURL = %q, want default %q", baseURL, defaultKimiBaseURL)
	}
}

func TestKimiConfig_HonoursOverrides(t *testing.T) {
	t.Setenv("MOONSHOT_API_KEY", "k")
	t.Setenv("KIMI_MODEL", "kimi-k2.7-code")
	t.Setenv("KIMI_BASE_URL", "https://gateway.internal/v1")

	_, model, baseURL, err := kimiConfig()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if model != "kimi-k2.7-code" {
		t.Errorf("model = %q, want the override", model)
	}
	if baseURL != "https://gateway.internal/v1" {
		t.Errorf("baseURL = %q, want the override", baseURL)
	}
}

// === Kimi identity / availability ===

func TestKimi_NameAndAvailability(t *testing.T) {
	k := &Kimi{}
	if k.Name() != "kimi" {
		t.Errorf("Name() = %q, want kimi", k.Name())
	}

	t.Setenv("MOONSHOT_API_KEY", "")
	if k.Available() {
		t.Error("Available() should be false without MOONSHOT_API_KEY")
	}

	t.Setenv("MOONSHOT_API_KEY", "present")
	if !k.Available() {
		t.Error("Available() should be true with MOONSHOT_API_KEY set")
	}
}

// === Kimi deploy paths (missing key) ===

func TestKimi_DeployPathsRequireAPIKey(t *testing.T) {
	t.Setenv("MOONSHOT_API_KEY", "")
	k := &Kimi{}
	ctx := &discovery.ProjectContext{}

	if _, err := k.Deploy(testAgent(), ctx, "task"); err == nil {
		t.Error("Deploy should fail without an API key")
	}
	if _, _, err := k.DeployWithUsage(testAgent(), ctx, "task"); err == nil {
		t.Error("DeployWithUsage should fail without an API key")
	}
	if _, _, err := k.DeployRawWithUsage("composed", "task"); err == nil {
		t.Error("DeployRawWithUsage should fail without an API key")
	}
}

// === Kimi deploy paths (against a stub API) ===

func TestKimi_DeployWithUsage_ReturnsContentAndUsage(t *testing.T) {
	srv := sseChatServer(t, []string{"Hello", " world"}, 11, 5)
	t.Setenv("MOONSHOT_API_KEY", "test-key")
	t.Setenv("KIMI_BASE_URL", srv.URL)

	k := &Kimi{}
	out, usage, err := k.DeployWithUsage(testAgent(), &discovery.ProjectContext{}, "task")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "Hello") || !strings.Contains(out, "world") {
		t.Errorf("output = %q, want the streamed chunks", out)
	}
	if usage == nil {
		t.Fatal("expected usage info")
	}
	if usage.PromptTokens != 11 || usage.CompletionTokens != 5 {
		t.Errorf("usage = %d/%d, want 11/5", usage.PromptTokens, usage.CompletionTokens)
	}
}

func TestKimi_Deploy_ReturnsContent(t *testing.T) {
	srv := sseChatServer(t, []string{"answer"}, 3, 2)
	t.Setenv("MOONSHOT_API_KEY", "test-key")
	t.Setenv("KIMI_BASE_URL", srv.URL)

	out, err := (&Kimi{}).Deploy(testAgent(), &discovery.ProjectContext{}, "task")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "answer") {
		t.Errorf("output = %q", out)
	}
}

func TestKimi_DeployRawWithUsage_SendsComposedPrompt(t *testing.T) {
	srv := sseChatServer(t, []string{"raw-ok"}, 7, 1)
	t.Setenv("MOONSHOT_API_KEY", "test-key")
	t.Setenv("KIMI_BASE_URL", srv.URL)

	out, usage, err := (&Kimi{}).DeployRawWithUsage("pre-composed prompt", "task")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "raw-ok") {
		t.Errorf("output = %q", out)
	}
	if usage == nil || usage.TotalTokens != 8 {
		t.Errorf("usage = %+v, want total 8", usage)
	}
}

func TestKimi_DeployWithUsage_WrapsAPIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprint(w, `{"error":{"message":"bad key"}}`)
	}))
	defer srv.Close()

	t.Setenv("MOONSHOT_API_KEY", "test-key")
	t.Setenv("KIMI_BASE_URL", srv.URL)

	if _, _, err := (&Kimi{}).DeployWithUsage(testAgent(), &discovery.ProjectContext{}, "task"); err == nil {
		t.Fatal("expected an error for a 401 response")
	}
}

// Kimi must satisfy the optional usage interfaces, since the flywheel relies on
// them to record token cost.
func TestKimi_ImplementsUsageInterfaces(t *testing.T) {
	var k any = &Kimi{}
	if _, ok := k.(Backend); !ok {
		t.Error("Kimi must implement Backend")
	}
	if _, ok := k.(UsageReporter); !ok {
		t.Error("Kimi must implement UsageReporter")
	}
	if _, ok := k.(RawUsageReporter); !ok {
		t.Error("Kimi must implement RawUsageReporter")
	}
}

// === envHasValue / fail-closed availability ===

func TestEnvHasValue(t *testing.T) {
	tests := []struct {
		name  string
		value string
		set   bool
		want  bool
	}{
		{name: "unset", set: false, want: false},
		{name: "empty", value: "", set: true, want: false},
		{name: "whitespace only", value: "   ", set: true, want: false},
		{name: "tab only", value: "\t", set: true, want: false},
		{name: "real value", value: "sk-abc123", set: true, want: true},
		{name: "padded value", value: "  sk-abc123  ", set: true, want: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			const key = "TEST_MOONBASE_HASVALUE"
			if tt.set {
				t.Setenv(key, tt.value)
			} else {
				os.Unsetenv(key)
			}
			if got := envHasValue(key); got != tt.want {
				t.Errorf("envHasValue(%q) = %v, want %v", tt.value, got, tt.want)
			}
		})
	}
}

// An exported-but-empty API key must not make a backend look usable — otherwise
// auto-detection selects it and every deploy fails with "not set".
func TestAPIKeyBackends_UnavailableWhenKeyIsEmpty(t *testing.T) {
	tests := []struct {
		name    string
		envKey  string
		backend Backend
	}{
		{name: "openai", envKey: "OPENAI_API_KEY", backend: &OpenAI{}},
		{name: "anthropic", envKey: "ANTHROPIC_API_KEY", backend: &Anthropic{}},
		{name: "kimi", envKey: "MOONSHOT_API_KEY", backend: &Kimi{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv(tt.envKey, "")
			if tt.backend.Available() {
				t.Errorf("%s reported available with an empty API key", tt.name)
			}

			t.Setenv(tt.envKey, "real-key")
			if !tt.backend.Available() {
				t.Errorf("%s reported unavailable with a real API key", tt.name)
			}
		})
	}
}
