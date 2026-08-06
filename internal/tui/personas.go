package tui

import (
	"image/color"
	"strings"

	"charm.land/lipgloss/v2"
)

// AgentPersona is the display persona used to label an operative's pipeline
// feedback, so the operator can tell who is speaking and in what voice.
// Designations are the authentic KND character names from agents/*.md.
type AgentPersona struct {
	Operative   string // display name, e.g. "Numbuh 4"
	Designation string // character name, e.g. "Wallabee Beatles"
	Personality string // short voice/personality tagline
	Color       color.Color
}

// agentPersonas is keyed by agent id (matching Phase.AgentName). The taglines
// distill each operative's Identity section into one line of voice.
var agentPersonas = map[string]AgentPersona{
	"numbuh-0":    {"Numbuh 0", "Monty Uno", "The founder — guards the whole system.", lipgloss.Color("#F97316")},
	"numbuh-1":    {"Numbuh 1", "Nigel Uno", "British, decisive. Requirements first, no loose ends.", lipgloss.Color("#FF6B6B")},
	"numbuh-2":    {"Numbuh 2", "Hoagie Gilligan", "Inventor. Loves an elegant blueprint.", lipgloss.Color("#4ECDC4")},
	"numbuh-3":    {"Numbuh 3", "Kuki Sanban", "Cheerful. Writes humane, readable code.", lipgloss.Color("#A8E6CF")},
	"numbuh-4":    {"Numbuh 4", "Wallabee Beatles", "Australian, blunt, brave. \"Does it hold when I hit it?\"", lipgloss.Color("#FFE66D")},
	"numbuh-5":    {"Numbuh 5", "Abigail Lincoln", "Cool and calm. Nothing ships past her half-baked.", lipgloss.Color("#C4B5FD")},
	"numbuh-9":    {"Numbuh 9", "Maurice", "Veteran. Bridges the old world and the new.", lipgloss.Color("#22D3EE")},
	"numbuh-13":   {"Numbuh 13", "Numbuh 13", "Unlucky by name. Breaks things on purpose.", lipgloss.Color("#F59E0B")},
	"numbuh-86":   {"Numbuh 86", "Fanny Fulbright", "Irish, fiery. If it's dead, it gets decommissioned.", lipgloss.Color("#FB7185")},
	"numbuh-274":  {"Numbuh 274", "Chad Dickson", "By the book. Thinks like an attacker.", lipgloss.Color("#EF4444")},
	"numbuh-362":  {"Numbuh 362", "Rachel T. McKenzie", "Supreme Leader. Owns production.", lipgloss.Color("#06B6D4")},
	"numbuh-999":  {"Numbuh 999", "Mrs. Uno", "Cartographer. Maps everything in writing.", lipgloss.Color("#93C5FD")},
	"sector-z":    {"Sector Z", "The Lost Operatives", "Ghosts of code past. Excavate the ancient.", lipgloss.Color("#9CA3AF")},
	"knd-council": {"KND Council", "Kids Next Door", "The whole council, assembled.", lipgloss.Color("#FFD700")},
}

// personaKey normalizes an operative label or agent id to a persona map key.
// Accepts "Numbuh 4", "numbuh-4", "Sector Z", "sector-z", "KND Council", etc.
func personaKey(s string) string {
	k := strings.ToLower(strings.TrimSpace(s))
	return strings.ReplaceAll(k, " ", "-")
}

// personaFor resolves the persona for an operative display name or agent id.
// Unknown agents still get a usable label so feedback is never anonymous.
func personaFor(nameOrID string) AgentPersona {
	if p, ok := agentPersonas[personaKey(nameOrID)]; ok {
		return p
	}
	label := strings.TrimSpace(nameOrID)
	if label == "" {
		label = "Operative"
	}
	return AgentPersona{Operative: label, Color: lipgloss.Color("#00FF88")}
}
