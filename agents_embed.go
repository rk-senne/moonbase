// Package moonbase is the module root. It embeds the canonical agent
// definitions so the compiled binary can install them from any directory
// without needing a repository checkout on disk.
//
// The embed directive must live here (repository root, alongside agents/)
// because go:embed paths cannot traverse upward with "..".
package moonbase

import (
	"embed"
	"io/fs"
)

//go:embed agents/*.md
var embeddedAgents embed.FS

// AgentsFS returns the embedded agent .md files as a filesystem rooted directly
// at the agent files, so entries read as "numbuh-4.md" rather than
// "agents/numbuh-4.md". The agents are frozen into the binary at build time.
func AgentsFS() (fs.FS, error) {
	return fs.Sub(embeddedAgents, "agents")
}
