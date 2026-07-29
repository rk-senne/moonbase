# Production Code Standards

## Zero Tolerance

- **No TODOs** — if it's not done, don't merge it. Track in an issue instead.
- **No placeholders** — `// implement later` is a bug. Ship complete or don't ship.
- **No swallowed errors** — every `error` must be returned, wrapped, or logged. Never `_ = fn()`.
- **No hardcoded values** — use `internal/config` or constants with clear names.

## Error Handling

- Wrap with context: `fmt.Errorf("loading agent %s: %w", name, err)`
- Return errors to the caller; let the top-level decide how to present them.
- Log at the boundary (CLI layer, TUI layer), not deep in library code.
- Sentinel errors use `var ErrNotFound = errors.New("not found")` pattern.
- Check errors immediately after the call — no deferred error checks.

## Input Validation

- Validate at public function boundaries (exported functions, CLI args, config parsing).
- Fail fast — return clear errors before doing work.
- Agent IDs: `[a-zA-Z0-9-]` only. File paths: sanitize traversal attempts.
- Never trust user input in shell commands — use `exec.Command` with explicit args.

## Resource Cleanup

- `defer f.Close()` immediately after successful open.
- `defer cancel()` immediately after `context.WithCancel`/`WithTimeout`.
- Long-running goroutines must respect context cancellation.
- File permissions: `0600` for user data, `0700` for directories.

## Performance

- No unbounded allocations — pre-allocate slices when size is known.
- Paginate or limit any list that could grow (agents are bounded; history is not).
- Avoid repeated file reads in hot paths — cache or pass data down.
- Profile before optimizing — don't guess at bottlenecks.

## Concurrency Safety

- Document goroutine-safety in doc comments: `// Safe for concurrent use.`
- Protect shared state with `sync.Mutex` or `sync.RWMutex`.
- Prefer channels for coordination, mutexes for state protection.
- Never pass pointers to goroutines without synchronization.

## Observability

- Use `log/slog` (structured logging) — not `fmt.Println` or `log.Printf`.
- Log at boundaries: command entry, pipeline phase transitions, errors.
- Include context: agent name, phase, duration for pipeline operations.
- Logs go to file only — never pollute stdout/stderr (TUI owns those).

## Security

- No secrets in code, config files, or logs. Environment variables only.
- SafeEnv for child processes — allowlist, don't denylist.
- Validate all external input (CLI args, file content, agent frontmatter).
- Fail closed — if validation is ambiguous, reject.
