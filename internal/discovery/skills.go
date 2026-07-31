package discovery

import (
	"errors"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
	"sync"

	"gopkg.in/yaml.v3"
)

// SkillMeta holds lightweight skill metadata loaded at startup.
// Only ~100 tokens per skill in the composed prompt.
type SkillMeta struct {
	Name        string // Unique skill identifier (from frontmatter or filename)
	Description string // What the skill provides and when to use it
	Path        string // Absolute path to the skill file
	Legacy      bool   // True if skill lacks YAML frontmatter (backward compat)
}

// skillEntry is an internal registry entry combining metadata with cached content.
type skillEntry struct {
	meta    SkillMeta
	content string // empty until LoadContent() called
	loaded  bool
}

// SkillRegistry indexes skills by metadata and loads content on demand.
// Safe for concurrent use.
type SkillRegistry struct {
	mu     sync.RWMutex
	skills map[string]*skillEntry // keyed by normalized name
	order  []string              // insertion order for deterministic listing
}

// skillFrontmatter is the YAML structure expected in skill file frontmatter.
type skillFrontmatter struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
}

// validSkillName matches only lowercase alphanumeric and hyphens.
var validSkillName = regexp.MustCompile(`^[a-z0-9-]+$`)

// skillRequestPattern matches @skill(name) references in text.
var skillRequestPattern = regexp.MustCompile(`@skill\(([a-z0-9-]+)\)`)

// maxFrontmatterRead is the maximum bytes read from a file to extract frontmatter.
// Frontmatter for skills should be ≤200 bytes; 1KB gives generous headroom
// without reading multi-KB instruction bodies.
const maxFrontmatterRead = 1024

var (
	// ErrSkillNotFound is returned when a skill name is not in the registry.
	ErrSkillNotFound = errors.New("skill not found")
	// ErrSkillNoFrontmatter is returned when a skill file lacks YAML frontmatter.
	ErrSkillNoFrontmatter = errors.New("skill missing YAML frontmatter")
	// ErrSkillInvalidName is returned when a skill name fails validation.
	ErrSkillInvalidName = errors.New("skill name invalid")
)

// NewSkillRegistry creates an empty skill registry.
func NewSkillRegistry() *SkillRegistry {
	return &SkillRegistry{
		skills: make(map[string]*skillEntry),
	}
}

// Register adds a skill to the registry. If a skill with the same name already
// exists, the first registration wins (duplicates are silently dropped).
func (r *SkillRegistry) Register(meta SkillMeta) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.skills[meta.Name]; exists {
		return // first registration wins
	}
	r.skills[meta.Name] = &skillEntry{meta: meta}
	r.order = append(r.order, meta.Name)
}

// List returns metadata for all discovered skills in discovery order.
func (r *SkillRegistry) List() []SkillMeta {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]SkillMeta, 0, len(r.order))
	for _, name := range r.order {
		if entry, ok := r.skills[name]; ok {
			result = append(result, entry.meta)
		}
	}
	return result
}

// Get returns metadata for a single skill by name, or nil if not found.
func (r *SkillRegistry) Get(name string) *SkillMeta {
	r.mu.RLock()
	defer r.mu.RUnlock()

	entry, ok := r.skills[name]
	if !ok {
		return nil
	}
	meta := entry.meta
	return &meta
}

// LoadContent reads the skill body from disk (cached after first load).
// Returns content with frontmatter stripped. Respects maxSkillSize truncation.
func (r *SkillRegistry) LoadContent(name string) (string, error) {
	r.mu.RLock()
	entry, ok := r.skills[name]
	if !ok {
		r.mu.RUnlock()
		return "", fmt.Errorf("loading skill %q: %w", name, ErrSkillNotFound)
	}
	if entry.loaded {
		content := entry.content
		r.mu.RUnlock()
		return content, nil
	}
	r.mu.RUnlock()

	// Cache miss — read from disk under write lock.
	r.mu.Lock()
	defer r.mu.Unlock()

	// Double-check after acquiring write lock (another goroutine may have loaded it).
	if entry.loaded {
		return entry.content, nil
	}

	data, err := os.ReadFile(entry.meta.Path)
	if err != nil {
		return "", fmt.Errorf("loading skill %q: %w", name, err)
	}

	body := stripFrontmatter(string(data))
	body = strings.TrimSpace(body)
	if len(body) > maxSkillSize {
		body = body[:maxSkillSize] + "\n...(truncated)"
	}

	entry.content = body
	entry.loaded = true
	return entry.content, nil
}

