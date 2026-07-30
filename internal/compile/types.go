// Package compile translates moonbase Agent structs into Kiro-native agent JSON.
// The compiled JSON is a derived artifact — never hand-edited, always regenerated
// from the markdown agent definitions in agents/.
package compile

// KiroAgent is the target JSON structure for a compiled Kiro-native agent.
// Field names and JSON tags match the schema validated by `kiro-cli agent validate`.
type KiroAgent struct {
	Name          string                    `json:"name"`
	Description   string                    `json:"description,omitempty"`
	Prompt        string                    `json:"prompt"`
	Tools         []string                  `json:"tools,omitempty"`
	AllowedTools  []string                  `json:"allowedTools,omitempty"`
	ToolsSettings *KiroToolsSettings        `json:"toolsSettings,omitempty"`
	Hooks         map[string][]KiroHook     `json:"hooks,omitempty"`
	MCPServers    map[string]KiroMCPServer  `json:"mcpServers,omitempty"`
}

// KiroToolsSettings configures tool-level permissions.
type KiroToolsSettings struct {
	Shell *KiroShellSettings `json:"shell,omitempty"`
	Write *KiroWriteSettings `json:"write,omitempty"`
}

// KiroShellSettings configures shell tool permissions.
type KiroShellSettings struct {
	AllowedCommands  []string `json:"allowedCommands,omitempty"`
	DeniedCommands   []string `json:"deniedCommands,omitempty"`
	AutoAllowReadonly bool    `json:"autoAllowReadonly,omitempty"`
	DenyByDefault    bool    `json:"denyByDefault,omitempty"`
}

// KiroWriteSettings configures write tool permissions.
type KiroWriteSettings struct {
	AllowedPaths []string `json:"allowedPaths,omitempty"`
	DeniedPaths  []string `json:"deniedPaths,omitempty"`
}

// KiroHook defines a lifecycle hook command in Kiro's schema.
type KiroHook struct {
	Command   string `json:"command"`
	Matcher   string `json:"matcher,omitempty"`
	TimeoutMs int    `json:"timeout_ms,omitempty"`
}

// KiroMCPServer defines an MCP server in Kiro's schema.
type KiroMCPServer struct {
	Command string            `json:"command"`
	Args    []string          `json:"args,omitempty"`
	Env     map[string]string `json:"env,omitempty"`
}
