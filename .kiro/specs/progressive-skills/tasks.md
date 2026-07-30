# Tasks: Progressive Skill Loading

## Milestone 1: Skill Registry Foundation

### Task 1: Create SkillMeta type and frontmatter parser
- **Requirements:** AC-1.1, AC-1.2, AC-1.3, AC-2.1
- **Files:** `internal/discovery/skills.go`
- **Action:** Define `SkillMeta` struct (Name, Description, Path, Legacy bool). Implement `parseFrontmatterOnly(path string) (SkillMeta, error)` that opens the file, reads up to 1KB, extracts YAML frontmatter between `---` delimiters, unmarshals into `struct{Name, Description string}`, validates name (non-empty, ≤64 chars, `[a-z0-9-]`). Return error for missing/invalid frontmatter. Use `gopkg.in/yaml.v3`.
- **Test:** `TestParseFrontmatterOnly_Valid`, `TestParseFrontmatterOnly_MissingFrontmatter`, `TestParseFrontmatterOnly_EmptyName`, `TestParseFrontmatterOnly_NameTooLong`, `TestParseFrontmatterOnly_LargeFile` (body beyond 1KB doesn't affect parsing)
- **Status:** pending

### Task 2: Create SkillRegistry with List/Get/LoadContent
- **Requirements:** AC-4.1, AC-4.2, AC-4.3
- **Files:** `internal/discovery/skills.go`
- **Action:** Implement `SkillRegistry` struct with `sync.RWMutex`, internal `map[string]*skillEntry`, and `[]string` for insertion order. Methods: `NewSkillRegistry() *SkillRegistry`, `Register(meta SkillMeta)`, `List() []SkillMeta`, `Get(name string) *SkillMeta`, `Names() []string`, `LoadContent(name string) (string, error)`. LoadContent reads full file from `meta.Path`, strips frontmatter, truncates to `maxSkillSize`, caches result in entry. Thread-safe: RLock for reads, Lock for cache writes.
- **Test:** `TestSkillRegistry_Register`, `TestSkillRegistry_List_PreservesOrder`, `TestSkillRegistry_Get_Found`, `TestSkillRegistry_Get_NotFound`, `TestSkillRegistry_LoadContent_CachesResult`, `TestSkillRegistry_LoadContent_FileDeleted`, `TestSkillRegistry_ConcurrentAccess`
- **Status:** pending

### Task 3: Create unit tests for skills.go
- **Requirements:** AC-1.1, AC-1.2, AC-1.3, AC-4.1, AC-4.3
- **Files:** `internal/discovery/skills_test.go`
- **Action:** Write table-driven tests for frontmatter parsing and registry operations. Use `t.TempDir()` for fixture files. Create test skills with and without frontmatter. Verify concurrent access with `t.Parallel()` and multiple goroutines.
- **Test:** `go test -race ./internal/discovery/...` passes
- **Status:** pending

### Checkpoint: Registry Foundation
- [ ] `SkillMeta` type defined with Name, Description, Path, Legacy fields
- [ ] `parseFrontmatterOnly` reads ≤1KB, extracts YAML, validates
- [ ] `SkillRegistry` supports Register/List/Get/Names/LoadContent
- [ ] LoadContent caches after first read, strips frontmatter
- [ ] All tests pass with `-race`
- [ ] `go build ./...` passes

---

## Milestone 2: Discovery Integration

### Task 4: Add discoverSkillRegistry to discovery.go
- **Requirements:** AC-2.1, AC-2.2, AC-2.3, AC-4.2
- **Files:** `internal/discovery/discovery.go`
- **Action:** Implement `discoverSkillRegistry(projectDir string) *SkillRegistry`. Walk `.kiro/skills/` (same directory structure as current `discoverSkills`): flat `.md` files and subdirectory `.md` files. For each file: call `parseFrontmatterOnly`. On success: register. On error (no frontmatter): create legacy `SkillMeta` with name derived from filename, `Legacy: true`, register it. Add `SkillRegistry *SkillRegistry` field to `ProjectContext`. Call `discoverSkillRegistry` in `Discover()`. Keep existing `discoverSkills()` call temporarily for backward compatibility.
- **Test:** `TestDiscover_SkillRegistry_Populated`, `TestDiscover_SkillRegistry_LegacyFallback`, `TestDiscover_SkillRegistry_EmptyDir`, `TestDiscover_SkillRegistry_MissingDir`
- **Status:** pending

### Task 5: Refactor ComposePrompt skill section to catalog
- **Requirements:** AC-3.3
- **Files:** `internal/discovery/compose.go`
- **Action:** Replace the current `--- PROJECT SKILLS ---` section (which injects full content) with `--- AVAILABLE SKILLS ---` section that lists only name + description in a table. Add instruction line: "Request a skill with @skill(name) to load its full content into your context." Guard: if `SkillRegistry` is nil or empty, fall back to existing `[]SkillFile` injection (backward compat for callers that haven't migrated). If registry is present and non-empty, prefer catalog over legacy injection.
- **Test:** `TestComposePrompt_SkillCatalog_ShowsMetadataOnly`, `TestComposePrompt_SkillCatalog_FallbackToLegacy`, `TestComposePrompt_SkillCatalog_EmptyRegistry`
- **Status:** pending

### Task 6: Update existing discovery tests
- **Requirements:** AC-2.3
- **Files:** `internal/discovery/discovery_test.go`, `internal/discovery/discovery_new_test.go`
- **Action:** Update any tests that assert on `ProjectContext.Skills` to also verify `ProjectContext.SkillRegistry` is populated. Ensure existing tests still pass (backward compat). Add test that legacy skills (no frontmatter) still appear in `ProjectContext.Skills`.
- **Test:** `go test -race ./internal/discovery/...` — all existing + new tests pass
- **Status:** pending

### Checkpoint: Discovery Integration
- [ ] `Discover()` populates both `Skills` (legacy) and `SkillRegistry` (new)
- [ ] `ComposePrompt` emits catalog (metadata only) when registry is present
- [ ] Catalog includes per-skill name + description + @skill() instruction
- [ ] Backward compat: old code using `ProjectContext.Skills` still works
- [ ] All existing tests pass
- [ ] `go build ./...` passes

---

## Milestone 3: On-Demand Loading Pipeline

### Task 7: Implement @skill() pattern scanner
- **Requirements:** AC-3.1, AC-3.2
- **Files:** `internal/discovery/skills.go` (or `internal/pipeline/skill_resolve.go` if pipeline package exists)
- **Action:** Implement `ExtractSkillRequests(text string) []string` — regex scan for `@skill\(([a-z0-9-]+)\)` pattern. Return unique skill names found. Implement `ResolveSkills(registry *SkillRegistry, names []string) ([]ResolvedSkill, []string)` — returns loaded content for found skills and list of not-found names.
- **Test:** `TestExtractSkillRequests_SingleMatch`, `TestExtractSkillRequests_MultipleMatches`, `TestExtractSkillRequests_NoMatches`, `TestExtractSkillRequests_InvalidChars_Ignored`, `TestResolveSkills_FoundAndNotFound`
- **Status:** pending

### Task 8: Integrate skill injection into compose flow
- **Requirements:** AC-3.1, AC-3.2
- **Files:** `internal/discovery/compose.go`
- **Action:** Add `ComposeWithSkills(agentPrompt string, context *ProjectContext, task string, requestedSkills []string) string` function. This calls `ComposePrompt` and then appends `--- SKILL: {name} ---` sections for each requested skill. Keeps the cacheable prefix optimization intact (skills go after task, in the dynamic section). Update `ComposePrompt` to accept a variadic `...ComposeOption` for skill injection (or use the separate function approach).
- **Test:** `TestComposeWithSkills_InjectsContent`, `TestComposeWithSkills_SkipsMissing`, `TestComposeWithSkills_RespectsMaxSize`
- **Status:** pending

### Task 9: Pre-scan task text for skill references
- **Requirements:** AC-3.2
- **Files:** `internal/discovery/compose.go`
- **Action:** In `ComposePrompt` (or the calling layer), pre-scan the `task` parameter for `@skill(...)` patterns. If found, auto-load and inject those skills. This handles single-shot deploys where the operative can't interactively request skills (e.g., `moonbase deploy 4 "use @skill(docker-build) to review the Dockerfile"`).
- **Test:** `TestComposePrompt_TaskContainsSkillRef_AutoInjects`
- **Status:** pending

### Checkpoint: On-Demand Loading
- [ ] `@skill(name)` pattern extracted from text via regex
- [ ] `ResolveSkills` loads content for valid names, reports missing
- [ ] `ComposeWithSkills` appends skill sections after task
- [ ] Task text pre-scanned for skill references (single-shot support)
- [ ] All tests pass with `-race`
- [ ] `go build ./...` passes

---

## Milestone 4: Kiro-Native skill:// Emission

### Task 10: Implement EmitKiroSkillResources
- **Requirements:** AC-5.1, AC-5.2, AC-5.3
- **Files:** `internal/discovery/kiro_skills.go`
- **Action:** Implement `EmitKiroSkillResources(registry *SkillRegistry, outputDir string) error`. For each skill in the registry: create `outputDir/{name}/` directory (0700). If source is already directory-style (path ends with `SKILL.md`): copy file to `outputDir/{name}/SKILL.md`. If source is flat file: read content, ensure YAML frontmatter is present, write to `outputDir/{name}/SKILL.md`. Return error wrapping any failures (but continue on individual file errors).
- **Test:** `TestEmitKiroSkillResources_CreatesDirectories`, `TestEmitKiroSkillResources_FlatFileConverted`, `TestEmitKiroSkillResources_DirectoryStyleCopied`, `TestEmitKiroSkillResources_EmptyRegistry`, `TestEmitKiroSkillResources_OutputDirCreated`
- **Status:** pending

### Task 11: Create kiro_skills_test.go
- **Requirements:** AC-5.1, AC-5.2, AC-5.3
- **Files:** `internal/discovery/kiro_skills_test.go`
- **Action:** Write tests for `EmitKiroSkillResources`. Use `t.TempDir()` for both source skills and output. Verify directory structure, file content, frontmatter presence, permissions.
- **Test:** `go test -race ./internal/discovery/...` passes
- **Status:** pending

### Task 12: Update moonbase init scaffolding
- **Requirements:** AC-5.3, AC-1.1
- **Files:** `cmd/moonbase/init.go`
- **Action:** Update skills scaffolding to create an example skill in Kiro-native directory format: `.kiro/skills/example/SKILL.md` with proper YAML frontmatter (`name: example`, `description: Example skill — replace with your domain knowledge`). Update the `skillsReadme` constant to document the new format and `@skill()` protocol. Remove old flat-file README-only scaffolding.
- **Test:** Manual: `moonbase init` in temp dir creates `.kiro/skills/example/SKILL.md` with valid frontmatter
- **Status:** pending

### Checkpoint: Kiro-Native Emission
- [ ] `EmitKiroSkillResources` creates `{name}/SKILL.md` directory structure
- [ ] Flat files converted to directory style with frontmatter preserved
- [ ] `moonbase init` scaffolds example skill in Kiro-native format
- [ ] All emission tests pass
- [ ] `go build ./...` passes

---

## Milestone 5: Documentation & Validation

### Task 13: Update README skills section
- **Requirements:** AC-3.2, AC-3.3
- **Files:** `README.md`
- **Action:** Add a "Skills" section under Key Capabilities documenting: skill file format (frontmatter), progressive loading behavior, `@skill(name)` protocol for operatives, and skill:// emission for Kiro-native mode. Keep brief — link to the spec for details.
- **Test:** README renders correctly, no broken markdown
- **Status:** pending

### Task 14: End-to-end integration validation
- **Requirements:** All ACs
- **Files:** N/A (manual validation)
- **Action:** In the moonbase project itself: (1) create `.kiro/skills/go-testing/SKILL.md` with frontmatter; (2) run `go test ./internal/discovery/...` confirming registry picks it up; (3) verify `ComposePrompt` outputs catalog not full content; (4) verify `@skill(go-testing)` in task text triggers injection; (5) run `moonbase lint` — still passes.
- **Test:** All quality gates pass: `go build ./...`, `go vet ./...`, `go test -race ./...`
- **Status:** pending

### Final Checkpoint
- [ ] All 14 tasks complete
- [ ] `go build ./...` passes
- [ ] `go test -race ./...` passes (all packages)
- [ ] `moonbase lint` passes (14 agents valid)
- [ ] No new TODOs introduced
- [ ] CHANGELOG.md updated under `[Unreleased]`
