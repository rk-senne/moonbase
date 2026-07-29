package templates

import "strings"

// GenerateSteeringFiles returns a map of filename→content for the steering
// files that moonbase init generates by default. Content is adapted to the
// detected stack. Always produces exactly 6 files regardless of stack.
//
// The data-access-performance standard is intentionally NOT included here — it
// is an opt-in steering file (see GenerateDataAccessPerformance) for projects
// that have a data layer that warrants it.
func GenerateSteeringFiles(stack, buildTool, testCmd string) map[string]string {
	normalized := normalizeStack(stack)

	return map[string]string{
		"dev-rules.md":            generateDevRules(normalized),
		"production-standards.md": generateProductionStandards(normalized),
		"test-alignment.md":       generateTestAlignment(normalized),
		"reasoning-protocol.md":   generateReasoningProtocol(),
		"quality-gates.md":        generateQualityGates(normalized),
		"changelog.md":            generateChangelog(),
	}
}

// GenerateDataAccessPerformance returns the opt-in data-access & performance
// steering standard, adapted to the detected stack. This is generated only when
// explicitly requested (e.g. 'moonbase init --data-access') for projects with a
// data layer that warrants it — not part of the default steering set.
func GenerateDataAccessPerformance(stack string) string {
	return generateDataAccessPerformance(normalizeStack(stack))
}

// normalizeStack maps various stack identifiers to canonical names.
func normalizeStack(stack string) string {
	s := strings.ToLower(strings.TrimSpace(stack))
	switch {
	case s == "go" || s == "golang":
		return "go"
	case s == "java" || s == "kotlin" || s == "spring" || s == "springboot":
		return "java"
	case s == "javascript" || s == "typescript" || s == "js" || s == "ts" || s == "node" || s == "nodejs":
		return "javascript"
	case s == "python" || s == "py":
		return "python"
	case s == "rust" || s == "rs":
		return "rust"
	default:
		return "generic"
	}
}

// --- dev-rules.md ---

func generateDevRules(stack string) string {
	switch stack {
	case "go":
		return devRulesGo
	case "java":
		return devRulesJava
	case "javascript":
		return devRulesJavaScript
	case "python":
		return devRulesPython
	case "rust":
		return devRulesRust
	default:
		return devRulesGeneric
	}
}

const devRulesGo = `# Development Rules

## Stack
- Go 1.22+
- Build: ` + "`go build ./...`" + `
- Test: ` + "`go test ./...`" + `
- Lint: ` + "`go vet ./...`" + `

## Conventions
- Errors: wrap with context using ` + "`" + `fmt.Errorf("what failed: %w", err)` + "`" + `
- Naming: Go standard (camelCase locals, PascalCase exports)
- Tests: ` + "`*_test.go`" + ` in same package, table-driven where applicable
- No global state — pass dependencies through constructors
- Interfaces: define where consumed, not where implemented
- Files: one responsibility per file, max ~300 lines before splitting
`

const devRulesJava = `# Development Rules

## Stack
- Java 17+
- Build: ` + "`mvn clean package`" + ` / ` + "`./gradlew build`" + `
- Test: ` + "`mvn test`" + ` / ` + "`./gradlew test`" + `
- Framework: Spring Boot (if detected)

## Conventions
- Constructor injection (final fields, no @Autowired on constructors)
- Package-private over public for Spring components
- Immutable value objects where possible
- DTOs separate from domain entities
- Preconditions.checkNotNull/checkArgument for validation
- One class per file, class name matches filename
- Use Optional for nullable returns, never return null
`

const devRulesJavaScript = `# Development Rules

## Stack
- TypeScript preferred over raw JavaScript
- Build: ` + "`npm run build`" + `
- Test: ` + "`npm test`" + `
- Lint: ` + "`npm run lint`" + ` (ESLint + Prettier)

## Conventions
- Strict mode always (` + "`" + `"strict": true` + "`" + ` in tsconfig)
- Prefer ` + "`const`" + ` over ` + "`let`" + `, never ` + "`var`" + `
- Named exports over default exports
- Error handling: use typed errors, never throw raw strings
- Async/await over raw promises
- Pure functions where possible, side effects at edges
- Small modules: one concept per file
`

