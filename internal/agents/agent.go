// Package agents provides loading, parsing, and registry management for KND
// operative definitions. Each operative is a single .md file with YAML frontmatter
// containing metadata and a markdown body containing the agent's system prompt.
package agents

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
	Hooks            *HooksConfig   `yaml:"hooks,omitempty"`    // Lifecycle hook commands
	PipelinePosition *int           `yaml:"pipeline_position,omitempty"` // Position in core pipeline (nil = not a pipeline agent)
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
	OnActivate []Hook `yaml:"on_activate"` // Commands to run when the agent is activated
}

// Hook represents a lifecycle hook command.
type Hook struct {
	Command   string `yaml:"command"`    // Shell command to execute
	TimeoutMs int    `yaml:"timeout_ms"` // Maximum execution time in milliseconds
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
