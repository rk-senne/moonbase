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

// maxTotalComposedSize is the absolute maximum size of a composed prompt (512KB).
// Guards against OOM if many large files are discovered.
const maxTotalComposedSize = 512 * 1024

// ComposePrompt builds the full prompt for an agent deployment.
// Order: project steering rules → agent prompt → spec context → task
//
// SECURITY TRUST BOUNDARY:
// Content from steering and spec files is injected into the prompt structure.
// This content is read from the local filesystem (.kiro/ directory) which is
// under the project owner's control. However, injected content could
// theoretically attempt to override agent instructions via prompt injection.
//
// Mitigations:
// 1. Content is wrapped in clearly delimited sections (--- PROJECT RULES --- etc.)
// 2. The agent prompt (identity/instructions) comes AFTER steering, so the agent's
//    core directives have final authority in the prompt order
// 3. Content is size-limited to prevent context window exhaustion
// 4. Frontmatter is stripped to avoid YAML injection into the prompt
// 5. Total composed size is capped to prevent OOM
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
			if len(content) > maxSteeringSize {
				content = content[:maxSteeringSize] + "\n...(truncated for size)"
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
				content := f.Content
				if len(content) > maxSpecSize {
					content = content[:maxSpecSize] + "\n...(truncated)"
				}
				specBlock.WriteString(fmt.Sprintf("### %s.md\n\n%s\n\n", f.Type, content))
			}
		}

		specBlock.WriteString("--- END PROJECT SPEC CONTEXT ---\n")
		sections = append(sections, specBlock.String())
	}

	// 4. Stack info (brief)
	if context != nil && context.Stack.Language != "" {
		stackInfo := fmt.Sprintf("\n--- PROJECT STACK ---\nLanguage: %s | Build: %s | Test: %s\n--- END PROJECT STACK ---\n",
			context.Stack.Language, context.Stack.BuildTool, context.Stack.TestCommand)
		sections = append(sections, stackInfo)
	}

	composed := strings.Join(sections, "\n")

	// 5. Task (if provided — this is what the user actually wants done)
	if task != "" {
		composed += fmt.Sprintf("\n\n--- TASK ---\n%s\n--- END TASK ---\n", task)
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