const devRulesPython = `# Development Rules

## Stack
- Python 3.10+
- Build: ` + "`pip install -e .`" + ` / ` + "`poetry install`" + `
- Test: ` + "`python -m pytest`" + `
- Lint: ` + "`ruff check .`" + ` / ` + "`mypy .`" + `

## Conventions
- Type hints on all public function signatures
- Dataclasses or Pydantic for structured data
- Context managers for resource cleanup
- f-strings for formatting
- No bare ` + "`except:`" + ` — always specify exception type
- Docstrings on all public functions (Google style)
- One class per module for complex classes
`

const devRulesRust = `# Development Rules

## Stack
- Rust (latest stable)
- Build: ` + "`cargo build`" + `
- Test: ` + "`cargo test`" + `
- Lint: ` + "`cargo clippy -- -D warnings`" + `

## Conventions
- Error handling: ` + "`Result<T, E>`" + ` with thiserror/anyhow
- Prefer ` + "`&str`" + ` over ` + "`String`" + ` in function parameters
- Derive Clone, Debug, PartialEq where sensible
- No ` + "`.unwrap()`" + ` in production code — use ` + "`?`" + ` or explicit handling
- Module structure mirrors responsibility
- Unsafe code requires a // SAFETY: comment
- Ownership: prefer borrowing over cloning
`

const devRulesGeneric = `# Development Rules

## Stack
- Identify and document the primary language and framework
- Build: run the project's standard build command
- Test: run the project's standard test command
- Lint: run the project's standard linter

## Conventions
- Follow the existing code style in the project
- Wrap errors with context — include what operation failed
- One responsibility per file
- No global mutable state — pass dependencies explicitly
- Validate inputs at public boundaries
- Document public APIs
`

// --- production-standards.md ---

func generateProductionStandards(stack string) string {
	errorHandling := productionErrorHandling(stack)
	resourceCleanup := productionResourceCleanup(stack)

	return `# Production Code Standards

## Zero Tolerance
- **No TODOs** — if it's not done, don't merge it. Track in an issue instead.
- **No placeholders** — "implement later" is a bug. Ship complete or don't ship.
- **No swallowed errors** — every error must be handled. Never silently ignore failures.
- **No hardcoded values** — use configuration, constants, or environment variables.

## Error Handling
` + errorHandling + `
## Input Validation
- Validate at public function/method boundaries.
- Fail fast — return clear errors before doing work.
- Never trust user input in shell commands or queries.

## Resource Cleanup
` + resourceCleanup + `
## Performance
- No unbounded allocations — paginate or limit lists that can grow.
- Profile before optimizing — don't guess at bottlenecks.
- Avoid repeated I/O in hot paths — cache or pass data down.

## Security
- No secrets in code, config files, or logs. Environment variables only.
- Validate all external input.
- Fail closed — if validation is ambiguous, reject.
`
}

func productionErrorHandling(stack string) string {
	switch stack {
	case "go":
		return `- Wrap with context: ` + "`" + `fmt.Errorf("what failed: %w", err)` + "`" + `
- Use sentinel errors: ` + "`" + `var ErrNotFound = errors.New("not found")` + "`" + `
- Check errors immediately after the call — no deferred error checks.
- Return errors to the caller; let the top-level decide how to present them.

`
	case "java":
		return `- Checked exceptions for recoverable errors, unchecked for programming errors.
- Wrap with context — never catch-and-ignore.
- Log or rethrow, never both.
- Use specific exception types over generic Exception.

`
	case "javascript":
		return `- Use typed Error subclasses for different failure modes.
- Async errors propagate with await — never swallow ` + "`.catch(() => {})`" + `.
- Always handle promise rejections.
- Include context in error messages.

`
	case "python":
		return `- Use specific exception types — never bare ` + "`except:`" + `.
- Chain exceptions with ` + "`raise ... from ...`" + ` to preserve context.
- Log or raise, never both silently.
- Custom exceptions inherit from domain-specific base classes.

`
	case "rust":
		return `- Use ` + "`Result<T, E>`" + ` for all fallible operations.
- Propagate with ` + "`?`" + ` operator.
- Use ` + "`thiserror`" + ` for library errors, ` + "`anyhow`" + ` for applications.
- No ` + "`.unwrap()`" + ` or ` + "`.expect()`" + ` in production paths.

`
	default:
		return `- Wrap errors with context — include what operation failed and why.
- Never silently ignore errors — handle, propagate, or log them.
- Use specific error types where the language supports it.
- Return errors to the caller; let the top-level decide presentation.

`
	}
}

