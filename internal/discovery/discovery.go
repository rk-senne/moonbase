// Package discovery implements project context detection for moonbase agents.
// It scans the working directory for .kiro/specs/, .kiro/steering/, build configs,
// and README files, assembling a ProjectContext that agents use to understand
// the project they are working in.
package discovery

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ProjectContext holds discovered information about the project an agent is working in.
type ProjectContext struct {
	Specs    []SpecFile     // Spec documents found in .kiro/specs/{feature}/
	Steering []SteeringFile // Steering rules from .kiro/steering/ (manual-inclusion files excluded)
	Skills   []SkillFile    // Skill definitions from .kiro/skills/
	Prompts  []PromptFile   // Stored prompts from .kiro/prompts/
	Stack    StackInfo      // Detected technology stack from build config files
	README   string         // Project README content (truncated to 2000 chars)
	RootDir  string         // Absolute path to the project root directory
}

// SpecFile represents a spec document from .kiro/specs/{feature}/.
type SpecFile struct {
	Feature string // Feature directory name (e.g., "moonbase-v2")
	Type    string // Spec type: "requirements", "design", or "tasks"
	Path    string // Absolute file path
	Content string // Full file content
}

// SteeringFile represents a project-wide rule from .kiro/steering/.
type SteeringFile struct {
	Name    string // Filename without extension (e.g., "dev-rules")
	Path    string // Absolute file path
	Content string // Full file content (including frontmatter if present)
}

// SkillFile represents a skill/knowledge document from .kiro/skills/.
type SkillFile struct {
	Name    string // Skill name (derived from filename or directory)
	Path    string // Absolute file path
	Content string // Full file content
}

// PromptFile represents a stored prompt from .kiro/prompts/.
type PromptFile struct {
	Name    string // Prompt name (derived from filename)
	Path    string // Absolute file path
	Content string // Full file content
}

// StackInfo holds detected technology stack information.
type StackInfo struct {
	Language    string // Primary language: "go", "java", "javascript", "python", "rust", etc.
	BuildTool   string // Build tool: "make", "maven", "npm", "cargo", etc.
	TestCommand string // Detected test command (e.g., "go test ./...")
	BuildFile   string // Absolute path to the build config file that was detected
}

// Discover scans a project directory for specs, steering rules, and stack info.
// It returns a ProjectContext populated with everything it finds.
// If nothing is found, it returns an empty (but valid) context — not an error.
func Discover(projectDir string) (*ProjectContext, error) {
	ctx := &ProjectContext{RootDir: projectDir}

	// Discover specs
	specs, err := discoverSpecs(projectDir)
	if err == nil {
		ctx.Specs = specs
	}

	// Discover steering
	steering, err := discoverSteering(projectDir)
	if err == nil {
		ctx.Steering = steering
	}

	// Discover skills
	skills, err := discoverSkills(projectDir)
	if err == nil {
		ctx.Skills = skills
	}

	// Discover prompts
	prompts, err := discoverPrompts(projectDir)
	if err == nil {
		ctx.Prompts = prompts
	}

	// Detect stack
	ctx.Stack = detectStack(projectDir)

	// Read README (truncated)
	ctx.README = readREADME(projectDir)

	return ctx, nil
}

// HasSpecs returns true if any spec files were found.
func (pc *ProjectContext) HasSpecs() bool {
	return len(pc.Specs) > 0
}

// HasSteering returns true if any steering rules were found.
func (pc *ProjectContext) HasSteering() bool {
	return len(pc.Steering) > 0
}

// HasSkills returns true if any skill files were found.
func (pc *ProjectContext) HasSkills() bool {
	return len(pc.Skills) > 0
}

// HasPrompts returns true if any stored prompts were found.
func (pc *ProjectContext) HasPrompts() bool {
	return len(pc.Prompts) > 0
}

// Summary returns a brief text summary of what was discovered.
func (pc *ProjectContext) Summary() string {
	var parts []string
	if pc.HasSpecs() {
		features := make(map[string]bool)
		for _, s := range pc.Specs {
			features[s.Feature] = true
		}
		featureList := make([]string, 0, len(features))
		for f := range features {
			featureList = append(featureList, f)
		}
		parts = append(parts, fmt.Sprintf("Specs: %s", strings.Join(featureList, ", ")))
	}
	if pc.HasSteering() {
		names := make([]string, 0, len(pc.Steering))
		for _, s := range pc.Steering {
			names = append(names, s.Name)
		}
		parts = append(parts, fmt.Sprintf("Steering: %s", strings.Join(names, ", ")))
	}
	if pc.HasSkills() {
		parts = append(parts, fmt.Sprintf("Skills: %d", len(pc.Skills)))
	}
	if pc.HasPrompts() {
		parts = append(parts, fmt.Sprintf("Prompts: %d", len(pc.Prompts)))
	}
	if pc.Stack.Language != "" {
		parts = append(parts, fmt.Sprintf("Stack: %s/%s", pc.Stack.Language, pc.Stack.BuildTool))
	}
	if len(parts) == 0 {
		return "No project context discovered"
	}
	return strings.Join(parts, " | ")
}

