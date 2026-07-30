# Design: Progressive Skill Loading

## Architecture Decision

Replace the current eager `discoverSkills()` → full-content-in-memory → bulk-inject pattern with a three-level progressive loading model. The skill registry parses only YAML frontmatter at startup, stores metadata + paths, and defers content reads until an operative explicitly requests a skill. The compose layer switches from injecting skill bodies to emitting a lightweight catalog.

**Key ADR:** Skills follow the same progressive disclosure architecture as Kiro's native `skill://` model (Level 1: metadata always loaded, Level 2: instructions on trigger, Level 3: resources as needed). This ensures that when moonbase emits `skill://` resources for the Kiro backend, the mental model and file structure are already aligned — no conversion needed.

---

## Files Affected

| File | Change Type | Purpose |
|------|------------|---------|
| `internal/discovery/skills.go` | new | `SkillMeta`, `SkillRegistry`, frontmatter parsing, deferred loading |
| `internal/discovery/skills_test.go` | new | Unit tests for registry, parsing, loading, caching |
| `internal/discovery/discovery.go` | modify | Replace `discoverSkills()` with `discoverSkillRegistry()`, add `SkillRegistry` to `ProjectContext` |
| `internal/discovery/compose.go` | modify | Replace eager skill injection with catalog section + on-demand hook |
| `internal/discovery/kiro_skills.go` | new | `EmitKiroSkillResources()` — writes skill:// resource files |
| `internal/discovery/kiro_skills_test.go` | new | Tests for skill:// emission |
| `cmd/moonbase/init.go` | modify | Update scaffolding to create skills with YAML frontmatter |
| `internal/discovery/discovery_test.go` | modify | Update existing tests for new skill behavior |

---

## Component Designs

### 1. Skill File Format

Skills are markdown files with YAML frontmatter. The frontmatter is the machine-readable metadata; the body is the skill content (instructions, examples, guidance).

```markdown
---
name: docker-build
description: Docker multi-stage build patterns, layer caching, and CI integration. Use when working with Dockerfiles or container builds.
---

# Docker Build Patterns

## Multi-Stage Builds
...
```

**Directory structure (both supported):**

```
.kiro/skills/
├── docker-build.md              ← flat file (name from frontmatter)
├── git-workflow/
│   └── SKILL.md                 ← directory-style (Kiro-native compatible)
└── legacy-no-frontmatter.md     ← fallback: name derived from filename
```

### 2. SkillMeta and SkillRegistry

```go
// internal/discovery/skills.go

// SkillMeta holds lightweight skill metadata loaded at startup.
// Only ~100 tokens per skill in the composed prompt.
type SkillMeta struct {
    Name        string // Unique skill identifier (from frontmatter or filename)
    Description string // What the skill provides and when to use it
    Path        string // Absolute path to the skill file
    Legacy      bool   // True if skill lacks YAML frontmatter (backward compat)
}

// SkillRegistry indexes skills by metadata and loads content on demand.
// Safe for concurrent use.
type SkillRegistry struct {
    mu      sync.RWMutex
    skills  map[string]*skillEntry // keyed by normalized name
    order   []string              // insertion order for deterministic listing
}

type skillEntry struct {
    meta    SkillMeta
    content string // empty until LoadContent() called
    loaded  bool
}

// List returns metadata for all discovered skills in discovery order.
func (r *SkillRegistry) List() []SkillMeta

// Get returns metadata for a single skill by name, or nil if not found.
func (r *SkillRegistry) Get(name string) *SkillMeta

// LoadContent reads the skill body from disk (cached after first load).
// Returns content with frontmatter stripped. Respects maxSkillSize.
func (r *SkillRegistry) LoadContent(name string) (string, error)

// Names returns all registered skill names.
func (r *SkillRegistry) Names() []string
```

**Concurrency:** Read-heavy workload (list/get dominate, load is rare). `sync.RWMutex` protects the cache. Content loading takes a write lock only on cache-miss.

