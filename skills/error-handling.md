---
name: error-handling
description: Error handling discipline — wrap with context, sentinel errors, fail fast, never swallow errors, log at boundaries only.
---

# Error Handling

## Wrap with Context

Every error returned must carry enough context to diagnose without a debugger:

```go
// Bad — caller has no idea what failed
return err

// Good — caller knows what operation failed and on what input
return fmt.Errorf("loading agent %s: %w", name, err)
```

Wrapping rules:
- Include the operation: "opening", "parsing", "writing", "connecting to".
- Include the identifying input: filename, ID, URL.
- Use `%w` to preserve the error chain for `errors.Is`/`errors.As`.
- Do not double-wrap: if the callee already wrapped, just return.

## Sentinel Errors

Define package-level sentinel errors for conditions callers need to branch on:

```go
var (
    ErrNotFound     = errors.New("not found")
    ErrUnauthorized = errors.New("unauthorized")
    ErrConflict     = errors.New("conflict")
)
```

- Callers use `errors.Is(err, ErrNotFound)` to detect the condition.
- Never compare error strings — use sentinels or custom types.
- Keep sentinel names short, prefixed with `Err`.

## Fail Fast

- Validate inputs at the top of the function, before doing any work.
- Return errors immediately — do not continue with partial state.
- Group validation: check all fields, return all errors (not one at a time).

```go
func CreateUser(req CreateUserRequest) (*User, error) {
    if req.Email == "" {
        return nil, fmt.Errorf("creating user: %w", ErrMissingEmail)
    }
    if !isValidEmail(req.Email) {
        return nil, fmt.Errorf("creating user: %w", ErrInvalidEmail)
    }
    // Only proceed after all preconditions are met
    ...
}
```

## Never Swallow Errors

```go
// NEVER do this in production code
_ = file.Close()
json.Unmarshal(data, &v) // ignoring return

// Always handle or propagate
if err := file.Close(); err != nil {
    return fmt.Errorf("closing config file: %w", err)
}
```

- Every `error` return must be checked on the next line.
- `_ = fn()` is a bug in production code (acceptable only in tests).
- If an error truly cannot be handled, log it at the boundary with context.

## Log at Boundaries Only

- Library/package code: return errors, never log them.
- Boundary code (CLI handlers, HTTP handlers, TUI): log and present.
- Logging inside a library means the error gets logged multiple times.

```go
// internal/pipeline — returns error, does NOT log
func (p *Pipeline) Run(ctx context.Context) error {
    if err := p.analyze(ctx); err != nil {
        return fmt.Errorf("pipeline analysis: %w", err)
    }
    return nil
}

// cmd/moonbase — boundary, logs the error
func runMission(task string) {
    if err := pipeline.Run(ctx); err != nil {
        slog.Error("mission failed", "task", task, "error", err)
        os.Exit(1)
    }
}
```

## Error Types for Rich Context

When callers need structured error information beyond a sentinel:

```go
type ValidationError struct {
    Field   string
    Message string
}

func (e *ValidationError) Error() string {
    return fmt.Sprintf("validation: %s: %s", e.Field, e.Message)
}
```

Use `errors.As` to extract typed errors in callers.
