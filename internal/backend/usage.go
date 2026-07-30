// Usage reporting types for token consumption observability.
//
// This file defines the optional interfaces backends can implement to report
// token usage alongside their responses. The pattern mirrors RawDeployer:
// callers check with a type assertion at call time and degrade gracefully
// when usage is unavailable (nil return, not an error).
//
// Backends that talk to APIs returning usage data (OpenAI, Anthropic, Kimi)
// implement these interfaces. Backends that shell out to CLI tools (kiro-cli,
// codex, ollama) or use clipboard do not — they return nil usage.
package backend

import (
	"github.com/rk-senne/moonbase/internal/agents"
	"github.com/rk-senne/moonbase/internal/discovery"
)

// UsageInfo holds token consumption from a single backend call.
// Nil UsageInfo means the backend doesn't report usage (not an error).
type UsageInfo struct {
	PromptTokens     int    // Tokens in the prompt/input
	CompletionTokens int    // Tokens in the completion/output
	TotalTokens      int    // Sum (some APIs report this independently)
	Model            string // Model name that served the request
}

// UsageReporter is an optional interface backends implement to return token usage
// alongside the standard Deploy response. Check with type assertion:
//
//	if ur, ok := be.(UsageReporter); ok { output, usage, err := ur.DeployWithUsage(...) }
//
// This does NOT replace the Backend interface — it's additive.
type UsageReporter interface {
	DeployWithUsage(agent agents.Agent, context *discovery.ProjectContext, task string) (string, *UsageInfo, error)
}

// RawUsageReporter is the usage-aware variant of RawDeployer for pre-composed prompts.
// Check with type assertion:
//
//	if raw, ok := be.(RawUsageReporter); ok { output, usage, err := raw.DeployRawWithUsage(...) }
type RawUsageReporter interface {
	DeployRawWithUsage(composed string, task string) (string, *UsageInfo, error)
}