// Names returns all registered skill names in discovery order.
func (r *SkillRegistry) Names() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]string, len(r.order))
	copy(result, r.order)
	return result
}

// Len returns the number of registered skills.
func (r *SkillRegistry) Len() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.order)
}

// parseFrontmatterOnly reads up to maxFrontmatterRead bytes from a file to
// extract YAML frontmatter. Returns SkillMeta with Name/Description populated.
// Returns ErrSkillNoFrontmatter if the file lacks frontmatter, or
// ErrSkillInvalidName if the name fails validation.
func parseFrontmatterOnly(path string) (SkillMeta, error) {
	f, err := os.Open(path)
	if err != nil {
		return SkillMeta{}, fmt.Errorf("opening skill %s: %w", path, err)
	}
	defer f.Close()

	buf := make([]byte, maxFrontmatterRead)
	n, err := f.Read(buf)
	if err != nil && err != io.EOF {
		return SkillMeta{}, fmt.Errorf("reading skill %s: %w", path, err)
	}
	buf = buf[:n]

	// Trim leading whitespace
	content := strings.TrimLeft(string(buf), "\n\r")

	// Must start with ---
	if !strings.HasPrefix(content, "---") {
		return SkillMeta{}, fmt.Errorf("skill %s: %w", path, ErrSkillNoFrontmatter)
	}

	// Find closing ---
	rest := content[3:]
	if len(rest) > 0 && rest[0] == '\n' {
		rest = rest[1:]
	}

	closeIdx := strings.Index(rest, "\n---")
	if closeIdx == -1 {
		return SkillMeta{}, fmt.Errorf("skill %s: %w (missing closing ---)", path, ErrSkillNoFrontmatter)
	}

	yamlContent := rest[:closeIdx]

	var fm skillFrontmatter
	if err := yaml.Unmarshal([]byte(yamlContent), &fm); err != nil {
		return SkillMeta{}, fmt.Errorf("skill %s: parsing frontmatter: %w", path, err)
	}

	// Validate name
	name := normalizeSkillName(fm.Name)
	if name == "" {
		return SkillMeta{}, fmt.Errorf("skill %s: %w (empty name)", path, ErrSkillInvalidName)
	}
	if len(name) > 64 {
		return SkillMeta{}, fmt.Errorf("skill %s: %w (exceeds 64 chars)", path, ErrSkillInvalidName)
	}
	if !validSkillName.MatchString(name) {
		return SkillMeta{}, fmt.Errorf("skill %s: %w (must match [a-z0-9-])", path, ErrSkillInvalidName)
	}

	return SkillMeta{
		Name:        name,
		Description: fm.Description,
		Path:        path,
	}, nil
}

// normalizeSkillName converts a skill name to the canonical form.
// It lowercases and replaces underscores/spaces with hyphens.
func normalizeSkillName(name string) string {
	name = strings.TrimSpace(name)
	name = strings.ToLower(name)
	name = strings.ReplaceAll(name, "_", "-")
	name = strings.ReplaceAll(name, " ", "-")
	return name
}

// ExtractSkillRequests scans text for @skill(name) patterns and returns
// unique skill names found.
func ExtractSkillRequests(text string) []string {
	matches := skillRequestPattern.FindAllStringSubmatch(text, -1)
	if len(matches) == 0 {
		return nil
	}

	seen := make(map[string]bool, len(matches))
	var names []string
	for _, m := range matches {
		name := m[1]
		if !seen[name] {
			seen[name] = true
			names = append(names, name)
		}
	}
	return names
}

// ResolvedSkill holds a skill name and its loaded content.
type ResolvedSkill struct {
	Name    string
	Content string
}

// ResolveSkills loads content for the given skill names from the registry.
// Returns resolved skills and a list of names that were not found.
func ResolveSkills(registry *SkillRegistry, names []string) ([]ResolvedSkill, []string) {
	if registry == nil {
		return nil, names
	}

	var resolved []ResolvedSkill
	var notFound []string

	for _, name := range names {
		content, err := registry.LoadContent(name)
		if err != nil {
			notFound = append(notFound, name)
			continue
		}
		resolved = append(resolved, ResolvedSkill{Name: name, Content: content})
	}

	return resolved, notFound
}

// ParseFrontmatterOnlyFromPath is the exported entry point for parsing skill
// YAML frontmatter from a file path. It returns SkillMeta with Name,
// Description, and Path populated. Returns ErrSkillNoFrontmatter if the file
// lacks frontmatter, or ErrSkillInvalidName if the name fails validation.
func ParseFrontmatterOnlyFromPath(path string) (SkillMeta, error) {
	return parseFrontmatterOnly(path)
}
