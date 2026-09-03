package main

import "github.com/rk-senne/moonbase/internal/agents"

// agentLookup is the only capability the mission pipeline needs from the agent
// registry: resolving an agent by name.
//
// Declared here, at the point of use, rather than exported from internal/agents —
// that keeps this package depending on one method instead of the registry's full
// eleven-method surface (Interface Segregation). *agents.Registry satisfies it
// without any changes, so this narrows the dependency without adding indirection
// or requiring a test double.
type agentLookup interface {
	GetByName(name string) *agents.Agent
}

// Compile-time assertion that the real registry satisfies the narrowed interface.
var _ agentLookup = (*agents.Registry)(nil)
