package compile

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// WriteAgent writes the compiled KiroAgent JSON and companion prompt file to targetDir.
// Produces <name>.json (indented, 0644) and <name>.prompt.md (0644).
// Guards against path traversal in the agent name.
func WriteAgent(ka *KiroAgent, promptBody string, targetDir string) error {
	if ka.Name == "" {
		return fmt.Errorf("agent name is empty")
	}

	// Guard: reject path traversal in name
	if strings.Contains(ka.Name, "/") || strings.Contains(ka.Name, "\\") ||
		strings.Contains(ka.Name, "..") || ka.Name == "." {
		return fmt.Errorf("agent name %q contains path traversal characters", ka.Name)
	}

	// Ensure target directory exists
	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		return fmt.Errorf("creating target directory %s: %w", targetDir, err)
	}

	// Write JSON file
	jsonPath := filepath.Join(targetDir, ka.Name+".json")
	jsonData, err := json.MarshalIndent(ka, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling agent %s to JSON: %w", ka.Name, err)
	}
	// Append newline for POSIX compliance
	jsonData = append(jsonData, '\n')
	if err := os.WriteFile(jsonPath, jsonData, 0o644); err != nil {
		return fmt.Errorf("writing %s: %w", jsonPath, err)
	}

	// Write companion prompt file
	promptPath := filepath.Join(targetDir, ka.Name+".prompt.md")
	if err := os.WriteFile(promptPath, []byte(promptBody), 0o644); err != nil {
		return fmt.Errorf("writing %s: %w", promptPath, err)
	}

	return nil
}
