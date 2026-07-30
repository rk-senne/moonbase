# Requirements: Progressive Skill Loading

## Overview

Moonbase currently loads all skill content eagerly at discovery time — every `.md` file in `.kiro/skills/` is read into memory and injected into the composed prompt regardless of whether the operative needs it. This wastes context window tokens and violates Anthropic's context-management guidance: "load metadata upfront, full content on demand" (Anthropic Platform Docs, *Manage tool context* + *Agent Skills — Progressive Disclosure*).

This spec introduces a three-level progressive skill loading model aligned with Kiro's `skill://` resource convention:

| Level | When Loaded | What |
|-------|-------------|------|
| L1: Metadata | Always (startup) | `name`, `description`, `path` — ~100 tokens per skill |
| L2: Instructions | On-demand (operative references skill) | Full `SKILL.md` body content |
| L3: Resources | On-demand (skill body references sub-file) | Bundled scripts, schemas, templates |

**Current state (confirmed by reading `internal/discovery/discovery.go`):** `discoverSkills()` calls `os.ReadFile(path)` for every `.md` file and stores full content in `SkillFile.Content`. `compose.go` injects up to `maxSkills=5` skills (each truncated to `maxSkillSize=2000`) into every prompt. This is **eager loading** — no selective injection exists today.

---

## User Stories

### US-1: Context-Efficient Skill Discovery
As an operative, I want only skill metadata (name + description) loaded at startup so that my effective context window isn't consumed by skill content I never reference.

### US-2: On-Demand Skill Injection
As an operative, I want to request a specific skill by name and have its full content injected into my context at that point — not before.

### US-3: Skill Authoring with Frontmatter
As a project maintainer, I want skills defined as markdown files with YAML frontmatter (`name`, `description`) so that moonbase can index them without reading the full body.

### US-4: Skill Registry API
As a moonbase developer, I want a `SkillRegistry` in `internal/discovery/` that exposes metadata for lookup and deferred content loading, so the pipeline and compose layer can request skills on demand.

### US-5: Kiro-Native Skill Resources
As a Kiro CLI user, I want moonbase to emit skills as `skill://` resources in the generated `.kiro/` config so that Kiro's native progressive disclosure applies when running agents through the Kiro backend.

---

## Acceptance Criteria

### AC-1: Skill File Format

#### AC-1.1: YAML Frontmatter Required
- **WHEN** a `.md` file in `.kiro/skills/` or its subdirectories has YAML frontmatter with `name` and `description`
- **THEN** the skill registry indexes it with metadata only (no body content loaded)
- **SHALL** parse frontmatter using the same `---` delimited format as agent files

#### AC-1.2: Missing Frontmatter Fallback
- **WHEN** a `.md` file lacks YAML frontmatter (legacy format)
- **THEN** the registry derives `name` from the filename (strip `.md`, use directory prefix for subdirs)
- **SHALL** set `description` to empty string and mark the skill as `legacy: true`
- **SHALL** log a structured warning: `"skill missing frontmatter"` with path

#### AC-1.3: Frontmatter Validation
- **WHEN** frontmatter is present but `name` is empty or exceeds 64 characters
- **THEN** the skill is skipped with a validation warning
- **SHALL NOT** crash or abort discovery of other skills

---

### AC-2: Metadata-Only Discovery

#### AC-2.1: Startup Loads Metadata Only
- **WHEN** `Discover()` or `DiscoverSkills()` is called
- **THEN** only frontmatter (name, description) and file path are stored in the registry
- **SHALL NOT** call `os.ReadFile()` on the full file body during discovery
- **SHALL** read only the first 1KB of each file to extract frontmatter (optimization)

#### AC-2.2: Registry Exposes Metadata
- **WHEN** the pipeline or compose layer queries the skill registry
- **THEN** it receives `[]SkillMeta{Name, Description, Path}` — no content field
- **SHALL** support listing all skills and lookup by name

#### AC-2.3: Backward-Compatible ProjectContext
- **WHEN** existing code accesses `ProjectContext.Skills`
- **THEN** behavior is preserved (returns metadata) but `Content` field is empty by default
- **SHALL** add `SkillRegistry` field to `ProjectContext` for the new API

---

### AC-3: On-Demand Content Loading

#### AC-3.1: Load Full Content by Name
- **WHEN** an operative (or the compose layer) calls `registry.LoadSkill(name)`
- **THEN** the full file content is read from disk and returned
- **SHALL** strip frontmatter before returning body content
- **SHALL** respect existing `maxSkillSize` truncation (2000 bytes default)

