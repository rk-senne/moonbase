// Package moonbase embeds the curated skills library so the compiled binary can
// scaffold them into any project via `moonbase init` without needing a
// repository checkout on disk.
//
// The embed directive must live here (repository root, alongside skills/)
// because go:embed paths cannot traverse upward with "..".
package moonbase

import (
	"embed"
	"io/fs"
)

//go:embed skills/*.md
var embeddedSkills embed.FS

// SkillsFS returns the embedded skill .md files as a filesystem rooted directly
// at the skill files, so entries read as "docker-build.md" rather than
// "skills/docker-build.md". The skills are frozen into the binary at build time.
func SkillsFS() (fs.FS, error) {
	return fs.Sub(embeddedSkills, "skills")
}
