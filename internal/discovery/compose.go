package discovery

import (
	"fmt"
	"strings"
)

// maxSteeringSize is the maximum bytes of steering content included per file.
// Prevents excessively large steering files from consuming the entire context window
// or being used as a prompt injection amplification vector.
const maxSteeringSize = 8000

// maxSpecSize is the maximum bytes of spec content included per file.
const maxSpecSize = 1500

// maxEagerSkillSize is the maximum runes of a legacy (no-frontmatter) skill
// injected eagerly into every prompt. These are not requested by the agent, so
// they stay tightly bounded to protect the context window.
const maxEagerSkillSize = 2000

// maxSkillContentSize is the maximum runes of a skill loaded on demand through
// the progressive registry (via @skill(name)). The agent explicitly asked for
// this content, so silently cutting it defeats the purpose of the request; the
// budget matches maxSteeringSize. The curated library's largest skill is ~3,100
// runes, so this leaves substantial headroom.
const maxSkillContentSize = 8000

// maxReadmeSize is the maximum runes of README content included as stack context.
const maxReadmeSize = 2000

// maxSkills is the maximum number of skills included in a prompt.
// Prevents context bloat when many skills are defined.
const maxSkills = 5

// maxTotalComposedSize is the absolute maximum size of a composed prompt (512KB).
// Guards against OOM if many large files are discovered.
const maxTotalComposedSize = 512 * 1024

