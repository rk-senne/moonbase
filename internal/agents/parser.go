package agents

import (
	"bytes"
	"errors"
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

var (
	// ErrNoFrontmatter is returned when a file does not begin with --- YAML frontmatter.
	ErrNoFrontmatter = errors.New("no YAML frontmatter found")
	// ErrMalformedFrontmatter is returned when the closing --- delimiter is missing.
	ErrMalformedFrontmatter = errors.New("malformed YAML frontmatter (missing closing ---)")
)

// frontmatterDelim is the YAML frontmatter delimiter
var frontmatterDelim = []byte("---")

// SplitFrontmatter splits a markdown file with YAML frontmatter into its parts.
// Expected format:
//
//	---
//	yaml content
//	---
//	markdown body
//
// Returns (yamlBytes, bodyBytes, error).
func SplitFrontmatter(content []byte) ([]byte, []byte, error) {
	content = bytes.TrimLeft(content, "\n\r")

	// Must start with ---
	if !bytes.HasPrefix(content, frontmatterDelim) {
		return nil, nil, ErrNoFrontmatter
	}

	// Find the closing ---
	rest := content[len(frontmatterDelim):]

	// Skip the newline after opening ---
	if len(rest) > 0 && rest[0] == '\n' {
		rest = rest[1:]
	} else if len(rest) > 1 && rest[0] == '\r' && rest[1] == '\n' {
		rest = rest[2:]
	}

	// Find closing delimiter (must be at start of a line)
	closeIdx := bytes.Index(rest, append([]byte("\n"), frontmatterDelim...))
	if closeIdx == -1 {
		// Try \r\n
		closeIdx = bytes.Index(rest, append([]byte("\r\n"), frontmatterDelim...))
		if closeIdx == -1 {
			return nil, nil, ErrMalformedFrontmatter
		}
	}

	yamlBytes := rest[:closeIdx]
	remaining := rest[closeIdx+1:] // skip the \n before ---

	// Skip the closing --- and any trailing newline
	remaining = bytes.TrimPrefix(remaining, frontmatterDelim)
	if len(remaining) > 0 && remaining[0] == '\n' {
		remaining = remaining[1:]
	} else if len(remaining) > 1 && remaining[0] == '\r' && remaining[1] == '\n' {
		remaining = remaining[2:]
	}

	return yamlBytes, remaining, nil
}

// ParseAgentFile reads a .md agent file and returns a parsed Agent.
// The YAML frontmatter populates the struct fields.
// The markdown body (everything after frontmatter) is stored in Agent.Prompt.
func ParseAgentFile(path string) (*Agent, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading agent file %s: %w", path, err)
	}

	yamlBytes, body, err := SplitFrontmatter(content)
	if err != nil {
		return nil, fmt.Errorf("parsing frontmatter in %s: %w", path, err)
	}

	var agent Agent
	if err := yaml.Unmarshal(yamlBytes, &agent); err != nil {
		return nil, fmt.Errorf("parsing YAML in %s: %w", path, err)
	}

	agent.Prompt = string(body)
	agent.FilePath = path

	return &agent, nil
}
