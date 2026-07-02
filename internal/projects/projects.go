package projects

import (
	"os"
	"path/filepath"
	"sort"
)

type Project struct {
	Name string
	Path string
	Type string // go, node, java, rust, git
}

var manifests = map[string]string{
	"go.mod":       "go",
	"package.json": "node",
	"pom.xml":      "java",
	"Cargo.toml":   "rust",
	"build.gradle": "java",
}

// Discover scans common developer workspace directories for projects.
// Checks multiple conventional roots rather than hardcoding one person's layout.
func Discover() []Project {
	home, _ := os.UserHomeDir()
	roots := []string{
		filepath.Join(home, "Workspace"),
		filepath.Join(home, "Projects"),
		filepath.Join(home, "Developer"),
		filepath.Join(home, "dev"),
		filepath.Join(home, "src"),
	}

	var projects []Project
	for _, root := range roots {
		projects = append(projects, scanRoot(root)...)
	}

	sort.Slice(projects, func(i, j int) bool {
		return projects[i].Name < projects[j].Name
	})
	return projects
}

// scanRoot scans a root directory one level deep for project directories.
// Skips roots that don't exist.
func scanRoot(root string) []Project {
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil
	}

	var projects []Project
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		dir := filepath.Join(root, e.Name())
		ptype := detectType(dir)
		if ptype != "" {
			projects = append(projects, Project{
				Name: e.Name(),
				Path: dir,
				Type: ptype,
			})
		}
		// Also scan one level deeper for grouped workspaces (e.g., ~/Workspace/Personal/*)
		subEntries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, sub := range subEntries {
			if !sub.IsDir() {
				continue
			}
			subDir := filepath.Join(dir, sub.Name())
			subType := detectType(subDir)
			if subType != "" {
				projects = append(projects, Project{
					Name: sub.Name(),
					Path: subDir,
					Type: subType,
				})
			}
		}
	}
	return projects
}

func detectType(dir string) string {
	for file, ptype := range manifests {
		if _, err := os.Stat(filepath.Join(dir, file)); err == nil {
			return ptype
		}
	}
	// Fallback: has .git
	if _, err := os.Stat(filepath.Join(dir, ".git")); err == nil {
		return "git"
	}
	return ""
}