func productionResourceCleanup(stack string) string {
	switch stack {
	case "go":
		return `- ` + "`defer f.Close()`" + ` immediately after successful open.
- ` + "`defer cancel()`" + ` immediately after ` + "`context.WithCancel`" + `/` + "`WithTimeout`" + `.
- Long-running goroutines must respect context cancellation.
- File permissions: ` + "`0600`" + ` for user data, ` + "`0700`" + ` for directories.

`
	case "java":
		return `- try-with-resources for all AutoCloseable objects.
- ` + "`@PreDestroy`" + ` for Spring lifecycle cleanup.
- Close database connections, HTTP clients, and streams explicitly.
- Use connection pools — never create connections per-request.

`
	case "javascript":
		return `- ` + "`try/finally`" + ` for cleanup that must always run.
- ` + "`using`" + ` declarations (Stage 3) for disposable resources.
- AbortController for fetch cancellation.
- Clear intervals and timeouts when components unmount.

`
	case "python":
		return `- ` + "`with`" + ` statement (context managers) for file and network resources.
- ` + "`atexit`" + ` for process-level cleanup.
- Close database connections in finally blocks.
- Use contextlib for custom resource management.

`
	case "rust":
		return `- Drop trait handles cleanup automatically (RAII pattern).
- Use scoped guards for temporary state changes.
- Explicit ` + "`.close()`" + ` only when Drop timing matters.
- Arc/Rc for shared ownership — avoid reference cycles.

`
	default:
		return `- Clean up resources (files, connections, handles) when done with them.
- Use the language's idiomatic cleanup pattern (try/finally, defer, with, RAII).
- Never leave resources open across function boundaries without good reason.
- Set appropriate file permissions for user data.

`
	}
}

// --- test-alignment.md ---

func generateTestAlignment(stack string) string {
	testLocation := testAlignmentLocation(stack)
	testQuality := testAlignmentQuality(stack)

	return `# Test Alignment

## Principle
Tests are part of the spec. If a feature isn't tested, it isn't done.
Tests document behavior — a reader should understand the contract from tests alone.

## Rules
- Every public function/method has at least one test.
- Table-driven / parameterized tests for functions with 3+ distinct cases.
` + testLocation + `
## Cross-Check Matrix
| Change Type | Required Test |
|---|---|
| New public function/method | Unit test: happy path + error path |
| New CLI command / API endpoint | Integration test exercising full flow |
| Bug fix | Regression test reproducing the bug first |
| New config option | Test for default value + override |
| New data model / schema | Validation tests for valid + invalid |

## Test Quality
- Tests assert behavior, not implementation.
- One logical assertion per test case.
- No test interdependence — each test sets up its own state.
` + testQuality
}

func testAlignmentLocation(stack string) string {
	switch stack {
	case "go":
		return `- Test files: ` + "`*_test.go`" + ` in same package.
- ` + "`go test -race ./...`" + ` must pass — no data races tolerated.
- Integration tests use build tag: ` + "`//go:build integration`" + `

`
	case "java":
		return `- Test files: ` + "`src/test/java/`" + ` mirroring main source structure.
- Use @ExtendWith(MockitoExtension.class) for unit tests.
- @SpringBootTest for integration tests.

`
	case "javascript":
		return `- Test files: ` + "`*.test.ts`" + ` / ` + "`*.spec.ts`" + ` colocated or in ` + "`__tests__/`" + `.
- Runner: Jest or Vitest.
- Coverage target: >80%.

`
	case "python":
		return `- Test files: ` + "`tests/`" + ` directory mirroring source structure.
- Use pytest fixtures for setup/teardown.
- ` + "`python -m pytest --cov`" + ` for coverage.

`
	case "rust":
		return `- Unit tests: ` + "`#[cfg(test)] mod tests`" + ` in same file.
- Integration tests: ` + "`tests/`" + ` directory at crate root.
- ` + "`cargo test`" + ` must pass with no warnings.

`
	default:
		return `- Test files live near the code they test.
- Integration tests are clearly separated from unit tests.
- Run the full test suite before merging.

`
	}
}