**Cache bound:** The registry caches loaded content indefinitely during a single process run. Since moonbase is a short-lived CLI tool (not a daemon), unbounded growth is acceptable — a typical project has <20 skills. If this changes, add LRU eviction.

### 3. Frontmatter Parsing

```go
// parseFrontmatterOnly reads up to 1KB from a file to extract YAML frontmatter.
// Returns SkillMeta with Name/Description populated, or an error if missing/invalid.
func parseFrontmatterOnly(path string) (SkillMeta, error) {
    // Open file, read first 1024 bytes
    // Find opening "---\n" and closing "\n---\n"
    // Unmarshal the YAML between delimiters into struct{Name, Description string}
    // Validate: name non-empty, <=64 chars, matches [a-z0-9-]
}
```

**Why 1KB:** Frontmatter for skills should be ≤200 bytes. Reading 1KB gives generous headroom without reading multi-KB instruction bodies. This is the key optimization — `os.ReadFile` (current approach) reads the entire file.

### 4. Discovery Integration

```go
// In discovery.go — replace current discoverSkills()

func discoverSkillRegistry(projectDir string) *SkillRegistry {
    registry := NewSkillRegistry()
    skillsDir := filepath.Join(projectDir, ".kiro", "skills")

    // Walk directory structure (same traversal as current discoverSkills)
    // For each .md file:
    //   1. parseFrontmatterOnly(path)
    //   2. If success: registry.Register(meta)
    //   3. If frontmatter missing: register as legacy (name from filename)
    //   4. If validation error: log warning, skip

    return registry
}
```

**ProjectContext change:**

```go
type ProjectContext struct {
    // ... existing fields ...
    Skills        []SkillFile     // DEPRECATED: kept for backward compat, populated lazily
    SkillRegistry *SkillRegistry  // New: progressive loading registry
}
```

### 5. Compose Layer Changes

**Before (eager injection):**
```
--- PROJECT SKILLS ---
The following domain knowledge is available:

### docker-build
[full content up to 2000 chars]

### git-workflow
[full content up to 2000 chars]
--- END PROJECT SKILLS ---
```

**After (catalog + on-demand protocol):**
```
--- AVAILABLE SKILLS ---
The following skills are available. Request any skill with @skill(name) to load its full content.

| Skill | Description |
|-------|-------------|
| docker-build | Docker multi-stage build patterns, layer caching, and CI integration |
| git-workflow | Branch strategy, PR templates, commit conventions |

--- END AVAILABLE SKILLS ---
```

**On-demand injection:** When the pipeline detects `@skill(name)` in an operative's output (or the task references a skill), `LoadContent(name)` is called and the content is appended to the next compose pass as:

```
--- SKILL: docker-build ---
[full skill body, frontmatter stripped, max 2000 chars]
--- END SKILL ---
```

### 6. Operative Request Protocol

Operatives see the skill catalog in their prompt and can request skills using:

```
@skill(docker-build)
```

**Pipeline handling:**
1. Phase output is scanned for `@skill(...)` patterns (regex: `@skill\(([a-z0-9-]+)\)`)
2. Matched skill names are resolved against the registry
3. Content is loaded and injected into the operative's context for the next turn
4. If skill not found: include a brief "Skill '{name}' not found. Available: ..." message

**For non-interactive (single-shot) deploys:** The compose layer pre-scans the task text for skill references and pre-loads them. This handles `moonbase deploy 4 "review the docker-build skill patterns"`.

### 7. Kiro-Native skill:// Emission

