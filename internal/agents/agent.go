package agents

// Agent represents a KND operative loaded from a .md file with YAML frontmatter.
type Agent struct {
	// Frontmatter fields (parsed from YAML)
	Name             string         `yaml:"name"`
	Designation      string         `yaml:"designation"`
	Role             string         `yaml:"role"`
	Description      string         `yaml:"description"`
	Tools            []string       `yaml:"tools"`
	AutoTools        []string       `yaml:"auto_tools"`
	Shell            *ShellConfig   `yaml:"shell,omitempty"`
	Write            *WriteConfig   `yaml:"write,omitempty"`
	Routing          *RoutingConfig `yaml:"routing,omitempty"`
	Hooks            *HooksConfig   `yaml:"hooks,omitempty"`
	PipelinePosition *int           `yaml:"pipeline_position,omitempty"`
	Shortcut         string         `yaml:"shortcut"`
	Triggers         *string        `yaml:"triggers,omitempty"`

	// Derived fields (not from YAML)
	Prompt   string `yaml:"-"` // Full markdown body (the system prompt)
	FilePath string `yaml:"-"` // Source file path
}

// ShellConfig defines shell tool permissions for an agent.
type ShellConfig struct {
	AllowedCommands []string `yaml:"allowed_commands"`
	ReadOnly        bool     `yaml:"read_only"`
}

// WriteConfig defines file write permissions for an agent.
type WriteConfig struct {
	Auto             []string `yaml:"auto"`
	Denied           []string `yaml:"denied"`
	RequiresApproval []string `yaml:"requires_approval"`
}

// RoutingConfig defines which agents this operative can route to.
type RoutingConfig struct {
	Available []string `yaml:"available"`
	Trusted   []string `yaml:"trusted"`
}

// HooksConfig defines lifecycle hooks.
type HooksConfig struct {
	OnActivate []Hook `yaml:"on_activate"`
}

// Hook represents a lifecycle hook command.
type Hook struct {
	Command   string `yaml:"command"`
	TimeoutMs int    `yaml:"timeout_ms"`
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