func testAlignmentQuality(stack string) string {
	switch stack {
	case "go":
		return `- Use ` + "`t.Helper()`" + ` in test utility functions.
- Use ` + "`t.TempDir()`" + ` for file system tests — never write to real project dirs.
- Use ` + "`testdata/`" + ` for fixture files.
`
	case "java":
		return `- Use AssertJ for fluent, readable assertions.
- Mock external dependencies, not the class under test.
- Use @Nested for grouping related test cases.
`
	case "javascript":
		return `- Use ` + "`describe`" + `/` + "`it`" + ` blocks for clear test organization.
- Mock external modules at the boundary, not internals.
- Prefer ` + "`toEqual`" + ` for deep equality, ` + "`toBe`" + ` for identity.
`
	case "python":
		return `- Use pytest fixtures over setUp/tearDown methods.
- Use ` + "`tmp_path`" + ` fixture for file system tests.
- Parametrize with ` + "`@pytest.mark.parametrize`" + `.
`
	case "rust":
		return `- Use ` + "`assert_eq!`" + ` and ` + "`assert_ne!`" + ` for clear failure messages.
- Use ` + "`#[should_panic]`" + ` for expected panics.
- Use ` + "`tempfile`" + ` crate for file system tests.
`
	default:
		return `- Use the language's standard assertion library.
- Isolate tests from external systems (mock or use in-memory alternatives).
- Use temporary directories for file system tests.
`
	}
}

// --- reasoning-protocol.md (100% universal) ---

func generateReasoningProtocol() string {
	return reasoningProtocol
}

const reasoningProtocol = `# Reasoning Protocol

## Task Scaling

Calibrate effort to complexity:

| Complexity | Action |
|---|---|
| Trivial (typo, rename, one-liner) | Fix directly, verify builds |
| Simple (add field, new test, small refactor) | Read context → implement → test |
| Complex (new feature, architecture change) | Full protocol below |

## Full Protocol (Complex Tasks)

### 1. Retrieval First
- Read the relevant code before making claims about it.
- Check ` + "`.kiro/specs/`" + ` for existing requirements and design decisions.
- Check ` + "`.kiro/steering/`" + ` for project conventions.
- Identify the boundaries the change touches.

### 2. Alternative Generation
- Generate 2–3 approaches for any non-trivial design decision.
- Evaluate trade-offs: complexity, testability, blast radius, reversibility.
- Select with reasoning — state why the chosen approach wins.

### 3. Adversarial Review
Before considering work done, challenge it:
- **Hidden assumptions** — what am I taking for granted?
- **Edge cases** — empty input, nil/null, max-length, concurrent access?
- **Scaling risks** — does this work with 1 item? 1000? 1 million?
- **Integration risk** — does this break existing callers?
- **Security** — does this introduce injection, traversal, or leakage?

### 4. Self-Verification
- Did I update tests? Docs? Changelog?
- Did I say "this is safe" without evidence?
- Does the change compile and tests pass?
- Does it align with the original request — no scope creep?

## Workspace Awareness
- Follow existing patterns in the code you're modifying.
- Don't introduce new dependencies without justification.
- Don't change architecture as a side-effect of a feature.
`

// --- quality-gates.md ---

