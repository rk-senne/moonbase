// Package agents provides loading, parsing, and registry management for KND
// operative definitions. Each operative is a single .md file with YAML frontmatter
// containing metadata and a markdown body containing the agent's system prompt.
package agents

// MCPServerConfig defines an MCP server available to the agent.
type MCPServerConfig struct {
	Name         string            `yaml:"name"`                          // Unique server name (required)
	Command      string            `yaml:"command"`                       // Command to launch the MCP server (required)
	Args         []string          `yaml:"args,omitempty"`                // Arguments for the command
	Env          map[string]string `yaml:"env,omitempty"`                 // Environment variables for the server process
	AllowedTools []string          `yaml:"allowed_tools,omitempty"`       // Scoped tool filtering for this server
}

// Agent represents a KND operative loaded from a .md file with YAML frontmatter.
type Agent struct {
	// --- Frontmatter fields (parsed from YAML) ---

	Name             string         `yaml:"name"`               // Unique identifier (e.g., "numbuh-4")
	Designation      string         `yaml:"designation"`        // Character name (e.g., "Wallabee Beatles")
	Role             string         `yaml:"role"`               // Functional role (e.g., "QA / Verification")
	Description      string         `yaml:"description"`        // Brief description for display
	Tools            []string       `yaml:"tools"`              // Allowed tools (read, shell, grep, etc.)
	AutoTools        []string       `yaml:"auto_tools"`         // Tools that auto-approve without confirmation
	Shell            *ShellConfig   `yaml:"shell,omitempty"`    // Shell command permissions
	Write            *WriteConfig   `yaml:"write,omitempty"`    // File write permissions
	Routing          *RoutingConfig `yaml:"routing,omitempty"`  // Which agents this operative can hand off to
	Hooks            *HooksConfig      `yaml:"hooks,omitempty"`         // Lifecycle hook commands
	Guardrails       *GuardrailsConfig `yaml:"guardrails,omitempty"`    // Runtime guardrails
	Handoff          *HandoffConfig    `yaml:"handoff,omitempty"`       // Handoff format between agents
	OutputSchema     string            `yaml:"output_schema,omitempty"` // Expected output format hint (e.g. 'json', 'markdown', 'structured')
	PipelinePosition *int              `yaml:"pipeline_position,omitempty"` // Position in core pipeline (nil = not a pipeline agent)
	MCPServers       []MCPServerConfig `yaml:"mcp_servers,omitempty"`       // MCP servers available to this agent
	Shortcut         string         `yaml:"shortcut"`           // Keyboard shortcut in TUI
	Triggers         *string        `yaml:"triggers,omitempty"` // Trigger conditions for conditional specialists

	// --- Derived fields (not from YAML) ---

	Prompt   string `yaml:"-"` // Full markdown body (the agent's system prompt)
	FilePath string `yaml:"-"` // Absolute path to the source .md file
	Source   string `yaml:"-"` // Where this agent was loaded from: "built-in", "user", "project"
}

// ShellConfig defines shell tool permissions for an agent.
type ShellConfig struct {
	AllowedCommands []string `yaml:"allowed_commands"` // Allowlisted shell commands
	ReadOnly        bool     `yaml:"read_only"`        // If true, only read-only commands are permitted
}

// WriteConfig defines file write permissions for an agent.
type WriteConfig struct {
	Auto             []string `yaml:"auto"`              // Glob patterns auto-approved for writing
	Denied           []string `yaml:"denied"`            // Glob patterns denied for writing
	RequiresApproval []string `yaml:"requires_approval"` // Glob patterns requiring human approval
}

// RoutingConfig defines which agents this operative can hand off to.
type RoutingConfig struct {
	Available []string `yaml:"available"` // Agents this operative can route to
	Trusted   []string `yaml:"trusted"`   // Agents whose output is accepted without verification
}

// HooksConfig defines lifecycle hooks for an agent.
type HooksConfig struct {
	OnActivate  []Hook `yaml:"on_activate"`             // Commands to run when the agent is activated
	PreToolUse  []Hook `yaml:"pre_tool_use,omitempty"`  // Run before tool calls (exit 2 = block)
	PostToolUse []Hook `yaml:"post_tool_use,omitempty"` // Run after tool calls
	OnComplete  []Hook `yaml:"on_complete,omitempty"`   // Run when agent finishes
}

// Hook represents a lifecycle hook command.
type Hook struct {
	Command   string `yaml:"command"`    // Shell command to execute
	TimeoutMs int    `yaml:"timeout_ms"` // Maximum execution time in milliseconds
}

// GuardrailsConfig defines runtime guardrails for an agent.
// Pattern adapted from OpenAI Agents SDK (tripwire guardrails) and AWS Bedrock
// Guardrails (layered defense). These are evaluated by the pipeline orchestrator
// and backends that support enforcement.
type GuardrailsConfig struct {
	MaxTurns    int      `yaml:"max_turns,omitempty"`    // Max LLM turns before forced stop (0=unlimited)
	MaxOutput   int      `yaml:"max_output,omitempty"`   // Max output chars (0=unlimited)
	InputRules  []string `yaml:"input_rules,omitempty"`  // Input validation rules (regex patterns to reject)
	OutputRules []string `yaml:"output_rules,omitempty"` // Output validation rules (must-contain patterns)
	StopWords   []string `yaml:"stop_words,omitempty"`   // Words that trigger immediate stop
}

// HandoffConfig defines the handoff format between agents.
// Based on OpenAI Agents SDK handoff primitive and Kiro's spawn/delegation pattern.
type HandoffConfig struct {
	Format   string   `yaml:"format,omitempty"`   // 'structured' or 'freeform' (default: freeform)
	Required []string `yaml:"required,omitempty"` // Required fields in handoff (e.g. ["next_agent", "reason", "risk"])
}

// HasShell returns true if the agent has shell tool access.
func (a *Agent) HasShell() bool {
	return a.Shell != nil
}

// HasWrite returns true if the agent has write tool access.
func (a *Agent) HasWrite() bool {
	return a.Write != nil
}

// IsPipeline returns true if the agent has a pipeline position (core pipeline agent).
func (a *Agent) IsPipeline() bool {
	return a.PipelinePosition != nil
}

// IsConditional returns true if the agent has trigger conditions (conditional specialist).
func (a *Agent) IsConditional() bool {
	return a.Triggers != nil
}

// HasGuardrails returns true if guardrails are configured for this agent.
func (a *Agent) HasGuardrails() bool {
	return a.Guardrails != nil
}

// MaxTurnsLimit returns the max LLM turns limit, or 0 if guardrails are not configured.
func (a *Agent) MaxTurnsLimit() int {
	if a.Guardrails == nil {
		return 0
	}
	return a.Guardrails.MaxTurns
}

// MaxOutputLimit returns the max output chars limit, or 0 if guardrails are not configured.
func (a *Agent) MaxOutputLimit() int {
	if a.Guardrails == nil {
		return 0
	}
	return a.Guardrails.MaxOutput
}

// HasMCPServers returns true if the agent has MCP servers configured.
func (a *Agent) HasMCPServers() bool {
	return len(a.MCPServers) > 0
}

// HasPreToolHooks returns true if the agent has pre-tool-use hooks configured.
func (a *Agent) HasPreToolHooks() bool {
	return a.Hooks != nil && len(a.Hooks.PreToolUse) > 0
}

// HasPostToolHooks returns true if the agent has post-tool-use hooks configured.
func (a *Agent) HasPostToolHooks() bool {
	return a.Hooks != nil && len(a.Hooks.PostToolUse) > 0
}