// discoverSpecs finds all spec files in .kiro/specs/
func discoverSpecs(projectDir string) ([]SpecFile, error) {
	specsDir := filepath.Join(projectDir, ".kiro", "specs")
	if _, err := os.Stat(specsDir); os.IsNotExist(err) {
		return nil, nil
	}

	var specs []SpecFile

	// Walk spec directories
	entries, err := os.ReadDir(specsDir)
	if err != nil {
		return nil, fmt.Errorf("reading specs dir: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() || strings.HasPrefix(entry.Name(), "_") {
			continue
		}

		feature := entry.Name()
		featureDir := filepath.Join(specsDir, feature)

		// Look for standard spec files
		specTypes := []string{"requirements.md", "design.md", "tasks.md"}
		for _, specType := range specTypes {
			path := filepath.Join(featureDir, specType)
			if content, err := os.ReadFile(path); err == nil {
				typeName := strings.TrimSuffix(specType, ".md")
				specs = append(specs, SpecFile{
					Feature: feature,
					Type:    typeName,
					Path:    path,
					Content: string(content),
				})
			}
		}
	}

	return specs, nil
}

// detectStack detects the project's technology stack from build config files.
func detectStack(projectDir string) StackInfo {
	stackDetectors := []struct {
		file        string
		language    string
		buildTool   string
		testCommand string
	}{
		{"go.mod", "go", "go", "go test ./..."},
		{"Cargo.toml", "rust", "cargo", "cargo test"},
		{"pom.xml", "java", "maven", "mvn test"},
		{"build.gradle", "java", "gradle", "./gradlew test"},
		{"build.gradle.kts", "kotlin", "gradle", "./gradlew test"},
		{"package.json", "javascript", "npm", "npm test"},
		{"pyproject.toml", "python", "pip", "python -m pytest"},
		{"requirements.txt", "python", "pip", "python -m pytest"},
		{"Makefile", "", "make", "make test"},
	}

	for _, d := range stackDetectors {
		path := filepath.Join(projectDir, d.file)
		if _, err := os.Stat(path); err == nil {
			return StackInfo{
				Language:    d.language,
				BuildTool:   d.buildTool,
				TestCommand: d.testCommand,
				BuildFile:   path,
			}
		}
	}

	return StackInfo{}
}

// readREADME reads the project README, truncated to 2000 chars.
func readREADME(projectDir string) string {
	readmeNames := []string{"README.md", "readme.md", "README", "README.txt"}

	for _, name := range readmeNames {
		path := filepath.Join(projectDir, name)
		content, err := os.ReadFile(path)
		if err == nil {
			s := string(content)
			if len(s) > 2000 {
				s = s[:2000] + "\n...(truncated)"
			}
			return s
		}
	}

	return ""
}

// discoverSkills finds skill/knowledge files in .kiro/skills/.
// It loads SKILL.md from the skills directory itself, and any *.md files
// from subdirectories within .kiro/skills/.
func discoverSkills(projectDir string) ([]SkillFile, error) {
	skillsDir := filepath.Join(projectDir, ".kiro", "skills")
	if _, err := os.Stat(skillsDir); os.IsNotExist(err) {
		return nil, nil
	}

	var skills []SkillFile

	entries, err := os.ReadDir(skillsDir)
	if err != nil {
		return nil, fmt.Errorf("reading skills dir: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			// Read *.md files from subdirectories
			subDir := filepath.Join(skillsDir, entry.Name())
			subEntries, err := os.ReadDir(subDir)
			if err != nil {
				continue
			}
			for _, subEntry := range subEntries {
				if subEntry.IsDir() || !strings.HasSuffix(strings.ToLower(subEntry.Name()), ".md") {
					continue
				}
				path := filepath.Join(subDir, subEntry.Name())
				content, err := os.ReadFile(path)
				if err != nil {
					continue
				}
				name := entry.Name() + "/" + strings.TrimSuffix(subEntry.Name(), ".md")
				skills = append(skills, SkillFile{
					Name:    name,
					Path:    path,
					Content: string(content),
				})
			}
		} else if strings.ToUpper(entry.Name()) == "SKILL.MD" || strings.HasSuffix(strings.ToLower(entry.Name()), ".md") {
			// Load SKILL.md or any top-level .md files
			path := filepath.Join(skillsDir, entry.Name())
			content, err := os.ReadFile(path)
			if err != nil {
				continue
			}
			name := strings.TrimSuffix(entry.Name(), ".md")
			skills = append(skills, SkillFile{
				Name:    name,
				Path:    path,
				Content: string(content),
			})
		}
	}

	return skills, nil
}

// discoverPrompts finds stored prompt files in .kiro/prompts/.
// It loads all *.md files from the prompts directory.
func discoverPrompts(projectDir string) ([]PromptFile, error) {
	promptsDir := filepath.Join(projectDir, ".kiro", "prompts")
	if _, err := os.Stat(promptsDir); os.IsNotExist(err) {
		return nil, nil
	}

	var prompts []PromptFile

	entries, err := os.ReadDir(promptsDir)
	if err != nil {
		return nil, fmt.Errorf("reading prompts dir: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(strings.ToLower(entry.Name()), ".md") {
			continue
		}
		path := filepath.Join(promptsDir, entry.Name())
		content, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		name := strings.TrimSuffix(entry.Name(), ".md")
		prompts = append(prompts, PromptFile{
			Name:    name,
			Path:    path,
			Content: string(content),
		})
	}

	return prompts, nil
}
