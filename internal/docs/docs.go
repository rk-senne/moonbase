package docs

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	"charm.land/glamour/v2"
)

type Doc struct {
	Name string
	Path string
}

// Discover finds markdown documentation in the project
func Discover() []Doc {
	var docs []Doc
	seen := map[string]bool{}

	// Search these directories
	dirs := []string{"docs", ".kiro/steering", "wiki", "spec", "."}

	for _, dir := range dirs {
		filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return nil
			}
			if info.IsDir() {
				// Skip hidden dirs (except .kiro)
				if strings.HasPrefix(info.Name(), ".") && info.Name() != ".kiro" {
					return filepath.SkipDir
				}
				// Skip deep nesting
				if strings.Count(path, string(os.PathSeparator)) > 3 {
					return filepath.SkipDir
				}
				return nil
			}
			ext := strings.ToLower(filepath.Ext(path))
			if ext != ".md" && ext != ".txt" {
				return nil
			}
			if seen[path] {
				return nil
			}
			seen[path] = true
			docs = append(docs, Doc{
				Name: path,
				Path: path,
			})
			return nil
		})
	}

	sort.Slice(docs, func(i, j int) bool {
		return docs[i].Name < docs[j].Name
	})
	return docs
}

// Render reads a markdown file and renders it for the terminal
func Render(path string, width int) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}

	r, err := glamour.NewTermRenderer(
		glamour.WithStandardStyle("dark"),
		glamour.WithWordWrap(width),
	)
	if err != nil {
		return string(data), nil // fallback to raw
	}

	out, err := r.Render(string(data))
	if err != nil {
		return string(data), nil
	}
	return out, nil
}
