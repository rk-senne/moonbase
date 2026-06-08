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

// Discover scans ~/Workspace for projects
func Discover() []Project {
	home, _ := os.UserHomeDir()
	roots := []string{
		filepath.Join(home, "Workspace", "Personal"),
		filepath.Join(home, "Workspace", "Projects"),
	}

	var projects []Project
	for _, root := range roots {
		entries, err := os.ReadDir(root)
		if err != nil {
			continue
		}
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
		}
	}

	sort.Slice(projects, func(i, j int) bool {
		return projects[i].Name < projects[j].Name
	})
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
