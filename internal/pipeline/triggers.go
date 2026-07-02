package pipeline

import (
	"strings"
)

// TriggerResult indicates whether a conditional specialist should be invoked.
type TriggerResult struct {
	Invoke bool
	Reason string
}

// EvaluateTrigger checks if a conditional specialist's trigger conditions are met
// based on the pipeline context.
//
// triggerSpec is the agent's `triggers` field from frontmatter.
// context is the accumulated pipeline state.
func EvaluateTrigger(triggerSpec string, context *PipelineContext) TriggerResult {
	if triggerSpec == "" {
		return TriggerResult{Invoke: false, Reason: "no trigger conditions defined"}
	}

	lower := strings.ToLower(triggerSpec)
	reasons := []string{}

	// Check file count trigger (e.g., Numbuh 0: ">5 files changed")
	if strings.Contains(lower, ">5 files") || strings.Contains(lower, "5 files") {
		if len(context.FilesChanged) > 5 {
			reasons = append(reasons, "more than 5 files changed")
		}
	}

	// Check for keyword-based triggers by scanning phase outputs
	allOutputs := gatherOutputs(context)
	allOutputsLower := strings.ToLower(allOutputs)

	// Security triggers
	if containsAny(lower, "auth", "secret", "permission", "injection", "cve", "security") {
		if containsAny(allOutputsLower, "auth", "authentication", "authorization",
			"secret", "credential", "token", "password", "api_key", "apikey",
			"permission", "rbac", "acl",
			"injection", "xss", "csrf", "sql injection",
			"cve-", "vulnerability") {
			reasons = append(reasons, "security-related content detected in pipeline output")
		}
	}

	// Deployment/infra triggers
	if containsAny(lower, "ci/cd", "docker", "infra", "deploy", "env var", "runtime") {
		if containsAny(allOutputsLower, "dockerfile", "docker-compose", "docker",
			"github/workflows", "gitlab-ci", "jenkinsfile", "ci/cd", "pipeline",
			"deploy", "deployment", "infrastructure",
			"env var", "environment variable", "env_", "process.env") {
			reasons = append(reasons, "deployment/infrastructure content detected")
		}
	}

	// Migration triggers
	if containsAny(lower, "upgrade", "migration", "breaking change", "deprecat", "framework") {
		if containsAny(allOutputsLower, "upgrade", "migration", "migrate",
			"breaking change", "deprecated", "deprecation",
			"version bump", "major version") {
			reasons = append(reasons, "migration/upgrade content detected")
		}
	}

	// Core logic / architecture triggers
	if containsAny(lower, "core logic", "orchestration", "pipeline", "architecture", "pattern") {
		if containsAny(allOutputsLower, "architecture", "refactor core",
			"new pattern", "design pattern", "orchestrat",
			"state machine", "pipeline") {
			reasons = append(reasons, "core logic/architecture change detected")
		}
	}

	// Dead code / cleanup triggers
	if containsAny(lower, "dead code", "unused", "stale", "duplicate", "deprecated", "zombie") {
		if containsAny(allOutputsLower, "dead code", "unused", "unreferenced",
			"stale", "duplicate", "deprecated", "zombie", "todo", "fixme", "hack") {
			reasons = append(reasons, "tech debt/cleanup signals detected")
		}
	}

	// Edge case / chaos triggers
	if containsAny(lower, "edge case", "fragile", "user-facing", "parser", "state machine") {
		if containsAny(allOutputsLower, "edge case", "boundary", "fragile",
			"user input", "user-facing", "parser", "parsing",
			"state machine", "state transition") {
			reasons = append(reasons, "edge case/fragility signals detected")
		}
	}

	// Documentation triggers
	if containsAny(lower, "readme", "api doc", "adr", "changelog", "onboarding") {
		if containsAny(allOutputsLower, "readme", "documentation",
			"api doc", "adr", "changelog", "onboarding",
			"undocumented", "needs documentation") {
			reasons = append(reasons, "documentation need detected")
		}
	}

	// Legacy/archaeology triggers
	if containsAny(lower, "old", "mysterious", "undocumented", "legacy", "nobody-knows") {
		if containsAny(allOutputsLower, "legacy", "old code", "mysterious",
			"undocumented", "nobody knows", "ancient",
			"git blame", "historical") {
			reasons = append(reasons, "legacy code signals detected")
		}
	}

	if len(reasons) > 0 {
		return TriggerResult{
			Invoke: true,
			Reason: strings.Join(reasons, "; "),
		}
	}

	return TriggerResult{
		Invoke: false,
		Reason: "trigger conditions not met",
	}
}

// gatherOutputs combines all phase outputs into a single string for scanning.
func gatherOutputs(context *PipelineContext) string {
	var parts []string
	for _, output := range context.PhaseOutputs {
		parts = append(parts, output)
	}
	// Include file names as they're significant signals
	parts = append(parts, strings.Join(context.FilesChanged, " "))
	return strings.Join(parts, "\n")
}

// containsAny checks if the text contains any of the given substrings.
func containsAny(text string, substrings ...string) bool {
	for _, s := range substrings {
		if strings.Contains(text, s) {
			return true
		}
	}
	return false
}
