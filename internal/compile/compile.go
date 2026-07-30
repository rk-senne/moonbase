package compile

import (
	"fmt"

	"github.com/rk-senne/moonbase/internal/agents"
)

// Compile translates a moonbase Agent into a KiroAgent struct and the prompt body.
// Returns the compiled struct, prompt body (markdown content), and any error.
//
// Mapping rules (authoritative — from Phase 0 schema validation):
//   - name -> name; description -> description
//   - markdown body -> companion .prompt.md file, prompt = "file://<name>.prompt.md"
//   - tools -> tools; auto_tools -> allowedTools (omit if empty)
//   - shell.allowed_commands -> toolsSettings.shell.allowedCommands
//   - shell.read_only==true -> toolsSettings.shell.denyByDefault=true + autoAllowReadonly=true
//   - write.auto -> toolsSettings.write.allowedPaths
//   - write.denied -> toolsSettings.write.deniedPaths
//   - hooks.on_activate -> hooks["agentSpawn"]
//   - hooks.pre_tool_use -> hooks["preToolUse"]
//   - hooks.post_tool_use -> hooks["postToolUse"]
//   - hooks.on_complete -> hooks["stop"]
//   - mcp_servers -> mcpServers map keyed by name
//   - mcp_servers[].allowed_tools -> append "@<name>/<tool>" to top-level allowedTools
//
// Omitted (moonbase-only): routing, pipeline_position, triggers, shortcut,
// guardrails, handoff, output_schema.
func Compile(agent agents.Agent) (*KiroAgent, string, error) {
	if agent.Name == "" {
		return nil, "", fmt.Errorf("agent has empty name")
	}

	ka := &KiroAgent{
		Name:        agent.Name,
		Description: agent.Description,
		Prompt:      fmt.Sprintf("file://%s.prompt.md", agent.Name),
	}

	// Tools: direct copy
	if len(agent.Tools) > 0 {
		ka.Tools = make([]string, len(agent.Tools))
		copy(ka.Tools, agent.Tools)
	}

	// AllowedTools: from auto_tools (omit if empty per R-3.4)
	var allowedTools []string
	if len(agent.AutoTools) > 0 {
		allowedTools = make([]string, len(agent.AutoTools))
		copy(allowedTools, agent.AutoTools)
	}

	// ToolsSettings
	ka.ToolsSettings = compileToolsSettings(agent)

	// Hooks
	ka.Hooks = compileHooks(agent)

	// MCP Servers
	if len(agent.MCPServers) > 0 {
		ka.MCPServers = make(map[string]KiroMCPServer, len(agent.MCPServers))
		for _, srv := range agent.MCPServers {
			ka.MCPServers[srv.Name] = KiroMCPServer{
				Command: srv.Command,
				Args:    srv.Args,
				Env:     srv.Env,
			}
			// Append @<name>/<tool> to allowedTools for each allowed_tools entry
			for _, tool := range srv.AllowedTools {
				allowedTools = append(allowedTools, fmt.Sprintf("@%s/%s", srv.Name, tool))
			}
		}
	}

	// Set allowedTools only if non-empty
	if len(allowedTools) > 0 {
		ka.AllowedTools = allowedTools
	}

	return ka, agent.Prompt, nil
}

// compileToolsSettings translates shell and write configs.
func compileToolsSettings(agent agents.Agent) *KiroToolsSettings {
	var ts KiroToolsSettings
	hasSettings := false

	if agent.Shell != nil {
		shell := &KiroShellSettings{}
		if len(agent.Shell.AllowedCommands) > 0 {
			shell.AllowedCommands = make([]string, len(agent.Shell.AllowedCommands))
			copy(shell.AllowedCommands, agent.Shell.AllowedCommands)
		}
		if agent.Shell.ReadOnly {
			// Map read_only to denyByDefault + autoAllowReadonly
			// (no toolset field, no readOnly field — see Phase 0 corrections)
			shell.DenyByDefault = true
			shell.AutoAllowReadonly = true
		}
		ts.Shell = shell
		hasSettings = true
	}

	if agent.Write != nil {
		write := &KiroWriteSettings{}
		if len(agent.Write.Auto) > 0 {
			write.AllowedPaths = make([]string, len(agent.Write.Auto))
			copy(write.AllowedPaths, agent.Write.Auto)
		}
		if len(agent.Write.Denied) > 0 {
			write.DeniedPaths = make([]string, len(agent.Write.Denied))
			copy(write.DeniedPaths, agent.Write.Denied)
		}
		// Only include write settings if there's something to say
		if len(write.AllowedPaths) > 0 || len(write.DeniedPaths) > 0 {
			ts.Write = write
			hasSettings = true
		}
	}

	if !hasSettings {
		return nil
	}
	return &ts
}

// compileHooks translates moonbase hook config to Kiro hook format.
// Kiro lifecycle names: agentSpawn, preToolUse, postToolUse, stop
func compileHooks(agent agents.Agent) map[string][]KiroHook {
	if agent.Hooks == nil {
		return nil
	}

	hooks := make(map[string][]KiroHook)

	if len(agent.Hooks.OnActivate) > 0 {
		hooks["agentSpawn"] = convertHooks(agent.Hooks.OnActivate)
	}
	if len(agent.Hooks.PreToolUse) > 0 {
		hooks["preToolUse"] = convertHooks(agent.Hooks.PreToolUse)
	}
	if len(agent.Hooks.PostToolUse) > 0 {
		hooks["postToolUse"] = convertHooks(agent.Hooks.PostToolUse)
	}
	if len(agent.Hooks.OnComplete) > 0 {
		hooks["stop"] = convertHooks(agent.Hooks.OnComplete)
	}

	if len(hooks) == 0 {
		return nil
	}
	return hooks
}

// convertHooks converts a slice of moonbase Hook to KiroHook.
func convertHooks(src []agents.Hook) []KiroHook {
	result := make([]KiroHook, len(src))
	for i, h := range src {
		result[i] = KiroHook{
			Command:   h.Command,
			TimeoutMs: h.TimeoutMs,
		}
	}
	return result
}
