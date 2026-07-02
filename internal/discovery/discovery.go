package discovery

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ProjectContext holds discovered information about the project an agent is working in.
type ProjectContext struct {
	// Specs found in .kiro/specs/
	Specs []SpecFile

	// Steering rules from .kiro/steering/ (with inclusion filtering applied)
	Steering []SteeringFile

	// Detected stack info (from build configs)
	Stack StackInfo

	// README content (first 2000 chars)
	README string

	// Root directory of the project
	RootDir string
}

// SpecFile represents a spec document from .kiro/specs/{feature}/
type SpecFile struct {
	Feature  string // e.g., "moonbase-v2"
	Type     string // "requirements", "design", "tasks"
	Path     string // absolute path
	Content  string // file content
}

// SteeringFile represents a project-wide rule from .kiro/steering/
type SteeringFile struct {
	Name    string // filename without extension
	Path    string // absolute path
	Content string // file content
}

// StackInfo holds detected technology stack information.
type StackInfo struct {
	Language    string // "go", "java", "javascript", "python", "rust", etc.
	BuildTool   string // "make", "maven", "npm", "cargo", etc.
	TestCommand string // detected test command
	BuildFile   string // path to the build config file
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
