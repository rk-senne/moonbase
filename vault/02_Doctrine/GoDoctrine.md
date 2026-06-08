# Go Doctrine

Go-specific engineering standards for Moonbase.

---

## Core Philosophy

Go favours clarity over cleverness. Follow this.

- Write code that reads like prose, not poetry.
- Accept Go's verbosity as a feature, not a flaw.
- Prefer the standard library over third-party packages.
- If a third-party package is needed, it must justify its weight.

---

## Style

Follow Effective Go and Go Code Review Comments.

- `gofmt` is non-negotiable. All code must be formatted.
- Use `golint` / `staticcheck` conventions.
- Variable names: short in small scopes, descriptive in larger scopes.
- Package names: short, lowercase, no underscores, singular.
- Exported names must have doc comments.
- Error messages: lowercase, no punctuation, no "failed to" prefix unless wrapping.

---

## Error Handling

- Always handle errors. Never ignore them with `_`.
- Wrap errors with context: `fmt.Errorf("loading config: %w", err)`.
- Return errors up. Let the caller decide.
- Use sentinel errors or custom types only when callers need to distinguish failure modes.
- Do not panic in library code. Panic only in main or truly unrecoverable situations.

---

## Package Design

- Packages should do one thing well.
- Avoid circular dependencies.
- Internal packages for implementation details.
- Accept interfaces, return structs.
- Keep exported surface area small.

---

## Concurrency

- Do not start goroutines without a shutdown path.
- Use `context.Context` for cancellation and timeouts.
- Prefer channels for communication, mutexes for state protection.
- Never leak goroutines. Every goroutine must have a termination condition.
- Use `sync.WaitGroup` or `errgroup` for coordinating concurrent work.

---

## Testing

- Table-driven tests are the default for function testing.
- Test files live next to the code: `foo_test.go` beside `foo.go`.
- Use `testify` only if the project already uses it. Otherwise, standard `testing` package.
- Test behaviour, not implementation. Test the public API.
- Use `t.Helper()` in test helpers.
- Name test cases clearly: `TestLoadConfig_MissingFile`, `TestParse_EmptyInput`.

---

## Dependencies

- `go mod tidy` must pass cleanly.
- Pin dependencies via `go.sum`.
- Prefer standard library: `net/http`, `encoding/json`, `os`, `fmt`, `strings`, `path/filepath`.
- Evaluate dependencies by: maintenance activity, dependency count, license, security history.
- Avoid dependencies that pull in the world for one function.

---

## Project Structure (Moonbase)

```
cmd/moonbase/         — main entry point
internal/             — private implementation packages
  tui/                — terminal UI (Bubble Tea)
  chat/               — chat/streaming logic
  agents/             — agent registry and loading
  backend/            — AI provider backends
  config/             — configuration loading
  pipeline/           — agent pipeline orchestration
agents/               — agent definitions (Profile.md, config.json, prompt)
doctrine/             — shared engineering doctrine
docs/                 — project documentation
bin/                  — compiled binary
```

- `cmd/` contains only main.
- `internal/` contains all private logic.
- No business logic in `cmd/`.
- No global state. Pass dependencies explicitly.

---

## Bubble Tea (TUI)

Moonbase uses Bubble Tea for its terminal UI.

- Models own their state. Messages drive updates. Views are pure renders.
- Keep `Update()` methods focused. Delegate to sub-models.
- Use `tea.Cmd` for side effects. Never perform I/O in `Update()` directly.
- Test models by sending messages and asserting state, not by rendering.

---

## Go Proverbs (Reference)

- Don't communicate by sharing memory; share memory by communicating.
- Concurrency is not parallelism.
- The bigger the interface, the weaker the abstraction.
- Make the zero value useful.
- A little copying is better than a little dependency.
- Clear is better than clever.
- Errors are values.
- Don't just check errors, handle them gracefully.

---

## Final Rule

If the Go standard library can do it, use the standard library.

If you must add a dependency, document why.

If you must be clever, document why harder.
