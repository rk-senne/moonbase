package moonbase

import (
	"io/fs"
	"regexp"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

// skillFrontmatterTest is the expected YAML structure in skill frontmatter.
type skillFrontmatterTest struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
}

// validSkillNamePattern matches only lowercase alphanumeric and hyphens.
var validSkillNamePattern = regexp.MustCompile(`^[a-z0-9-]+$`)

func TestSkillsFS_ReturnsValidFS(t *testing.T) {
	sfs, err := SkillsFS()
	if err != nil {
		t.Fatalf("SkillsFS() error: %v", err)
	}
	if sfs == nil {
		t.Fatal("SkillsFS() returned nil")
	}
}

func TestSkillsFS_ContainsAtLeast10Skills(t *testing.T) {
	sfs, err := SkillsFS()
	if err != nil {
		t.Fatalf("SkillsFS() error: %v", err)
	}
	entries, err := fs.Glob(sfs, "*.md")
	if err != nil {
		t.Fatalf("Glob error: %v", err)
	}
	if len(entries) < 10 {
		t.Fatalf("expected >=10 embedded skills, got %d", len(entries))
	}
}

func TestSkillsFS_AllSkillsHaveValidFrontmatter(t *testing.T) {
	sfs, err := SkillsFS()
	if err != nil {
		t.Fatalf("SkillsFS() error: %v", err)
	}
	entries, err := fs.Glob(sfs, "*.md")
	if err != nil {
		t.Fatalf("Glob error: %v", err)
	}

	tests := make([]struct {
		filename string
		stem     string
	}, 0, len(entries))
	for _, e := range entries {
		stem := strings.TrimSuffix(e, ".md")
		tests = append(tests, struct {
			filename string
			stem     string
		}{filename: e, stem: stem})
	}

	for _, tt := range tests {
		t.Run(tt.stem, func(t *testing.T) {
			data, err := fs.ReadFile(sfs, tt.filename)
			if err != nil {
				t.Fatalf("reading %s: %v", tt.filename, err)
			}
			content := string(data)

			// Must start with ---
			trimmed := strings.TrimLeft(content, "\n\r")
			if !strings.HasPrefix(trimmed, "---") {
				t.Fatalf("%s: missing YAML frontmatter opening ---", tt.filename)
			}

			// Extract YAML block
			rest := trimmed[3:]
			if len(rest) > 0 && rest[0] == '\n' {
				rest = rest[1:]
			}
			closeIdx := strings.Index(rest, "\n---")
			if closeIdx == -1 {
				t.Fatalf("%s: missing YAML frontmatter closing ---", tt.filename)
			}
			yamlBlock := rest[:closeIdx]

			var fm skillFrontmatterTest
			if err := yaml.Unmarshal([]byte(yamlBlock), &fm); err != nil {
				t.Fatalf("%s: parsing frontmatter: %v", tt.filename, err)
			}

			// name must be non-empty
			if fm.Name == "" {
				t.Errorf("%s: frontmatter name is empty", tt.filename)
			}

			// name must match [a-z0-9-]
			if !validSkillNamePattern.MatchString(fm.Name) {
				t.Errorf("%s: name %q does not match [a-z0-9-]", tt.filename, fm.Name)
			}

			// name must equal filename stem
			if fm.Name != tt.stem {
				t.Errorf("%s: name %q does not match filename stem %q", tt.filename, fm.Name, tt.stem)
			}

			// name must be <=64 chars
			if len(fm.Name) > 64 {
				t.Errorf("%s: name %q exceeds 64 chars", tt.filename, fm.Name)
			}

			// description must be non-empty
			if fm.Description == "" {
				t.Errorf("%s: frontmatter description is empty", tt.filename)
			}

			// body must exist (non-trivial content after frontmatter)
			body := strings.TrimSpace(rest[closeIdx+4:])
			if len(body) < 50 {
				t.Errorf("%s: body is too short (%d chars), expected substantial content", tt.filename, len(body))
			}
		})
	}
}