func generateQualityGates(stack string) string {
	buildCommands := qualityGatesBuildCommands(stack)

	return `# Quality Gates

## Must Pass Before Done

### Build & Test
` + buildCommands + `
### Code Quality
- [ ] No new TODOs introduced
- [ ] No ignored/swallowed errors in production code
- [ ] All new public functions/methods have documentation
- [ ] Error messages include context
- [ ] Resources are cleaned up properly

### Documentation
- [ ] CHANGELOG.md updated for user-visible changes
- [ ] README updated if interface changed

### Git Discipline
- [ ] Commit messages: ` + "`type(scope): description`" + `
- [ ] Types: feat, fix, refactor, test, docs, chore, perf
- [ ] One logical change per commit
- [ ] No secrets, no generated files, no binary artifacts

## Four-Lens Review
1. **Contract Fidelity** — does the code match the spec?
2. **Architecture Erosion** — does this respect boundaries and patterns?
3. **Completeness** — edge cases handled? Tests written? Docs updated?
4. **Intention Alignment** — does this solve what was actually asked for?

## Risk Classification
| Risk | Response |
|---|---|
| LOW (cosmetic, docs, test-only) | Self-merge after gates pass |
| MEDIUM (new feature, refactor) | Peer review + gates |
| HIGH (security, core logic, data) | Design review + gates + manual test |
| CRITICAL (breaking change, data loss) | Human approval required |
`
}

func qualityGatesBuildCommands(stack string) string {
	switch stack {
	case "go":
		return "```bash\ngo build ./...           # compiles cleanly\ngo vet ./...             # no suspicious constructs\ngo test -race ./...      # all tests pass, no data races\n```\n"
	case "java":
		return "```bash\nmvn clean verify         # or: ./gradlew check\nmvn test                 # all tests pass\n```\n"
	case "javascript":
		return "```bash\nnpm run build            # compiles cleanly\nnpm run lint             # no lint errors\nnpm test                 # all tests pass\nnpm run type-check       # no type errors (if TypeScript)\n```\n"
	case "python":
		return "```bash\nruff check .             # no lint errors\nmypy .                   # no type errors\npython -m pytest --cov   # all tests pass with coverage\n```\n"
	case "rust":
		return "```bash\ncargo build              # compiles cleanly\ncargo clippy -- -D warnings  # no lint warnings\ncargo test               # all tests pass\n```\n"
	default:
		return "```bash\n# Run the project's build command\n# Run the project's linter\n# Run the project's test suite\n```\n"
	}
}

// --- changelog.md (100% universal) ---

func generateChangelog() string {
	return changelog
}

const changelog = `# Changelog Discipline

## Location
` + "`CHANGELOG.md`" + ` at repository root. Always updated as part of the change, not after.

## Format
` + "```" + `markdown
## [Unreleased]

### Added
- feat(scope): description of what was added

### Fixed
- fix(scope): description of what was fixed

### Changed
- refactor(scope): description of what changed
` + "```" + `

## Rules
- Newest first — latest changes at the top.
- One bullet per change — prefix with conventional commit type.
- Record WHY, not just WHAT.
- No secrets — never reference API keys or internal URLs.
- User-visible changes only — skip internal refactors unless they change behavior.

## When to Update
| Change Type | Update Changelog? |
|---|---|
| New feature or command | Yes |
| Bug fix | Yes |
| Performance improvement | Yes (if user-noticeable) |
| Internal refactor | No |
| Test-only changes | No |
| Dependency update | Only if it fixes a bug or adds capability |
`

// --- data-access-performance.md ---

