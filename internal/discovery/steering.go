package discovery

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// steeringFrontmatter represents the YAML frontmatter in steering files.
type steeringFrontmatter struct {
	Inclusion string `yaml:"inclusion"`
}

// discoverSteering finds and loads steering files from .kiro/steering/.
// Files with `inclusion: manual` in their frontmatter are excluded.
func discoverSteering(projectDir string) ([]SteeringFile, error) {
	steeringDir := filepath.Join(projectDir, ".kiro", "steering")
	if _, err := os.Stat(steeringDir); os.IsNotExist(err) {
		return nil, nil
	}

	files, err := filepath.Glob(filepath.Join(steeringDir, "*.md"))
	if err != nil {
		return nil, fmt.Errorf("globbing steering files: %w", err)
	}

	var steering []SteeringFile

	for _, file := range files {
		content, err := os.ReadFile(file)
		if err != nil {
			continue
		}

		// Check inclusion policy
		if shouldExclude(content) {
			continue
		}

		name := strings.TrimSuffix(filepath.Base(file), ".md")
		steering = append(steering, SteeringFile{
			Name:    name,
			Path:    file,
			Content: string(content),
		})
	}

	return steering, nil
}

// shouldExclude checks if a steering file has `inclusion: manual` in its frontmatter.
// If so, it should be excluded from automatic loading.
func shouldExclude(content []byte) bool {
	content = bytes.TrimLeft(content, "\n\r")

	// Must start with ---
	if !bytes.HasPrefix(content, []byte("---")) {
		return false // No frontmatter = include by default
	}

	// Find closing ---
	rest := content[3:]
	if len(rest) > 0 && rest[0] == '\n' {
		rest = rest[1:]
	}

	closeIdx := bytes.Index(rest, []byte("\n---"))
	if closeIdx == -1 {
		return false // Malformed frontmatter = include by default
	}

	yamlContent := rest[:closeIdx]

	var fm steeringFrontmatter
	if err := yaml.Unmarshal(yamlContent, &fm); err != nil {
		return false // Parse error = include by default
	}

	return fm.Inclusion == "manual"
}
