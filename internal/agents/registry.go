package agents

import (
	"encoding/json"
	"os"
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
	return Agent{Name: "Unknown", Description: "Operative not found"}
}

func (r *Registry) All() []Agent {
	return r.agents
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

	files, err := filepath.Glob(filepath.Join(dir, "*.json"))
	if err != nil {
		return nil, err
	}

	for _, file := range files {
		data, err := os.ReadFile(file)
		if err != nil {
			continue
		}
		var agent Agent
		if err := json.Unmarshal(data, &agent); err != nil {
			continue
		}
		// Skip knd-council from roster (it's a meta-agent)
		if agent.Name == "knd-council" {
			continue
		}
		agents = append(agents, agent)
	}

	// Sort by KND operative order
	sort.Slice(agents, func(i, j int) bool {
		return sortOrder(agentName(agents[i])) < sortOrder(agentName(agents[j]))
	})

	return agents, nil
}

func agentName(a Agent) string {
	return strings.ToLower(a.Name)
}