// ComposePrompt builds the full prompt for an agent deployment.
// Order: project steering rules → agent prompt → spec context → skills → stack → task
//
// PROMPT CACHING OPTIMIZATION:
// This ordering is deliberately designed for LLM prompt caching (e.g., Claude's
// cache_control). Caching works by prefix matching — the more requests that share
// a common prefix, the higher the cache hit rate. We place static/slow-changing
// content first (steering, agent prompt) and dynamic/per-request content last (task).
// Use ComposeCacheablePrefix() to get everything except the task, which backends
// can mark as the cacheable prefix boundary.
//
// SECURITY TRUST BOUNDARY:
// Content from steering and spec files is injected into the prompt structure.
// This content is read from the local filesystem (.kiro/ directory) which is
// under the project owner's control. However, injected content could
// theoretically attempt to override agent instructions via prompt injection.
//
// Mitigations:
//  1. Content is wrapped in clearly delimited sections (--- PROJECT RULES --- etc.)
//  2. The agent prompt (identity/instructions) comes AFTER steering, so the agent's
//     core directives have final authority in the prompt order
//  3. Content is size-limited to prevent context window exhaustion
//  4. Frontmatter is stripped to avoid YAML injection into the prompt
//  5. Total composed size is capped to prevent OOM
func ComposePrompt(agentPrompt string, context *ProjectContext, task string) string {
	var sections []string

	// 1. Steering rules (project-wide conventions)
	if context != nil && context.HasSteering() {
		var steeringBlock strings.Builder
		steeringBlock.WriteString("--- PROJECT RULES ---\n")
		steeringBlock.WriteString("Follow these project-specific rules and conventions:\n\n")
		for _, s := range context.Steering {
			steeringBlock.WriteString(fmt.Sprintf("### %s\n\n", s.Name))
			// Strip frontmatter from steering content before including
			content := stripFrontmatter(s.Content)
			// SECURITY: Truncate oversized steering files to prevent context exhaustion
			// (rune-safe: byte slicing can split a multi-byte char → invalid UTF-8)
			if r := []rune(content); len(r) > maxSteeringSize {
				content = string(r[:maxSteeringSize]) + "\n...(truncated for size)"
			}
			steeringBlock.WriteString(content)
			steeringBlock.WriteString("\n\n")
		}
		steeringBlock.WriteString("--- END PROJECT RULES ---\n")
		sections = append(sections, steeringBlock.String())
	}

	// 2. Agent prompt (the agent's full identity and operating protocol)
	sections = append(sections, agentPrompt)

	// 3. Spec context (if specs exist for the current project)
	if context != nil && context.HasSpecs() {
		var specBlock strings.Builder
		specBlock.WriteString("\n--- PROJECT SPEC CONTEXT ---\n")
		specBlock.WriteString("The following specs exist for this project. Reference AC-IDs when applicable:\n\n")

		// Group by feature
		features := make(map[string][]SpecFile)
		for _, s := range context.Specs {
			features[s.Feature] = append(features[s.Feature], s)
		}

		for feature, files := range features {
			specBlock.WriteString(fmt.Sprintf("## Feature: %s\n\n", feature))
			for _, f := range files {
				// SECURITY: Truncate per-file spec content
				// (rune-safe: byte slicing can split a multi-byte char → invalid UTF-8)
				content := f.Content
				if r := []rune(content); len(r) > maxSpecSize {
					content = string(r[:maxSpecSize]) + "\n...(truncated)"
				}
				specBlock.WriteString(fmt.Sprintf("### %s.md\n\n%s\n\n", f.Type, content))
			}
		}

		specBlock.WriteString("--- END PROJECT SPEC CONTEXT ---\n")
		sections = append(sections, specBlock.String())
	}

	// 4. Skills — progressive loading: catalog for registry skills, eager for legacy.
	// Registry skills: emit a lightweight catalog (name + description only).
	// Legacy skills (no frontmatter): inject full content for backward compat.
	if context != nil && context.HasSkills() {
		var skillBlock strings.Builder

		// Registry skills → catalog (metadata only, ~100 tokens each)
		if context.HasRegistrySkills() {
			skillBlock.WriteString("\n--- AVAILABLE SKILLS ---\n")
			skillBlock.WriteString("The following skills are available. Request any skill with @skill(name) to load its full content.\n\n")
			skillBlock.WriteString("| Skill | Description |\n")
			skillBlock.WriteString("|-------|-------------|\n")
			for _, meta := range context.SkillRegistry.List() {
				desc := meta.Description
				if desc == "" {
					desc = "(no description)"
				}
				skillBlock.WriteString(fmt.Sprintf("| %s | %s |\n", meta.Name, desc))
			}
			skillBlock.WriteString("\n--- END AVAILABLE SKILLS ---\n")
		}

		// Legacy skills (no frontmatter) → inject full content for backward compat
		if len(context.Skills) > 0 {
			skillBlock.WriteString("\n--- PROJECT SKILLS ---\n")
			skillBlock.WriteString("The following domain knowledge is available:\n\n")

			limit := len(context.Skills)
			if limit > maxSkills {
				limit = maxSkills
			}
			for _, skill := range context.Skills[:limit] {
				skillBlock.WriteString(fmt.Sprintf("### %s\n", skill.Name))
				skillBlock.WriteString(truncateRunes(skill.Content, maxEagerSkillSize))
				skillBlock.WriteString("\n\n")
			}
			skillBlock.WriteString("--- END PROJECT SKILLS ---\n")
		}

		sections = append(sections, skillBlock.String())
	}

	// 5. Stack info (brief)
	if context != nil && context.Stack.Language != "" {
		stackInfo := fmt.Sprintf("\n--- PROJECT STACK ---\nLanguage: %s | Build: %s | Test: %s\n--- END PROJECT STACK ---\n",
			context.Stack.Language, context.Stack.BuildTool, context.Stack.TestCommand)
		sections = append(sections, stackInfo)
	}

	composed := strings.Join(sections, "\n")

	// 6. Task (if provided — this is what the user actually wants done)
	if task != "" {
		composed += fmt.Sprintf("\n\n--- TASK ---\n%s\n--- END TASK ---\n", task)
	}

	// 7. On-demand skill injection: pre-scan task (and agent prompt) for @skill(name)
	// references and inject loaded content. This handles single-shot deploys where the
	// operative cannot interactively request skills (e.g., `moonbase deploy 4 "use
	// @skill(docker-build) to review the Dockerfile"`).
	//
	// NOTE: Full multi-turn pipeline scanning is out of scope for this implementation.
	// The compose-time pre-scan satisfies single-shot deploys (AC-3.2/3.3).
	if context != nil && context.HasRegistrySkills() {
		// Scan task and agent prompt for skill references
		scanText := task + " " + agentPrompt
		requested := ExtractSkillRequests(scanText)
		if len(requested) > 0 {
			resolved, notFound := ResolveSkills(context.SkillRegistry, requested)
			for _, skill := range resolved {
				composed += fmt.Sprintf("\n--- SKILL: %s ---\n%s\n--- END SKILL ---\n", skill.Name, skill.Content)
			}
			if len(notFound) > 0 {
				available := strings.Join(context.SkillRegistry.Names(), ", ")
				composed += fmt.Sprintf("\n--- SKILL RESOLUTION ---\nSkill(s) not found: %s. Available: %s\n--- END SKILL RESOLUTION ---\n",
					strings.Join(notFound, ", "), available)
			}
		}
	}

	// SECURITY: Final size guard — prevent OOM from accumulated content
	if len(composed) > maxTotalComposedSize {
		composed = composed[:maxTotalComposedSize] + "\n...(prompt truncated — exceeded maximum size)"
	}

	return composed
}

// stripFrontmatter removes YAML frontmatter from content if present.
func stripFrontmatter(content string) string {
	trimmed := strings.TrimLeft(content, "\n\r")
	if !strings.HasPrefix(trimmed, "---") {
		return content
	}

	// Find closing ---
	rest := trimmed[3:]
	if len(rest) > 0 && rest[0] == '\n' {
		rest = rest[1:]
	}

	closeIdx := strings.Index(rest, "\n---")
	if closeIdx == -1 {
		return content // No closing = return as-is
	}

	// Skip past the closing --- and any trailing newline
	body := rest[closeIdx+4:]
	if len(body) > 0 && body[0] == '\n' {
		body = body[1:]
	}

	return body
}

// ComposeCacheablePrefix returns the cacheable portion of a composed prompt —
// everything EXCEPT the task. This enables backends that support prompt caching
// (e.g., Claude's cache_control, OpenAI's cached prompts) to mark the boundary
// between the stable prefix and the per-request task suffix.
//
// Prompt caching works by prefix matching: requests sharing a common prefix hit
// the cache. By separating the prefix (steering + agent + specs + skills + stack)
// from the task, backends can cache the expensive static context and only process
// the new task portion on each request.
func ComposeCacheablePrefix(agentPrompt string, context *ProjectContext) string {
	return ComposePrompt(agentPrompt, context, "")
}