```go
// internal/discovery/kiro_skills.go

// EmitKiroSkillResources writes skill entries compatible with Kiro's skill:// protocol.
// Each skill becomes a directory under outputDir with a SKILL.md containing
// the original frontmatter + body.
//
// Output structure:
//   outputDir/
//     docker-build/
//       SKILL.md    ← original content (frontmatter + body)
//     git-workflow/
//       SKILL.md    ← original content
//
// This function is independently callable without the full kiro-native-interop
// integration. It produces the file structure that Kiro's progressive disclosure
// expects.
func EmitKiroSkillResources(registry *SkillRegistry, outputDir string) error {
    for _, meta := range registry.List() {
        skillDir := filepath.Join(outputDir, meta.Name)
        os.MkdirAll(skillDir, 0o700)

        // For directory-style skills: copy as-is
        // For flat-file skills: read source and write to SKILL.md in new directory
        // Ensure YAML frontmatter with name + description is present
    }
}
```

**Integration point:** When `kiro-native-interop` is implemented, it will call `EmitKiroSkillResources()` to populate `.kiro/skills/` in Kiro-native format. Until then, `moonbase init` can optionally run this to ensure skills are structured for Kiro compatibility.

---

## Data Flow

### Startup (Discovery)
```
.kiro/skills/**/*.md
    ↓ parseFrontmatterOnly (1KB read per file)
SkillRegistry{name → SkillMeta{name, description, path}}
    ↓ stored in
ProjectContext.SkillRegistry
```

### Prompt Composition
```
ComposePrompt(agent, context, task)
    ↓ context.SkillRegistry.List()
"--- AVAILABLE SKILLS ---" catalog section (~100 tokens per skill)
    ↓ (no full content injected)
Composed prompt (lighter by ~2000 × N tokens vs. current)
```

### On-Demand Loading
```
Operative output: "I need the docker patterns. @skill(docker-build)"
    ↓ pipeline scans for @skill(...)
registry.LoadContent("docker-build")
    ↓ os.ReadFile → strip frontmatter → truncate to maxSkillSize
"--- SKILL: docker-build ---" injected into next turn
```

---

## Edge Cases

| Scenario | Handling |
|----------|----------|
| Skill file deleted between discovery and load | Return `ErrSkillNotFound`, log warning |
| Skill content exceeds `maxSkillSize` | Truncate with `...(truncated)` suffix (same as today) |
| Multiple skills with same name (flat + subdir) | First discovered wins; log duplicate warning |
| Operative requests non-existent skill | Return list of available names in error message |
| No skills directory exists | Empty registry, no error, catalog section omitted |
| Legacy skills without frontmatter | Load eagerly into `Skills` field for backward compat; not in registry catalog |
| Circular `@skill()` references | Skills don't reference other skills — no recursion possible |

---

## Migration Strategy

1. **Phase 1 (this spec):** Add `SkillRegistry` alongside existing `[]SkillFile`. Both are populated. New compose path uses registry; old `Skills` field still available for any callers.
2. **Phase 2 (next release):** Deprecation warning on `ProjectContext.Skills` direct access.
3. **Phase 3 (v-next):** Remove `[]SkillFile` field and `discoverSkills()` function entirely.

This ensures zero breaking changes for existing code that reads `ProjectContext.Skills`.

---

## Performance Impact

| Metric | Before | After |
|--------|--------|-------|
| Disk reads at startup | `N × full file reads` | `N × 1KB reads` |
| Prompt tokens (skills section) | `~2000 × min(N, 5)` tokens | `~100 × N` tokens (catalog only) |
| On-demand load | N/A | Single file read per requested skill |
| Memory at rest | Full content of all skills | Metadata only (~200 bytes per skill) |

For a project with 10 skills averaging 3KB each: startup reads drop from ~30KB to ~10KB, and prompt overhead drops from ~10,000 tokens to ~1,000 tokens.

---

## Security Considerations

- Frontmatter parsing uses `gopkg.in/yaml.v3` (same as agent parsing) — no new dependencies
- File paths are validated: no path traversal beyond `.kiro/skills/`
- Skill content is treated as untrusted text in the prompt (same trust model as today)
- `@skill()` names validated against `[a-z0-9-]` before registry lookup (no injection)
- Emitted skill:// directories written with `0700` permissions
