package discovery

import (
	"fmt"
	"strings"
)

// ComposePrompt builds the full prompt for an agent deployment.
// Order: project steering rules → agent prompt → spec context → task
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
				// Include a summary (first 1500 chars per spec file)
				content := f.Content
				if len(content) > 1500 {
					content = content[:1500] + "\n...(truncated)"
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
