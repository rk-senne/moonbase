package tui

import (
	"strings"

	"github.com/rk-senne/moonbase/internal/agents"
)

// agentEntry represents a sidebar roster entry.
type agentEntry struct {
	key   string
	name  string
	role  string
	index int
}

// agentGroup represents a grouped section in the sidebar.
type agentGroup struct {
	title   string
	entries []agentEntry
}

// buildSidebarGroups constructs sidebar roster from the registry.
func buildSidebarGroups(reg *agents.Registry) []agentGroup {
	allAgents := reg.All()

	var coreEntries []agentEntry
	var specialistEntries []agentEntry
	var metaEntries []agentEntry

	for i, a := range allAgents {
		key := extractSidebarKey(a.Name)
		name := a.Designation
		if name == "" {
			name = a.Name
		}
		role := a.Role
		if len(role) > 10 {
			role = role[:10]
		}

		entry := agentEntry{key: key, name: name, role: role, index: i}

		if a.PipelinePosition != nil && *a.PipelinePosition >= 0 && *a.PipelinePosition <= 5 {
			coreEntries = append(coreEntries, entry)
		} else if a.Name == "knd-council" || a.Name == "sector-z" {
			metaEntries = append(metaEntries, entry)
		} else {
			specialistEntries = append(specialistEntries, entry)
		}
	}

	var groups []agentGroup
	if len(coreEntries) > 0 {
		groups = append(groups, agentGroup{title: "SECTOR V", entries: coreEntries})
	}
	if len(specialistEntries) > 0 {
		groups = append(groups, agentGroup{title: "SPECIALISTS", entries: specialistEntries})
	}
	if len(metaEntries) > 0 {
		groups = append(groups, agentGroup{title: "META", entries: metaEntries})
	}

	// Fallback if registry is empty
	if len(groups) == 0 {
		groups = []agentGroup{
			{"SECTOR V", []agentEntry{
				{"0", "Numbuh 0", "Overseer", -1},
				{"1", "Numbuh 1", "Analyst", -1},
				{"2", "Numbuh 2", "Architect", -1},
				{"3", "Numbuh 3", "Implement", -1},
				{"4", "Numbuh 4", "QA", -1},
				{"5", "Numbuh 5", "Reviewer", -1},
			}},
		}
	}

	return groups
}

// extractSidebarKey returns a display key for the sidebar from an agent name.
func extractSidebarKey(name string) string {
	switch name {
	case "knd-council":
		return "K"
	case "sector-z":
		return "Z"
	default:
		if strings.HasPrefix(name, "numbuh-") {
			num := strings.TrimPrefix(name, "numbuh-")
			return num
		}
		return string(strings.ToUpper(name[:1]))
	}
}