#### AC-3.2: Operative Skill Request Protocol
- **WHEN** an operative's prompt contains `@skill(name)` or references a skill by name
- **THEN** the pipeline injects the skill content into the next compose pass
- **SHALL** document this protocol in the operative's available skill list

#### AC-3.3: Skill Catalog in Prompt
- **WHEN** the composed prompt is built and skills exist
- **THEN** include a `--- AVAILABLE SKILLS ---` section listing name + description only
- **SHALL** instruct the operative: "Request a skill with @skill(name) to load its full content"
- **SHALL** replace the current eager skill injection block in `compose.go`

---

### AC-4: Skill Registry Implementation

#### AC-4.1: SkillRegistry Type
- **WHEN** the `internal/discovery` package is imported
- **THEN** a `SkillRegistry` type is available with methods: `List()`, `Get(name)`, `LoadContent(name)`
- **SHALL** be safe for concurrent use (read-heavy, write-rare)

#### AC-4.2: Registry Initialization
- **WHEN** `Discover()` is called
- **THEN** `ProjectContext.SkillRegistry` is populated from `.kiro/skills/`
- **SHALL** handle missing directory gracefully (empty registry, no error)

#### AC-4.3: Caching Loaded Content
- **WHEN** `LoadContent(name)` is called multiple times for the same skill
- **THEN** the second call returns cached content (no repeated disk I/O)
- **SHALL** cache entries are bounded (evict after 20 skills to prevent unbounded memory)

---

### AC-5: Kiro-Native skill:// Emission

#### AC-5.1: Generate Skill Resources
- **WHEN** `moonbase init` or a future `moonbase kiro-export` runs
- **THEN** each discovered skill is emitted as a `skill://` resource reference in `.kiro/` config
- **SHALL** emit path relative to project root: `skill://skills/{name}/SKILL.md`
- **SHALL** include `name` and `description` in the resource metadata

#### AC-5.2: Independent of kiro-native-interop
- **WHEN** the kiro-native-interop spec is not yet implemented
- **THEN** skill:// emission is a standalone function that writes to a known file path
- **SHALL** be callable independently: `discovery.EmitKiroSkillResources(registry, outputPath)`

#### AC-5.3: Format Alignment
- **WHEN** skill resources are emitted
- **THEN** the format matches Kiro's skill structure: directory with `SKILL.md` + YAML frontmatter
- **SHALL** `moonbase init` scaffold skills in this structure by default

---

## Scope

### In Scope
- `SkillMeta` and `SkillRegistry` types in `internal/discovery/`
- Refactor `discoverSkills()` to metadata-only with deferred loading
- Refactor `ComposePrompt()` skill section from eager injection to catalog + on-demand
- Operative `@skill(name)` request protocol and documentation
- Skill file format: markdown with YAML frontmatter (`name`, `description`)
- `skill://` resource emission function
- Updated `moonbase init` scaffolding for new skill format
- Tests for all new types and behaviors

### Out of Scope
- Full kiro-native-interop backend (separate spec; this is independently implementable)
- Skill marketplace or remote skill fetching
- Runtime package installation within skills
- Skill versioning or dependency resolution
- Changes to the agent `.md` format

---

## Dependencies

| Dependency | Status | Impact |
|-----------|--------|--------|
| `gopkg.in/yaml.v3` | Already in go.mod | YAML frontmatter parsing |
| kiro-native-interop spec | Not yet written | AC-5 emits skill:// resources; full integration deferred |
| Existing `internal/discovery/` | Stable | Refactoring existing functions |

---

## Risks

| Risk | Mitigation |
|------|-----------|
| Breaking existing skill loading for projects without frontmatter | AC-1.2 fallback: derive name from filename, load eagerly for legacy skills |
| Operatives don't know to request skills | AC-3.3: catalog section in prompt with clear instructions |
| Repeated disk reads hurt performance in hot paths | AC-4.3: LRU cache with bounded size |
| Large skill files exceed context when loaded | Existing `maxSkillSize` truncation preserved |
| skill:// format changes upstream | AC-5.2: emission is standalone; format is a simple directory + frontmatter convention |

---

## Context Management Rationale

This design directly applies Anthropic's documented guidance:

1. **"Load metadata upfront, full content on demand"** — *Agent Skills, Progressive Disclosure* (platform.claude.com): Level 1 metadata (~100 tokens per skill) is always loaded; Level 2 instructions load only when triggered.

2. **"Tool search keeps definitions out of context until asked for"** — *Manage Tool Context* (platform.claude.com): The skill catalog pattern mirrors tool_search — operatives see a lightweight index and request specific skills by name.

3. **Prompt caching optimization** — The catalog (static, small) stays in the cacheable prefix; injected skill content (dynamic, per-request) goes after the cache boundary.
