package agents

import (
	"path/filepath"
	"sort"
	"strings"

	"github.com/rk-senne/moonbase/internal/logging"
)

// AgentSource constants for identifying where an agent was loaded from.
const (
	SourceBuiltIn = "built-in"
	SourceUser    = "user"
	SourceProject = "project"
)

// Registry holds all loaded agents and provides lookup by name, index, or role.
type Registry struct {
	dir    string
	agents []Agent
}

// NewRegistry creates a new Registry that will load agents from the given directory.
func NewRegistry(dir string) *Registry {
	return &Registry{dir: dir}
}

// LoadSync loads agents from the registry's directory synchronously.
// It stores the loaded agents in the registry and returns any error encountered.
// This is the preferred method for non-TUI consumers that don't need tea.Cmd.
func (r *Registry) LoadSync() error {
	agents, err := loadFromDir(r.dir)
	if err != nil {
		return err
	}
	for i := range agents {
		if agents[i].Source == "" {
			agents[i].Source = SourceBuiltIn
		}
	}
	r.agents = agents
	return nil
}

// LoadMultipleDirsSync loads agents from multiple directories and merges them synchronously.
// Priority order: later directories override earlier ones (same agent name).
// This is the preferred method for non-TUI consumers that don't need tea.Cmd.
func (r *Registry) LoadMultipleDirsSync(dirs ...string) error {
	merged, err := loadAndMergeDirs(dirs...)
	if err != nil {
		return err
	}
	r.agents = merged
	return nil
}

// ReloadMultipleDirs synchronously reloads agents from multiple directories.
func (r *Registry) ReloadMultipleDirs(dirs ...string) {
	merged, err := loadAndMergeDirs(dirs...)
	if err == nil {
		r.agents = merged
	}
}

// Reload synchronously reloads all agents from disk.
func (r *Registry) Reload() {
	agents, err := loadFromDir(r.dir)
	if err == nil {
		for i := range agents {
			if agents[i].Source == "" {
				agents[i].Source = SourceBuiltIn
			}
		}
		r.agents = agents
	}
}

// Count returns the number of loaded agents.
func (r *Registry) Count() int {
	return len(r.agents)
}

// Get returns the agent at the given index, or a placeholder if out of bounds.
func (r *Registry) Get(index int) Agent {
	if index >= 0 && index < len(r.agents) {
		return r.agents[index]
	}
	return Agent{Name: "unknown", Description: "Operative not found"}
}

// All returns all loaded agents in display order.
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

// DefaultAgentOrder defines the canonical display order for agents in the TUI sidebar.
// This is the single source of truth — config.DefaultConfig() references this slice.
var DefaultAgentOrder = []string{
	"numbuh-0", "numbuh-1", "numbuh-2", "numbuh-3", "numbuh-4", "numbuh-5",
	"numbuh-362", "numbuh-274", "numbuh-86", "numbuh-999", "numbuh-13",
	"knd-council", "sector-z", "numbuh-9",
}

func sortOrder(name string) int {
	for i, n := range DefaultAgentOrder {
		if n == name {
			return i
		}
	}
	return 99
}

// loadAndMergeDirs loads agents from multiple directories and merges them.
// Directories are processed in order; later directories override earlier ones for same-name agents.
// Source tags are assigned based on directory position:
//   - dirs[0] = built-in
//   - dirs[1] = user (~/.moonbase/agents/)
//   - dirs[2] = project (.kiro/agents/)
func loadAndMergeDirs(dirs ...string) ([]Agent, error) {
	sourceLabels := []string{SourceBuiltIn, SourceUser, SourceProject}

	// Map for deduplication (name → agent)
	agentMap := make(map[string]Agent)

	for i, dir := range dirs {
		if dir == "" {
			continue
		}

		agents, err := loadFromDir(dir)
		if err != nil {
			// Non-built-in dirs may not exist — skip gracefully
			if i > 0 {
				continue
			}
			return nil, err
		}

		source := SourceBuiltIn
		if i < len(sourceLabels) {
			source = sourceLabels[i]
		}

		for j := range agents {
			agents[j].Source = source
			name := agents[j].Name

			// Log warning if overriding a pipeline agent
			if existing, exists := agentMap[name]; exists {
				if existing.IsPipeline() && source != SourceBuiltIn {
					if logging.Logger != nil {
						logging.Logger.Warn("agent overrides built-in pipeline agent", "source", source, "agent", name)
					}
				}
			}
			agentMap[name] = agents[j]
		}
	}

	// Flatten map to slice
	result := make([]Agent, 0, len(agentMap))
	for _, a := range agentMap {
		result = append(result, a)
	}

	// Sort by KND operative order
	sort.Slice(result, func(i, j int) bool {
		return sortOrder(strings.ToLower(result[i].Name)) < sortOrder(strings.ToLower(result[j].Name))
	})

	return result, nil
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
			if logging.Logger != nil {
				logging.Logger.Warn("found .json agent files but no .md files; JSON format is deprecated", "count", len(jsonFiles))
			}
		}
		return nil, nil
	}

	for _, file := range mdFiles {
		agent, err := ParseAgentFile(file)
		if err != nil {
			if logging.Logger != nil {
				logging.Logger.Warn("skipping agent file", "file", filepath.Base(file), "error", err)
			}
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
