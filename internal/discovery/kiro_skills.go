package discovery

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// EmitKiroSkillResources writes skill entries compatible with Kiro's skill://
// protocol. Each skill becomes a directory under outputDir with a SKILL.md
// containing the YAML frontmatter (name + description) plus the body content.
//
// Output structure:
//
//	outputDir/
//	  docker-build/
//	    SKILL.md    ← frontmatter + body
//	  git-workflow/
//	    SKILL.md    ← frontmatter + body
//
// This function is independently callable without the full kiro-native-interop
// integration. It produces the file structure that Kiro's progressive disclosure
// expects. Directory permissions are 0700, file permissions are 0600.
func EmitKiroSkillResources(registry *SkillRegistry, outputDir string) error {
	if registry == nil || registry.Len() == 0 {
		return nil
	}

	skills := registry.List()
	var errs []string

	for _, meta := range skills {
		skillDir := filepath.Join(outputDir, meta.Name)
		if err := os.MkdirAll(skillDir, 0o700); err != nil {
			errs = append(errs, fmt.Sprintf("creating dir %s: %v", skillDir, err))
			continue
		}

		// Check if the source is already directory-style (path ends with SKILL.md)
		var content []byte
		if strings.ToUpper(filepath.Base(meta.Path)) == "SKILL.MD" {
			// Directory-style skill — read source as-is
			data, err := os.ReadFile(meta.Path)
			if err != nil {
				errs = append(errs, fmt.Sprintf("reading skill %s: %v", meta.Name, err))
				continue
			}
			content = data
		} else {
			// Flat-file skill — read source and ensure frontmatter is present
			data, err := os.ReadFile(meta.Path)
			if err != nil {
				errs = append(errs, fmt.Sprintf("reading skill %s: %v", meta.Name, err))
				continue
			}
			// Check if frontmatter already exists
			trimmed := strings.TrimLeft(string(data), "\n\r")
			if strings.HasPrefix(trimmed, "---") {
				content = data
			} else {
				// Prepend frontmatter
				var b strings.Builder
				b.WriteString("---\n")
				b.WriteString(fmt.Sprintf("name: %s\n", meta.Name))
				if meta.Description != "" {
					b.WriteString(fmt.Sprintf("description: %s\n", meta.Description))
				}
				b.WriteString("---\n")
				b.WriteString(string(data))
				content = []byte(b.String())
			}
		}

		outPath := filepath.Join(skillDir, "SKILL.md")
		if err := os.WriteFile(outPath, content, 0o600); err != nil {
			errs = append(errs, fmt.Sprintf("writing skill %s: %v", meta.Name, err))
			continue
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("emitting skill resources: %s", strings.Join(errs, "; "))
	}
	return nil
}
