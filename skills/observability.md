---
name: observability
description: Observability practices — structured logging at boundaries, metrics and traces, context propagation, and never logging secrets or PII.
---

# Observability

## Structured Logging

Use structured logging (key-value pairs) instead of string interpolation:

```go
// Bad — unstructured, hard to parse
log.Printf("failed to load agent %s: %v", name, err)

// Good — structured, queryable, machine-parseable
slog.Error("loading agent failed",
    "agent", name,
    "error", err,
    "duration_ms", elapsed.Milliseconds(),
)
```

Use `log/slog` (Go 1.21+) with JSON handler for production:
```go
handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
    Level: slog.LevelInfo,
})
slog.SetDefault(slog.New(handler))
```

## Log at Boundaries Only

- HTTP/CLI handlers: log request start, completion, and errors.
- Pipeline phase transitions: log phase entry and exit with duration.
- Library/internal code: return errors, never log directly.
- One log event per boundary crossing — avoid duplicates up the stack.

## Log Levels

| Level | Use For |
|-------|---------|
| ERROR | Operation failed, requires attention |
| WARN | Degraded but recoverable (retry succeeded, fallback used) |
| INFO | Significant state changes (started, completed, deployed) |
| DEBUG | Diagnostic detail (disabled in production by default) |

## Never Log Secrets or PII

- Tokens, API keys, passwords: never include in log output.
- Emails, names, IPs: redact or hash if logging is necessary.
- Request bodies: log only safe fields, never raw payloads with credentials.
- Use allowlists for what CAN be logged, not denylists.

```go
// Bad — leaks the token
slog.Info("authenticated", "token", token)

// Good — log only non-sensitive identifiers
slog.Info("authenticated", "user_id", userID, "token_prefix", token[:8])
```

## Context Propagation

- Pass `request_id` / `trace_id` through the entire call chain.
- Include it in every log entry for correlated debugging.
- Use `context.Context` to carry request-scoped values.

## Metrics

Key metrics for any service:
- RED method: Rate (requests/sec), Errors (error rate), Duration (latency p50/p95/p99).
- USE method for resources: Utilization, Saturation, Errors.
- Instrument at the handler level — not deep inside business logic.
- Use histograms for latency, counters for throughput and errors.

## Alerting Principles

- Alert on symptoms (high error rate), not causes (disk full).
- Every alert must be actionable — if you cannot act, it is noise.
- Include runbook links in alert annotations.
- Page on user-facing impact; ticket on internal degradation.
