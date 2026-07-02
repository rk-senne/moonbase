package agents

import (
	"log"
	"path/filepath"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// AgentsLoadedMsg is sent when agents finish loading
type AgentsLoadedMsg struct {
	Agents []Agent
	Err    error
}

// Registry holds all loaded agents
type Registry struct {
	dir    string
	agents []Agent
}

func NewRegistry(dir string) *Registry {
	return &Registry{dir: dir}
}

func (r *Registry) Load() tea.Cmd {
	return func() tea.Msg {
		agents, err := loadFromDir(r.dir)
		if err != nil {
			return AgentsLoadedMsg{Err: err}
		}
		r.agents = agents
		return AgentsLoadedMsg{Agents: agents}
	}
}

func (r *Registry) Reload() {
	agents, err := loadFromDir(r.dir)
	if err == nil {
		r.agents = agents
	}
}

func (r *Registry) Count() int {
	return len(r.agents)
}

func (r *Registry) Get(index int) Agent {
	if index >= 0 && index < len(r.agents) {
		return r.agents[index]
	}
	return Agent{Name: "unknown", Description: "Operative not found"}
}

func (r *Registry) All() []Agent {
	return r.agents
}

// GetByName returns an agent by name, or nil if not found.
func (r *Registry) GetByName(name string) *Agent {
	for i := range r.agents {
		if r.agents[i].Name == name {
			return &r.agents[i]
		}
	}
	return nil
}

// PipelineAgents returns agents sorted by pipeline_position (core pipeline only).
func (r *Registry) PipelineAgents() []Agent {
	var pipeline []Agent
	for _, a := range r.agents {
		if a.IsPipeline() {
			pipeline = append(pipeline, a)
		}
	}
	sort.Slice(pipeline, func(i, j int) bool {
		return *pipeline[i].PipelinePosition < *pipeline[j].PipelinePosition
	})
	return pipeline
}

// Specialists returns agents that are conditional (have trigger conditions).
func (r *Registry) Specialists() []Agent {
	var specialists []Agent
	for _, a := range r.agents {
		if a.IsConditional() {
			specialists = append(specialists, a)
		}
	}
	return specialists
}

// agentOrder defines display order for the sidebar
var agentOrder = []string{
	"numbuh-0", "numbuh-1", "numbuh-2", "numbuh-3", "numbuh-4", "numbuh-5",
	"numbuh-362", "numbuh-274", "numbuh-86", "numbuh-999", "numbuh-13",
	"knd-council", "sector-z", "numbuh-9",
}

func sortOrder(name string) int {
	for i, n := range agentOrder {
		if n == name {
			return i
		}
	}
	return 99
}

func loadFromDir(dir string) ([]Agent, error) {
	var agents []Agent

	// Load .md files (new format)
	mdFiles, err := filepath.Glob(filepath.Join(dir, "*.md"))
	if err != nil {
		return nil, err
	}

	if len(mdFiles) == 0 {
		// Check if old .json files exist and warn
		jsonFiles, _ := filepath.Glob(filepath.Join(dir, "*.json"))
		if len(jsonFiles) > 0 {
			log.Printf("WARNING: Found %d .json agent files but no .md files. JSON format is deprecated. Convert to .md with YAML frontmatter.", len(jsonFiles))
		}
		return nil, nil
	}

	for _, file := range mdFiles {
		agent, err := ParseAgentFile(file)
		if err != nil {
			log.Printf("WARNING: skipping %s: %v", filepath.Base(file), err)
			continue
		}
		agents = append(agents, *agent)
	}

	// Sort by KND operative order
	sort.Slice(agents, func(i, j int) bool {
		return sortOrder(strings.ToLower(agents[i].Name)) < sortOrder(strings.ToLower(agents[j].Name))
	})

	return agents, nil
}