// generateDataAccessPerformance returns the data-access efficiency standard.
// The universal core is the same across stacks; the ORM/lazy-loading guidance
// is adapted to the detected stack (it only applies where an ORM exists).
func generateDataAccessPerformance(stack string) string {
	ormGuidance := dataAccessORMGuidance(stack)

	return `# Data Access & Performance

## Principle
Push work to the data layer. Memory is not a query engine, and an innocent-looking
loop is where O(N²) hides. Efficiency is a design decision, not an afterthought.

## Rules

### Bound Every Query
- **No unbounded queries** — always paginate or ` + "`LIMIT`" + ` results that could grow.
- A query that returns "all rows" is a production incident waiting for enough data.
- Default to a page size; require an explicit opt-in for larger fetches.

### Push Down, Don't Pull Up
- Filter, sort, and aggregate at the **data layer** (SQL, index, query engine) —
  not in application memory — for any collection that could exceed ~100 items.
- ` + "`ORDER BY`" + ` in the query, backed by an index. Never fetch-then-sort-in-memory
  for unbounded sets.
- Aggregations (` + "`COUNT`" + `, ` + "`SUM`" + `, ` + "`GROUP BY`" + `) belong in the query, not a loop.

### No N+1
- Fetch related data in **batches or joins**, never one query per item in a loop.
- A loop that issues a query per iteration is an N+1 problem — fix it before merge.

` + ormGuidance + `
### Complexity Budget
- Aim for O(1) or O(N) on hot paths. O(N log N) (e.g. sorting) is acceptable when
  the work genuinely requires it — but know which one you're writing and why.
- No hidden O(N²): nested loops over the same growing collection are a red flag.
- Pre-allocate slices/collections when the size is known.

## Explicitly Allowed
- In-memory operations on **bounded, small collections** (< ~100 items) — sorting a
  config list or a fixed set of options in memory is fine.
- O(N log N) algorithms where the problem requires ordering or searching.
- Caching computed results that are expensive to regenerate.

## Red Flags (Reject in Review)
- ` + "`SELECT *`" + ` with no ` + "`LIMIT`" + ` on a table that grows.
- Sorting or filtering a database result set in application memory.
- A query inside a loop.
- Loading an entire table/collection to count or aggregate it.
`
}

// dataAccessORMGuidance returns lazy-loading / eager-fetch guidance for stacks
// that commonly use an ORM. Stacks without a dominant ORM get generic advice.
func dataAccessORMGuidance(stack string) string {
	switch stack {
	case "java":
		return `### ORM Discipline (JPA / Hibernate)
- **No lazy loading across boundaries** — lazy associations that resolve during
  serialization cause N+1 and ` + "`LazyInitializationException`" + `. Fetch explicitly.
- Use fetch joins or ` + "`@EntityGraph`" + ` to load what you need in one query.
- Set ` + "`spring.jpa.open-in-view=false`" + ` — OSIV hides N+1 behind view rendering.
- Prefer projections (DTOs) over loading full entities for read-only views.
`
	case "python":
		return `### ORM Discipline (Django / SQLAlchemy)
- **No lazy loading in loops** — use ` + "`select_related`" + `/` + "`prefetch_related`" + ` (Django)
  or ` + "`joinedload`" + `/` + "`selectinload`" + ` (SQLAlchemy) to avoid N+1.
- Never iterate a queryset and access related objects without prefetching.
- Use ` + "`.only()`" + `/` + "`.values()`" + ` to fetch just the columns you need.
`
	case "javascript":
		return `### ORM Discipline (Prisma / TypeORM / Sequelize)
- **No lazy loading in loops** — use ` + "`include`" + ` (Prisma) or eager ` + "`relations`" + `
  (TypeORM) to batch related data in one query.
- Never ` + "`await`" + ` a relation lookup inside a ` + "`.map()`" + `/` + "`for`" + ` over rows — that's N+1.
- Select specific fields rather than hydrating whole entities for read views.
`
	case "go":
		return `### Data Access Discipline
- No ORM lazy loading — Go data access is explicit, keep it that way.
- Build set-based queries; never issue a query per row in a loop.
- Use ` + "`sqlx.In`" + ` / batch queries for "fetch many by IDs" patterns.
- Scan only the columns you need; avoid ` + "`SELECT *`" + ` in library code.
`
	case "rust":
		return `### Data Access Discipline (Diesel / SQLx / SeaORM)
- Prefer explicit joins over per-row relation loads.
- Use ` + "`.load()`" + ` with a single composed query rather than looping queries.
- Batch "fetch by IDs" with ` + "`WHERE id = ANY($1)`" + ` rather than N queries.
- Select specific columns; avoid loading full rows for read-only views.
`
	default:
		return `### ORM / Data Access Discipline
- If using an ORM, prefer **eager/explicit fetching** over lazy loading.
  Lazy loading hides performance cliffs behind innocent property access.
- Batch related-data fetches — never issue one query per item in a loop.
- Select only the fields you need for the operation at hand.
`
	}
}
