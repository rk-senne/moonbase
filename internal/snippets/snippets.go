package snippets

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type Snippet struct {
	Name    string `json:"name"`
	Content string `json:"content"`
	Agent   string `json:"agent,omitempty"` // optional: agent-specific
}

var snippetsPath string

func init() {
	home, _ := os.UserHomeDir()
	snippetsPath = filepath.Join(home, ".config", "moonbase", "snippets.json")
}

func Load() []Snippet {
	data, err := os.ReadFile(snippetsPath)
	if err != nil {
		return nil
	}
	var s []Snippet
	json.Unmarshal(data, &s)
	return s
}

func Save(s []Snippet) error {
	os.MkdirAll(filepath.Dir(snippetsPath), 0755)
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(snippetsPath, data, 0644)
}

func Add(name, content, agent string) error {
	all := Load()
	all = append(all, Snippet{Name: name, Content: content, Agent: agent})
	return Save(all)
}

func ForAgent(agent string) []Snippet {
	all := Load()
	var result []Snippet
	for _, s := range all {
		if s.Agent == "" || s.Agent == agent {
			result = append(result, s)
		}
	}
	return result
}
