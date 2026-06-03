package agents

// Agent represents a KND operative configuration
type Agent struct {
	Name            string            `json:"name"`
	Description     string            `json:"description"`
	Prompt          string            `json:"prompt"`
	Tools           []string          `json:"tools"`
	AllowedTools    []string          `json:"allowedTools"`
	ToolsSettings   map[string]any    `json:"toolsSettings"`
	Resources       []string          `json:"resources"`
	Hooks           map[string][]Hook `json:"hooks"`
	KeyboardShortcut string           `json:"keyboardShortcut"`
	WelcomeMessage  string            `json:"welcomeMessage"`
}

// Hook represents a lifecycle hook
type Hook struct {
	Command   string `json:"command"`
	TimeoutMs int    `json:"timeout_ms"`
}
