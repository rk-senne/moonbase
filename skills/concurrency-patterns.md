---
name: concurrency-patterns
description: Go concurrency patterns — goroutine lifecycle, context cancellation, mutex vs channel selection, and data race prevention.
---

# Concurrency Patterns

## Goroutine Lifecycle

Every goroutine must have a clear shutdown path:

```go
func (s *Server) Start(ctx context.Context) {
    go func() {
        ticker := time.NewTicker(30 * time.Second)
        defer ticker.Stop()
        for {
            select {
            case <-ctx.Done():
                return // clean exit
            case <-ticker.C:
                s.cleanup()
            }
        }
    }()
}
```

Rules:
- Never launch a goroutine without a way to stop it.
- Pass `context.Context` for cancellation — not bare channels.
- `defer cancel()` immediately after `context.WithCancel`/`WithTimeout`.
- Document goroutine-safety: `// Safe for concurrent use.`

## Context Cancellation

```go
ctx, cancel := context.WithTimeout(parentCtx, 5*time.Second)
defer cancel() // always defer immediately

result, err := longOperation(ctx)
if errors.Is(err, context.DeadlineExceeded) {
    // handle timeout specifically
}
```

- Propagate context through the call chain.
- Check `ctx.Err()` before expensive operations in loops.
- Cancel as soon as the result is no longer needed (prevents resource leaks).

## Mutex vs Channel

Use mutexes for protecting shared state:
```go
type Registry struct {
    mu    sync.RWMutex
    items map[string]*Item
}

func (r *Registry) Get(key string) *Item {
    r.mu.RLock()
    defer r.mu.RUnlock()
    return r.items[key]
}
```

Use channels for coordination and communication:
```go
func fanOut(ctx context.Context, jobs []Job, workers int) <-chan Result {
    results := make(chan Result, len(jobs))
    sem := make(chan struct{}, workers)
    var wg sync.WaitGroup
    for _, job := range jobs {
        wg.Add(1)
        go func(j Job) {
            defer wg.Done()
            sem <- struct{}{}
            defer func() { <-sem }()
            results <- process(ctx, j)
        }(job)
    }
    go func() { wg.Wait(); close(results) }()
    return results
}
```

Decision guide:
- Protecting a map/slice/struct field: `sync.Mutex` or `sync.RWMutex`.
- Signaling between goroutines: channels.
- One-time initialization: `sync.Once`.
- Waiting for N goroutines: `sync.WaitGroup`.

## Avoiding Data Races

Common race sources and fixes:

| Race Source | Fix |
|-------------|-----|
| Shared map without lock | `sync.RWMutex` around reads/writes |
| Loop variable captured in goroutine | Pass as function parameter |
| Slice append from multiple goroutines | Mutex or pre-allocate with index |
| Struct field written by one, read by many | `sync.RWMutex` or atomic |

## Bounded Concurrency

Limit concurrent work with a semaphore pattern:

```go
sem := make(chan struct{}, maxConcurrency)
var wg sync.WaitGroup
for _, task := range tasks {
    wg.Add(1)
    sem <- struct{}{} // block if at capacity
    go func(t Task) {
        defer wg.Done()
        defer func() { <-sem }()
        execute(t)
    }(task)
}
wg.Wait()
```
